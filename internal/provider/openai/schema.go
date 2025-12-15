package openai

import (
	"encoding/json"

	"github.com/openai/openai-go"
	ai "github.com/spetersoncode/gains"
)

func buildOpenAISchemaFormat(schema *ai.ResponseSchema) openai.ChatCompletionNewParamsResponseFormatUnion {
	var schemaMap map[string]any
	json.Unmarshal(schema.Schema, &schemaMap)

	name := schema.Name
	if name == "" {
		name = "response_schema"
	}

	strict := true // Default to strict

	// OpenAI strict mode requires additionalProperties: false on all objects
	if strict {
		addAdditionalPropertiesFalse(schemaMap)
	}

	return openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
			Type: "json_schema",
			JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:        name,
				Description: openai.String(schema.Description),
				Schema:      schemaMap,
				Strict:      openai.Bool(strict),
			},
		},
	}
}

// addAdditionalPropertiesFalse recursively adds additionalProperties: false to all object schemas.
// This is required by OpenAI's strict mode.
func addAdditionalPropertiesFalse(schema map[string]any) {
	if schema == nil {
		return
	}

	// If this is an object type, add additionalProperties: false
	if schemaType, ok := schema["type"].(string); ok && schemaType == "object" {
		schema["additionalProperties"] = false
	}

	// Recurse into properties
	if props, ok := schema["properties"].(map[string]any); ok {
		for _, propSchema := range props {
			if propMap, ok := propSchema.(map[string]any); ok {
				addAdditionalPropertiesFalse(propMap)
			}
		}
	}

	// Recurse into array items
	if items, ok := schema["items"].(map[string]any); ok {
		addAdditionalPropertiesFalse(items)
	}
}
