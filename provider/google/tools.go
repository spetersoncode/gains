package google

import (
	"encoding/json"
	"fmt"

	"github.com/spetersoncode/gains"
	"google.golang.org/genai"
)

func convertTools(tools []gains.Tool) []*genai.Tool {
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

func convertToolChoice(choice gains.ToolChoice) *genai.ToolConfig {
	switch choice {
	case gains.ToolChoiceNone:
		return &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeNone,
			},
		}
	case gains.ToolChoiceRequired:
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

func extractToolCalls(parts []*genai.Part) []gains.ToolCall {
	var calls []gains.ToolCall
	for i, part := range parts {
		if part.FunctionCall != nil {
			args, _ := json.Marshal(part.FunctionCall.Args)
			calls = append(calls, gains.ToolCall{
				ID:        fmt.Sprintf("call_%d_%s", i, part.FunctionCall.Name),
				Name:      part.FunctionCall.Name,
				Arguments: string(args),
			})
		}
	}
	return calls
}
