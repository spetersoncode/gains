package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	ai "github.com/spetersoncode/gains"
)

// ApprovalDecision represents a user's decision on a tool call.
type ApprovalDecision struct {
	ToolCallID string // ID of the tool call being approved/rejected
	Approved   bool   // Whether the tool call was approved
	Reason     string // Reason for rejection (empty if approved)
}

// ApprovalBroker manages async tool approvals for AG-UI integration.
// It receives approval decisions from the frontend via a channel and
// routes them to waiting approval requests.
//
// Usage:
//
//	broker := agent.NewApprovalBroker()
//	go func() {
//	    for decision := range frontendDecisions {
//	        broker.Decide(decision)
//	    }
//	}()
//	result, err := agent.Run(ctx, messages,
//	    agent.WithApprover(broker.Approver()),
//	)
type ApprovalBroker struct {
	mu       sync.Mutex
	pending  map[string]chan ApprovalDecision
	timeout  time.Duration
	onSubmit func(call ai.ToolCall) // Called when approval is submitted
}

// NewApprovalBroker creates a new ApprovalBroker.
// The default timeout for waiting on decisions is 5 minutes.
func NewApprovalBroker() *ApprovalBroker {
	return &ApprovalBroker{
		pending: make(map[string]chan ApprovalDecision),
		timeout: 5 * time.Minute,
	}
}

// ApprovalBrokerOption configures an ApprovalBroker.
type ApprovalBrokerOption func(*ApprovalBroker)

// WithApprovalTimeout sets the timeout for waiting on approval decisions.
func WithApprovalTimeout(d time.Duration) ApprovalBrokerOption {
	return func(b *ApprovalBroker) {
		b.timeout = d
	}
}

// WithOnSubmit sets a callback that's called when an approval request is submitted.
// This is useful for tracking or logging approval requests.
func WithOnSubmit(fn func(call ai.ToolCall)) ApprovalBrokerOption {
	return func(b *ApprovalBroker) {
		b.onSubmit = fn
	}
}

// NewApprovalBrokerWith creates a new ApprovalBroker with options.
func NewApprovalBrokerWith(opts ...ApprovalBrokerOption) *ApprovalBroker {
	b := NewApprovalBroker()
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Approver returns an ApproverFunc that can be used with WithApprover.
// The returned function blocks until a decision is received or the context
// is cancelled.
func (b *ApprovalBroker) Approver() ApproverFunc {
	return func(ctx context.Context, call ai.ToolCall) (bool, string) {
		return b.waitForDecision(ctx, call)
	}
}

// Decide sends an approval decision to the broker.
// The decision will be routed to the waiting approval request for the
// specified tool call ID.
//
// Returns an error if there is no pending approval for the given tool call ID.
func (b *ApprovalBroker) Decide(decision ApprovalDecision) error {
	b.mu.Lock()
	ch, ok := b.pending[decision.ToolCallID]
	b.mu.Unlock()

	if !ok {
		return fmt.Errorf("no pending approval for tool call %q", decision.ToolCallID)
	}

	// Non-blocking send - if the channel is already closed or full, it's a no-op
	select {
	case ch <- decision:
	default:
	}

	return nil
}

// Approve is a convenience method to approve a tool call.
func (b *ApprovalBroker) Approve(toolCallID string) error {
	return b.Decide(ApprovalDecision{
		ToolCallID: toolCallID,
		Approved:   true,
	})
}

// Reject is a convenience method to reject a tool call.
func (b *ApprovalBroker) Reject(toolCallID, reason string) error {
	return b.Decide(ApprovalDecision{
		ToolCallID: toolCallID,
		Approved:   false,
		Reason:     reason,
	})
}

// PendingCount returns the number of pending approval requests.
func (b *ApprovalBroker) PendingCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pending)
}

// HasPending returns true if there are any pending approval requests.
func (b *ApprovalBroker) HasPending() bool {
	return b.PendingCount() > 0
}

// waitForDecision registers a pending approval and waits for a decision.
func (b *ApprovalBroker) waitForDecision(ctx context.Context, call ai.ToolCall) (bool, string) {
	// Create a channel for this specific tool call
	ch := make(chan ApprovalDecision, 1)

	// Register the pending approval
	b.mu.Lock()
	b.pending[call.ID] = ch
	b.mu.Unlock()

	// Clean up when done
	defer func() {
		b.mu.Lock()
		delete(b.pending, call.ID)
		close(ch)
		b.mu.Unlock()
	}()

	// Call the onSubmit callback if set
	if b.onSubmit != nil {
		b.onSubmit(call)
	}

	// Wait for decision with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	select {
	case decision := <-ch:
		return decision.Approved, decision.Reason
	case <-timeoutCtx.Done():
		if ctx.Err() != nil {
			return false, "approval cancelled"
		}
		return false, "approval timeout"
	}
}
