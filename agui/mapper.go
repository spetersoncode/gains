package agui

import (
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
)

// Custom event names for gains-specific workflow events.
// These are emitted as AG-UI CUSTOM events with a name and value.
const (
	// CustomEventRouteSelected is emitted when a route is chosen in a Router step.
	// Value contains: stepName (string), routeName (string)
	CustomEventRouteSelected = "gains.route_selected"

	// CustomEventLoopIteration is emitted at the start of each loop iteration.
	// Value contains: stepName (string), iteration (int)
	CustomEventLoopIteration = "gains.loop_iteration"
)

// Mapper converts gains events to AG-UI events.
// With the unified event system, this is now a true 1:1 mapping -
// each gains event maps to exactly one AG-UI event.
//
// The mapper tracks run depth to handle nested RunStart/RunEnd events
// (e.g., from workflows containing agents or sub-agents). Only the outermost
// run lifecycle events are mapped to AG-UI RUN_STARTED/RUN_FINISHED.
//
// Create a new Mapper for each run using NewMapper. The Mapper is not
// safe for concurrent use - each goroutine should have its own Mapper.
type Mapper struct {
	threadID     string
	runID        string
	runDepth     int  // Tracks nesting depth of runs
	initialState any  // Optional initial state to emit after first RunStart
	stateEmitted bool // Track whether initial state has been emitted
}

// MapperOption configures a Mapper.
type MapperOption func(*Mapper)

// WithInitialState configures the mapper to emit a STATE_SNAPSHOT event
// with the given state immediately after RUN_STARTED. This ensures the
// frontend has the initial state for shared state synchronization.
func WithInitialState(state any) MapperOption {
	return func(m *Mapper) {
		m.initialState = state
	}
}

