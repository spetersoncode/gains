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

func TestMapper_NestedRuns(t *testing.T) {
	t.Run("nested RunStart/RunEnd only emits outermost", func(t *testing.T) {
		m := NewMapper("thread-1", "run-1")

		// First run start - should emit
		r1 := m.MapEvent(event.Event{Type: event.RunStart})
		if r1 == nil {
			t.Fatal("expected RUN_STARTED for first run")
		}
		if r1.Type() != events.EventTypeRunStarted {
			t.Errorf("expected RUN_STARTED, got %s", r1.Type())
		}
		if m.RunDepth() != 1 {
			t.Errorf("expected depth 1, got %d", m.RunDepth())
		}

		// Nested run start - should return nil
		r2 := m.MapEvent(event.Event{Type: event.RunStart})
		if r2 != nil {
			t.Errorf("expected nil for nested run, got %s", r2.Type())
		}
		if m.RunDepth() != 2 {
			t.Errorf("expected depth 2, got %d", m.RunDepth())
		}

		// Nested run end - should return nil
		r3 := m.MapEvent(event.Event{Type: event.RunEnd})
		if r3 != nil {
			t.Errorf("expected nil for nested run end, got %s", r3.Type())
		}
		if m.RunDepth() != 1 {
			t.Errorf("expected depth 1, got %d", m.RunDepth())
		}

		// First run end - should emit
		r4 := m.MapEvent(event.Event{Type: event.RunEnd})
		if r4 == nil {
			t.Fatal("expected RUN_FINISHED for first run end")
		}
		if r4.Type() != events.EventTypeRunFinished {
			t.Errorf("expected RUN_FINISHED, got %s", r4.Type())
		}
		if m.RunDepth() != 0 {
			t.Errorf("expected depth 0, got %d", m.RunDepth())
		}
	})

	t.Run("RunError emits regardless of depth", func(t *testing.T) {
		m := NewMapper("thread-1", "run-1")

		// Start outer run
		m.MapEvent(event.Event{Type: event.RunStart})
		// Start nested run
		m.MapEvent(event.Event{Type: event.RunStart})
		if m.RunDepth() != 2 {
			t.Fatalf("expected depth 2, got %d", m.RunDepth())
		}

		// Error in nested run should still emit
		r := m.MapEvent(event.Event{Type: event.RunError, Error: errors.New("nested error")})
		if r == nil {
			t.Fatal("expected RUN_ERROR event")
		}
		if r.Type() != events.EventTypeRunError {
			t.Errorf("expected RUN_ERROR, got %s", r.Type())
		}
	})

	t.Run("MapStream filters nested runs", func(t *testing.T) {
		m := NewMapper("thread-1", "run-1")

		input := make(chan event.Event, 20)

		// Simulate: outer run -> nested run -> message -> end nested -> end outer
		input <- event.Event{Type: event.RunStart}                              // outer: emits
		input <- event.Event{Type: event.RunStart}                              // nested: filtered
		input <- event.Event{Type: event.MessageStart, MessageID: "msg-1"}      // emits
		input <- event.Event{Type: event.MessageDelta, MessageID: "msg-1", Delta: "Hi"} // emits
		input <- event.Event{Type: event.MessageEnd, MessageID: "msg-1"}        // emits
		input <- event.Event{Type: event.RunEnd}                                // nested: filtered
		input <- event.Event{Type: event.RunEnd}                                // outer: emits
		close(input)

		output := m.MapStream(input)

		var received []events.EventType
		for ev := range output {
			received = append(received, ev.Type())
		}

		expected := []events.EventType{
			events.EventTypeRunStarted,
			events.EventTypeTextMessageStart,
			events.EventTypeTextMessageContent,
			events.EventTypeTextMessageEnd,
			events.EventTypeRunFinished,
		}

		if len(received) != len(expected) {
			t.Fatalf("expected %d events, got %d: %v", len(expected), len(received), received)
		}

		for i, e := range expected {
			if received[i] != e {
				t.Errorf("event %d: expected %s, got %s", i, e, received[i])
			}
		}
	})
}

