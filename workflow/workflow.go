package workflow

import (
	"context"

	"github.com/spetersoncode/gains/store"
)

// Workflow is the top-level orchestrator that wraps a root step.
// It provides the primary entry point for workflow execution.
type Workflow struct {
	name string
	root Step
}

// New creates a new workflow with a root step.
func New(name string, root Step) *Workflow {
	return &Workflow{name: name, root: root}
}

// Name returns the workflow name.
func (w *Workflow) Name() string { return w.name }

// Run executes the workflow synchronously.
func (w *Workflow) Run(ctx context.Context, state *store.Store, opts ...Option) (*Result, error) {
	if state == nil {
		state = store.New(nil)
	}

	stepResult, err := w.root.Run(ctx, state, opts...)
	if err != nil {
		termination := TerminationError
		if ctx.Err() == context.Canceled {
			termination = TerminationCancelled
		} else if ctx.Err() == context.DeadlineExceeded {
			termination = TerminationTimeout
		}
		return &Result{
			WorkflowName: w.name,
			State:        state,
			Error:        err,
			Termination:  termination,
		}, err
	}

	return &Result{
		WorkflowName: w.name,
		State:        state,
		Output:       stepResult.Output,
		Usage:        stepResult.Usage,
		Termination:  TerminationComplete,
	}, nil
}

// RunStream executes the workflow and returns an event channel.
func (w *Workflow) RunStream(ctx context.Context, state *store.Store, opts ...Option) <-chan Event {
	if state == nil {
		state = store.New(nil)
	}

	return w.root.RunStream(ctx, state, opts...)
}
