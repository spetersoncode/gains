package agui

import (
	"context"
	"sync"
	"testing"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
)

func TestParseApprovalInput(t *testing.T) {
	t.Run("parses approval", func(t *testing.T) {
		data := []byte(`{"toolCallId": "call-123", "approved": true}`)
		input, err := ParseApprovalInput(data)
		if err != nil {
			t.Fatal(err)
		}
		if input.ToolCallID != "call-123" {
			t.Errorf("expected toolCallId 'call-123', got %q", input.ToolCallID)
		}
		if !input.Approved {
			t.Error("expected approved=true")
		}
	})

	t.Run("parses rejection with reason", func(t *testing.T) {
		data := []byte(`{"toolCallId": "call-456", "approved": false, "reason": "too risky"}`)
		input, err := ParseApprovalInput(data)
		if err != nil {
			t.Fatal(err)
		}
		if input.ToolCallID != "call-456" {
			t.Errorf("expected toolCallId 'call-456', got %q", input.ToolCallID)
		}
		if input.Approved {
			t.Error("expected approved=false")
		}
		if input.Reason != "too risky" {
			t.Errorf("expected reason 'too risky', got %q", input.Reason)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		data := []byte(`{invalid}`)
		_, err := ParseApprovalInput(data)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestApprovalInput_ToDecision(t *testing.T) {
	input := &ApprovalInput{
		ToolCallID: "call-789",
		Approved:   false,
		Reason:     "blocked by policy",
	}

	decision := input.ToDecision()

	if decision.ToolCallID != "call-789" {
		t.Errorf("expected ToolCallID 'call-789', got %q", decision.ToolCallID)
	}
	if decision.Approved {
		t.Error("expected Approved=false")
	}
	if decision.Reason != "blocked by policy" {
		t.Errorf("expected Reason 'blocked by policy', got %q", decision.Reason)
	}
}

func TestHandleApproval(t *testing.T) {
	broker := agent.NewApprovalBroker()
	approver := broker.Approver()

	var wg sync.WaitGroup
	wg.Add(1)

	var approved bool

	go func() {
		defer wg.Done()
		approved, _ = approver(context.Background(), ai.ToolCall{
			ID:   "handle-test",
			Name: "test_tool",
		})
	}()

	time.Sleep(10 * time.Millisecond)

	err := HandleApproval(broker, &ApprovalInput{
		ToolCallID: "handle-test",
		Approved:   true,
	})
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	if !approved {
		t.Error("expected approved=true")
	}
}

func TestHandleApprovalJSON(t *testing.T) {
	broker := agent.NewApprovalBroker()
	approver := broker.Approver()

	var wg sync.WaitGroup
	wg.Add(1)

	var approved bool
	var reason string

	go func() {
		defer wg.Done()
		approved, reason = approver(context.Background(), ai.ToolCall{
			ID:   "json-test",
			Name: "test_tool",
		})
	}()

	time.Sleep(10 * time.Millisecond)

	data := []byte(`{"toolCallId": "json-test", "approved": false, "reason": "user rejected"}`)
	err := HandleApprovalJSON(broker, data)
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	if approved {
		t.Error("expected approved=false")
	}
	if reason != "user rejected" {
		t.Errorf("expected reason 'user rejected', got %q", reason)
	}
}
