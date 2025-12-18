package agent

import (
	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/internal/store"
)

// Event is an alias to the unified event type.
// Agent events use these event.Type values:
//   - event.RunStart, event.RunEnd, event.RunError
//   - event.StepStart, event.StepEnd
//   - event.MessageStart, event.MessageDelta, event.MessageEnd
//   - event.ToolCallStart, event.ToolCallArgs, event.ToolCallEnd, event.ToolCallResult
//   - event.ToolCallApproved, event.ToolCallRejected, event.ToolCallExecuting
type Event = event.Event

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

	// TerminationClientToolCall indicates the model called a client-side tool.
	// The frontend should execute the tool and resume with the result.
	TerminationClientToolCall TerminationReason = "client_tool_call"
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

	// PendingClientToolCalls contains tool calls awaiting client execution.
	// These are set when Termination is TerminationClientToolCall.
	PendingClientToolCalls []ai.ToolCall
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