// NewMapper creates a new Mapper for a single run.
// The threadID and runID are used in lifecycle events (RUN_STARTED, RUN_FINISHED).
// Use WithInitialState to emit an initial STATE_SNAPSHOT after RUN_STARTED.
func NewMapper(threadID, runID string, opts ...MapperOption) *Mapper {
	if threadID == "" {
		threadID = events.GenerateThreadID()
	}
	if runID == "" {
		runID = events.GenerateRunID()
	}
	m := &Mapper{
		threadID: threadID,
		runID:    runID,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// ThreadID returns the thread ID for this mapper.
func (m *Mapper) ThreadID() string {
	return m.threadID
}

// RunID returns the run ID for this mapper.
func (m *Mapper) RunID() string {
	return m.runID
}

// RunDepth returns the current nesting depth of runs.
// Returns 0 when no runs are active, 1 during a top-level run, etc.
func (m *Mapper) RunDepth() int {
	return m.runDepth
}

// RunStarted returns a RUN_STARTED event.
func (m *Mapper) RunStarted() events.Event {
	return events.NewRunStartedEvent(m.threadID, m.runID)
}

// RunFinished returns a RUN_FINISHED event.
func (m *Mapper) RunFinished() events.Event {
	return events.NewRunFinishedEvent(m.threadID, m.runID)
}

// RunError returns a RUN_ERROR event.
func (m *Mapper) RunError(err error) events.Event {
	msg := "unknown error"
	if err != nil {
		msg = err.Error()
	}
	return events.NewRunErrorEvent(msg)
}

// StateSnapshot returns a STATE_SNAPSHOT event with the given state.
func (m *Mapper) StateSnapshot(state any) events.Event {
	return events.NewStateSnapshotEvent(state)
}

// StateDelta returns a STATE_DELTA event with the given JSON Patch operations.
func (m *Mapper) StateDelta(patches ...event.JSONPatch) events.Event {
	return events.NewStateDeltaEvent(toAGUIPatches(patches))
}

// MessagesSnapshot returns a MESSAGES_SNAPSHOT event with the given messages.
func (m *Mapper) MessagesSnapshot(messages []ai.Message) events.Event {
	return events.NewMessagesSnapshotEvent(FromGainsMessages(messages))
}

// MapStream wraps a gains event channel and yields AG-UI events.
// Events that have no AG-UI equivalent (returning nil from MapEvent) are filtered out.
// The returned channel closes when the input channel closes.
//
// If the mapper was created with WithInitialState, a STATE_SNAPSHOT event is
// automatically emitted after the first RUN_STARTED event.
func (m *Mapper) MapStream(input <-chan event.Event) <-chan events.Event {
	output := make(chan events.Event, 100)
	go func() {
		defer close(output)
		for e := range input {
			if aguiEvent := m.MapEvent(e); aguiEvent != nil {
				output <- aguiEvent

				// Emit initial state snapshot after first RUN_STARTED
				if aguiEvent.Type() == events.EventTypeRunStarted && m.initialState != nil && !m.stateEmitted {
					m.stateEmitted = true
					output <- m.StateSnapshot(m.initialState)
				}
			}
		}
	}()
	return output
}

// MapEvent converts a unified gains event to an AG-UI event.
// This is a true 1:1 mapping - each gains event maps to exactly one AG-UI event.
// Returns nil for events that have no AG-UI equivalent.
//
// For nested runs (e.g., workflows containing agents), only the outermost
// RunStart/RunEnd events are mapped to AG-UI lifecycle events.
func (m *Mapper) MapEvent(e event.Event) events.Event {
	switch e.Type {
	// Run lifecycle - track depth for nested runs
	case event.RunStart:
		m.runDepth++
		if m.runDepth == 1 {
			// Only emit for the outermost run
			return m.RunStarted()
		}
		return nil
	case event.RunEnd:
		m.runDepth--
		if m.runDepth == 0 {
			// Only emit when returning to outermost level
			return m.RunFinished()
		}
		return nil
	case event.RunError:
		// Errors always bubble up regardless of nesting depth
		return m.RunError(e.Error)

	// Step lifecycle
	case event.StepStart:
		return events.NewStepStartedEvent(e.StepName)
	case event.StepEnd:
		return events.NewStepFinishedEvent(e.StepName)
	case event.StepSkipped:
		// Emit as finished (skipped steps are immediately done)
		return events.NewStepFinishedEvent(e.StepName)

	// Message lifecycle
	case event.MessageStart:
		return events.NewTextMessageStartEvent(
			e.MessageID,
			events.WithRole(RoleAssistant),
		)
	case event.MessageDelta:
		return events.NewTextMessageContentEvent(e.MessageID, e.Delta)
	case event.MessageEnd:
		return events.NewTextMessageEndEvent(e.MessageID)

	// Tool call lifecycle
	case event.ToolCallStart:
		if e.ToolCall == nil {
			return nil
		}
		return events.NewToolCallStartEvent(e.ToolCall.ID, e.ToolCall.Name)
	case event.ToolCallArgs:
		if e.ToolCall == nil {
			return nil
		}
		return events.NewToolCallArgsEvent(e.ToolCall.ID, e.ToolCall.Arguments)
	case event.ToolCallEnd:
		if e.ToolCall == nil {
			return nil
		}
		return events.NewToolCallEndEvent(e.ToolCall.ID)
	case event.ToolCallResult:
		if e.ToolCall == nil || e.ToolResult == nil {
			return nil
		}
		messageID := events.GenerateMessageID()
		return events.NewToolCallResultEvent(messageID, e.ToolCall.ID, e.ToolResult.Content)

	// Tool approval (gains-specific, no direct AG-UI equivalent)
	case event.ToolCallApproved, event.ToolCallRejected, event.ToolCallExecuting:
		return nil

	// Workflow-specific
	case event.ParallelStart:
		return events.NewStepStartedEvent(e.StepName)
	case event.ParallelEnd:
		return events.NewStepFinishedEvent(e.StepName)
	case event.RouteSelected:
		// Map to AG-UI custom event for route observability
		return events.NewCustomEvent(CustomEventRouteSelected,
			events.WithValue(map[string]any{
				"stepName":  e.StepName,
				"routeName": e.RouteName,
			}))
	case event.LoopIteration:
		// Map to AG-UI custom event for loop observability
		return events.NewCustomEvent(CustomEventLoopIteration,
			events.WithValue(map[string]any{
				"stepName":  e.StepName,
				"iteration": e.Iteration,
			}))

	// State synchronization
	case event.StateSnapshot:
		return events.NewStateSnapshotEvent(e.State)
	case event.StateDelta:
		return events.NewStateDeltaEvent(toAGUIPatches(e.StatePatches))
	case event.MessagesSnapshot:
		return events.NewMessagesSnapshotEvent(FromGainsMessages(e.Messages))

	default:
		return nil
	}
}

// toAGUIPatches converts gains JSONPatch operations to AG-UI JSONPatchOperation.
func toAGUIPatches(patches []event.JSONPatch) []events.JSONPatchOperation {
	if len(patches) == 0 {
		return nil
	}
	result := make([]events.JSONPatchOperation, len(patches))
	for i, p := range patches {
		result[i] = events.JSONPatchOperation{
			Op:    string(p.Op),
			Path:  p.Path,
			Value: p.Value,
			From:  p.From,
		}
	}
	return result
}
