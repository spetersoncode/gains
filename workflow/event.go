package workflow

import (
	"time"

	"github.com/spetersoncode/gains"
)

// EventType identifies the kind of event occurring during workflow execution.
type EventType string

const (
	// EventWorkflowStart fires when the workflow begins.
	EventWorkflowStart EventType = "workflow_start"

	// EventWorkflowComplete fires when the workflow finishes.
	EventWorkflowComplete EventType = "workflow_complete"

	// EventStepStart fires when a step begins execution.
	EventStepStart EventType = "step_start"

	// EventStepComplete fires when a step finishes successfully.
	EventStepComplete EventType = "step_complete"

	// EventStepSkipped fires when a step is skipped (e.g., routing).
	EventStepSkipped EventType = "step_skipped"

	// EventStreamDelta fires for streaming content from LLM.
	EventStreamDelta EventType = "stream_delta"

	// EventToolCall fires when a tool is called within a step.
	EventToolCall EventType = "tool_call"

	// EventParallelStart fires when parallel execution begins.
	EventParallelStart EventType = "parallel_start"

	// EventParallelComplete fires when all parallel branches complete.
	EventParallelComplete EventType = "parallel_complete"

	// EventRouteSelected fires when a route is chosen.
	EventRouteSelected EventType = "route_selected"

	// EventError fires when an error occurs.
	EventError EventType = "error"
)

// Event represents an observable occurrence during workflow execution.
type Event struct {
	// Type identifies the kind of event.
	Type EventType

	// StepName identifies the step that produced this event.
	StepName string

	// Delta contains streaming content for EventStreamDelta.
	Delta string

	// Result contains step result for EventStepComplete.
	Result *StepResult

	// ToolCall contains tool call info for EventToolCall.
	ToolCall *gains.ToolCall

	// ParallelResults contains results from parallel execution.
	ParallelResults map[string]*StepResult

	// RouteName identifies the selected route for EventRouteSelected.
	RouteName string

	// Error contains the error for EventError.
	Error error

	// Message contains additional context.
	Message string

	// Timestamp is when the event occurred.
	Timestamp time.Time
}

// StepResult contains the output of a step execution.
type StepResult struct {
	// StepName identifies which step produced this result.
	StepName string

	// Output is the primary output value (optional).
	Output any

	// Response contains the LLM response if the step used one.
	Response *gains.Response

	// Usage aggregates token usage if applicable.
	Usage gains.Usage

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
	Usage gains.Usage

	// Termination indicates why execution stopped.
	Termination TerminationReason

	// Error contains any error that caused termination.
	Error error
}

// emit sends an event with timestamp to the channel.
func emit(ch chan<- Event, event Event) {
	event.Timestamp = time.Now()
	select {
	case ch <- event:
	default:
		// Channel full - don't block
	}
}
