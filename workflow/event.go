package workflow

import (
	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
)

// Event is an alias to the unified event type.
// Workflow events use these event.Type values:
//   - event.RunStart, event.RunEnd, event.RunError
//   - event.StepStart, event.StepEnd, event.StepSkipped
//   - event.MessageStart, event.MessageDelta, event.MessageEnd
//   - event.ToolCallStart, event.ToolCallArgs, event.ToolCallEnd, event.ToolCallResult
//   - event.ParallelStart, event.ParallelEnd
//   - event.RouteSelected, event.LoopIteration
type Event = event.Event

// StepResult contains the output of a step execution.
type StepResult struct {
	// StepName identifies which step produced this result.
	StepName string

	// Output is the primary output value (optional).
	Output any

	// Response contains the LLM response if the step used one.
	Response *ai.Response

	// Usage aggregates token usage if applicable.
	Usage ai.Usage

	// Metadata holds step-specific metadata.
	Metadata map[string]any
}

// TerminationReason indicates why the workflow stopped.
type TerminationReason string

const (
	// TerminationComplete indicates normal completion.
	TerminationComplete TerminationReason = "complete"

	// TerminationTimeout indicates the deadline was exceeded.
	TerminationTimeout TerminationReason = "timeout"

	// TerminationCancelled indicates context cancellation.
	TerminationCancelled TerminationReason = "cancelled"

	// TerminationError indicates an error occurred.
	TerminationError TerminationReason = "error"
)

// Result represents the final outcome of workflow execution.
type Result struct {
	// WorkflowName identifies the workflow.
	WorkflowName string

	// State contains the final state after execution.
	State *State

	// Output is the primary output from the workflow.
	Output any

	// Usage aggregates token usage across all steps.
	Usage ai.Usage

	// Termination indicates why execution stopped.
	Termination TerminationReason

	// Error contains any error that caused termination.
	Error error
}
