package workflow

import (
	"context"
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
	err := w.root.Run(ctx, state, opts...)
	if err != nil {
		termination := TerminationError
		if ctx.Err() == context.Canceled {
			termination = TerminationCancelled
		} else if ctx.Err() == context.DeadlineExceeded {
			termination = TerminationTimeout
		}
		return &Result[S]{
			WorkflowName: w.name,
			State:        state,
			Error:        err,
			Termination:  termination,
		}, err
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
