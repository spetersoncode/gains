package workflow

import (
	"context"

	"github.com/spetersoncode/gains/event"
)

// ExitCondition determines when the loop should stop.
// Return true to EXIT the loop, false to continue iterating.
// The iteration parameter is 1-indexed (first iteration is 1).
type ExitCondition[S any] func(ctx context.Context, state *S, iteration int) bool

// LoopOption configures a Loop.
type LoopOption func(*loopConfig)

type loopConfig struct {
	maxIters int
}

// WithMaxIterations sets the maximum number of loop iterations.
// Default is 10.
func WithMaxIterations(n int) LoopOption {
	return func(c *loopConfig) {
		c.maxIters = n
	}
}

// Loop repeatedly executes a step until a condition returns true.
// Use for iterative refinement workflows where steps need to repeat
// based on evaluation results stored in state.
type Loop[S any] struct {
	name          string
	step          Step[S]
	exitCondition ExitCondition[S]
	maxIters      int
}

// NewLoopWithExitCondition creates a loop with a custom exit condition.
// Use this for complex conditions that can't be expressed with the simple helpers.
//
// The exit condition receives the context, state, and current iteration (1-indexed).
// Return true to exit the loop, false to continue.
//
// Example:
//
//	loop := NewLoopWithExitCondition[MyState]("refine", step,
//	    func(ctx context.Context, s *MyState, iter int) bool {
//	        return iter >= 5 || s.Quality > 0.95
//	    },
//	)
func NewLoopWithExitCondition[S any](
	name string,
	step Step[S],
	exitCondition ExitCondition[S],
	opts ...LoopOption,
) *Loop[S] {
	cfg := &loopConfig{maxIters: 10}
	for _, opt := range opts {
		opt(cfg)
	}
	return &Loop[S]{
		name:          name,
		step:          step,
		exitCondition: exitCondition,
		maxIters:      cfg.maxIters,
	}
}

// NewLoopUntil creates a loop that exits when the predicate returns true.
// This is the recommended way to create loops for most use cases.
//
// Example:
//
//	loop := NewLoopUntil[MyState]("process", step,
//	    func(s *MyState) bool { return s.Done },
//	)
func NewLoopUntil[S any](
	name string,
	step Step[S],
	predicate func(*S) bool,
	opts ...LoopOption,
) *Loop[S] {
	return NewLoopWithExitCondition(name, step, func(_ context.Context, s *S, _ int) bool {
		return predicate(s)
	}, opts...)
}

// NewLoopWhile creates a loop that continues while the predicate returns true.
// Exits when the predicate returns false.
//
// Example:
//
//	loop := NewLoopWhile[MyState]("retry", step,
//	    func(s *MyState) bool { return s.NeedsRetry },
//	)
func NewLoopWhile[S any](
	name string,
	step Step[S],
	predicate func(*S) bool,
	opts ...LoopOption,
) *Loop[S] {
	return NewLoopWithExitCondition(name, step, func(_ context.Context, s *S, _ int) bool {
		return !predicate(s) // exit when predicate is false
	}, opts...)
}

// NewLoopN creates a loop that executes exactly n times.
//
// Example:
//
//	loop := NewLoopN[MyState]("retry", step, 3)  // retry 3 times
func NewLoopN[S any](name string, step Step[S], n int) *Loop[S] {
	return NewLoopWithExitCondition(name, step, func(_ context.Context, _ *S, iter int) bool {
		return iter >= n
	}, WithMaxIterations(n))
}

// Name returns the loop name.
func (l *Loop[S]) Name() string { return l.name }

// Run executes the step repeatedly until the exit condition returns true.
func (l *Loop[S]) Run(ctx context.Context, state *S, opts ...Option) error {
	options := ApplyOptions(opts...)

	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	for i := 1; i <= l.maxIters; i++ {
		if err := ctx.Err(); err != nil {
			return &StepError{StepName: l.name, Err: err}
		}

		stepCtx := ctx
		if options.StepTimeout > 0 {
			var cancel context.CancelFunc
			stepCtx, cancel = context.WithTimeout(ctx, options.StepTimeout)
			defer cancel()
		}

		err := l.step.Run(stepCtx, state, opts...)
		if err != nil {
			if options.ErrorHandler != nil {
				handlerErr := options.ErrorHandler(ctx, l.step.Name(), err)
				if handlerErr != nil {
					// Handler wants to propagate (possibly transformed) error
					return &StepError{StepName: l.name, Err: handlerErr}
				}
				// Handler suppressed the error (returned nil)
				if options.ContinueOnError {
					continue
				}
				// Error suppressed, stop successfully
				return nil
			}
			// No handler - propagate original error
			return &StepError{StepName: l.name, Err: err}
		}

		// Check exit condition after step execution
		if l.exitCondition(ctx, state, i) {
			return nil
		}
	}

	return ErrMaxIterationsExceeded
}

// RunStream executes the step repeatedly and emits events.
func (l *Loop[S]) RunStream(ctx context.Context, state *S, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		options := ApplyOptions(opts...)

		if options.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, options.Timeout)
			defer cancel()
		}

		event.Emit(ch, Event{Type: event.RunStart, StepName: l.name})

		for i := 1; i <= l.maxIters; i++ {
			event.Emit(ch, Event{Type: event.LoopIteration, StepName: l.name, Iteration: i})

			if err := ctx.Err(); err != nil {
				event.Emit(ch, Event{Type: event.RunError, StepName: l.name, Error: err})
				return
			}

			stepCtx := ctx
			if options.StepTimeout > 0 {
				var cancel context.CancelFunc
				stepCtx, cancel = context.WithTimeout(ctx, options.StepTimeout)
				defer cancel()
			}

			// Forward events from step
			stepEvents := l.step.RunStream(stepCtx, state, opts...)
			var stepError error

			for ev := range stepEvents {
				ch <- ev
				if ev.Type == event.RunError {
					stepError = ev.Error
				}
			}

			if stepError != nil {
				if options.ErrorHandler != nil {
					handlerErr := options.ErrorHandler(ctx, l.step.Name(), stepError)
					if handlerErr != nil {
						// Handler wants to propagate - emit the handler's error
						event.Emit(ch, Event{Type: event.RunError, StepName: l.name, Error: handlerErr})
						return
					}
					// Handler suppressed the error
					if options.ContinueOnError {
						continue
					}
					// Error suppressed, stop successfully
					event.Emit(ch, Event{Type: event.RunEnd, StepName: l.name})
					return
				}
				// No handler - error was already emitted by step, just stop
				return
			}

			// Check exit condition after step execution
			if l.exitCondition(ctx, state, i) {
				event.Emit(ch, Event{
					Type:     event.RunEnd,
					StepName: l.name,
				})
				return
			}
		}

		// Max iterations exceeded
		event.Emit(ch, Event{Type: event.RunError, StepName: l.name, Error: ErrMaxIterationsExceeded})
	}()

	return ch
}
