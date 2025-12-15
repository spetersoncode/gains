package anthropic

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/spetersoncode/gains"
)

func convertMessages(messages []gains.Message) ([]anthropic.MessageParam, []anthropic.TextBlockParam) {
	var result []anthropic.MessageParam
	var system []anthropic.TextBlockParam

	for _, msg := range messages {
		switch msg.Role {
		case gains.RoleSystem:
			system = append(system, anthropic.TextBlockParam{Text: msg.Content})
		case gains.RoleUser:
			if msg.HasParts() {
				blocks := convertPartsToAnthropicBlocks(msg.Parts)
				result = append(result, anthropic.MessageParam{
					Role:    anthropic.MessageParamRoleUser,
					Content: blocks,
				})
			} else {
				result = append(result, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
			}
		case gains.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				// Assistant message with tool calls
				var blocks []anthropic.ContentBlockParamUnion
				if msg.Content != "" {
					blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
				}
				for _, tc := range msg.ToolCalls {
					var input any
					json.Unmarshal([]byte(tc.Arguments), &input)
					blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Name))
				}
				result = append(result, anthropic.MessageParam{
					Role:    anthropic.MessageParamRoleAssistant,
					Content: blocks,
				})
			} else {
				result = append(result, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
			}
		case gains.RoleTool:
			// Tool results are sent as user messages with tool_result blocks
			var blocks []anthropic.ContentBlockParamUnion
			for _, tr := range msg.ToolResults {
				blocks = append(blocks, anthropic.NewToolResultBlock(tr.ToolCallID, tr.Content, tr.IsError))
			}
			result = append(result, anthropic.MessageParam{
				Role:    anthropic.MessageParamRoleUser,
				Content: blocks,
			})
		default:
			result = append(result, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		}
	}

	return result, system
}

func convertPartsToAnthropicBlocks(parts []gains.ContentPart) []anthropic.ContentBlockParamUnion {
	var blocks []anthropic.ContentBlockParamUnion
	for _, part := range parts {
		switch part.Type {
		case gains.ContentPartTypeText:
			blocks = append(blocks, anthropic.NewTextBlock(part.Text))
		case gains.ContentPartTypeImage:
			if part.ImageURL != "" {
				// URL-based image
				blocks = append(blocks, anthropic.NewImageBlock(anthropic.URLImageSourceParam{
					URL: part.ImageURL,
				}))
			} else if part.Base64 != "" {
				// Base64-encoded image
				mediaType := part.MimeType
				if mediaType == "" {
					mediaType = "image/jpeg" // Default
				}
				blocks = append(blocks, anthropic.NewImageBlockBase64(mediaType, part.Base64))
			}
		}
	}
	return blocks
}
