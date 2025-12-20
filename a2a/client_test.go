package a2a

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_SendMessage(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected JSON content type")
			}

			// Parse request
			var req jsonRPCRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}

			if req.Method != "tasks/send" {
				t.Errorf("expected tasks/send, got %s", req.Method)
			}

			// Send response
			task := &Task{
				Kind:      "task",
				ID:        "task-123",
				ContextID: "ctx-1",
				Status:    NewTaskStatus(TaskStateCompleted),
			}
			msg := NewMessage(MessageRoleAgent, NewTextPart("Hello back!"))
			task.Status.Message = &msg

			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
			}
			resp.Result, _ = json.Marshal(task)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// Create client
		client := NewClient(server.URL)

		// Send message
		task, err := client.SendMessage(context.Background(), SendMessageRequest{
			Message: NewMessage(MessageRoleUser, NewTextPart("Hello")),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if task.ID != "task-123" {
			t.Errorf("expected task-123, got %s", task.ID)
		}
		if task.Status.State != TaskStateCompleted {
			t.Errorf("expected completed, got %v", task.Status.State)
		}
		if task.Status.Message == nil || task.Status.Message.TextContent() != "Hello back!" {
			t.Errorf("unexpected message content")
		}
	})

	t.Run("RPC error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      "1",
				Error: &jsonRPCError{
					Code:    -32600,
					Message: "Invalid request",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		_, err := client.SendText(context.Background(), "Hello")

		if err == nil {
			t.Fatal("expected error")
		}
		if err.Error() != "RPC error -32600: Invalid request" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(server.URL)
		_, err := client.SendText(context.Background(), "Hello")

		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestClient_SendText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify params contains a text part
		var params SendMessageRequest
		paramsJSON, _ := json.Marshal(req.Params)
		json.Unmarshal(paramsJSON, &params)

		if params.Message.TextContent() != "Test query" {
			t.Errorf("expected 'Test query', got %q", params.Message.TextContent())
		}

		// Return task
		task := &Task{ID: "task-1", Status: NewTaskStatus(TaskStateCompleted)}
		resp := jsonRPCResponse{JSONRPC: "2.0", ID: req.ID}
		resp.Result, _ = json.Marshal(task)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	task, err := client.SendText(context.Background(), "Test query")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ID != "task-1" {
		t.Errorf("expected task-1, got %s", task.ID)
	}
}

func TestClient_WithHTTPClient(t *testing.T) {
	customClient := &http.Client{}
	client := NewClient("http://example.com", WithHTTPClient(customClient))

	if client.httpClient != customClient {
		t.Error("custom HTTP client not set")
	}
}
