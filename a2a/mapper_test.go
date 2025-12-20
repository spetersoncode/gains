package a2a

import (
	"errors"
	"testing"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
)

func TestMapper_Lifecycle(t *testing.T) {
	m := NewMapper("task-1", "ctx-1")

	if m.TaskID() != "task-1" {
		t.Errorf("TaskID = %q, want task-1", m.TaskID())
	}
	if m.ContextID() != "ctx-1" {
		t.Errorf("ContextID = %q, want ctx-1", m.ContextID())
	}
	if m.State() != TaskStateSubmitted {
		t.Errorf("State = %v, want %v", m.State(), TaskStateSubmitted)
	}
}

func TestMapper_StatusUpdates(t *testing.T) {
	m := NewMapper("task-1", "ctx-1")

	// Submitted
	submitted := m.Submitted()
	if submitted.Status.State != TaskStateSubmitted {
		t.Errorf("Submitted state = %v, want %v", submitted.Status.State, TaskStateSubmitted)
	}
	if submitted.Final {
		t.Error("Submitted should not be final")
	}

	// Working
	working := m.Working()
	if working.Status.State != TaskStateWorking {
		t.Errorf("Working state = %v, want %v", working.Status.State, TaskStateWorking)
	}
	if working.Final {
		t.Error("Working should not be final")
	}

	// Input Required
	input := m.InputRequired("Please provide more info")
	if input.Status.State != TaskStateInputRequired {
		t.Errorf("InputRequired state = %v, want %v", input.Status.State, TaskStateInputRequired)
	}
	if input.Status.Message == nil {
		t.Error("InputRequired should have message")
	}
	if input.Final {
		t.Error("InputRequired should not be final")
	}

	// Completed
	msg := NewMessage(MessageRoleAgent, NewTextPart("Done!"))
	completed := m.Completed(&msg)
	if completed.Status.State != TaskStateCompleted {
		t.Errorf("Completed state = %v, want %v", completed.Status.State, TaskStateCompleted)
	}
	if !completed.Final {
		t.Error("Completed should be final")
	}

	// Failed
	failed := m.Failed("Something went wrong")
	if failed.Status.State != TaskStateFailed {
		t.Errorf("Failed state = %v, want %v", failed.Status.State, TaskStateFailed)
	}
	if !failed.Final {
		t.Error("Failed should be final")
	}

	// Canceled
	canceled := m.Canceled()
	if canceled.Status.State != TaskStateCanceled {
		t.Errorf("Canceled state = %v, want %v", canceled.Status.State, TaskStateCanceled)
	}
	if !canceled.Final {
		t.Error("Canceled should be final")
	}
}

func TestMapper_MapEvent_RunLifecycle(t *testing.T) {
	m := NewMapper("task-1", "ctx-1")

	// RunStart -> Working
	evt := m.MapEvent(event.Event{Type: event.RunStart})
	if evt == nil {
		t.Fatal("RunStart should emit event")
	}
	update, ok := evt.(TaskStatusUpdateEvent)
	if !ok {
		t.Fatal("Event should be TaskStatusUpdateEvent")
	}
	if update.Status.State != TaskStateWorking {
		t.Errorf("State = %v, want %v", update.Status.State, TaskStateWorking)
	}

	// RunEnd -> Completed
	evt = m.MapEvent(event.Event{Type: event.RunEnd})
	if evt == nil {
		t.Fatal("RunEnd should emit event")
	}
	update, ok = evt.(TaskStatusUpdateEvent)
	if !ok {
		t.Fatal("Event should be TaskStatusUpdateEvent")
	}
	if update.Status.State != TaskStateCompleted {
		t.Errorf("State = %v, want %v", update.Status.State, TaskStateCompleted)
	}
	if !update.Final {
		t.Error("RunEnd should be final")
	}
}

func TestMapper_MapEvent_NestedRuns(t *testing.T) {
	m := NewMapper("task-1", "ctx-1")

	// Outer run starts
	evt := m.MapEvent(event.Event{Type: event.RunStart})
	if evt == nil {
		t.Error("Outer RunStart should emit")
	}

	// Inner run starts (e.g., sub-agent)
	evt = m.MapEvent(event.Event{Type: event.RunStart})
	if evt != nil {
		t.Error("Inner RunStart should not emit")
	}

	// Inner run ends
	evt = m.MapEvent(event.Event{Type: event.RunEnd})
	if evt != nil {
		t.Error("Inner RunEnd should not emit")
	}

	// Outer run ends
	evt = m.MapEvent(event.Event{Type: event.RunEnd})
	if evt == nil {
		t.Error("Outer RunEnd should emit")
	}
}

