package agui

import (
	"encoding/json"

	ai "github.com/spetersoncode/gains"
)

// Tool represents a tool definition from the AG-UI protocol.
// Frontend applications send these to define capabilities available to agents.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// ToGainsTool converts an AG-UI tool to a gains Tool.
func (t Tool) ToGainsTool() ai.Tool {
	return ai.Tool{
		Name:        t.Name,
		Description: t.Description,
		Parameters:  t.Parameters,
	}
}

// ParseTools parses a slice of any (from JSON unmarshaling) into Tool structs.
// This handles the Tools field from RunAgentInput which is []any.
func ParseTools(raw []any) ([]Tool, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	// Re-marshal and unmarshal to get proper typing
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	var tools []Tool
	if err := json.Unmarshal(data, &tools); err != nil {
		return nil, err
	}

	return tools, nil
}

// ToGainsTools converts a slice of AG-UI tools to gains tools.
func ToGainsTools(tools []Tool) []ai.Tool {
	if len(tools) == 0 {
		return nil
	}

	result := make([]ai.Tool, len(tools))
	for i, t := range tools {
		result[i] = t.ToGainsTool()
	}
	return result
}

// ToolNames extracts the names from a slice of tools.
func ToolNames(tools []Tool) []string {
	if len(tools) == 0 {
		return nil
	}

	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	return names
}
