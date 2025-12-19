package agui

import (
	"context"
	"testing"
	"time"

	"github.com/spetersoncode/gains/agent"
)

func TestParseUserInputInput(t *testing.T) {
	t.Run("parse confirm response", func(t *testing.T) {
		data := []byte(`{"requestId":"req-123","confirmed":true}`)
		input, err := ParseUserInputInput(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if input.RequestID != "req-123" {
			t.Errorf("expected requestId 'req-123', got %q", input.RequestID)
		}
		if !input.Confirmed {
			t.Error("expected confirmed to be true")
		}
	})

	t.Run("parse text response", func(t *testing.T) {
		data := []byte(`{"requestId":"req-456","value":"user text"}`)
		input, err := ParseUserInputInput(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if input.Value != "user text" {
			t.Errorf("expected value 'user text', got %q", input.Value)
		}
	})

	t.Run("parse cancelled response", func(t *testing.T) {
		data := []byte(`{"requestId":"req-789","cancelled":true}`)
		input, err := ParseUserInputInput(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !input.Cancelled {
			t.Error("expected cancelled to be true")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		data := []byte(`{invalid}`)
		_, err := ParseUserInputInput(data)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestUserInputInput_ToResponse(t *testing.T) {
	input := &UserInputInput{
		RequestID: "req-123",
		Value:     "test value",
		Confirmed: true,
		Cancelled: false,
	}

	response := input.ToResponse()

	if response.RequestID != "req-123" {
		t.Errorf("expected RequestID 'req-123', got %q", response.RequestID)
	}
	if response.Value != "test value" {
		t.Errorf("expected Value 'test value', got %q", response.Value)
	}
	if !response.Confirmed {
		t.Error("expected Confirmed to be true")
	}
	if response.Cancelled {
		t.Error("expected Cancelled to be false")
	}
}

func TestHandleUserInput(t *testing.T) {
	broker := agent.NewUserInputBrokerWith(agent.WithInputTimeout(100 * time.Millisecond))

	// Track the request ID via callback
	var capturedReqID string
	broker2 := agent.NewUserInputBrokerWith(
		agent.WithInputTimeout(100*time.Millisecond),
		agent.WithOnInputSubmit(func(req agent.UserInputRequest) {
			capturedReqID = req.ID
		}),
	)

	// Start a request in background
	done := make(chan bool)
	go func() {
		broker2.RequestConfirm(context.Background(), "Test", "Test message")
		done <- true
	}()

	// Wait for callback to capture the ID
	time.Sleep(20 * time.Millisecond)

	if capturedReqID == "" {
		t.Fatal("no request ID captured")
	}

	input := &UserInputInput{
		RequestID: capturedReqID,
		Confirmed: true,
	}

	err := HandleUserInput(broker2, input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	<-done

	// Test error case: no pending request
	err = HandleUserInput(broker, &UserInputInput{RequestID: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent request")
	}
}

func TestHandleUserInputJSON(t *testing.T) {
	broker := agent.NewUserInputBrokerWith(agent.WithInputTimeout(100 * time.Millisecond))

	// Test error case: invalid JSON
	err := HandleUserInputJSON(broker, []byte(`{invalid}`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	// Test error case: no pending request
	err = HandleUserInputJSON(broker, []byte(`{"requestId":"nonexistent"}`))
	if err == nil {
		t.Error("expected error for nonexistent request")
	}
}
