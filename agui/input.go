package agui

import (
	"encoding/json"
	"errors"

	ai "github.com/spetersoncode/gains"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
)

// RunAgentInput represents the AG-UI protocol request for running an agent.
// This mirrors the AG-UI protocol specification and is transport-agnostic.
type RunAgentInput struct {
	ThreadID       string           `json:"thread_id"`
	RunID          string           `json:"run_id"`
	Messages       []events.Message `json:"messages"`
	Tools          []any            `json:"tools,omitempty"`           // Frontend-provided tools
	Context        []any            `json:"context,omitempty"`         // Context items
	State          any              `json:"state,omitempty"`           // State
	ForwardedProps any              `json:"forwarded_props,omitempty"` // Forwarded props
}

// PreparedInput contains validated and converted input ready for agent execution.
type PreparedInput struct {
	ThreadID  string
	RunID     string
	Messages  []ai.Message
	Tools     []Tool   // Parsed frontend tools
	ToolNames []string // Tool names for cleanup tracking
	State     any      // Raw state from frontend
}

// ErrNoMessages is returned when the input contains no messages.
var ErrNoMessages = errors.New("no messages provided")

// Prepare validates the input and converts it to gains types.
// Returns ErrNoMessages if Messages is empty.
// Returns an error if tool parsing fails.
func (r *RunAgentInput) Prepare() (*PreparedInput, error) {
	// Convert messages
	messages := ToGainsMessages(r.Messages)
	if len(messages) == 0 {
		return nil, ErrNoMessages
	}

	result := &PreparedInput{
		ThreadID: r.ThreadID,
		RunID:    r.RunID,
		Messages: messages,
		State:    r.State,
	}

	// Parse frontend tools if provided
	if len(r.Tools) > 0 {
		tools, err := ParseTools(r.Tools)
		if err != nil {
			return nil, err
		}
		result.Tools = tools
		result.ToolNames = ToolNames(tools)
	}

	return result, nil
}

// GainsTools converts the parsed frontend tools to gains tools.
// Returns nil if no tools were parsed.
func (p *PreparedInput) GainsTools() []ai.Tool {
	return ToGainsTools(p.Tools)
}

// DecodeState decodes the raw state into a typed struct.
// Returns the zero value of T if State is nil.
func DecodeState[T any](input *PreparedInput) (T, error) {
	var result T
	if input.State == nil {
		return result, nil
	}

	// Re-marshal and unmarshal to get proper typing
	data, err := json.Marshal(input.State)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}

	return result, nil
}

// MustDecodeState is like DecodeState but panics on error.
func MustDecodeState[T any](input *PreparedInput) T {
	result, err := DecodeState[T](input)
	if err != nil {
		panic("agui: failed to decode state: " + err.Error())
	}
	return result
}
