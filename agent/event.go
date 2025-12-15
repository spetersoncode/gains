package agent

import (
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/internal/store"
)

// EventType identifies the kind of event occurring during agent execution.
type EventType string

const (
	// EventStepStart fires at the beginning of each agent iteration.
	EventStepStart EventType = "step_start"

	// EventStreamDelta fires for each streaming token during model response.
	EventStreamDelta EventType = "stream_delta"

	// EventToolCallRequested fires when the model requests a tool call.
	EventToolCallRequested EventType = "tool_call_requested"

	// EventToolCallApproved fires when a tool call is approved (human-in-the-loop).
	EventToolCallApproved EventType = "tool_call_approved"

	// EventToolCallRejected fires when a tool call is rejected by the approver.
	EventToolCallRejected EventType = "tool_call_rejected"

	// EventToolCallStarted fires before executing a tool handler.
	EventToolCallStarted EventType = "tool_call_started"

	// EventToolResult fires after a tool handler completes.
	EventToolResult EventType = "tool_result"

	// EventStepComplete fires at the end of each agent iteration.
	EventStepComplete EventType = "step_complete"

	// EventAgentComplete fires when the agent finishes execution.
	EventAgentComplete EventType = "agent_complete"

	// EventError fires when an error occurs during execution.
	EventError EventType = "error"
)

// Event represents an observable occurrence during agent execution.
type Event struct {
	// Type identifies the kind of event.
	Type EventType

	// Step is the current iteration number (1-indexed).
	Step int

	// Delta contains streaming content for EventStreamDelta events.
	Delta string

	// ToolCall contains the tool call for tool-related events.
	ToolCall *ai.ToolCall

	// ToolResult contains the result for EventToolResult events.
	ToolResult *ai.ToolResult

	// Response contains the model response for EventStepComplete and EventAgentComplete.
	Response *ai.Response

	// Error contains the error for EventError events.
	Error error

	// Message contains additional context (e.g., rejection reason).
	Message string

	// Timestamp is when the event occurred.
	Timestamp time.Time
}

// TerminationReason indicates why the agent stopped execution.
type TerminationReason string

const (
	// TerminationComplete indicates normal completion (no more tool calls).
	TerminationComplete TerminationReason = "complete"

	// TerminationMaxSteps indicates the step limit was reached.
	TerminationMaxSteps TerminationReason = "max_steps"

	// TerminationTimeout indicates the context deadline was exceeded.
	TerminationTimeout TerminationReason = "timeout"

	// TerminationCustom indicates a custom stop predicate returned true.
	TerminationCustom TerminationReason = "custom"

	// TerminationRejected indicates all tool calls were rejected.
	TerminationRejected TerminationReason = "rejected"

	// TerminationError indicates an unrecoverable error occurred.
	TerminationError TerminationReason = "error"

	// TerminationCancelled indicates context cancellation.
	TerminationCancelled TerminationReason = "cancelled"
)

// Result represents the final outcome of an agent execution.
type Result struct {
	// Response is the final response from the model.
	Response *ai.Response

	// history contains the complete conversation history (private).
	history *store.MessageStore

	// Steps is the number of iterations completed.
	Steps int

	// Termination indicates why execution stopped.
	Termination TerminationReason

	// TotalUsage aggregates token usage across all steps.
	TotalUsage ai.Usage

	// Error contains any error that caused termination (if applicable).
	Error error
}

// Messages returns the conversation history as a slice.
func (r *Result) Messages() []ai.Message {
	if r.history == nil {
		return nil
	}
	return r.history.Messages()
}

// MessageCount returns the number of messages in the conversation history.
func (r *Result) MessageCount() int {
	if r.history == nil {
		return 0
	}
	return r.history.Len()
}

// LastMessages returns the last n messages from the conversation history.
// If n exceeds the total message count, all messages are returned.
func (r *Result) LastMessages(n int) []ai.Message {
	if r.history == nil {
		return nil
	}
	msgs := r.history.Messages()
	if n >= len(msgs) {
		return msgs
	}
	return msgs[len(msgs)-n:]
}
