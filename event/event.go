// Package event provides a unified event system for streaming responses
// across client, agent, and workflow packages. The event types are designed
// for 1:1 mapping with the AG-UI protocol.
package event

import (
	"context"
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

	// MessagesSnapshot fires to send the complete message history to the frontend.
	MessagesSnapshot Type = "messages_snapshot"
)

// Activity events (AG-UI human-in-the-loop)
const (
	// ActivitySnapshot fires to send the complete activity state to the frontend.
	// Used for tool approval UI, loading states, and other transient activities.
	ActivitySnapshot Type = "activity_snapshot"

	// ActivityDelta fires to send incremental activity state changes.
	ActivityDelta Type = "activity_delta"
)

// ActivityType categorizes activities for AG-UI ACTIVITY events.
type ActivityType string

// Activity type constants for human-in-the-loop interactions.
const (
	// ActivityToolApproval indicates a tool call awaiting user approval.
	ActivityToolApproval ActivityType = "tool_approval"

	// ActivityLoading indicates a loading/processing state.
	ActivityLoading ActivityType = "loading"

	// ActivityUserInput indicates a request for user input.
	ActivityUserInput ActivityType = "user_input"
)

// ApprovalStatus represents the status of a tool approval request.
type ApprovalStatus string

// Approval status constants.
const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

// ToolApprovalActivity represents the state of a tool approval request.
// This is the content structure for ActivityToolApproval events.
type ToolApprovalActivity struct {
	ToolCallID string         `json:"toolCallId"`
	ToolName   string         `json:"toolName"`
	Arguments  string         `json:"arguments"`
	Status     ApprovalStatus `json:"status"`
	Reason     string         `json:"reason,omitempty"` // Reason for rejection
}

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

	// Messages contains the complete message history for MessagesSnapshot events.
	Messages []ai.Message

	// Activity fields for ActivitySnapshot and ActivityDelta events.
	ActivityID      string       // Unique ID for activity correlation
	Activity        ActivityType // Type of activity (tool_approval, loading, etc.)
	ActivityContent any          // Content for the activity (e.g., ToolApprovalActivity)
	ActivityPatches []JSONPatch  // Patches for ActivityDelta events

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

// EmitSnapshot is a convenience function that sends a StateSnapshot event.
// Use this in tool handlers to sync state with the frontend:
//
//	func handleTool(ctx context.Context, args Args) (string, error) {
//	    state.Progress = 50
//	    event.EmitSnapshot(eventCh, state)
//	    return "done", nil
//	}
func EmitSnapshot(ch chan<- Event, state any) {
	Emit(ch, NewStateSnapshot(state))
}

// EmitDelta is a convenience function that sends a StateDelta event.
// Use this in tool handlers for efficient incremental state updates:
//
//	event.EmitDelta(eventCh,
//	    event.Replace("/progress", 50),
//	    event.Add("/items/-", "new item"),
//	)
func EmitDelta(ch chan<- Event, patches ...JSONPatch) {
	Emit(ch, NewStateDelta(patches...))
}

// EmitField is a convenience function that sends a StateDelta event
// for a single field update. Use this for simple field changes:
//
//	event.EmitField(eventCh, "/progress", 75)
func EmitField(ch chan<- Event, path string, value any) {
	Emit(ch, NewStateDelta(Replace(path, value)))
}

// NewMessagesSnapshot creates a MessagesSnapshot event with the given messages.
func NewMessagesSnapshot(messages []ai.Message) Event {
	return Event{
		Type:     MessagesSnapshot,
		Messages: messages,
	}
}

// EmitMessagesSnapshot is a convenience function that sends a MessagesSnapshot event.
// Use this to sync the complete conversation history with the frontend:
//
//	event.EmitMessagesSnapshot(eventCh, conversation.Messages())
func EmitMessagesSnapshot(ch chan<- Event, messages []ai.Message) {
	Emit(ch, NewMessagesSnapshot(messages))
}

// NewActivitySnapshot creates an ActivitySnapshot event.
// The activityID should be unique for each activity instance and is used
// for correlation between snapshot and delta events.
func NewActivitySnapshot(activityID string, activityType ActivityType, content any) Event {
	return Event{
		Type:            ActivitySnapshot,
		ActivityID:      activityID,
		Activity:        activityType,
		ActivityContent: content,
	}
}

// NewActivityDelta creates an ActivityDelta event.
// Use the same activityID as the corresponding ActivitySnapshot to update it.
func NewActivityDelta(activityID string, activityType ActivityType, patches ...JSONPatch) Event {
	return Event{
		Type:            ActivityDelta,
		ActivityID:      activityID,
		Activity:        activityType,
		ActivityPatches: patches,
	}
}

// NewToolApprovalPending creates an ActivitySnapshot event for a pending tool approval.
// The frontend will display this as a tool approval request with approve/reject buttons.
func NewToolApprovalPending(toolCallID, toolName, arguments string) Event {
	return NewActivitySnapshot(toolCallID, ActivityToolApproval, ToolApprovalActivity{
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Arguments:  arguments,
		Status:     ApprovalPending,
	})
}

