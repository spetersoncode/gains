// Package event provides a unified event system for streaming responses
// across client, agent, and workflow packages. The event types are designed
// for 1:1 mapping with the AG-UI protocol.
package event

import (
	"time"

	ai "github.com/spetersoncode/gains"
)

// Type identifies the kind of event.
type Type string

// Run lifecycle events
const (
	// RunStart fires when execution begins (agent run, workflow run, or chat stream).
	RunStart Type = "run_start"

	// RunEnd fires when execution completes successfully.
	RunEnd Type = "run_end"

	// RunError fires when an unrecoverable error occurs.
	RunError Type = "run_error"
)

// Step lifecycle events (agent/workflow only)
const (
	// StepStart fires when a step begins.
	StepStart Type = "step_start"

	// StepEnd fires when a step completes.
	StepEnd Type = "step_end"

	// StepSkipped fires when a step is skipped (e.g., routing).
	StepSkipped Type = "step_skipped"
)

// Message lifecycle events
const (
	// MessageStart fires when an assistant message begins.
	MessageStart Type = "message_start"

	// MessageDelta fires for each streaming token.
	MessageDelta Type = "message_delta"

	// MessageEnd fires when an assistant message completes.
	MessageEnd Type = "message_end"
)

// Tool call lifecycle events
const (
	// ToolCallStart fires when a tool call begins (contains tool name).
	ToolCallStart Type = "tool_call_start"

	// ToolCallArgs fires with tool call arguments.
	ToolCallArgs Type = "tool_call_args"

	// ToolCallEnd fires when tool call transmission is complete.
	ToolCallEnd Type = "tool_call_end"

	// ToolCallResult fires with the tool execution result.
	ToolCallResult Type = "tool_call_result"
)

// Tool approval events (agent only)
const (
	// ToolCallApproved fires when a tool call is approved (human-in-the-loop).
	ToolCallApproved Type = "tool_call_approved"

	// ToolCallRejected fires when a tool call is rejected.
	ToolCallRejected Type = "tool_call_rejected"

	// ToolCallExecuting fires before tool handler execution begins.
	ToolCallExecuting Type = "tool_call_executing"
)

// Workflow-specific events
const (
	// ParallelStart fires when parallel execution begins.
	ParallelStart Type = "parallel_start"

	// ParallelEnd fires when all parallel branches complete.
	ParallelEnd Type = "parallel_end"

	// RouteSelected fires when a route is chosen.
	RouteSelected Type = "route_selected"

	// LoopIteration fires at the start of each loop iteration.
	LoopIteration Type = "loop_iteration"
)

// Retry events
const (
	// RetryAttempt fires when a retry attempt starts.
	RetryAttempt Type = "retry_attempt"

	// RetryFailed fires when an attempt fails (may or may not retry).
	RetryFailed Type = "retry_failed"

	// RetryScheduled fires when a retry is scheduled after a failure.
	RetryScheduled Type = "retry_scheduled"

	// RetrySuccess fires when an attempt succeeds.
	RetrySuccess Type = "retry_success"

	// RetryExhausted fires when all retry attempts are exhausted.
	RetryExhausted Type = "retry_exhausted"
)

// State synchronization events (AG-UI shared state)
const (
	// StateSnapshot fires to send the complete state to the frontend.
	StateSnapshot Type = "state_snapshot"

	// StateDelta fires to send incremental state changes using JSON Patch.
	StateDelta Type = "state_delta"
)

// PatchOp represents a JSON Patch operation type (RFC 6902).
type PatchOp string

// JSON Patch operation types.
const (
	PatchAdd     PatchOp = "add"
	PatchRemove  PatchOp = "remove"
	PatchReplace PatchOp = "replace"
	PatchMove    PatchOp = "move"
	PatchCopy    PatchOp = "copy"
	PatchTest    PatchOp = "test"
)

// JSONPatch represents a JSON Patch operation (RFC 6902).
type JSONPatch struct {
	Op    PatchOp `json:"op"`              // Operation type
	Path  string  `json:"path"`            // JSON Pointer path
	Value any     `json:"value,omitempty"` // Value for add, replace, test
	From  string  `json:"from,omitempty"`  // Source path for move, copy
}

// Event represents an observable occurrence during streaming execution.
// This unified type is used by client, agent, and workflow packages.
type Event struct {
	// Type identifies the kind of event.
	Type Type

	// MessageID identifies the message for Start/Delta/End correlation.
	MessageID string

	// Delta contains streaming content for MessageDelta events.
	Delta string

	// Response contains the complete response for MessageEnd and RunEnd events.
	Response *ai.Response

	// ToolCall contains the tool call for tool-related events.
	ToolCall *ai.ToolCall

	// ToolResult contains the result for ToolCallResult events.
	ToolResult *ai.ToolResult

	// Step is the current iteration number (1-indexed) for agent events.
	Step int

	// StepName identifies the step for workflow events.
	StepName string

	// RouteName identifies the selected route for RouteSelected events.
	RouteName string

	// Iteration is the loop iteration (1-indexed) for LoopIteration events.
	Iteration int

	// Attempt is the retry attempt number (1-indexed) for retry events.
	Attempt int

	// Error contains the error for RunError events.
	Error error

	// Message contains additional context (e.g., rejection reason, termination reason).
	Message string

	// PendingToolCalls contains tool calls awaiting client-side execution.
	// Set on RunEnd events when termination is due to client tool calls.
	PendingToolCalls []ai.ToolCall

	// State contains the full state for StateSnapshot events.
	State any

	// StatePatches contains JSON Patch operations for StateDelta events.
	StatePatches []JSONPatch

	// Timestamp is when the event occurred.
	Timestamp time.Time
}

// emit sends an event with timestamp to the channel (non-blocking).
func Emit(ch chan<- Event, e Event) {
	e.Timestamp = time.Now()
	select {
	case ch <- e:
	default:
		// Channel full - don't block
	}
}

// NewChannel creates a buffered event channel with standard capacity.
func NewChannel() chan Event {
	return make(chan Event, 100)
}

// NewStateSnapshot creates a StateSnapshot event with the given state.
func NewStateSnapshot(state any) Event {
	return Event{
		Type:  StateSnapshot,
		State: state,
	}
}

// NewStateDelta creates a StateDelta event with the given patches.
func NewStateDelta(patches ...JSONPatch) Event {
	return Event{
		Type:         StateDelta,
		StatePatches: patches,
	}
}

// Replace creates a JSON Patch replace operation.
func Replace(path string, value any) JSONPatch {
	return JSONPatch{Op: PatchReplace, Path: path, Value: value}
}

// Add creates a JSON Patch add operation.
func Add(path string, value any) JSONPatch {
	return JSONPatch{Op: PatchAdd, Path: path, Value: value}
}

// Remove creates a JSON Patch remove operation.
func Remove(path string) JSONPatch {
	return JSONPatch{Op: PatchRemove, Path: path}
}

// Move creates a JSON Patch move operation.
func Move(from, path string) JSONPatch {
	return JSONPatch{Op: PatchMove, From: from, Path: path}
}

// Copy creates a JSON Patch copy operation.
func Copy(from, path string) JSONPatch {
	return JSONPatch{Op: PatchCopy, From: from, Path: path}
}

// Test creates a JSON Patch test operation.
func Test(path string, value any) JSONPatch {
	return JSONPatch{Op: PatchTest, Path: path, Value: value}
}
