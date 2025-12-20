package a2a

import (
	"testing"

	ai "github.com/spetersoncode/gains"
)

func TestToGainsMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    Message
		wantRole ai.Role
		wantText string
	}{
		{
			name: "user text message",
			input: NewMessage(MessageRoleUser,
				NewTextPart("Hello, world!"),
			),
			wantRole: ai.RoleUser,
			wantText: "Hello, world!",
		},
		{
			name: "agent text message",
			input: NewMessage(MessageRoleAgent,
				NewTextPart("Hi there!"),
			),
			wantRole: ai.RoleAssistant,
			wantText: "Hi there!",
		},
		{
			name: "multi-part text message",
			input: NewMessage(MessageRoleUser,
				NewTextPart("Hello, "),
				NewTextPart("world!"),
			),
			wantRole: ai.RoleUser,
			wantText: "Hello, world!",
		},
		{
			name: "message with data part ignored",
			input: NewMessage(MessageRoleUser,
				NewTextPart("Check this: "),
				NewDataPart(map[string]any{"key": "value"}),
			),
			wantRole: ai.RoleUser,
			wantText: "Check this: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToGainsMessage(tt.input)
			if got.Role != tt.wantRole {
				t.Errorf("Role = %v, want %v", got.Role, tt.wantRole)
			}
			if got.Content != tt.wantText {
				t.Errorf("Content = %q, want %q", got.Content, tt.wantText)
			}
		})
	}
}

func TestFromGainsMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    ai.Message
		wantRole MessageRole
		wantText string
	}{
		{
			name: "user message",
			input: ai.Message{
				Role:    ai.RoleUser,
				Content: "Hello!",
			},
			wantRole: MessageRoleUser,
			wantText: "Hello!",
		},
		{
			name: "assistant message",
			input: ai.Message{
				Role:    ai.RoleAssistant,
				Content: "Hi there!",
			},
			wantRole: MessageRoleAgent,
			wantText: "Hi there!",
		},
		{
			name: "system message maps to agent",
			input: ai.Message{
				Role:    ai.RoleSystem,
				Content: "You are a helpful assistant.",
			},
			wantRole: MessageRoleAgent,
			wantText: "You are a helpful assistant.",
		},
		{
			name: "tool message maps to agent",
			input: ai.Message{
				Role:    ai.RoleTool,
				Content: "Tool result",
			},
			wantRole: MessageRoleAgent,
			wantText: "Tool result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromGainsMessage(tt.input)
			if got.Role != tt.wantRole {
				t.Errorf("Role = %v, want %v", got.Role, tt.wantRole)
			}
			if len(got.Parts) == 0 {
				if tt.wantText != "" {
					t.Errorf("Parts empty, want text %q", tt.wantText)
				}
				return
			}
			tp, ok := got.Parts[0].(TextPart)
			if !ok {
				t.Errorf("First part is not TextPart")
				return
			}
			if tp.Text != tt.wantText {
				t.Errorf("Text = %q, want %q", tp.Text, tt.wantText)
			}
		})
	}
}

func TestToGainsMessages(t *testing.T) {
	a2aMsgs := []Message{
		NewMessage(MessageRoleUser, NewTextPart("Hi")),
		NewMessage(MessageRoleAgent, NewTextPart("Hello")),
		NewMessage(MessageRoleUser, NewTextPart("How are you?")),
	}

	gainsMsgs := ToGainsMessages(a2aMsgs)

	if len(gainsMsgs) != 3 {
		t.Fatalf("len = %d, want 3", len(gainsMsgs))
	}

	expected := []struct {
		role    ai.Role
		content string
	}{
		{ai.RoleUser, "Hi"},
		{ai.RoleAssistant, "Hello"},
		{ai.RoleUser, "How are you?"},
	}

	for i, want := range expected {
		if gainsMsgs[i].Role != want.role {
			t.Errorf("msg[%d].Role = %v, want %v", i, gainsMsgs[i].Role, want.role)
		}
		if gainsMsgs[i].Content != want.content {
			t.Errorf("msg[%d].Content = %q, want %q", i, gainsMsgs[i].Content, want.content)
		}
	}
}