// NewToolApprovalApproved creates an ActivityDelta event to mark a tool as approved.
func NewToolApprovalApproved(toolCallID string) Event {
	return NewActivityDelta(toolCallID, ActivityToolApproval,
		Replace("/status", string(ApprovalApproved)),
	)
}

// NewToolApprovalRejected creates an ActivityDelta event to mark a tool as rejected.
func NewToolApprovalRejected(toolCallID, reason string) Event {
	return NewActivityDelta(toolCallID, ActivityToolApproval,
		Replace("/status", string(ApprovalRejected)),
		Replace("/reason", reason),
	)
}

// EmitToolApprovalPending emits a tool approval pending activity.
func EmitToolApprovalPending(ch chan<- Event, toolCallID, toolName, arguments string) {
	Emit(ch, NewToolApprovalPending(toolCallID, toolName, arguments))
}

// EmitToolApprovalApproved emits a tool approval approved activity update.
func EmitToolApprovalApproved(ch chan<- Event, toolCallID string) {
	Emit(ch, NewToolApprovalApproved(toolCallID))
}

// EmitToolApprovalRejected emits a tool approval rejected activity update.
func EmitToolApprovalRejected(ch chan<- Event, toolCallID, reason string) {
	Emit(ch, NewToolApprovalRejected(toolCallID, reason))
}

// Context-based event forwarding for nested runs

// forwardChannelKey is the context key for event forwarding channels.
type forwardChannelKey struct{}

// ForwardChannel is the type of channel used for event forwarding.
type ForwardChannel = chan<- Event

// WithForwardChannel returns a new context with the given event channel for forwarding.
// Tool handlers can use ForwardChannelFromContext to retrieve this channel and
// forward sub-run events to the parent event stream.
func WithForwardChannel(ctx context.Context, ch chan<- Event) context.Context {
	return context.WithValue(ctx, forwardChannelKey{}, ch)
}

// ForwardChannelFromContext retrieves the event forwarding channel from the context.
// Returns nil if no channel is set.
func ForwardChannelFromContext(ctx context.Context) chan<- Event {
	ch, _ := ctx.Value(forwardChannelKey{}).(chan<- Event)
	return ch
}

// User input activity types and helpers

// UserInputActivity represents the state of a user input request.
// This is the content structure for ActivityUserInput events.
type UserInputActivity struct {
	RequestID   string   `json:"requestId"`
	Type        string   `json:"type"`                  // "confirm", "text", "choice"
	Title       string   `json:"title,omitempty"`
	Message     string   `json:"message"`
	Choices     []string `json:"choices,omitempty"`
	Default     string   `json:"default,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Status      string   `json:"status"` // "pending", "responded", "cancelled", "timeout"
	Value       string   `json:"value,omitempty"`
	Confirmed   bool     `json:"confirmed,omitempty"`
}

// NewUserInputPending creates an ActivitySnapshot event for a pending user input request.
// The frontend will display this as an input dialog appropriate for the type.
func NewUserInputPending(requestID, inputType, title, message string, choices []string, defaultVal, placeholder string) Event {
	return NewActivitySnapshot(requestID, ActivityUserInput, UserInputActivity{
		RequestID:   requestID,
		Type:        inputType,
		Title:       title,
		Message:     message,
		Choices:     choices,
		Default:     defaultVal,
		Placeholder: placeholder,
		Status:      "pending",
	})
}

// NewUserInputResponded creates an ActivityDelta event to mark an input as responded.
func NewUserInputResponded(requestID, value string, confirmed bool) Event {
	return NewActivityDelta(requestID, ActivityUserInput,
		Replace("/status", "responded"),
		Replace("/value", value),
		Replace("/confirmed", confirmed),
	)
}

// NewUserInputCancelled creates an ActivityDelta event to mark an input as cancelled.
func NewUserInputCancelled(requestID string) Event {
	return NewActivityDelta(requestID, ActivityUserInput,
		Replace("/status", "cancelled"),
	)
}

// NewUserInputTimeout creates an ActivityDelta event to mark an input as timed out.
func NewUserInputTimeout(requestID string) Event {
	return NewActivityDelta(requestID, ActivityUserInput,
		Replace("/status", "timeout"),
	)
}

// EmitUserInputPending emits a user input pending activity.
func EmitUserInputPending(ch chan<- Event, requestID, inputType, title, message string, choices []string, defaultVal, placeholder string) {
	Emit(ch, NewUserInputPending(requestID, inputType, title, message, choices, defaultVal, placeholder))
}

// EmitUserInputResponded emits a user input responded activity update.
func EmitUserInputResponded(ch chan<- Event, requestID, value string, confirmed bool) {
	Emit(ch, NewUserInputResponded(requestID, value, confirmed))
}

// EmitUserInputCancelled emits a user input cancelled activity update.
func EmitUserInputCancelled(ch chan<- Event, requestID string) {
	Emit(ch, NewUserInputCancelled(requestID))
}

// EmitUserInputTimeout emits a user input timeout activity update.
func EmitUserInputTimeout(ch chan<- Event, requestID string) {
	Emit(ch, NewUserInputTimeout(requestID))
}