func TestMapper_WithInitialState(t *testing.T) {
	t.Run("emits STATE_SNAPSHOT after RUN_STARTED", func(t *testing.T) {
		initialState := map[string]any{
			"progress": 0,
			"items":    []string{},
		}
		m := NewMapper("thread-1", "run-1", WithInitialState(initialState))

		input := make(chan event.Event, 10)
		input <- event.Event{Type: event.RunStart}
		input <- event.Event{Type: event.MessageStart, MessageID: "msg-1"}
		input <- event.Event{Type: event.MessageEnd, MessageID: "msg-1"}
		input <- event.Event{Type: event.RunEnd}
		close(input)

		output := m.MapStream(input)

		var received []events.EventType
		for ev := range output {
			received = append(received, ev.Type())
		}

		expected := []events.EventType{
			events.EventTypeRunStarted,
			events.EventTypeStateSnapshot, // Emitted after RUN_STARTED
			events.EventTypeTextMessageStart,
			events.EventTypeTextMessageEnd,
			events.EventTypeRunFinished,
		}

		if len(received) != len(expected) {
			t.Fatalf("expected %d events, got %d: %v", len(expected), len(received), received)
		}

		for i, e := range expected {
			if received[i] != e {
				t.Errorf("event %d: expected %s, got %s", i, e, received[i])
			}
		}
	})

	t.Run("only emits once for nested runs", func(t *testing.T) {
		m := NewMapper("thread-1", "run-1", WithInitialState(map[string]any{"x": 1}))

		input := make(chan event.Event, 20)
		input <- event.Event{Type: event.RunStart}  // outer - triggers snapshot
		input <- event.Event{Type: event.RunStart}  // nested - filtered
		input <- event.Event{Type: event.RunEnd}    // nested end - filtered
		input <- event.Event{Type: event.RunEnd}    // outer end
		close(input)

		output := m.MapStream(input)

		var stateSnapshots int
		for ev := range output {
			if ev.Type() == events.EventTypeStateSnapshot {
				stateSnapshots++
			}
		}

		if stateSnapshots != 1 {
			t.Errorf("expected 1 STATE_SNAPSHOT, got %d", stateSnapshots)
		}
	})

	t.Run("no snapshot without WithInitialState", func(t *testing.T) {
		m := NewMapper("thread-1", "run-1") // No initial state

		input := make(chan event.Event, 10)
		input <- event.Event{Type: event.RunStart}
		input <- event.Event{Type: event.RunEnd}
		close(input)

		output := m.MapStream(input)

		var received []events.EventType
		for ev := range output {
			received = append(received, ev.Type())
		}

		expected := []events.EventType{
			events.EventTypeRunStarted,
			events.EventTypeRunFinished,
		}

		if len(received) != len(expected) {
			t.Fatalf("expected %d events, got %d: %v", len(expected), len(received), received)
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

func TestMapper_MapEvent_CustomWorkflowEvents(t *testing.T) {
	m := NewMapper("thread-1", "run-1")

	t.Run("RouteSelected maps to CUSTOM event", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type:      event.RouteSelected,
			StepName:  "router_step",
			RouteName: "route_a",
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeCustom {
			t.Errorf("expected CUSTOM, got %s", result.Type())
		}
	})

	t.Run("LoopIteration maps to CUSTOM event", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type:      event.LoopIteration,
			StepName:  "loop_step",
			Iteration: 3,
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeCustom {
			t.Errorf("expected CUSTOM, got %s", result.Type())
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

func TestMapper_MapStream(t *testing.T) {
	m := NewMapper("thread-1", "run-1")

	t.Run("maps events and filters nils", func(t *testing.T) {
		input := make(chan event.Event, 10)

		// Send mix of mappable and non-mappable events
		input <- event.Event{Type: event.RunStart}
		input <- event.Event{Type: event.ToolCallApproved}  // maps to nil (gains-specific)
		input <- event.Event{Type: event.MessageStart, MessageID: "msg-1"}
		input <- event.Event{Type: event.ToolCallExecuting} // maps to nil (gains-specific)
		input <- event.Event{Type: event.MessageDelta, MessageID: "msg-1", Delta: "Hello"}
		input <- event.Event{Type: event.RouteSelected}     // maps to CUSTOM event
		input <- event.Event{Type: event.MessageEnd, MessageID: "msg-1"}
		input <- event.Event{Type: event.RunEnd}
		close(input)

		output := m.MapStream(input)

		var received []events.EventType
		for ev := range output {
			received = append(received, ev.Type())
		}

		expected := []events.EventType{
			events.EventTypeRunStarted,
			events.EventTypeTextMessageStart,
			events.EventTypeTextMessageContent,
			events.EventTypeCustom, // RouteSelected now maps to CUSTOM
			events.EventTypeTextMessageEnd,
			events.EventTypeRunFinished,
		}

		if len(received) != len(expected) {
			t.Fatalf("expected %d events, got %d: %v", len(expected), len(received), received)
		}

		for i, e := range expected {
			if received[i] != e {
				t.Errorf("event %d: expected %s, got %s", i, e, received[i])
			}
		}
	})

	t.Run("closes output when input closes", func(t *testing.T) {
		input := make(chan event.Event)
		output := m.MapStream(input)

		close(input)

		// Output should close after input closes
		_, open := <-output
		if open {
			t.Error("expected output channel to be closed")
		}
	})

	t.Run("handles empty input", func(t *testing.T) {
		input := make(chan event.Event)
		close(input)

		output := m.MapStream(input)

		var count int
		for range output {
			count++
		}

		if count != 0 {
			t.Errorf("expected 0 events, got %d", count)
		}
	})
}

func TestMapper_StateEvents(t *testing.T) {
	m := NewMapper("thread-1", "run-1")

	t.Run("StateSnapshot helper", func(t *testing.T) {
		state := map[string]any{
			"progress": 50,
			"items":    []string{"a", "b"},
		}

		ev := m.StateSnapshot(state)
		if ev.Type() != events.EventTypeStateSnapshot {
			t.Errorf("expected STATE_SNAPSHOT, got %s", ev.Type())
		}
	})

	t.Run("StateDelta helper", func(t *testing.T) {
		ev := m.StateDelta(
			event.Replace("/progress", 75),
			event.Add("/items/-", "c"),
		)
		if ev.Type() != events.EventTypeStateDelta {
			t.Errorf("expected STATE_DELTA, got %s", ev.Type())
		}
	})
}

func TestMapper_MapEvent_State(t *testing.T) {
	m := NewMapper("thread-1", "run-1")

	t.Run("StateSnapshot maps to STATE_SNAPSHOT", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type:  event.StateSnapshot,
			State: map[string]any{"progress": 100},
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeStateSnapshot {
			t.Errorf("expected STATE_SNAPSHOT, got %s", result.Type())
		}
	})

	t.Run("StateDelta maps to STATE_DELTA", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type: event.StateDelta,
			StatePatches: []event.JSONPatch{
				{Op: event.PatchReplace, Path: "/progress", Value: 50},
				{Op: event.PatchAdd, Path: "/items/-", Value: "new"},
			},
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeStateDelta {
			t.Errorf("expected STATE_DELTA, got %s", result.Type())
		}
	})

	t.Run("StateDelta with empty patches", func(t *testing.T) {
		result := m.MapEvent(event.Event{
			Type:         event.StateDelta,
			StatePatches: nil,
		})
		if result == nil {
			t.Fatal("expected event, got nil")
		}
		if result.Type() != events.EventTypeStateDelta {
			t.Errorf("expected STATE_DELTA, got %s", result.Type())
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
