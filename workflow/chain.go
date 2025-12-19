package workflow

import (
	"context"

	"github.com/spetersoncode/gains/event"
)

// Chain executes steps sequentially, passing state between them.
type Chain[S any] struct {
	name  string
	steps []Step[S]
}

// NewChain creates a sequential workflow.
func NewChain[S any](name string, steps ...Step[S]) *Chain[S] {
	return &Chain[S]{name: name, steps: steps}
}

// Name returns the chain name.
func (c *Chain[S]) Name() string { return c.name }

// Run executes steps sequentially.
func (c *Chain[S]) Run(ctx context.Context, state *S, opts ...Option) error {
	options := ApplyOptions(opts...)

	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	for _, step := range c.steps {
		if err := ctx.Err(); err != nil {
			return &StepError{StepName: step.Name(), Err: err}
		}

		stepCtx := ctx
		if options.StepTimeout > 0 {
			var cancel context.CancelFunc
			stepCtx, cancel = context.WithTimeout(ctx, options.StepTimeout)
			defer cancel()
		}

		err := step.Run(stepCtx, state, opts...)
		if err != nil {
			if options.ErrorHandler != nil {
				if handlerErr := options.ErrorHandler(ctx, step.Name(), err); handlerErr != nil {
					return &StepError{StepName: step.Name(), Err: handlerErr}
				}
				if options.ContinueOnError {
					continue
				}
			}
			return &StepError{StepName: step.Name(), Err: err}
		}
	}

	return nil
}

// RunStream executes steps sequentially and emits events.
func (c *Chain[S]) RunStream(ctx context.Context, state *S, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		options := ApplyOptions(opts...)

		if options.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, options.Timeout)
			defer cancel()
		}

		event.Emit(ch, Event{Type: event.RunStart, StepName: c.name})

		for _, step := range c.steps {
			if err := ctx.Err(); err != nil {
				event.Emit(ch, Event{Type: event.RunError, StepName: step.Name(), Error: err})
				return
			}

			stepCtx := ctx
			if options.StepTimeout > 0 {
				var cancel context.CancelFunc
				stepCtx, cancel = context.WithTimeout(ctx, options.StepTimeout)
				defer cancel()
			}

			// Forward events from step
			stepEvents := step.RunStream(stepCtx, state, opts...)
			var stepError error

			for ev := range stepEvents {
				ch <- ev
				if ev.Type == event.RunError {
					stepError = ev.Error
				}
			}

			if stepError != nil {
				if options.ErrorHandler != nil {
					if handlerErr := options.ErrorHandler(ctx, step.Name(), stepError); handlerErr == nil && options.ContinueOnError {
						continue
					}
				}
				return
			}
		}

		event.Emit(ch, Event{
			Type:     event.RunEnd,
			StepName: c.name,
		})
	}()

	return ch
}
