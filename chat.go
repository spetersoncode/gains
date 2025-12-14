package gains

import "context"

// ChatProvider defines the interface for AI chat providers.
type ChatProvider interface {
	// Chat sends a conversation and returns a complete response.
	Chat(ctx context.Context, messages []Message, opts ...Option) (*Response, error)

	// ChatStream sends a conversation and returns a channel of streaming events.
	// The channel is closed when the stream is complete or an error occurs.
	// Callers should check StreamEvent.Err for any errors.
	ChatStream(ctx context.Context, messages []Message, opts ...Option) (<-chan StreamEvent, error)
}
