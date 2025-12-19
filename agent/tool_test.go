package agent

import (
	"context"
	"testing"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/tool"
)

// mockChatClient is a simple mock for testing tool execution.
type mockChatClient struct {
	response *ai.Response
	err      error
}

func (m *mockChatClient) Chat(ctx context.Context, messages []ai.Message, opts ...ai.Option) (*ai.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockChatClient) ChatStream(ctx context.Context, messages []ai.Message, opts ...ai.Option) (<-chan event.Event, error) {
	ch := make(chan event.Event, 10)
	go func() {
		defer close(ch)
		event.Emit(ch, event.Event{Type: event.MessageStart, MessageID: "test-msg"})
		event.Emit(ch, event.Event{Type: event.MessageDelta, MessageID: "test-msg", Delta: "Hello"})
		event.Emit(ch, event.Event{Type: event.MessageEnd, MessageID: "test-msg", Response: m.response})
	}()
	return ch, nil
}

func TestNewTool_Basic(t *testing.T) {
	mockClient := &mockChatClient{
		response: &ai.Response{Content: "Test response"},
	}
	registry := tool.NewRegistry()
	subAgent := New(mockClient, registry)

	toolReg := NewTool("test-agent", subAgent,
		WithToolDescription("Test agent tool"),
		WithToolMaxSteps(3),
	)

	if toolReg.Tool.Name != "test-agent" {
		t.Errorf("expected tool name 'test-agent', got %q", toolReg.Tool.Name)
	}
	if toolReg.Tool.Description != "Test agent tool" {
		t.Errorf("unexpected description: %q", toolReg.Tool.Description)
	}
}

func TestWithToolEventForwarding_NoForwardChannel(t *testing.T) {
	mockClient := &mockChatClient{
		response: &ai.Response{Content: "Test response"},
	}
	registry := tool.NewRegistry()
	subAgent := New(mockClient, registry)

	toolReg := NewTool("test-agent", subAgent,
		WithToolEventForwarding(), // Enable forwarding
	)

	// Call without forwarding channel in context - should still work
	ctx := context.Background()
	result, err := toolReg.Handler(ctx, ai.ToolCall{
		ID:        "call-1",
		Name:      "test-agent",
		Arguments: `{"query": "test"}`,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Test response" {
		t.Errorf("expected 'Test response', got %q", result)
	}
}

func TestWithToolEventForwarding_WithForwardChannel(t *testing.T) {
	mockClient := &mockChatClient{
		response: &ai.Response{Content: "Streamed response"},
	}
	registry := tool.NewRegistry()
	subAgent := New(mockClient, registry)

	toolReg := NewTool("test-agent", subAgent,
		WithToolEventForwarding(),
	)

	// Create a forwarding channel and add it to context
	forwardCh := make(chan event.Event, 100)
	ctx := event.WithForwardChannel(context.Background(), forwardCh)

	// Run the tool handler
	result, err := toolReg.Handler(ctx, ai.ToolCall{
		ID:        "call-1",
		Name:      "test-agent",
		Arguments: `{"query": "test"}`,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Streamed response" {
		t.Errorf("expected 'Streamed response', got %q", result)
	}

	// Collect forwarded events
	close(forwardCh)
	var events []event.Event
	for ev := range forwardCh {
		events = append(events, ev)
	}

	// Should have received some events
	if len(events) == 0 {
		t.Error("expected forwarded events, got none")
	}

	// Should have RunStart and RunEnd at minimum
	hasRunStart := false
	hasRunEnd := false
	for _, ev := range events {
		if ev.Type == event.RunStart {
			hasRunStart = true
		}
		if ev.Type == event.RunEnd {
			hasRunEnd = true
		}
	}

	if !hasRunStart {
		t.Error("expected RunStart event to be forwarded")
	}
	if !hasRunEnd {
		t.Error("expected RunEnd event to be forwarded")
	}
}

func TestWithToolEventForwarding_Disabled(t *testing.T) {
	mockClient := &mockChatClient{
		response: &ai.Response{Content: "Test response"},
	}
	registry := tool.NewRegistry()
	subAgent := New(mockClient, registry)

	// Create tool WITHOUT event forwarding
	toolReg := NewTool("test-agent", subAgent)

	// Add forwarding channel to context
	forwardCh := make(chan event.Event, 100)
	ctx := event.WithForwardChannel(context.Background(), forwardCh)

	// Run the tool handler
	result, err := toolReg.Handler(ctx, ai.ToolCall{
		ID:        "call-1",
		Name:      "test-agent",
		Arguments: `{"query": "test"}`,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Test response" {
		t.Errorf("expected 'Test response', got %q", result)
	}

	// Should NOT have received any events since forwarding is disabled
	close(forwardCh)
	var events []event.Event
	for ev := range forwardCh {
		events = append(events, ev)
	}

	if len(events) != 0 {
		t.Errorf("expected no forwarded events when disabled, got %d", len(events))
	}
}

func TestEventContextHelpers(t *testing.T) {
	t.Run("WithForwardChannel and ForwardChannelFromContext", func(t *testing.T) {
		ch := make(chan event.Event, 10)
		ctx := event.WithForwardChannel(context.Background(), ch)

		retrieved := event.ForwardChannelFromContext(ctx)
		if retrieved == nil {
			t.Fatal("expected non-nil channel from context")
		}

		// Verify it's the same channel by sending an event
		go func() {
			retrieved <- event.Event{Type: event.RunStart}
		}()

		ev := <-ch
		if ev.Type != event.RunStart {
			t.Errorf("expected RunStart, got %v", ev.Type)
		}
	})

	t.Run("ForwardChannelFromContext returns nil when not set", func(t *testing.T) {
		ctx := context.Background()
		ch := event.ForwardChannelFromContext(ctx)
		if ch != nil {
			t.Error("expected nil channel from empty context")
		}
	})
}
