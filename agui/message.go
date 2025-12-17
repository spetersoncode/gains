package agui

import (
	ai "github.com/spetersoncode/gains"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
)

// Role constants matching AG-UI protocol.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"
)

// ToGainsMessages converts AG-UI messages to gains messages.
func ToGainsMessages(msgs []events.Message) []ai.Message {
	result := make([]ai.Message, 0, len(msgs))
	for _, msg := range msgs {
		result = append(result, ToGainsMessage(msg))
	}
	return result
}

// ToGainsMessage converts a single AG-UI message to a gains message.
func ToGainsMessage(msg events.Message) ai.Message {
	m := ai.Message{
		Role: toGainsRole(msg.Role),
	}

	// Set content if present
	if msg.Content != nil {
		m.Content = *msg.Content
	}

	// Convert tool calls (for assistant messages)
	if len(msg.ToolCalls) > 0 {
		m.ToolCalls = make([]ai.ToolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			m.ToolCalls[i] = ai.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
		}
	}

	// Convert tool result (for tool messages)
	if msg.ToolCallID != nil && msg.Content != nil {
		m.ToolResults = []ai.ToolResult{{
			ToolCallID: *msg.ToolCallID,
			Content:    *msg.Content,
		}}
	}

	return m
}

// FromGainsMessages converts gains messages to AG-UI messages.
func FromGainsMessages(msgs []ai.Message) []events.Message {
	result := make([]events.Message, 0, len(msgs))
	for i, msg := range msgs {
		result = append(result, FromGainsMessage(msg, i))
	}
	return result
}

// FromGainsMessage converts a single gains message to an AG-UI message.
// The index is used to generate a message ID if needed.
func FromGainsMessage(msg ai.Message, index int) events.Message {
	m := events.Message{
		ID:   events.GenerateMessageID(),
		Role: fromGainsRole(msg.Role),
	}

	// Set content
	if msg.Content != "" {
		m.Content = &msg.Content
	}

	// Convert tool calls (for assistant messages)
	if len(msg.ToolCalls) > 0 {
		m.ToolCalls = make([]events.ToolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			m.ToolCalls[i] = events.ToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: events.Function{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			}
		}
	}

	// Convert tool results (for tool messages)
	if len(msg.ToolResults) > 0 && len(msg.ToolResults) == 1 {
		m.ToolCallID = &msg.ToolResults[0].ToolCallID
		m.Content = &msg.ToolResults[0].Content
	}

	return m
}

// toGainsRole converts an AG-UI role string to a gains Role.
func toGainsRole(role string) ai.Role {
	switch role {
	case RoleUser:
		return ai.RoleUser
	case RoleAssistant:
		return ai.RoleAssistant
	case RoleSystem:
		return ai.RoleSystem
	case RoleTool:
		return ai.RoleTool
	default:
		return ai.RoleUser
	}
}

// fromGainsRole converts a gains Role to an AG-UI role string.
func fromGainsRole(role ai.Role) string {
	switch role {
	case ai.RoleUser:
		return RoleUser
	case ai.RoleAssistant:
		return RoleAssistant
	case ai.RoleSystem:
		return RoleSystem
	case ai.RoleTool:
		return RoleTool
	default:
		return RoleUser
	}
}
