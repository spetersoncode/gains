// Package chat provides the canonical ChatClient interface.
//
// This package exists to provide a unified interface that can be used across
// agent, workflow, and tool packages without import cycles. The interface
// combines both blocking Chat and streaming ChatStream methods.
//
// The [github.com/spetersoncode/gains/client.Client] type implements this interface.
package chat

import (
	"context"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
)

// Client defines the interface for high-level chat clients.
// This is the canonical interface used by agent, workflow, and tool packages.
//
// It provides both blocking (Chat) and streaming (ChatStream) methods.
// The streaming method returns rich [event.Event] with full lifecycle events
// (message start/delta/end, tool calls, etc).
type Client interface {
	// Chat sends a conversation and returns a complete response.
	Chat(ctx context.Context, messages []ai.Message, opts ...ai.Option) (*ai.Response, error)

	// ChatStream sends a conversation and returns a channel of rich events.
	ChatStream(ctx context.Context, messages []ai.Message, opts ...ai.Option) (<-chan event.Event, error)
}
