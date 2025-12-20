package google

import (
	"encoding/json"

	"google.golang.org/genai"
)

// ConvertJSONSchemaToGenaiSchema converts JSON Schema to Google genai Schema.
func ConvertJSONSchemaToGenaiSchema(schemaJSON json.RawMessage) *genai.Schema {
	if len(schemaJSON) == 0 {
		return nil
	}

	var schema map[string]any
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return nil
	}

	return convertSchemaObject(schema)
}

func convertSchemaObject(schema map[string]any) *genai.Schema {
	if schema == nil {
		return nil
	}

	result := &genai.Schema{}

	// Handle type
	if typeVal, ok := schema["type"].(string); ok {
		switch typeVal {
		case "string":
			result.Type = genai.TypeString
		case "number":
			result.Type = genai.TypeNumber
		case "integer":
			result.Type = genai.TypeInteger
		case "boolean":
			result.Type = genai.TypeBoolean
		case "array":
			result.Type = genai.TypeArray
		case "object":
			result.Type = genai.TypeObject
		}
	}

	// Handle description
	if desc, ok := schema["description"].(string); ok {
		result.Description = desc
	}

	// Handle enum
	if enumVal, ok := schema["enum"].([]any); ok {
		for _, e := range enumVal {
			if s, ok := e.(string); ok {
				result.Enum = append(result.Enum, s)
			}
		}
	}

	// Handle properties (for objects)
	if props, ok := schema["properties"].(map[string]any); ok {
		result.Properties = make(map[string]*genai.Schema)
		for name, propSchema := range props {
			if propMap, ok := propSchema.(map[string]any); ok {
				result.Properties[name] = convertSchemaObject(propMap)
			}
		}
	}

	// Handle required fields
	if required, ok := schema["required"].([]any); ok {
		for _, r := range required {
			if s, ok := r.(string); ok {
				result.Required = append(result.Required, s)
			}
		}
	}

	// Handle array items
	if items, ok := schema["items"].(map[string]any); ok {
		result.Items = convertSchemaObject(items)
	}

	return result
}
