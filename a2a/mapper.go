package a2a

import (
	"github.com/google/uuid"
	"github.com/spetersoncode/gains/event"
)

// Event represents an A2A streaming event (status update or artifact update).
type Event interface {
	isA2AEvent()
}

func (TaskStatusUpdateEvent) isA2AEvent()   {}
func (TaskArtifactUpdateEvent) isA2AEvent() {}

// Mapper converts gains events to A2A task status updates.
//
// A2A uses a task-centric model where all updates are framed as task status
// changes or artifact additions. This mapper accumulates message content
// and emits task status updates as the agent progresses.
//
// Create a new Mapper for each task using NewMapper. The Mapper is not
// safe for concurrent use - each goroutine should have its own Mapper.
type Mapper struct {
	taskID    string
	contextID string
	state     TaskState
	runDepth  int

	// Message accumulation
	currentMessageID string
	currentContent   string
	pendingParts     []Part
}

// NewMapper creates a new Mapper for a single task.
func NewMapper(taskID, contextID string) *Mapper {
	if taskID == "" {
		taskID = uuid.New().String()
	}
	if contextID == "" {
		contextID = uuid.New().String()
	}
	return &Mapper{
		taskID:    taskID,
		contextID: contextID,
		state:     TaskStateSubmitted,
	}
}

// TaskID returns the task ID for this mapper.
func (m *Mapper) TaskID() string {
	return m.taskID
}

// ContextID returns the context ID for this mapper.
func (m *Mapper) ContextID() string {
	return m.contextID
}

// State returns the current task state.
func (m *Mapper) State() TaskState {
	return m.state
}

// StatusUpdate creates a task status update event.
func (m *Mapper) StatusUpdate(state TaskState, msg *Message, final bool) TaskStatusUpdateEvent {
	m.state = state
	return NewTaskStatusUpdateEvent(
		m.taskID,
		m.contextID,
		NewTaskStatusWithMessage(state, msg),
		final,
	)
}

// ArtifactUpdate creates a task artifact update event.
func (m *Mapper) ArtifactUpdate(artifact Artifact) TaskArtifactUpdateEvent {
	return NewTaskArtifactUpdateEvent(m.taskID, m.contextID, artifact)
}

// Submitted returns a status update for the submitted state.
func (m *Mapper) Submitted() TaskStatusUpdateEvent {
	return m.StatusUpdate(TaskStateSubmitted, nil, false)
}

// Working returns a status update for the working state.
func (m *Mapper) Working() TaskStatusUpdateEvent {
	return m.StatusUpdate(TaskStateWorking, nil, false)
}

// InputRequired returns a status update requesting additional input.
func (m *Mapper) InputRequired(prompt string) TaskStatusUpdateEvent {
	msg := NewMessage(MessageRoleAgent, NewTextPart(prompt))
	return m.StatusUpdate(TaskStateInputRequired, &msg, false)
}

// Completed returns a final status update for successful completion.
func (m *Mapper) Completed(msg *Message) TaskStatusUpdateEvent {
	return m.StatusUpdate(TaskStateCompleted, msg, true)
}

// Failed returns a final status update for failure.
func (m *Mapper) Failed(errMsg string) TaskStatusUpdateEvent {
	msg := NewMessage(MessageRoleAgent, NewTextPart(errMsg))
	return m.StatusUpdate(TaskStateFailed, &msg, true)
}

// Canceled returns a final status update for cancellation.
func (m *Mapper) Canceled() TaskStatusUpdateEvent {
	return m.StatusUpdate(TaskStateCanceled, nil, true)
}

// MapStream wraps a gains event channel and yields A2A events.
// The returned channel closes when the input channel closes.
func (m *Mapper) MapStream(input <-chan event.Event) <-chan Event {
	output := make(chan Event, 100)
	go func() {
		defer close(output)
		for e := range input {
			if a2aEvent := m.MapEvent(e); a2aEvent != nil {
				output <- a2aEvent
			}
		}
	}()
	return output
}

// MapEvent converts a gains event to an A2A event.
// Returns nil for events that don't require an A2A update.
//
// For nested runs (e.g., workflows containing agents), only the outermost
// run lifecycle events trigger task state changes.
func (m *Mapper) MapEvent(e event.Event) Event {
	switch e.Type {
	// Run lifecycle - track depth for nested runs
	case event.RunStart:
		m.runDepth++
		if m.runDepth == 1 {
			return m.Working()
		}
		return nil

	case event.RunEnd:
		m.runDepth--
		if m.runDepth == 0 {
			// Finalize any pending message
			var msg *Message
			if m.currentContent != "" || len(m.pendingParts) > 0 {
				parts := m.pendingParts
				if m.currentContent != "" {
					parts = append([]Part{NewTextPart(m.currentContent)}, parts...)
				}
				finalMsg := NewMessage(MessageRoleAgent, parts...)
				msg = &finalMsg
			}
			return m.Completed(msg)
		}
		return nil

	case event.RunError:
		errMsg := "unknown error"
		if e.Error != nil {
			errMsg = e.Error.Error()
		}
		return m.Failed(errMsg)

	// Message lifecycle - accumulate content
	case event.MessageStart:
		m.currentMessageID = e.MessageID
		m.currentContent = ""
		return nil // No A2A event needed yet

	case event.MessageDelta:
		m.currentContent += e.Delta
		// Optionally emit intermediate status updates for long messages
		return nil

	case event.MessageEnd:
		// Message complete - could emit status update with current content
		// But we wait for RunEnd to emit the final message
		return nil

	// Tool calls - could be represented as artifacts or status messages
	case event.ToolCallStart, event.ToolCallArgs, event.ToolCallEnd:
		// Tool calls in progress - state is still "working"
		return nil

	case event.ToolCallResult:
		// Tool result received - could emit as artifact
		if e.ToolResult != nil {
			artifact := NewArtifact(
				e.ToolCall.Name,
				"Tool execution result",
				NewTextPart(e.ToolResult.Content),
			)
			// Add metadata about the tool call
			artifact.Metadata = map[string]any{
				"tool_call_id": e.ToolCall.ID,
				"tool_name":    e.ToolCall.Name,
				"is_error":     e.ToolResult.IsError,
			}
			return m.ArtifactUpdate(artifact)
		}
		return nil

	// Workflow events - no direct A2A mapping, but we stay in "working" state
	case event.StepStart, event.StepEnd, event.StepSkipped:
		return nil
	case event.ParallelStart, event.ParallelEnd:
		return nil
	case event.RouteSelected, event.LoopIteration:
		return nil

	default:
		return nil
	}
}

// CreateTask creates a Task object from the current mapper state.
func (m *Mapper) CreateTask() *Task {
	task := NewTask(m.taskID, m.contextID)
	task.Status = NewTaskStatus(m.state)
	return task
}

// CreateTaskWithHistory creates a Task with the given message history.
func (m *Mapper) CreateTaskWithHistory(history []Message) *Task {
	task := m.CreateTask()
	task.History = history
	return task
}
