package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	ai "github.com/spetersoncode/gains"
)

func TestApprovalBroker_ApproveReject(t *testing.T) {
	t.Run("approve via Decide", func(t *testing.T) {
		broker := NewApprovalBroker()
		approver := broker.Approver()

		var wg sync.WaitGroup
		wg.Add(1)

		var approved bool
		var reason string

		go func() {
			defer wg.Done()
			approved, reason = approver(context.Background(), ai.ToolCall{
				ID:   "call-123",
				Name: "test_tool",
			})
		}()

		// Give the goroutine time to register
		time.Sleep(10 * time.Millisecond)

		if err := broker.Decide(ApprovalDecision{
			ToolCallID: "call-123",
			Approved:   true,
		}); err != nil {
			t.Fatal(err)
		}

		wg.Wait()

		if !approved {
			t.Error("expected approved=true")
		}
		if reason != "" {
			t.Errorf("expected empty reason, got %q", reason)
		}
	})

	t.Run("reject via Decide", func(t *testing.T) {
		broker := NewApprovalBroker()
		approver := broker.Approver()

		var wg sync.WaitGroup
		wg.Add(1)

		var approved bool
		var reason string

		go func() {
			defer wg.Done()
			approved, reason = approver(context.Background(), ai.ToolCall{
				ID:   "call-456",
				Name: "dangerous_tool",
			})
		}()

		time.Sleep(10 * time.Millisecond)

		if err := broker.Reject("call-456", "too dangerous"); err != nil {
			t.Fatal(err)
		}

		wg.Wait()

		if approved {
			t.Error("expected approved=false")
		}
		if reason != "too dangerous" {
			t.Errorf("expected reason 'too dangerous', got %q", reason)
		}
	})

	t.Run("approve convenience method", func(t *testing.T) {
		broker := NewApprovalBroker()
		approver := broker.Approver()

		var wg sync.WaitGroup
		wg.Add(1)

		var approved bool

		go func() {
			defer wg.Done()
			approved, _ = approver(context.Background(), ai.ToolCall{
				ID:   "call-789",
				Name: "safe_tool",
			})
		}()

		time.Sleep(10 * time.Millisecond)

		if err := broker.Approve("call-789"); err != nil {
			t.Fatal(err)
		}

		wg.Wait()

		if !approved {
			t.Error("expected approved=true")
		}
	})
}

func TestApprovalBroker_PendingCount(t *testing.T) {
	broker := NewApprovalBroker()
	approver := broker.Approver()

	if broker.PendingCount() != 0 {
		t.Errorf("expected 0 pending, got %d", broker.PendingCount())
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		approver(context.Background(), ai.ToolCall{ID: "call-1"})
	}()

	go func() {
		defer wg.Done()
		approver(context.Background(), ai.ToolCall{ID: "call-2"})
	}()

	time.Sleep(20 * time.Millisecond)

	if broker.PendingCount() != 2 {
		t.Errorf("expected 2 pending, got %d", broker.PendingCount())
	}

	if !broker.HasPending() {
		t.Error("expected HasPending() to be true")
	}

	broker.Approve("call-1")
	broker.Approve("call-2")

	wg.Wait()

	if broker.PendingCount() != 0 {
		t.Errorf("expected 0 pending after approval, got %d", broker.PendingCount())
	}
}

func TestApprovalBroker_Timeout(t *testing.T) {
	broker := NewApprovalBrokerWith(
		WithApprovalTimeout(50 * time.Millisecond),
	)
	approver := broker.Approver()

	approved, reason := approver(context.Background(), ai.ToolCall{
		ID:   "timeout-call",
		Name: "slow_tool",
	})

	if approved {
		t.Error("expected approved=false on timeout")
	}
	if reason != "approval timeout" {
		t.Errorf("expected 'approval timeout', got %q", reason)
	}
}

func TestApprovalBroker_ContextCancellation(t *testing.T) {
	broker := NewApprovalBroker()
	approver := broker.Approver()

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	var approved bool
	var reason string

	go func() {
		defer wg.Done()
		approved, reason = approver(ctx, ai.ToolCall{
			ID:   "cancel-call",
			Name: "cancelled_tool",
		})
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	wg.Wait()

	if approved {
		t.Error("expected approved=false on cancellation")
	}
	if reason != "approval cancelled" {
		t.Errorf("expected 'approval cancelled', got %q", reason)
	}
}

func TestApprovalBroker_DecideNoPending(t *testing.T) {
	broker := NewApprovalBroker()

	err := broker.Decide(ApprovalDecision{
		ToolCallID: "nonexistent",
		Approved:   true,
	})

	if err == nil {
		t.Error("expected error for nonexistent tool call")
	}
}

func TestApprovalBroker_OnSubmitCallback(t *testing.T) {
	var submittedCalls []ai.ToolCall
	var mu sync.Mutex

	broker := NewApprovalBrokerWith(
		WithApprovalTimeout(50 * time.Millisecond),
		WithOnSubmit(func(call ai.ToolCall) {
			mu.Lock()
			submittedCalls = append(submittedCalls, call)
			mu.Unlock()
		}),
	)
	approver := broker.Approver()

	call := ai.ToolCall{
		ID:   "callback-call",
		Name: "callback_tool",
	}

	// This will timeout, but the callback should still be called
	approver(context.Background(), call)

	mu.Lock()
	defer mu.Unlock()

	if len(submittedCalls) != 1 {
		t.Fatalf("expected 1 submitted call, got %d", len(submittedCalls))
	}
	if submittedCalls[0].ID != "callback-call" {
		t.Errorf("expected call ID 'callback-call', got %q", submittedCalls[0].ID)
	}
}
