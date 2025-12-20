package a2a

import (
	"context"
	"errors"
	"testing"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/event"
)

// mockAgentRunner is a mock implementation of AgentRunner for testing.
type mockAgentRunner struct {
	runFunc       func(ctx context.Context, messages []ai.Message, opts ...agent.Option) (*agent.Result, error)
	runStreamFunc func(ctx context.Context, messages []ai.Message, opts ...agent.Option) <-chan event.Event
}

func (m *mockAgentRunner) Run(ctx context.Context, messages []ai.Message, opts ...agent.Option) (*agent.Result, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, messages, opts...)
	}
	return &agent.Result{}, nil
}

func (m *mockAgentRunner) RunStream(ctx context.Context, messages []ai.Message, opts ...agent.Option) <-chan event.Event {
	if m.runStreamFunc != nil {
		return m.runStreamFunc(ctx, messages, opts...)
	}
	ch := make(chan event.Event)
	close(ch)
	return ch
}

func TestAgentExecutor_Execute(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		mock := &mockAgentRunner{
			runFunc: func(ctx context.Context, messages []ai.Message, opts ...agent.Option) (*agent.Result, error) {
				// Verify the message was converted
				if len(messages) != 1 {
					t.Errorf("expected 1 message, got %d", len(messages))
				}
				if messages[0].Content != "Hello" {
					t.Errorf("expected content 'Hello', got %q", messages[0].Content)
				}

				return &agent.Result{
					Response: &ai.Response{Content: "Hi there!"},
				}, nil
			},
		}

		executor := NewAgentExecutor(mock)

		req := SendMessageRequest{
			Message: NewMessage(MessageRoleUser, NewTextPart("Hello")),
		}

		task, err := executor.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if task.Status.State != TaskStateCompleted {
			t.Errorf("expected completed state, got %v", task.Status.State)
		}
	})

	t.Run("execution error", func(t *testing.T) {
		mock := &mockAgentRunner{
			runFunc: func(ctx context.Context, messages []ai.Message, opts ...agent.Option) (*agent.Result, error) {
				return nil, errors.New("agent error")
			},
		}

		executor := NewAgentExecutor(mock)

		req := SendMessageRequest{
			Message: NewMessage(MessageRoleUser, NewTextPart("Hello")),
		}

		task, err := executor.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if task.Status.State != TaskStateFailed {
			t.Errorf("expected failed state, got %v", task.Status.State)
		}
	})
}

func TestAgentExecutor_ExecuteStream(t *testing.T) {
	mock := &mockAgentRunner{
		runStreamFunc: func(ctx context.Context, messages []ai.Message, opts ...agent.Option) <-chan event.Event {
			ch := make(chan event.Event, 10)
			go func() {
				defer close(ch)
				ch <- event.Event{Type: event.RunStart}
				ch <- event.Event{Type: event.MessageStart, MessageID: "msg-1"}
				ch <- event.Event{Type: event.MessageDelta, MessageID: "msg-1", Delta: "Hello"}
				ch <- event.Event{Type: event.MessageEnd, MessageID: "msg-1"}
				ch <- event.Event{Type: event.RunEnd}
			}()
			return ch
		},
	}

	executor := NewAgentExecutor(mock)

	req := SendMessageRequest{
		Message: NewMessage(MessageRoleUser, NewTextPart("Hi")),
	}

	events := executor.ExecuteStream(context.Background(), req)

	var receivedEvents []Event
	for evt := range events {
		receivedEvents = append(receivedEvents, evt)
	}

	// Should have: Working, Completed (only run start/end map to events)
	if len(receivedEvents) < 2 {
		t.Errorf("expected at least 2 events, got %d", len(receivedEvents))
	}

	// First should be Working
	if update, ok := receivedEvents[0].(TaskStatusUpdateEvent); ok {
		if update.Status.State != TaskStateWorking {
			t.Errorf("expected working state, got %v", update.Status.State)
		}
	} else {
		t.Errorf("expected TaskStatusUpdateEvent, got %T", receivedEvents[0])
	}

	// Last should be Completed
	if update, ok := receivedEvents[len(receivedEvents)-1].(TaskStatusUpdateEvent); ok {
		if update.Status.State != TaskStateCompleted {
			t.Errorf("expected completed state, got %v", update.Status.State)
		}
	}
}

// mockWorkflowRunner is a mock implementation of WorkflowRunner for testing.
type mockWorkflowRunner struct {
	runStreamFunc func(ctx context.Context, state any, opts ...interface{}) <-chan event.Event
}

func (m *mockWorkflowRunner) RunStream(ctx context.Context, state any, opts ...interface{}) <-chan event.Event {
	if m.runStreamFunc != nil {
		return m.runStreamFunc(ctx, state, opts...)
	}
	ch := make(chan event.Event)
	close(ch)
	return ch
}

