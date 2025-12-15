package gains

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToolChoiceConstants(t *testing.T) {
	assert.Equal(t, ToolChoice("auto"), ToolChoiceAuto)
	assert.Equal(t, ToolChoice("none"), ToolChoiceNone)
	assert.Equal(t, ToolChoice("required"), ToolChoiceRequired)
}

func TestNewToolResultMessage(t *testing.T) {
	t.Run("creates message with single result", func(t *testing.T) {
		result := ToolResult{
			ToolCallID: "call_abc123",
			Content:    "The weather is 72°F",
			IsError:    false,
		}

		msg := NewToolResultMessage(result)

		assert.Equal(t, RoleTool, msg.Role)
		assert.Len(t, msg.ToolResults, 1)
		assert.Equal(t, "call_abc123", msg.ToolResults[0].ToolCallID)
		assert.Equal(t, "The weather is 72°F", msg.ToolResults[0].Content)
		assert.False(t, msg.ToolResults[0].IsError)
	})

	t.Run("creates message with multiple results", func(t *testing.T) {
		results := []ToolResult{
			{ToolCallID: "call_1", Content: "Result 1", IsError: false},
			{ToolCallID: "call_2", Content: "Result 2", IsError: false},
			{ToolCallID: "call_3", Content: "Error occurred", IsError: true},
		}

		msg := NewToolResultMessage(results...)

		assert.Equal(t, RoleTool, msg.Role)
		assert.Len(t, msg.ToolResults, 3)
		assert.Equal(t, "call_1", msg.ToolResults[0].ToolCallID)
		assert.Equal(t, "call_2", msg.ToolResults[1].ToolCallID)
		assert.Equal(t, "call_3", msg.ToolResults[2].ToolCallID)
		assert.True(t, msg.ToolResults[2].IsError)
	})

	t.Run("creates message with no results", func(t *testing.T) {
		msg := NewToolResultMessage()

		assert.Equal(t, RoleTool, msg.Role)
		assert.Empty(t, msg.ToolResults)
	})

	t.Run("creates message with error result", func(t *testing.T) {
		result := ToolResult{
			ToolCallID: "call_error",
			Content:    "Function failed: connection timeout",
			IsError:    true,
		}

		msg := NewToolResultMessage(result)

		assert.Equal(t, RoleTool, msg.Role)
		assert.Len(t, msg.ToolResults, 1)
		assert.True(t, msg.ToolResults[0].IsError)
	})
}

func TestToolStruct(t *testing.T) {
	t.Run("creates tool with parameters", func(t *testing.T) {
		params := json.RawMessage(`{
			"type": "object",
			"properties": {
				"city": {"type": "string", "description": "City name"}
			},
			"required": ["city"]
		}`)

		tool := Tool{
			Name:        "get_weather",
			Description: "Get the current weather for a city",
			Parameters:  params,
		}

		assert.Equal(t, "get_weather", tool.Name)
		assert.Equal(t, "Get the current weather for a city", tool.Description)
		assert.NotNil(t, tool.Parameters)
	})

	t.Run("creates tool without parameters", func(t *testing.T) {
		tool := Tool{
			Name:        "get_time",
			Description: "Get the current time",
		}

		assert.Equal(t, "get_time", tool.Name)
		assert.Nil(t, tool.Parameters)
	})
}

func TestToolCallStruct(t *testing.T) {
	t.Run("creates tool call with arguments", func(t *testing.T) {
		call := ToolCall{
			ID:        "call_xyz789",
			Name:      "search",
			Arguments: `{"query": "best restaurants"}`,
		}

		assert.Equal(t, "call_xyz789", call.ID)
		assert.Equal(t, "search", call.Name)
		assert.Equal(t, `{"query": "best restaurants"}`, call.Arguments)
	})

	t.Run("creates tool call with empty arguments", func(t *testing.T) {
		call := ToolCall{
			ID:        "call_abc",
			Name:      "get_time",
			Arguments: "{}",
		}

		assert.Equal(t, "{}", call.Arguments)
	})
}

func TestToolResultStruct(t *testing.T) {
	t.Run("creates success result", func(t *testing.T) {
		result := ToolResult{
			ToolCallID: "call_123",
			Content:    `{"temperature": 72, "unit": "F"}`,
			IsError:    false,
		}

		assert.Equal(t, "call_123", result.ToolCallID)
		assert.Contains(t, result.Content, "temperature")
		assert.False(t, result.IsError)
	})

	t.Run("creates error result", func(t *testing.T) {
		result := ToolResult{
			ToolCallID: "call_456",
			Content:    "API rate limit exceeded",
			IsError:    true,
		}

		assert.True(t, result.IsError)
		assert.Equal(t, "API rate limit exceeded", result.Content)
	})
}
