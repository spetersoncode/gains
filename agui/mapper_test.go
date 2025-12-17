package agui

import (
	"errors"
	"testing"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
)

func TestNewMapper(t *testing.T) {
	t.Run("with provided IDs", func(t *testing.T) {
		m := NewMapper("thread-123", "run-456")
		if m.ThreadID() != "thread-123" {
			t.Errorf("expected thread ID 'thread-123', got %q", m.ThreadID())
		}
		if m.RunID() != "run-456" {
			t.Errorf("expected run ID 'run-456', got %q", m.RunID())
		}
	})

	t.Run("generates IDs when empty", func(t *testing.T) {
		m := NewMapper("", "")
		if m.ThreadID() == "" {
			t.Error("expected generated thread ID, got empty")
		}
		if m.RunID() == "" {
			t.Error("expected generated run ID, got empty")
		}
	})
}

func TestMapper_LifecycleEvents(t *testing.T) {
	m := NewMapper("thread-1", "run-1")

	t.Run("RunStarted", func(t *testing.T) {
		ev := m.RunStarted()
		if ev.Type() != events.EventTypeRunStarted {
			t.Errorf("expected RUN_STARTED, got %s", ev.Type())
		}
	})

	t.Run("RunFinished", func(t *testing.T) {
		ev := m.RunFinished()
		if ev.Type() != events.EventTypeRunFinished {
			t.Errorf("expected RUN_FINISHED, got %s", ev.Type())
		}
	})

	t.Run("RunError", func(t *testing.T) {
		ev := m.RunError(errors.New("test error"))
		if ev.Type() != events.EventTypeRunError {
			t.Errorf("expected RUN_ERROR, got %s", ev.Type())
		}
	})
}

func TestMapper_MapEvent_RunLifecycle(t *testing.T) {
	m := NewMapper("thread-1", "run-1")

	t.Run("RunStart maps to RUN_STARTED", func(t *testing.T) {
		result := m.MapEvent(event.Event{Type: event.RunStart})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeRunStarted {
			t.Errorf("expected RUN_STARTED, got %s", result.Type())
		}
	})

	t.Run("RunEnd maps to RUN_FINISHED", func(t *testing.T) {
		result := m.MapEvent(event.Event{Type: event.RunEnd})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeRunFinished {
			t.Errorf("expected RUN_FINISHED, got %s", result.Type())
		}
	})

	t.Run("RunError maps to RUN_ERROR", func(t *testing.T) {
		result := m.MapEvent(event.Event{Type: event.RunError, Error: errors.New("test")})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeRunError {
			t.Errorf("expected RUN_ERROR, got %s", result.Type())
		}
	})
}

func TestMapper_MapEvent_MessageLifecycle(t *testing.T) {
	m := NewMapper("thread-1", "run-1")

	t.Run("MessageStart maps to TEXT_MESSAGE_START", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type:      event.MessageStart,
			MessageID: "msg-1",
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeTextMessageStart {
			t.Errorf("expected TEXT_MESSAGE_START, got %s", result.Type())
		}
	})

	t.Run("MessageDelta maps to TEXT_MESSAGE_CONTENT", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type:      event.MessageDelta,
			MessageID: "msg-1",
			Delta:     "Hello",
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeTextMessageContent {
			t.Errorf("expected TEXT_MESSAGE_CONTENT, got %s", result.Type())
		}
	})

	t.Run("MessageEnd maps to TEXT_MESSAGE_END", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type:      event.MessageEnd,
			MessageID: "msg-1",
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeTextMessageEnd {
			t.Errorf("expected TEXT_MESSAGE_END, got %s", result.Type())
		}
	})
}

func TestMapper_MapEvent_StepLifecycle(t *testing.T) {
	m := NewMapper("thread-1", "run-1")

	t.Run("StepStart maps to STEP_STARTED", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type:     event.StepStart,
			StepName: "test_step",
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeStepStarted {
			t.Errorf("expected STEP_STARTED, got %s", result.Type())
		}
	})

	t.Run("StepEnd maps to STEP_FINISHED", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type:     event.StepEnd,
			StepName: "test_step",
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeStepFinished {
			t.Errorf("expected STEP_FINISHED, got %s", result.Type())
		}
	})
}

func TestMapper_MapEvent_ToolCallLifecycle(t *testing.T) {
	m := NewMapper("thread-1", "run-1")

	t.Run("ToolCallStart maps to TOOL_CALL_START", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type: event.ToolCallStart,
			ToolCall: &ai.ToolCall{
				ID:   "call-1",
				Name: "get_weather",
			},
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeToolCallStart {
			t.Errorf("expected TOOL_CALL_START, got %s", result.Type())
		}
	})

	t.Run("ToolCallArgs maps to TOOL_CALL_ARGS", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type: event.ToolCallArgs,
			ToolCall: &ai.ToolCall{
				ID:        "call-1",
				Name:      "get_weather",
				Arguments: `{"location": "NYC"}`,
			},
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeToolCallArgs {
			t.Errorf("expected TOOL_CALL_ARGS, got %s", result.Type())
		}
	})

	t.Run("ToolCallEnd maps to TOOL_CALL_END", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type: event.ToolCallEnd,
			ToolCall: &ai.ToolCall{
				ID:   "call-1",
				Name: "get_weather",
			},
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeToolCallEnd {
			t.Errorf("expected TOOL_CALL_END, got %s", result.Type())
		}
	})

	t.Run("ToolCallResult maps to TOOL_CALL_RESULT", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type: event.ToolCallResult,
			ToolCall: &ai.ToolCall{
				ID:   "call-1",
				Name: "get_weather",
			},
			ToolResult: &ai.ToolResult{
				ToolCallID: "call-1",
				Content:    `{"temp": 72}`,
			},
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeToolCallResult {
			t.Errorf("expected TOOL_CALL_RESULT, got %s", result.Type())
		}
	})
}