func TestWorkflowExecutor_Execute(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		mock := &mockWorkflowRunner{
			runStreamFunc: func(ctx context.Context, state any, opts ...interface{}) <-chan event.Event {
				// Verify the input was extracted
				input, ok := state.(map[string]any)
				if !ok {
					t.Errorf("expected map input, got %T", state)
				}
				if input["query"] != "Hello" {
					t.Errorf("expected query 'Hello', got %v", input["query"])
				}

				ch := make(chan event.Event, 10)
				go func() {
					defer close(ch)
					ch <- event.Event{Type: event.RunStart}
					ch <- event.Event{Type: event.RunEnd}
				}()
				return ch
			},
		}

		executor := NewWorkflowExecutor(mock)

		req := SendMessageRequest{
			Message: NewMessage(MessageRoleUser, NewTextPart("Hello")),
		}

		task, err := executor.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if task.Status.State != TaskStateCompleted {
			t.Errorf("expected completed state, got %v", task.Status.State)
		}
	})

	t.Run("execution error", func(t *testing.T) {
		mock := &mockWorkflowRunner{
			runStreamFunc: func(ctx context.Context, state any, opts ...interface{}) <-chan event.Event {
				ch := make(chan event.Event, 10)
				go func() {
					defer close(ch)
					ch <- event.Event{Type: event.RunError, Error: errors.New("workflow error")}
				}()
				return ch
			},
		}

		executor := NewWorkflowExecutor(mock)

		req := SendMessageRequest{
			Message: NewMessage(MessageRoleUser, NewTextPart("Hello")),
		}

		task, err := executor.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if task.Status.State != TaskStateFailed {
			t.Errorf("expected failed state, got %v", task.Status.State)
		}
	})
}

func TestWorkflowExecutor_ExecuteStream(t *testing.T) {
	mock := &mockWorkflowRunner{
		runStreamFunc: func(ctx context.Context, state any, opts ...interface{}) <-chan event.Event {
			ch := make(chan event.Event, 10)
			go func() {
				defer close(ch)
				ch <- event.Event{Type: event.RunStart}
				ch <- event.Event{Type: event.StepStart, StepName: "step1"}
				ch <- event.Event{Type: event.StepEnd, StepName: "step1"}
				ch <- event.Event{Type: event.RunEnd}
			}()
			return ch
		},
	}

	executor := NewWorkflowExecutor(mock)

	req := SendMessageRequest{
		Message: NewMessage(MessageRoleUser, NewTextPart("Hi")),
	}

	events := executor.ExecuteStream(context.Background(), req)

	var receivedEvents []Event
	for evt := range events {
		receivedEvents = append(receivedEvents, evt)
	}

	// Should have Working and Completed at minimum
	if len(receivedEvents) < 2 {
		t.Errorf("expected at least 2 events, got %d", len(receivedEvents))
	}
}

func TestMessageToWorkflowInput(t *testing.T) {
	t.Run("text message", func(t *testing.T) {
		msg := NewMessage(MessageRoleUser, NewTextPart("Hello world"))
		input := messageToWorkflowInput(msg)

		if input["query"] != "Hello world" {
			t.Errorf("expected query 'Hello world', got %v", input["query"])
		}
	})

	t.Run("data part", func(t *testing.T) {
		msg := NewMessage(MessageRoleUser,
			NewTextPart("Query"),
			NewDataPart(map[string]any{"key": "value", "num": 42}),
		)
		input := messageToWorkflowInput(msg)

		if input["query"] != "Query" {
			t.Errorf("expected query 'Query', got %v", input["query"])
		}
		if input["key"] != "value" {
			t.Errorf("expected key 'value', got %v", input["key"])
		}
		if input["num"] != 42 {
			t.Errorf("expected num 42, got %v", input["num"])
		}
	})

	t.Run("metadata", func(t *testing.T) {
		msg := NewMessage(MessageRoleUser, NewTextPart("Hello"))
		msg.Metadata = map[string]any{"meta": "data"}
		input := messageToWorkflowInput(msg)

		if input["meta"] != "data" {
			t.Errorf("expected meta 'data', got %v", input["meta"])
		}
	})
}

func TestGetContextID(t *testing.T) {
	t.Run("with context ID", func(t *testing.T) {
		contextID := "ctx-123"
		msg := NewMessage(MessageRoleUser, NewTextPart("Hello"))
		msg.ContextID = &contextID

		req := SendMessageRequest{Message: msg}
		got := getContextID(req)

		if got != "ctx-123" {
			t.Errorf("expected 'ctx-123', got %q", got)
		}
	})

	t.Run("without context ID", func(t *testing.T) {
		msg := NewMessage(MessageRoleUser, NewTextPart("Hello"))
		req := SendMessageRequest{Message: msg}
		got := getContextID(req)

		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})
}