func TestFromGainsMessages(t *testing.T) {
	gainsMsgs := []ai.Message{
		{Role: ai.RoleUser, Content: "Hi"},
		{Role: ai.RoleAssistant, Content: "Hello"},
	}

	a2aMsgs := FromGainsMessages(gainsMsgs)

	if len(a2aMsgs) != 2 {
		t.Fatalf("len = %d, want 2", len(a2aMsgs))
	}

	if a2aMsgs[0].Role != MessageRoleUser {
		t.Errorf("msg[0].Role = %v, want %v", a2aMsgs[0].Role, MessageRoleUser)
	}
	if a2aMsgs[1].Role != MessageRoleAgent {
		t.Errorf("msg[1].Role = %v, want %v", a2aMsgs[1].Role, MessageRoleAgent)
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that converting A2A -> gains -> A2A preserves meaning
	original := NewMessage(MessageRoleUser,
		NewTextPart("Hello, how can you help me?"),
	)

	gainsMsg := ToGainsMessage(original)
	roundTrip := FromGainsMessage(gainsMsg)

	if roundTrip.Role != original.Role {
		t.Errorf("Role changed: got %v, want %v", roundTrip.Role, original.Role)
	}

	if original.TextContent() != roundTrip.TextContent() {
		t.Errorf("Text changed: got %q, want %q", roundTrip.TextContent(), original.TextContent())
	}
}

func TestToolCallConversion(t *testing.T) {
	// Test gains message with tool calls converts to A2A data parts
	gainsMsg := ai.Message{
		Role: ai.RoleAssistant,
		ToolCalls: []ai.ToolCall{
			{
				ID:        "call-123",
				Name:      "get_weather",
				Arguments: `{"location": "NYC"}`,
			},
		},
	}

	a2aMsg := FromGainsMessage(gainsMsg)

	if len(a2aMsg.Parts) != 1 {
		t.Fatalf("Parts len = %d, want 1", len(a2aMsg.Parts))
	}

	dataPart, ok := a2aMsg.Parts[0].(DataPart)
	if !ok {
		t.Fatal("Part is not DataPart")
	}

	data, ok := dataPart.Data.(map[string]any)
	if !ok {
		t.Fatal("DataPart.Data is not map")
	}

	if data["type"] != "tool_call" {
		t.Errorf("type = %v, want tool_call", data["type"])
	}

	tc, ok := data["tool_call"].(map[string]any)
	if !ok {
		t.Fatal("tool_call is not map")
	}

	if tc["id"] != "call-123" {
		t.Errorf("id = %v, want call-123", tc["id"])
	}
	if tc["name"] != "get_weather" {
		t.Errorf("name = %v, want get_weather", tc["name"])
	}
}

func TestToolResultConversion(t *testing.T) {
	// Test gains message with tool results converts to A2A data parts
	gainsMsg := ai.Message{
		Role: ai.RoleTool,
		ToolResults: []ai.ToolResult{
			{
				ToolCallID: "call-123",
				Content:    "Sunny, 72F",
				IsError:    false,
			},
		},
	}

	a2aMsg := FromGainsMessage(gainsMsg)

	if len(a2aMsg.Parts) != 1 {
		t.Fatalf("Parts len = %d, want 1", len(a2aMsg.Parts))
	}

	dataPart, ok := a2aMsg.Parts[0].(DataPart)
	if !ok {
		t.Fatal("Part is not DataPart")
	}

	data, ok := dataPart.Data.(map[string]any)
	if !ok {
		t.Fatal("DataPart.Data is not map")
	}

	if data["type"] != "tool_result" {
		t.Errorf("type = %v, want tool_result", data["type"])
	}

	tr, ok := data["tool_result"].(map[string]any)
	if !ok {
		t.Fatal("tool_result is not map")
	}

	if tr["tool_call_id"] != "call-123" {
		t.Errorf("tool_call_id = %v, want call-123", tr["tool_call_id"])
	}
	if tr["content"] != "Sunny, 72F" {
		t.Errorf("content = %v, want Sunny, 72F", tr["content"])
	}
}