func TestMapper_MapEvent_Error(t *testing.T) {
	m := NewMapper("task-1", "ctx-1")

	_ = m.MapEvent(event.Event{Type: event.RunStart})

	testErr := errors.New("test error")
	evt := m.MapEvent(event.Event{Type: event.RunError, Error: testErr})

	if evt == nil {
		t.Fatal("RunError should emit event")
	}

	update, ok := evt.(TaskStatusUpdateEvent)
	if !ok {
		t.Fatal("Event should be TaskStatusUpdateEvent")
	}

	if update.Status.State != TaskStateFailed {
		t.Errorf("State = %v, want %v", update.Status.State, TaskStateFailed)
	}
	if !update.Final {
		t.Error("Error should be final")
	}
	if update.Status.Message == nil {
		t.Error("Error should have message")
	}
}

func TestMapper_MapEvent_ToolResult(t *testing.T) {
	m := NewMapper("task-1", "ctx-1")

	_ = m.MapEvent(event.Event{Type: event.RunStart})

	evt := m.MapEvent(event.Event{
		Type: event.ToolCallResult,
		ToolCall: &ai.ToolCall{
			ID:   "call-1",
			Name: "get_weather",
		},
		ToolResult: &ai.ToolResult{
			ToolCallID: "call-1",
			Content:    "Sunny, 72F",
		},
	})

	if evt == nil {
		t.Fatal("ToolCallResult should emit event")
	}

	artifact, ok := evt.(TaskArtifactUpdateEvent)
	if !ok {
		t.Fatal("Event should be TaskArtifactUpdateEvent")
	}

	if artifact.Artifact.Name != "get_weather" {
		t.Errorf("Artifact name = %q, want get_weather", artifact.Artifact.Name)
	}
}

func TestMapper_MapEvent_MessageAccumulation(t *testing.T) {
	m := NewMapper("task-1", "ctx-1")

	_ = m.MapEvent(event.Event{Type: event.RunStart})

	// Message events should not emit immediately
	evt := m.MapEvent(event.Event{Type: event.MessageStart, MessageID: "msg-1"})
	if evt != nil {
		t.Error("MessageStart should not emit")
	}

	evt = m.MapEvent(event.Event{Type: event.MessageDelta, MessageID: "msg-1", Delta: "Hello"})
	if evt != nil {
		t.Error("MessageDelta should not emit")
	}

	evt = m.MapEvent(event.Event{Type: event.MessageDelta, MessageID: "msg-1", Delta: " world"})
	if evt != nil {
		t.Error("MessageDelta should not emit")
	}

	evt = m.MapEvent(event.Event{Type: event.MessageEnd, MessageID: "msg-1"})
	if evt != nil {
		t.Error("MessageEnd should not emit")
	}

	// RunEnd should include accumulated message
	evt = m.MapEvent(event.Event{Type: event.RunEnd})
	if evt == nil {
		t.Fatal("RunEnd should emit")
	}

	update, ok := evt.(TaskStatusUpdateEvent)
	if !ok {
		t.Fatal("Event should be TaskStatusUpdateEvent")
	}

	if update.Status.Message == nil {
		t.Fatal("Completed status should include message")
	}

	if update.Status.Message.TextContent() != "Hello world" {
		t.Errorf("Message text = %q, want 'Hello world'", update.Status.Message.TextContent())
	}
}

func TestMapper_CreateTask(t *testing.T) {
	m := NewMapper("task-1", "ctx-1")

	task := m.CreateTask()

	if task.ID != "task-1" {
		t.Errorf("ID = %q, want task-1", task.ID)
	}
	if task.ContextID != "ctx-1" {
		t.Errorf("ContextID = %q, want ctx-1", task.ContextID)
	}
	if task.Kind != "task" {
		t.Errorf("Kind = %q, want task", task.Kind)
	}
	if task.Status.State != TaskStateSubmitted {
		t.Errorf("State = %v, want %v", task.Status.State, TaskStateSubmitted)
	}
}

func TestMapper_CreateTaskWithHistory(t *testing.T) {
	m := NewMapper("task-1", "ctx-1")

	history := []Message{
		NewMessage(MessageRoleUser, NewTextPart("Hello")),
		NewMessage(MessageRoleAgent, NewTextPart("Hi there!")),
	}

	task := m.CreateTaskWithHistory(history)

	if len(task.History) != 2 {
		t.Errorf("History len = %d, want 2", len(task.History))
	}
}

func TestTaskState_IsTerminal(t *testing.T) {
	terminal := []TaskState{
		TaskStateCompleted,
		TaskStateCanceled,
		TaskStateFailed,
		TaskStateRejected,
	}

	nonTerminal := []TaskState{
		TaskStateSubmitted,
		TaskStateWorking,
		TaskStateInputRequired,
		TaskStateAuthRequired,
	}

	for _, state := range terminal {
		if !state.IsTerminal() {
			t.Errorf("%v should be terminal", state)
		}
	}

	for _, state := range nonTerminal {
		if state.IsTerminal() {
			t.Errorf("%v should not be terminal", state)
		}
	}
}
