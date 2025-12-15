package anthropic

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/spetersoncode/gains"
)

// jsonResponseToolName is the name of the synthetic tool used for JSON mode.
const jsonResponseToolName = "__gains_json_response__"

func convertTools(tools []gains.Tool) []anthropic.ToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	result := make([]anthropic.ToolUnionParam, len(tools))
	for i, t := range tools {
		// Parse the JSON Schema to get the input schema
		var schema map[string]interface{}
		if len(t.Parameters) > 0 {
			json.Unmarshal(t.Parameters, &schema)
		}

		// Extract required as []string
		var required []string
		if reqVal, ok := schema["required"].([]interface{}); ok {
			for _, r := range reqVal {
				if s, ok := r.(string); ok {
					required = append(required, s)
				}
			}
		}

		inputSchema := anthropic.ToolInputSchemaParam{
			Properties: schema["properties"],
			Required:   required,
		}

		toolParam := anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: inputSchema,
		}

		result[i] = anthropic.ToolUnionParam{
			OfTool: &toolParam,
		}
	}
	return result
}

func convertToolChoice(choice gains.ToolChoice) anthropic.ToolChoiceUnionParam {
	switch choice {
	case gains.ToolChoiceNone:
		return anthropic.ToolChoiceUnionParam{
			OfNone: &anthropic.ToolChoiceNoneParam{},
		}
	case gains.ToolChoiceRequired:
		return anthropic.ToolChoiceUnionParam{
			OfAny: &anthropic.ToolChoiceAnyParam{},
		}
	default:
		return anthropic.ToolChoiceUnionParam{
			OfAuto: &anthropic.ToolChoiceAutoParam{},
		}
	}
}

func extractToolCalls(content []anthropic.ContentBlockUnion) []gains.ToolCall {
	var calls []gains.ToolCall
	for _, block := range content {
		if block.Type == "tool_use" {
			calls = append(calls, gains.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: string(block.Input),
			})
		}
	}
	return calls
}

func buildAnthropicJSONTool(options *gains.Options) (anthropic.ToolUnionParam, anthropic.ToolChoiceUnionParam) {
	var schema map[string]any
	if options.ResponseSchema != nil && len(options.ResponseSchema.Schema) > 0 {
		json.Unmarshal(options.ResponseSchema.Schema, &schema)
	} else {
		// Generic object schema for basic JSON mode
		schema = map[string]any{
			"type":                 "object",
			"additionalProperties": true,
		}
	}

	description := "Output the response as structured JSON"
	if options.ResponseSchema != nil && options.ResponseSchema.Description != "" {
		description = options.ResponseSchema.Description
	}

	// Extract required fields
	var required []string
	if reqVal, ok := schema["required"].([]any); ok {
		for _, r := range reqVal {
			if s, ok := r.(string); ok {
				required = append(required, s)
			}
		}
	}

	inputSchema := anthropic.ToolInputSchemaParam{
		Properties: schema["properties"],
		Required:   required,
	}

	tool := anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        jsonResponseToolName,
			Description: anthropic.String(description),
			InputSchema: inputSchema,
		},
	}

	toolChoice := anthropic.ToolChoiceUnionParam{
		OfTool: &anthropic.ToolChoiceToolParam{
			Name: jsonResponseToolName,
		},
	}

	return tool, toolChoice
}
