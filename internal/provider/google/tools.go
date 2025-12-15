package google

import (
	"encoding/json"
	"fmt"

	ai "github.com/spetersoncode/gains"
	"google.golang.org/genai"
)

func convertTools(tools []ai.Tool) []*genai.Tool {
	if len(tools) == 0 {
		return nil
	}

	funcs := make([]*genai.FunctionDeclaration, len(tools))
	for i, t := range tools {
		funcs[i] = &genai.FunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  convertJSONSchemaToGenaiSchema(t.Parameters),
		}
	}

	return []*genai.Tool{{FunctionDeclarations: funcs}}
}

func convertToolChoice(choice ai.ToolChoice) *genai.ToolConfig {
	switch choice {
	case ai.ToolChoiceNone:
		return &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeNone,
			},
		}
	case ai.ToolChoiceRequired:
		return &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAny,
			},
		}
	default:
		return &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAuto,
			},
		}
	}
}

func extractToolCalls(parts []*genai.Part) []ai.ToolCall {
	var calls []ai.ToolCall
	for i, part := range parts {
		if part.FunctionCall != nil {
			args, _ := json.Marshal(part.FunctionCall.Args)
			calls = append(calls, ai.ToolCall{
				ID:        fmt.Sprintf("call_%d_%s", i, part.FunctionCall.Name),
				Name:      part.FunctionCall.Name,
				Arguments: string(args),
			})
		}
	}
	return calls
}
