package gains

// Role represents the role of a message sender in a conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    Role
	Content string
}

// Response represents a complete response from a chat provider.
type Response struct {
	Content      string
	FinishReason string
	Usage        Usage
}

// Usage contains token usage information for a request.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// StreamEvent represents a single event in a streaming response.
type StreamEvent struct {
	// Delta contains the incremental content for this event.
	Delta string
	// Done indicates if this is the final event in the stream.
	Done bool
	// Response contains the final response data when Done is true.
	Response *Response
	// Err contains any error that occurred during streaming.
	Err error
}
