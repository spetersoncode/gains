package workflow

import (
	"context"
	"reflect"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
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

// NewLoopUntil creates a loop that exits when state[key] equals the target value.
// This is a convenience wrapper for common "exit when key equals X" patterns.
func NewLoopUntil(name string, step Step, key string, value any, opts ...LoopOption) *Loop {
	return NewLoop(name, step, func(ctx context.Context, state *State) bool {
		v, ok := state.Get(key)
		return ok && v == value
	}, opts...)
}

// NewLoopWhile creates a loop that continues while state[key] equals the target value.
// The loop exits when the key no longer equals the value (or is unset).
func NewLoopWhile(name string, step Step, key string, value any, opts ...LoopOption) *Loop {
	return NewLoop(name, step, func(ctx context.Context, state *State) bool {
		v, ok := state.Get(key)
		return !ok || v != value
	}, opts...)
}

// NewLoopUntilSet creates a loop that exits when state[key] is "truthy".
// A value is truthy if it exists and is non-nil, non-zero, non-empty.
func NewLoopUntilSet(name string, step Step, key string, opts ...LoopOption) *Loop {
	return NewLoop(name, step, func(ctx context.Context, state *State) bool {
		return isTruthy(state, key)
	}, opts...)
}

// isTruthy checks if a state key has a "truthy" value.
func isTruthy(state *State, key string) bool {
	v, ok := state.Get(key)
	if !ok || v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != ""
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Map, reflect.Array:
			return rv.Len() > 0
		default:
			return true // non-nil, exists = truthy
		}
	}
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

		event.Emit(ch, Event{Type: event.RunStart, StepName: l.name})

		var totalUsage ai.Usage

		for i := 1; i <= l.maxIters; i++ {
			state.Set(l.name+"_iteration", i)

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
			var stepResult *StepResult
			var stepError error

			for ev := range stepEvents {
				ch <- ev

				if ev.Type == event.StepEnd && ev.Response != nil {
					stepResult = &StepResult{
						StepName: l.step.Name(),
						Response: ev.Response,
						Usage:    ev.Response.Usage,
					}
				}
				if ev.Type == event.RunError {
					stepError = ev.Error
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

// IterationKey returns a typed key for the current iteration count.
// The key name follows the pattern "{loopName}_iteration".
func (l *Loop) IterationKey() Key[int] {
	return NewKey[int](l.name + "_iteration")
}

// NewLoopUntilKey creates a loop that exits when the typed key equals the target value.
// This provides compile-time type safety compared to NewLoopUntil.
func NewLoopUntilKey[T comparable](name string, step Step, key Key[T], value T, opts ...LoopOption) *Loop {
	return NewLoop(name, step, func(ctx context.Context, state *State) bool {
		v, ok := Get(state, key)
		return ok && v == value
	}, opts...)
}

// NewLoopWhileKey creates a loop that continues while the typed key equals the target value.
// The loop exits when the key no longer equals the value (or is unset).
func NewLoopWhileKey[T comparable](name string, step Step, key Key[T], value T, opts ...LoopOption) *Loop {
	return NewLoop(name, step, func(ctx context.Context, state *State) bool {
		v, ok := Get(state, key)
		return !ok || v != value
	}, opts...)
}

// NewLoopUntilKeySet creates a loop that exits when the typed key has a truthy value.
func NewLoopUntilKeySet[T any](name string, step Step, key Key[T], opts ...LoopOption) *Loop {
	return NewLoop(name, step, func(ctx context.Context, state *State) bool {
		return isTruthy(state, key.Name())
	}, opts...)
}
