package workflow

import (
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
// State contains all output from the workflow - access results via state fields.
type Result[S any] struct {
	// WorkflowName identifies the workflow.
	WorkflowName string

	// State contains the final state after execution.
	// All step outputs are stored in state fields via setters.
	State *S

	// Termination indicates why execution stopped.
	Termination TerminationReason

	// Error contains any error that caused termination.
	Error error
}
