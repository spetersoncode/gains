package workflow

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrWorkflowTimeout indicates the workflow exceeded its timeout.
	ErrWorkflowTimeout = errors.New("workflow: timeout exceeded")

	// ErrWorkflowCancelled indicates the workflow was cancelled.
	ErrWorkflowCancelled = errors.New("workflow: cancelled")

	// ErrNoRouteMatched indicates no route condition was satisfied.
	ErrNoRouteMatched = errors.New("workflow: no route matched")

	// ErrStepNotFound indicates a referenced step does not exist.
	ErrStepNotFound = errors.New("workflow: step not found")
)

// StepError wraps errors from step execution.
type StepError struct {
	StepName string
	Err      error
}

// Error returns a formatted error message including the step name.
func (e *StepError) Error() string {
	return fmt.Sprintf("workflow: step %q failed: %v", e.StepName, e.Err)
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *StepError) Unwrap() error {
	return e.Err
}

// ParallelError wraps errors from parallel execution.
type ParallelError struct {
	Errors map[string]error
}

// Error returns a formatted message summarizing the parallel execution failures.
func (e *ParallelError) Error() string {
	if len(e.Errors) == 0 {
		return "workflow: parallel execution failed"
	}
	if len(e.Errors) == 1 {
		for name, err := range e.Errors {
			return fmt.Sprintf("workflow: parallel step %q failed: %v", name, err)
		}
	}
	var names []string
	for name := range e.Errors {
		names = append(names, name)
	}
	return fmt.Sprintf("workflow: parallel execution failed with %d errors in steps: %s",
		len(e.Errors), strings.Join(names, ", "))
}

// Unwrap returns the first error for errors.Is/As compatibility.
func (e *ParallelError) Unwrap() error {
	for _, err := range e.Errors {
		return err
	}
	return nil
}
