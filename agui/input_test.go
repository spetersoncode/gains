package agui

import (
	"encoding/json"
	"testing"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
)

func TestRunAgentInput_Prepare(t *testing.T) {
	t.Run("valid input with messages", func(t *testing.T) {
		content := "Hello"
		input := RunAgentInput{
			ThreadID: "thread-1",
			RunID:    "run-1",
			Messages: []events.Message{
				{ID: "msg-1", Role: "user", Content: &content},
			},
		}

		prepared, err := input.Prepare()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if prepared.ThreadID != "thread-1" {
			t.Errorf("ThreadID = %q, want %q", prepared.ThreadID, "thread-1")
		}
		if prepared.RunID != "run-1" {
			t.Errorf("RunID = %q, want %q", prepared.RunID, "run-1")
		}
		if len(prepared.Messages) != 1 {
			t.Errorf("len(Messages) = %d, want 1", len(prepared.Messages))
		}
		if prepared.Messages[0].Content != "Hello" {
			t.Errorf("Messages[0].Content = %q, want %q", prepared.Messages[0].Content, "Hello")
		}
	})

	t.Run("empty messages returns error", func(t *testing.T) {
		input := RunAgentInput{
			ThreadID: "thread-1",
			RunID:    "run-1",
			Messages: []events.Message{},
		}

		_, err := input.Prepare()
		if err != ErrNoMessages {
			t.Errorf("error = %v, want ErrNoMessages", err)
		}
	})

	t.Run("nil messages returns error", func(t *testing.T) {
		input := RunAgentInput{
			ThreadID: "thread-1",
			RunID:    "run-1",
			Messages: nil,
		}

		_, err := input.Prepare()
		if err != ErrNoMessages {
			t.Errorf("error = %v, want ErrNoMessages", err)
		}
	})

	t.Run("with frontend tools", func(t *testing.T) {
		content := "Use my tool"
		input := RunAgentInput{
			ThreadID: "thread-1",
			RunID:    "run-1",
			Messages: []events.Message{
				{ID: "msg-1", Role: "user", Content: &content},
			},
			Tools: []any{
				map[string]any{
					"name":        "my_tool",
					"description": "A custom tool",
				},
			},
		}

		prepared, err := input.Prepare()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(prepared.Tools) != 1 {
			t.Fatalf("len(Tools) = %d, want 1", len(prepared.Tools))
		}
		if prepared.Tools[0].Name != "my_tool" {
			t.Errorf("Tools[0].Name = %q, want %q", prepared.Tools[0].Name, "my_tool")
		}
		if len(prepared.ToolNames) != 1 || prepared.ToolNames[0] != "my_tool" {
			t.Errorf("ToolNames = %v, want [my_tool]", prepared.ToolNames)
		}
	})

	t.Run("malformed tools returns error", func(t *testing.T) {
		content := "Hello"
		input := RunAgentInput{
			ThreadID: "thread-1",
			RunID:    "run-1",
			Messages: []events.Message{
				{ID: "msg-1", Role: "user", Content: &content},
			},
			// Invalid: tools should be objects, not strings
			Tools: []any{func() {}}, // Functions can't be marshaled
		}

		_, err := input.Prepare()
		if err == nil {
			t.Error("expected error for malformed tools")
		}
	})
}

func TestPreparedInput_GainsTools(t *testing.T) {
	t.Run("converts tools", func(t *testing.T) {
		prepared := &PreparedInput{
			Tools: []Tool{
				{Name: "tool1", Description: "desc1"},
				{Name: "tool2", Description: "desc2"},
			},
		}

		gainsTools := prepared.GainsTools()
		if len(gainsTools) != 2 {
			t.Fatalf("len(GainsTools) = %d, want 2", len(gainsTools))
		}
		if gainsTools[0].Name != "tool1" {
			t.Errorf("GainsTools[0].Name = %q, want %q", gainsTools[0].Name, "tool1")
		}
	})

	t.Run("empty tools returns nil", func(t *testing.T) {
		prepared := &PreparedInput{
			Tools: nil,
		}

		gainsTools := prepared.GainsTools()
		if gainsTools != nil {
			t.Errorf("GainsTools = %v, want nil", gainsTools)
		}
	})
}

