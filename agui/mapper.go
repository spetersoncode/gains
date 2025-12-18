package agui

import (
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"

	"github.com/spetersoncode/gains/event"
)

// Mapper converts gains events to AG-UI events.
// With the unified event system, this is now a true 1:1 mapping -
// each gains event maps to exactly one AG-UI event.
//
// Create a new Mapper for each run using NewMapper. The Mapper is not
// safe for concurrent use - each goroutine should have its own Mapper.
type Mapper struct {
	threadID string
	runID    string
}

// NewMapper creates a new Mapper for a single run.
// The threadID and runID are used in lifecycle events (RUN_STARTED, RUN_FINISHED).
func NewMapper(threadID, runID string) *Mapper {
	if threadID == "" {
		threadID = events.GenerateThreadID()
	}
	if runID == "" {
		runID = events.GenerateRunID()
	}
	return &Mapper{
		threadID: threadID,
		runID:    runID,
	}
}

// ThreadID returns the thread ID for this mapper.
func (m *Mapper) ThreadID() string {
	return m.threadID
}

// RunID returns the run ID for this mapper.
func (m *Mapper) RunID() string {
	return m.runID
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

// MapStream wraps a gains event channel and yields AG-UI events.
// Events that have no AG-UI equivalent (returning nil from MapEvent) are filtered out.
// The returned channel closes when the input channel closes.
func (m *Mapper) MapStream(input <-chan event.Event) <-chan events.Event {
	output := make(chan events.Event, 100)
	go func() {
		defer close(output)
		for e := range input {
			if aguiEvent := m.MapEvent(e); aguiEvent != nil {
				output <- aguiEvent
			}
		}
	}()
	return output
}

// MapEvent converts a unified gains event to an AG-UI event.
// This is a true 1:1 mapping - each gains event maps to exactly one AG-UI event.
// Returns nil for events that have no AG-UI equivalent.
func (m *Mapper) MapEvent(e event.Event) events.Event {
	switch e.Type {
	// Run lifecycle
	case event.RunStart:
		return m.RunStarted()
	case event.RunEnd:
		return m.RunFinished()
	case event.RunError:
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
		// No direct AG-UI equivalent, could use custom event
		return nil
	case event.LoopIteration:
		// No direct AG-UI equivalent, could use custom event
		return nil

	// State synchronization
	case event.StateSnapshot:
		return events.NewStateSnapshotEvent(e.State)
	case event.StateDelta:
		return events.NewStateDeltaEvent(toAGUIPatches(e.StatePatches))

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
