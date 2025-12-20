package a2a

import (
	ai "github.com/spetersoncode/gains"
)

// ToGainsMessages converts A2A messages to gains messages.
func ToGainsMessages(msgs []Message) []ai.Message {
	result := make([]ai.Message, 0, len(msgs))
	for _, msg := range msgs {
		result = append(result, ToGainsMessage(msg))
	}
	return result
}

// ToGainsMessage converts a single A2A message to a gains message.
func ToGainsMessage(msg Message) ai.Message {
	m := ai.Message{
		ID:   msg.MessageID,
		Role: toGainsRole(msg.Role),
	}

	// Extract text content from parts
	for _, part := range msg.Parts {
		switch p := part.(type) {
		case TextPart:
			m.Content += p.Text
		case DataPart:
			// Data parts might contain tool calls or tool results
			if data, ok := p.Data.(map[string]any); ok {
				if toolCalls := extractToolCalls(data); len(toolCalls) > 0 {
					m.ToolCalls = append(m.ToolCalls, toolCalls...)
				}
				if toolResults := extractToolResults(data); len(toolResults) > 0 {
					m.ToolResults = append(m.ToolResults, toolResults...)
				}
			}
		}
	}

	return m
}

// FromGainsMessages converts gains messages to A2A messages.
func FromGainsMessages(msgs []ai.Message) []Message {
	result := make([]Message, 0, len(msgs))
	for _, msg := range msgs {
		result = append(result, FromGainsMessage(msg))
	}
	return result
}

// FromGainsMessage converts a single gains message to an A2A message.
func FromGainsMessage(msg ai.Message) Message {
	m := NewMessage(fromGainsRole(msg.Role))
	if msg.ID != "" {
		m.MessageID = msg.ID
	}

	// Build parts from message content
	var parts []Part

	// Add text content
	if msg.Content != "" {
		parts = append(parts, NewTextPart(msg.Content))
	}

	// Add tool calls as data parts
	for _, tc := range msg.ToolCalls {
		parts = append(parts, NewDataPart(map[string]any{
			"type": "tool_call",
			"tool_call": map[string]any{
				"id":        tc.ID,
				"name":      tc.Name,
				"arguments": tc.Arguments,
			},
		}))
	}

	// Add tool results as data parts
	for _, tr := range msg.ToolResults {
		parts = append(parts, NewDataPart(map[string]any{
			"type": "tool_result",
			"tool_result": map[string]any{
				"tool_call_id": tr.ToolCallID,
				"content":      tr.Content,
				"is_error":     tr.IsError,
			},
		}))
	}

	m.Parts = parts
	return m
}

// toGainsRole converts an A2A role to a gains Role.
func toGainsRole(role MessageRole) ai.Role {
	switch role {
	case MessageRoleUser:
		return ai.RoleUser
	case MessageRoleAgent:
		return ai.RoleAssistant
	default:
		return ai.RoleUser
	}
}

// fromGainsRole converts a gains Role to an A2A role.
func fromGainsRole(role ai.Role) MessageRole {
	switch role {
	case ai.RoleUser:
		return MessageRoleUser
	case ai.RoleAssistant, ai.RoleSystem, ai.RoleTool:
		return MessageRoleAgent
	default:
		return MessageRoleUser
	}
}

// extractToolCalls extracts tool calls from a data part.
func extractToolCalls(data map[string]any) []ai.ToolCall {
	if data["type"] != "tool_call" {
		return nil
	}

	tc, ok := data["tool_call"].(map[string]any)
	if !ok {
		return nil
	}

	id, _ := tc["id"].(string)
	name, _ := tc["name"].(string)
	args, _ := tc["arguments"].(string)

	return []ai.ToolCall{{
		ID:        id,
		Name:      name,
		Arguments: args,
	}}
}

// extractToolResults extracts tool results from a data part.
func extractToolResults(data map[string]any) []ai.ToolResult {
	if data["type"] != "tool_result" {
		return nil
	}

	tr, ok := data["tool_result"].(map[string]any)
	if !ok {
		return nil
	}

	toolCallID, _ := tr["tool_call_id"].(string)
	content, _ := tr["content"].(string)
	isError, _ := tr["is_error"].(bool)

	return []ai.ToolResult{{
		ToolCallID: toolCallID,
		Content:    content,
		IsError:    isError,
	}}
}