func TestDecodeState(t *testing.T) {
	type MyState struct {
		Progress int      `json:"progress"`
		Items    []string `json:"items"`
	}

	t.Run("decodes state into struct", func(t *testing.T) {
		prepared := &PreparedInput{
			State: map[string]any{
				"progress": float64(50), // JSON numbers are float64
				"items":    []any{"a", "b"},
			},
		}

		state, err := DecodeState[MyState](prepared)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state.Progress != 50 {
			t.Errorf("Progress = %d, want 50", state.Progress)
		}
		if len(state.Items) != 2 || state.Items[0] != "a" {
			t.Errorf("Items = %v, want [a b]", state.Items)
		}
	})

	t.Run("nil state returns zero value", func(t *testing.T) {
		prepared := &PreparedInput{
			State: nil,
		}

		state, err := DecodeState[MyState](prepared)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state.Progress != 0 {
			t.Errorf("Progress = %d, want 0", state.Progress)
		}
		if state.Items != nil {
			t.Errorf("Items = %v, want nil", state.Items)
		}
	})

	t.Run("decodes into map", func(t *testing.T) {
		prepared := &PreparedInput{
			State: map[string]any{
				"key": "value",
			},
		}

		state, err := DecodeState[map[string]string](prepared)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state["key"] != "value" {
			t.Errorf("state[key] = %q, want %q", state["key"], "value")
		}
	})
}

func TestMustDecodeState(t *testing.T) {
	type MyState struct {
		Progress int `json:"progress"`
	}

	t.Run("returns decoded state", func(t *testing.T) {
		prepared := &PreparedInput{
			State: map[string]any{"progress": float64(100)},
		}

		state := MustDecodeState[MyState](prepared)
		if state.Progress != 100 {
			t.Errorf("Progress = %d, want 100", state.Progress)
		}
	})

	t.Run("panics on invalid state", func(t *testing.T) {
		prepared := &PreparedInput{
			State: func() {}, // Functions can't be marshaled
		}

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()

		MustDecodeState[MyState](prepared)
	})
}

func TestPreparedInput_State(t *testing.T) {
	t.Run("Prepare includes state", func(t *testing.T) {
		content := "Hello"
		input := RunAgentInput{
			ThreadID: "thread-1",
			RunID:    "run-1",
			Messages: []events.Message{
				{ID: "msg-1", Role: "user", Content: &content},
			},
			State: map[string]any{
				"progress": 0,
			},
		}

		prepared, err := input.Prepare()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if prepared.State == nil {
			t.Error("expected State to be set")
		}

		stateMap, ok := prepared.State.(map[string]any)
		if !ok {
			t.Fatalf("State is not map[string]any")
		}
		if stateMap["progress"] != 0 {
			t.Errorf("State[progress] = %v, want 0", stateMap["progress"])
		}
	})
}

func TestRunAgentInput_JSON(t *testing.T) {
	// Test that the struct marshals/unmarshals correctly
	jsonData := `{
		"thread_id": "thread-123",
		"run_id": "run-456",
		"messages": [
			{"id": "msg-1", "role": "user", "content": "Hello"}
		],
		"tools": [
			{"name": "search", "description": "Search the web"}
		]
	}`

	var input RunAgentInput
	if err := json.Unmarshal([]byte(jsonData), &input); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if input.ThreadID != "thread-123" {
		t.Errorf("ThreadID = %q, want %q", input.ThreadID, "thread-123")
	}
	if input.RunID != "run-456" {
		t.Errorf("RunID = %q, want %q", input.RunID, "run-456")
	}
	if len(input.Messages) != 1 {
		t.Errorf("len(Messages) = %d, want 1", len(input.Messages))
	}
	if len(input.Tools) != 1 {
		t.Errorf("len(Tools) = %d, want 1", len(input.Tools))
	}
}
