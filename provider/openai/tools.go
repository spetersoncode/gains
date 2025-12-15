package openai

import (
	"encoding/json"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
	"github.com/spetersoncode/gains"
)

func convertTools(tools []gains.Tool) []openai.ChatCompletionToolParam {
	if len(tools) == 0 {
		return nil
	}
	result := make([]openai.ChatCompletionToolParam, len(tools))
	for i, t := range tools {
		// Parse JSON schema to map for FunctionParameters
		var params shared.FunctionParameters
		if len(t.Parameters) > 0 {
			json.Unmarshal(t.Parameters, &params)
		}
		result[i] = openai.ChatCompletionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name:        t.Name,
				Description: openai.String(t.Description),
				Parameters:  params,
			},
		}
	}
	return result
}

func convertToolChoice(choice gains.ToolChoice) openai.ChatCompletionToolChoiceOptionUnionParam {
	switch choice {
	case gains.ToolChoiceNone:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("none"),
		}
	case gains.ToolChoiceRequired:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("required"),
		}
	default:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("auto"),
		}
	}
}

func extractToolCalls(msg openai.ChatCompletionMessage) []gains.ToolCall {
	if len(msg.ToolCalls) == 0 {
		return nil
	}
	result := make([]gains.ToolCall, len(msg.ToolCalls))
	for i, tc := range msg.ToolCalls {
		result[i] = gains.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		}
	}
	return result
}

func extractToolCallsFromAccumulator(toolCalls []openai.ChatCompletionMessageToolCall) []gains.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}
	result := make([]gains.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = gains.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		}
	}
	return result
}
