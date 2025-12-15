package workflow

import (
	"context"

	"github.com/spetersoncode/gains"
)

// Chain executes steps sequentially, passing state between them.
type Chain struct {
	name  string
	steps []Step
}

// NewChain creates a sequential workflow.
func NewChain(name string, steps ...Step) *Chain {
	return &Chain{name: name, steps: steps}
}

// Name returns the chain name.
func (c *Chain) Name() string { return c.name }

// Run executes steps sequentially.
func (c *Chain) Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error) {
	options := ApplyOptions(opts...)

	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	var totalUsage gains.Usage

	for _, step := range c.steps {
		if err := ctx.Err(); err != nil {
			return nil, &StepError{StepName: step.Name(), Err: err}
		}

		stepCtx := ctx
		if options.StepTimeout > 0 {
			var cancel context.CancelFunc
			stepCtx, cancel = context.WithTimeout(ctx, options.StepTimeout)
			defer cancel()
		}

		result, err := step.Run(stepCtx, state, opts...)
		if err != nil {
			if options.ErrorHandler != nil {
				if handlerErr := options.ErrorHandler(ctx, step.Name(), err); handlerErr != nil {
					return nil, &StepError{StepName: step.Name(), Err: handlerErr}
				}
				if options.ContinueOnError {
					continue
				}
			}
			return nil, &StepError{StepName: step.Name(), Err: err}
		}

		if options.OnStepComplete != nil {
			options.OnStepComplete(ctx, result)
		}

		totalUsage.InputTokens += result.Usage.InputTokens
		totalUsage.OutputTokens += result.Usage.OutputTokens
	}

	return &StepResult{
		StepName: c.name,
		Usage:    totalUsage,
	}, nil
}

// RunStream executes steps sequentially and emits events.
func (c *Chain) RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		options := ApplyOptions(opts...)

		if options.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, options.Timeout)
			defer cancel()
		}

		emit(ch, Event{Type: EventWorkflowStart, StepName: c.name})

		var totalUsage gains.Usage

		for _, step := range c.steps {
			if err := ctx.Err(); err != nil {
				emit(ch, Event{Type: EventError, StepName: step.Name(), Error: err})
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
					if handlerErr := options.ErrorHandler(ctx, step.Name(), stepError); handlerErr == nil && options.ContinueOnError {
						continue
					}
				}
				return
			}

			if stepResult != nil {
				totalUsage.InputTokens += stepResult.Usage.InputTokens
				totalUsage.OutputTokens += stepResult.Usage.OutputTokens
			}
		}

		emit(ch, Event{
			Type:     EventWorkflowComplete,
			StepName: c.name,
			Result: &StepResult{
				StepName: c.name,
				Usage:    totalUsage,
			},
		})
	}()

	return ch
}
