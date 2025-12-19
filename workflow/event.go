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
//   - event.StateSnapshot, event.StateDelta
type Event = event.Event

// StateEmitter allows workflow steps to emit state change notifications
// for AG-UI shared state synchronization. Steps can send full snapshots
// or incremental patches to keep the frontend in sync.
type StateEmitter interface {
	// EmitSnapshot sends a complete state snapshot to the frontend.
	// Use this for initial state or when delta tracking becomes complex.
	EmitSnapshot(state any)

	// EmitDelta sends incremental state changes using JSON Patch (RFC 6902).
	// More efficient than snapshots for small, frequent updates.
	EmitDelta(patches ...event.JSONPatch)
}

// channelEmitter implements StateEmitter by emitting events to a channel.
type channelEmitter struct {
	ch       chan<- Event
	stepName string
}

// NewChannelEmitter creates a StateEmitter that sends events to the given channel.
// The stepName is included in emitted events for tracing.
func NewChannelEmitter(ch chan<- Event, stepName string) StateEmitter {
	return &channelEmitter{ch: ch, stepName: stepName}
}

// EmitSnapshot sends a StateSnapshot event.
func (e *channelEmitter) EmitSnapshot(state any) {
	event.Emit(e.ch, Event{
		Type:     event.StateSnapshot,
		StepName: e.stepName,
		State:    state,
	})
}

// EmitDelta sends a StateDelta event with the given patches.
func (e *channelEmitter) EmitDelta(patches ...event.JSONPatch) {
	event.Emit(e.ch, Event{
		Type:         event.StateDelta,
		StepName:     e.stepName,
		StatePatches: patches,
	})
}

// noOpEmitter is a StateEmitter that discards all emissions.
// Used when no event channel is available (e.g., Run() without streaming).
type noOpEmitter struct{}

// NewNoOpEmitter creates a StateEmitter that discards all emissions.
func NewNoOpEmitter() StateEmitter {
	return &noOpEmitter{}
}

// EmitSnapshot does nothing.
func (e *noOpEmitter) EmitSnapshot(state any) {}

// EmitDelta does nothing.
func (e *noOpEmitter) EmitDelta(patches ...event.JSONPatch) {}

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
