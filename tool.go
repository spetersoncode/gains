package gains

import "encoding/json"

// Tool defines a function that can be called by the model.
type Tool struct {
	// Name is the unique identifier for the tool.
	Name string
	// Description explains what the tool does (helps the model decide when to use it).
	Description string
	// Parameters is a JSON Schema object defining the function parameters.
	Parameters json.RawMessage
}

// ToolCall represents a request from the model to invoke a tool.
type ToolCall struct {
	// ID is a unique identifier for this tool call (used to match results).
	ID string `json:"id"`
	// Name is the name of the tool to invoke.
	Name string `json:"name"`
	// Arguments is a JSON string containing the arguments to pass.
	Arguments string `json:"arguments"`
}

// ToolResult represents the result of executing a tool call.
type ToolResult struct {
	// ToolCallID matches the ID from the corresponding ToolCall.
	ToolCallID string `json:"toolCallId"`
	// Content is the result content to return to the model.
	Content string `json:"content"`
	// IsError indicates if the result represents an error.
	IsError bool `json:"isError,omitempty"`
}

// ToolChoice controls how the model uses tools.
type ToolChoice string

const (
	// ToolChoiceAuto lets the model decide when to use tools (default).
	ToolChoiceAuto ToolChoice = "auto"
	// ToolChoiceNone disables tool use for the request.
	ToolChoiceNone ToolChoice = "none"
	// ToolChoiceRequired forces the model to use a tool.
	ToolChoiceRequired ToolChoice = "required"
)

// NewToolResultMessage creates a message containing tool results.
// This is a convenience function for returning tool results to the model.
func NewToolResultMessage(results ...ToolResult) Message {
	return Message{
		Role:        RoleTool,
		ToolResults: results,
	}
}
