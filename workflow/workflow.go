package workflow

import (
	"context"

	"github.com/spetersoncode/gains/event"
)

// Workflow is the top-level orchestrator that wraps a root step.
// It provides the primary entry point for workflow execution.
type Workflow[S any] struct {
	name string
	root Step[S]
}

// New creates a new workflow with a root step.
func New[S any](name string, root Step[S]) *Workflow[S] {
	return &Workflow[S]{name: name, root: root}
}

// Name returns the workflow name.
func (w *Workflow[S]) Name() string { return w.name }

// Run executes the workflow synchronously.
// State is mutated in place - access results via state fields after completion.
// The state parameter must not be nil.
func (w *Workflow[S]) Run(ctx context.Context, state *S, opts ...Option) (*Result[S], error) {
	// Get observer from options if set
	options := ApplyOptions(opts...)
	obs := options.Observer

	// Use RunStream to get events
	eventCh := w.RunStream(ctx, state, opts...)

	var lastErr error
	for ev := range eventCh {
		// Forward event to observer if configured
		if obs != nil {
			obs.Observe(ctx, ev)
		}

		// Track errors
		if ev.Type == event.RunError {
			lastErr = ev.Error
		}
	}

	if lastErr != nil {
		termination := TerminationError
		if ctx.Err() == context.Canceled {
			termination = TerminationCancelled
		} else if ctx.Err() == context.DeadlineExceeded {
			termination = TerminationTimeout
		}
		return &Result[S]{
			WorkflowName: w.name,
			State:        state,
			Error:        lastErr,
			Termination:  termination,
		}, lastErr
	}

	return &Result[S]{
		WorkflowName: w.name,
		State:        state,
		Termination:  TerminationComplete,
	}, nil
}

// RunStream executes the workflow and returns an event channel.
// State is mutated in place during streaming.
// The state parameter must not be nil.
func (w *Workflow[S]) RunStream(ctx context.Context, state *S, opts ...Option) <-chan Event {
	return w.root.RunStream(ctx, state, opts...)
}
