package a2a

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRemoteTool_Tool(t *testing.T) {
	client := NewClient("http://example.com")
	rt := NewRemoteTool(client,
		WithToolName("my_agent"),
		WithToolDescription("Call my agent"),
	)

	tool := rt.Tool()

	if tool.Name != "my_agent" {
		t.Errorf("expected 'my_agent', got %q", tool.Name)
	}
	if tool.Description != "Call my agent" {
		t.Errorf("expected 'Call my agent', got %q", tool.Description)
	}
	if len(tool.Parameters) == 0 {
		t.Errorf("expected non-empty parameters")
	}
}

func TestRemoteTool_Handler(t *testing.T) {
	t.Run("successful call", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			task := &Task{
				ID:     "task-1",
				Status: NewTaskStatus(TaskStateCompleted),
			}
			msg := NewMessage(MessageRoleAgent, NewTextPart("The answer is 42"))
			task.Status.Message = &msg

			resp := jsonRPCResponse{JSONRPC: "2.0", ID: "1"}
			resp.Result, _ = json.Marshal(task)
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		rt := NewRemoteTool(client)
		handler := rt.Handler()

		input := json.RawMessage(`{"query": "What is the answer?"}`)
		result, err := handler(context.Background(), input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "The answer is 42" {
			t.Errorf("expected 'The answer is 42', got %q", result)
		}
	})

	t.Run("failed task", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			task := &Task{
				ID:     "task-1",
				Status: NewTaskStatus(TaskStateFailed),
			}
			msg := NewMessage(MessageRoleAgent, NewTextPart("Something went wrong"))
			task.Status.Message = &msg

			resp := jsonRPCResponse{JSONRPC: "2.0", ID: "1"}
			resp.Result, _ = json.Marshal(task)
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		rt := NewRemoteTool(client)
		handler := rt.Handler()

		input := json.RawMessage(`{"query": "Do something"}`)
		result, err := handler(context.Background(), input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "Remote agent failed: Something went wrong" {
			t.Errorf("unexpected result: %q", result)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		client := NewClient("http://example.com")
		rt := NewRemoteTool(client)
		handler := rt.Handler()

		input := json.RawMessage(`invalid json`)
		_, err := handler(context.Background(), input)

		if err == nil {
			t.Error("expected error for invalid input")
		}
	})

	t.Run("request error", func(t *testing.T) {
		client := NewClient("http://invalid.example.com:99999")
		rt := NewRemoteTool(client)
		handler := rt.Handler()

		input := json.RawMessage(`{"query": "test"}`)
		_, err := handler(context.Background(), input)

		if err == nil {
			t.Error("expected error for failed request")
		}
	})
}

func TestRemoteTool_Options(t *testing.T) {
	client := NewClient("http://example.com")

	customSchema := json.RawMessage(`{"type":"object","properties":{"custom":{"type":"string"}}}`)

	rt := NewRemoteTool(client,
		WithToolName("custom_tool"),
		WithToolDescription("A custom tool"),
		WithToolSchema(customSchema),
	)

	tool := rt.Tool()

	if tool.Name != "custom_tool" {
		t.Errorf("expected 'custom_tool', got %q", tool.Name)
	}
	if tool.Description != "A custom tool" {
		t.Errorf("expected 'A custom tool', got %q", tool.Description)
	}

	// Verify custom schema was used
	var schema map[string]any
	if err := json.Unmarshal(tool.Parameters, &schema); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in schema")
	}
	if _, ok := props["custom"]; !ok {
		t.Error("expected custom property in schema")
	}
}

// mockRegistry implements ToolRegistry for testing.
type mockRegistry struct {
	registered map[string]bool
}

func (m *mockRegistry) RegisterFunc(name, description string, schema json.RawMessage, handler func(ctx context.Context, input json.RawMessage) (string, error)) {
	if m.registered == nil {
		m.registered = make(map[string]bool)
	}
	m.registered[name] = true
}

func TestRemoteTool_Register(t *testing.T) {
	client := NewClient("http://example.com")
	rt := NewRemoteTool(client, WithToolName("my_remote_tool"))

	registry := &mockRegistry{}
	rt.Register(registry)

	if !registry.registered["my_remote_tool"] {
		t.Error("tool was not registered")
	}
}
