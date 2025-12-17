package gains

import "github.com/google/uuid"

// Role represents the role of a message sender in a conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// ContentPartType represents the type of content in a multimodal message part.
type ContentPartType string

const (
	ContentPartTypeText  ContentPartType = "text"
	ContentPartTypeImage ContentPartType = "image"
)

// ContentPart represents a single part of multimodal content.
// Use either Text (for text parts) or ImageURL/Base64 (for image parts).
type ContentPart struct {
	// Type indicates the content type: "text" or "image".
	Type ContentPartType `json:"type"`
	// Text contains the text content. Only used when Type is "text".
	Text string `json:"text,omitempty"`
	// ImageURL contains a URL to an image. Only used when Type is "image".
	// Mutually exclusive with Base64.
	ImageURL string `json:"imageUrl,omitempty"`
	// Base64 contains base64-encoded image data. Only used when Type is "image".
	// Mutually exclusive with ImageURL.
	Base64 string `json:"base64,omitempty"`
	// MimeType specifies the image format (e.g., "image/jpeg", "image/png").
	// Required when using Base64, optional for ImageURL (may be inferred).
	MimeType string `json:"mimeType,omitempty"`
}

// NewTextPart creates a text content part.
func NewTextPart(text string) ContentPart {
	return ContentPart{
		Type: ContentPartTypeText,
		Text: text,
	}
}

// NewImageURLPart creates an image content part from a URL.
func NewImageURLPart(url string) ContentPart {
	return ContentPart{
		Type:     ContentPartTypeImage,
		ImageURL: url,
	}
}

// NewImageBase64Part creates an image content part from base64 data.
func NewImageBase64Part(base64Data, mimeType string) ContentPart {
	return ContentPart{
		Type:     ContentPartTypeImage,
		Base64:   base64Data,
		MimeType: mimeType,
	}
}

// Message represents a single message in a conversation.
type Message struct {
	// ID is an optional unique identifier for the message.
	// Used for message correlation and AG-UI protocol compatibility.
	ID      string `json:"id,omitempty"`
	Role    Role   `json:"role"`
	Content string `json:"content,omitempty"`
	// Parts contains multimodal content parts (text, images).
	// If populated, Content is ignored for providers that support multimodal.
	Parts []ContentPart `json:"parts,omitempty"`
	// ToolCalls contains tool invocation requests from an assistant message.
	// Only populated when Role is RoleAssistant and the model wants to use tools.
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
	// ToolResults contains results from tool executions.
	// Only populated when Role is RoleTool.
	ToolResults []ToolResult `json:"toolResults,omitempty"`
}

// GenerateMessageID creates a unique message identifier.
func GenerateMessageID() string {
	return "msg-" + uuid.New().String()
}

// HasParts returns true if the message has multimodal content parts.
func (m Message) HasParts() bool {
	return len(m.Parts) > 0
}

// Response represents a complete response from a chat provider.
type Response struct {
	Content      string `json:"content,omitempty"`
	FinishReason string `json:"finishReason,omitempty"`
	Usage        Usage  `json:"usage"`
	// ToolCalls contains any tool invocation requests from the model.
	// Check if len(ToolCalls) > 0 to determine if tools should be executed.
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
}

// Usage contains token usage information for a request.
type Usage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
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
