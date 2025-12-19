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

// InitializeState creates a new state struct initialized from frontend state.
// This is the recommended way to create workflow state from AG-UI input:
//
//	input, err := runAgentInput.Prepare()
//	state, err := agui.InitializeState[MyState](input)
//	result, err := workflow.Run(ctx, state, opts...)
func InitializeState[T any](input *PreparedInput) (*T, error) {
	state, err := DecodeState[T](input)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// MustInitializeState is like InitializeState but panics on error.
func MustInitializeState[T any](input *PreparedInput) *T {
	state, err := InitializeState[T](input)
	if err != nil {
		panic("agui: failed to initialize state: " + err.Error())
	}
	return state
}

// MergeState merges frontend state into an existing state struct.
// Fields from the frontend state overwrite corresponding fields in state.
// This is useful when you have a pre-populated state with defaults:
//
//	state := &MyState{DefaultField: "value"}
//	agui.MergeState(state, input) // Overwrites with frontend values
func MergeState[T any](state *T, input *PreparedInput) error {
	if input.State == nil {
		return nil
	}

	decoded, err := DecodeState[T](input)
	if err != nil {
		return err
	}
	*state = decoded
	return nil
}

// MustMergeState is like MergeState but panics on error.
func MustMergeState[T any](state *T, input *PreparedInput) {
	if err := MergeState(state, input); err != nil {
		panic("agui: failed to merge state: " + err.Error())
	}
}

// RunWorkflowInput represents the AG-UI protocol request for running a workflow.
// This is designed for dispatching workflows by name with typed state.
type RunWorkflowInput struct {
	ThreadID       string `json:"thread_id"`
	RunID          string `json:"run_id"`
	WorkflowName   string `json:"workflow_name"`           // Name of workflow to execute
	State          any    `json:"state,omitempty"`         // Initial workflow state
	ForwardedProps any    `json:"forwarded_props,omitempty"`
}

// PreparedWorkflowInput contains validated workflow input ready for execution.
type PreparedWorkflowInput struct {
	ThreadID     string
	RunID        string
	WorkflowName string
	State        any // Raw state for workflow initialization
}

// ErrNoWorkflowName is returned when the workflow name is empty.
var ErrNoWorkflowName = errors.New("no workflow name provided")

// Prepare validates the workflow input.
// Returns ErrNoWorkflowName if WorkflowName is empty.
func (r *RunWorkflowInput) Prepare() (*PreparedWorkflowInput, error) {
	if r.WorkflowName == "" {
		return nil, ErrNoWorkflowName
	}

	return &PreparedWorkflowInput{
		ThreadID:     r.ThreadID,
		RunID:        r.RunID,
		WorkflowName: r.WorkflowName,
		State:        r.State,
	}, nil
}

// DecodeWorkflowState decodes the raw state into a typed struct.
// Returns the zero value of T if State is nil.
func DecodeWorkflowState[T any](input *PreparedWorkflowInput) (T, error) {
	var result T
	if input.State == nil {
		return result, nil
	}

	data, err := json.Marshal(input.State)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}

	return result, nil
}

// MustDecodeWorkflowState is like DecodeWorkflowState but panics on error.
func MustDecodeWorkflowState[T any](input *PreparedWorkflowInput) T {
	result, err := DecodeWorkflowState[T](input)
	if err != nil {
		panic("agui: failed to decode workflow state: " + err.Error())
	}
	return result
}