func TestMapper_MapEvent_ApprovalEventsReturnNil(t *testing.T) {
	m := NewMapper("thread-1", "run-1")

	t.Run("ToolCallApproved returns nil", func(t *testing.T) {
		result := m.MapEvent(event.Event{Type: event.ToolCallApproved})
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("ToolCallRejected returns nil", func(t *testing.T) {
		result := m.MapEvent(event.Event{Type: event.ToolCallRejected})
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("ToolCallExecuting returns nil", func(t *testing.T) {
		result := m.MapEvent(event.Event{Type: event.ToolCallExecuting})
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestToGainsMessage(t *testing.T) {
	t.Run("user message", func(t *testing.T) {
		content := "Hello"
		aguiMsg := events.Message{
			ID:      "msg-1",
			Role:    RoleUser,
			Content: &content,
		}

		gainsMsg := ToGainsMessage(aguiMsg)

		if gainsMsg.Role != ai.RoleUser {
			t.Errorf("expected RoleUser, got %v", gainsMsg.Role)
		}
		if gainsMsg.Content != "Hello" {
			t.Errorf("expected 'Hello', got %q", gainsMsg.Content)
		}
	})

	t.Run("assistant message with tool calls", func(t *testing.T) {
		aguiMsg := events.Message{
			ID:   "msg-1",
			Role: RoleAssistant,
			ToolCalls: []events.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: events.Function{
						Name:      "get_weather",
						Arguments: `{"location": "NYC"}`,
					},
				},
			},
		}

		gainsMsg := ToGainsMessage(aguiMsg)

		if gainsMsg.Role != ai.RoleAssistant {
			t.Errorf("expected RoleAssistant, got %v", gainsMsg.Role)
		}
		if len(gainsMsg.ToolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(gainsMsg.ToolCalls))
		}
		if gainsMsg.ToolCalls[0].Name != "get_weather" {
			t.Errorf("expected 'get_weather', got %q", gainsMsg.ToolCalls[0].Name)
		}
	})

	t.Run("tool result message", func(t *testing.T) {
		content := `{"temp": 72}`
		toolCallID := "call-1"
		aguiMsg := events.Message{
			ID:         "msg-1",
			Role:       RoleTool,
			Content:    &content,
			ToolCallID: &toolCallID,
		}

		gainsMsg := ToGainsMessage(aguiMsg)

		if gainsMsg.Role != ai.RoleTool {
			t.Errorf("expected RoleTool, got %v", gainsMsg.Role)
		}
		if len(gainsMsg.ToolResults) != 1 {
			t.Fatalf("expected 1 tool result, got %d", len(gainsMsg.ToolResults))
		}
		if gainsMsg.ToolResults[0].Content != `{"temp": 72}` {
			t.Errorf("expected content, got %q", gainsMsg.ToolResults[0].Content)
		}
	})
}

func TestFromGainsMessage(t *testing.T) {
	t.Run("user message", func(t *testing.T) {
		gainsMsg := ai.Message{
			Role:    ai.RoleUser,
			Content: "Hello",
		}

		aguiMsg := FromGainsMessage(gainsMsg, 0)

		if aguiMsg.Role != RoleUser {
			t.Errorf("expected 'user', got %q", aguiMsg.Role)
		}
		if aguiMsg.Content == nil || *aguiMsg.Content != "Hello" {
			t.Errorf("expected 'Hello', got %v", aguiMsg.Content)
		}
	})

	t.Run("assistant message with tool calls", func(t *testing.T) {
		gainsMsg := ai.Message{
			Role: ai.RoleAssistant,
			ToolCalls: []ai.ToolCall{
				{
					ID:        "call-1",
					Name:      "get_weather",
					Arguments: `{"location": "NYC"}`,
				},
			},
		}

		aguiMsg := FromGainsMessage(gainsMsg, 0)

		if aguiMsg.Role != RoleAssistant {
			t.Errorf("expected 'assistant', got %q", aguiMsg.Role)
		}
		if len(aguiMsg.ToolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(aguiMsg.ToolCalls))
		}
		if aguiMsg.ToolCalls[0].Function.Name != "get_weather" {
			t.Errorf("expected 'get_weather', got %q", aguiMsg.ToolCalls[0].Function.Name)
		}
	})
}
