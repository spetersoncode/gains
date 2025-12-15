package gains

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoleConstants(t *testing.T) {
	assert.Equal(t, Role("user"), RoleUser)
	assert.Equal(t, Role("assistant"), RoleAssistant)
	assert.Equal(t, Role("system"), RoleSystem)
	assert.Equal(t, Role("tool"), RoleTool)
}

func TestContentPartTypeConstants(t *testing.T) {
	assert.Equal(t, ContentPartType("text"), ContentPartTypeText)
	assert.Equal(t, ContentPartType("image"), ContentPartTypeImage)
}

func TestNewTextPart(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected ContentPart
	}{
		{
			name: "creates text part",
			text: "Hello, world!",
			expected: ContentPart{
				Type: ContentPartTypeText,
				Text: "Hello, world!",
			},
		},
		{
			name: "handles empty string",
			text: "",
			expected: ContentPart{
				Type: ContentPartTypeText,
				Text: "",
			},
		},
		{
			name: "handles multiline text",
			text: "line1\nline2\nline3",
			expected: ContentPart{
				Type: ContentPartTypeText,
				Text: "line1\nline2\nline3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			part := NewTextPart(tt.text)
			assert.Equal(t, tt.expected, part)
			assert.Empty(t, part.ImageURL)
			assert.Empty(t, part.Base64)
			assert.Empty(t, part.MimeType)
		})
	}
}

func TestNewImageURLPart(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected ContentPart
	}{
		{
			name: "creates image URL part",
			url:  "https://example.com/image.png",
			expected: ContentPart{
				Type:     ContentPartTypeImage,
				ImageURL: "https://example.com/image.png",
			},
		},
		{
			name: "handles data URL",
			url:  "data:image/png;base64,abc123",
			expected: ContentPart{
				Type:     ContentPartTypeImage,
				ImageURL: "data:image/png;base64,abc123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			part := NewImageURLPart(tt.url)
			assert.Equal(t, tt.expected, part)
			assert.Empty(t, part.Text)
			assert.Empty(t, part.Base64)
			assert.Empty(t, part.MimeType)
		})
	}
}

func TestNewImageBase64Part(t *testing.T) {
	tests := []struct {
		name       string
		base64Data string
		mimeType   string
		expected   ContentPart
	}{
		{
			name:       "creates base64 image part",
			base64Data: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB",
			mimeType:   "image/png",
			expected: ContentPart{
				Type:     ContentPartTypeImage,
				Base64:   "iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB",
				MimeType: "image/png",
			},
		},
		{
			name:       "handles jpeg",
			base64Data: "/9j/4AAQSkZJRgABAQAA",
			mimeType:   "image/jpeg",
			expected: ContentPart{
				Type:     ContentPartTypeImage,
				Base64:   "/9j/4AAQSkZJRgABAQAA",
				MimeType: "image/jpeg",
			},
		},
		{
			name:       "handles webp",
			base64Data: "UklGRlYAAABXRUJQVlA4",
			mimeType:   "image/webp",
			expected: ContentPart{
				Type:     ContentPartTypeImage,
				Base64:   "UklGRlYAAABXRUJQVlA4",
				MimeType: "image/webp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			part := NewImageBase64Part(tt.base64Data, tt.mimeType)
			assert.Equal(t, tt.expected, part)
			assert.Empty(t, part.Text)
			assert.Empty(t, part.ImageURL)
		})
	}
}

func TestMessageHasParts(t *testing.T) {
	tests := []struct {
		name     string
		message  Message
		expected bool
	}{
		{
			name: "returns true when parts present",
			message: Message{
				Parts: []ContentPart{NewTextPart("hello")},
			},
			expected: true,
		},
		{
			name: "returns true with multiple parts",
			message: Message{
				Parts: []ContentPart{
					NewTextPart("hello"),
					NewImageURLPart("https://example.com/img.png"),
				},
			},
			expected: true,
		},
		{
			name: "returns false when parts empty",
			message: Message{
				Parts: []ContentPart{},
			},
			expected: false,
		},
		{
			name: "returns false when parts nil",
			message: Message{
				Content: "text content",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.message.HasParts())
		})
	}
}

func TestMessageStruct(t *testing.T) {
	t.Run("creates user message with content", func(t *testing.T) {
		msg := Message{
			Role:    RoleUser,
			Content: "Hello",
		}
		assert.Equal(t, RoleUser, msg.Role)
		assert.Equal(t, "Hello", msg.Content)
		assert.False(t, msg.HasParts())
	})

	t.Run("creates assistant message with tool calls", func(t *testing.T) {
		msg := Message{
			Role: RoleAssistant,
			ToolCalls: []ToolCall{
				{ID: "call_1", Name: "get_weather", Arguments: `{"city":"NYC"}`},
			},
		}
		assert.Equal(t, RoleAssistant, msg.Role)
		assert.Len(t, msg.ToolCalls, 1)
		assert.Equal(t, "call_1", msg.ToolCalls[0].ID)
	})

	t.Run("creates tool message with results", func(t *testing.T) {
		msg := Message{
			Role: RoleTool,
			ToolResults: []ToolResult{
				{ToolCallID: "call_1", Content: "72°F", IsError: false},
			},
		}
		assert.Equal(t, RoleTool, msg.Role)
		assert.Len(t, msg.ToolResults, 1)
		assert.Equal(t, "72°F", msg.ToolResults[0].Content)
	})
}

func TestResponseStruct(t *testing.T) {
	t.Run("creates response with content", func(t *testing.T) {
		resp := Response{
			Content:      "Hello!",
			FinishReason: "stop",
			Usage: Usage{
				InputTokens:  10,
				OutputTokens: 5,
			},
		}
		assert.Equal(t, "Hello!", resp.Content)
		assert.Equal(t, "stop", resp.FinishReason)
		assert.Equal(t, 10, resp.Usage.InputTokens)
		assert.Equal(t, 5, resp.Usage.OutputTokens)
	})

	t.Run("creates response with tool calls", func(t *testing.T) {
		resp := Response{
			FinishReason: "tool_calls",
			ToolCalls: []ToolCall{
				{ID: "call_1", Name: "search"},
			},
		}
		assert.Len(t, resp.ToolCalls, 1)
	})
}

func TestStreamEventStruct(t *testing.T) {
	t.Run("creates delta event", func(t *testing.T) {
		event := StreamEvent{
			Delta: "Hello",
			Done:  false,
		}
		assert.Equal(t, "Hello", event.Delta)
		assert.False(t, event.Done)
		assert.Nil(t, event.Response)
		assert.Nil(t, event.Err)
	})

	t.Run("creates done event with response", func(t *testing.T) {
		event := StreamEvent{
			Done: true,
			Response: &Response{
				Content:      "Complete message",
				FinishReason: "stop",
			},
		}
		assert.True(t, event.Done)
		assert.NotNil(t, event.Response)
		assert.Equal(t, "Complete message", event.Response.Content)
	})

	t.Run("creates error event", func(t *testing.T) {
		event := StreamEvent{
			Err: assert.AnError,
		}
		assert.NotNil(t, event.Err)
	})
}

func TestUsageStruct(t *testing.T) {
	usage := Usage{
		InputTokens:  100,
		OutputTokens: 50,
	}
	assert.Equal(t, 100, usage.InputTokens)
	assert.Equal(t, 50, usage.OutputTokens)
}
