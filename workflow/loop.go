package workflow

import (
	"context"

	ai "github.com/spetersoncode/gains"
)

// LoopCondition evaluates state to determine if the loop should exit.
// Return true to exit the loop, false to continue iterating.
type LoopCondition func(ctx context.Context, state *State) bool

// LoopOption configures a Loop.
type LoopOption func(*Loop)

// WithMaxIterations sets the maximum number of loop iterations.
// Default is 10.
func WithMaxIterations(n int) LoopOption {
	return func(l *Loop) {
		l.maxIters = n
	}
}

// Loop repeatedly executes a step until a condition returns true.
// Use for iterative refinement workflows where steps need to repeat
// based on evaluation results stored in state.
type Loop struct {
	name      string
	step      Step
	condition LoopCondition
	maxIters  int
}

// NewLoop creates a loop that executes step until condition returns true.
// The condition is checked after each iteration. If the condition never
// returns true, the loop terminates after maxIterations (default 10)
// and returns ErrMaxIterationsExceeded.
//
// The loop stores the current iteration count in state under "{name}_iteration".
func NewLoop(name string, step Step, condition LoopCondition, opts ...LoopOption) *Loop {
	l := &Loop{
		name:      name,
		step:      step,
		condition: condition,
		maxIters:  10,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Name returns the loop name.
func (l *Loop) Name() string { return l.name }

// Run executes the step repeatedly until the condition returns true.
func (l *Loop) Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error) {
	options := ApplyOptions(opts...)

	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	var totalUsage ai.Usage

	for i := 1; i <= l.maxIters; i++ {
		state.Set(l.name+"_iteration", i)

		if err := ctx.Err(); err != nil {
			return nil, &StepError{StepName: l.name, Err: err}
		}

		stepCtx := ctx
		if options.StepTimeout > 0 {
			var cancel context.CancelFunc
			stepCtx, cancel = context.WithTimeout(ctx, options.StepTimeout)
			defer cancel()
		}

		result, err := l.step.Run(stepCtx, state, opts...)
		if err != nil {
			if options.ErrorHandler != nil {
				if handlerErr := options.ErrorHandler(ctx, l.step.Name(), err); handlerErr != nil {
					return nil, &StepError{StepName: l.name, Err: handlerErr}
				}
				if options.ContinueOnError {
					continue
				}
			}
			return nil, &StepError{StepName: l.name, Err: err}
		}

		if options.OnStepComplete != nil {
			options.OnStepComplete(ctx, result)
		}

		totalUsage.InputTokens += result.Usage.InputTokens
		totalUsage.OutputTokens += result.Usage.OutputTokens

		if l.condition(ctx, state) {
			return &StepResult{StepName: l.name, Usage: totalUsage}, nil
		}
	}

	return nil, ErrMaxIterationsExceeded
}

// RunStream executes the step repeatedly and emits events.
func (l *Loop) RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		options := ApplyOptions(opts...)

		if options.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, options.Timeout)
			defer cancel()
		}

		emit(ch, Event{Type: EventWorkflowStart, StepName: l.name})

		var totalUsage ai.Usage

		for i := 1; i <= l.maxIters; i++ {
			state.Set(l.name+"_iteration", i)

			emit(ch, Event{Type: EventLoopIteration, StepName: l.name, Iteration: i})

			if err := ctx.Err(); err != nil {
				emit(ch, Event{Type: EventError, StepName: l.name, Error: err})
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
			var stepResult *StepResult
			var stepError error

			for event := range stepEvents {
				ch <- event

				if event.Type == EventStepComplete && event.Result != nil {
					stepResult = event.Result
				}
				if event.Type == EventError {
					stepError = event.Error
				}
			}

			if stepError != nil {
				if options.ErrorHandler != nil {
					if handlerErr := options.ErrorHandler(ctx, l.step.Name(), stepError); handlerErr == nil && options.ContinueOnError {
						continue
					}
				}
				return
			}

			if stepResult != nil {
				totalUsage.InputTokens += stepResult.Usage.InputTokens
				totalUsage.OutputTokens += stepResult.Usage.OutputTokens
			}

			if l.condition(ctx, state) {
				emit(ch, Event{
					Type:     EventWorkflowComplete,
					StepName: l.name,
					Result: &StepResult{
						StepName: l.name,
						Usage:    totalUsage,
					},
				})
				return
			}
		}

		// Max iterations exceeded
		emit(ch, Event{Type: EventError, StepName: l.name, Error: ErrMaxIterationsExceeded})
	}()

	return ch
}
