package agui

import (
	"encoding/json"

	"github.com/spetersoncode/gains/agent"
)

// ApprovalInput represents an approval decision from the AG-UI frontend.
// This corresponds to a user action on a tool_approval activity.
type ApprovalInput struct {
	ToolCallID string `json:"toolCallId"`
	Approved   bool   `json:"approved"`
	Reason     string `json:"reason,omitempty"`
}

// ParseApprovalInput parses an approval decision from JSON.
func ParseApprovalInput(data []byte) (*ApprovalInput, error) {
	var input ApprovalInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, err
	}
	return &input, nil
}

// ToDecision converts an ApprovalInput to an agent.ApprovalDecision.
func (a *ApprovalInput) ToDecision() agent.ApprovalDecision {
	return agent.ApprovalDecision{
		ToolCallID: a.ToolCallID,
		Approved:   a.Approved,
		Reason:     a.Reason,
	}
}

// HandleApproval processes an approval input and sends it to the broker.
// This is a convenience function for AG-UI server handlers.
func HandleApproval(broker *agent.ApprovalBroker, input *ApprovalInput) error {
	return broker.Decide(input.ToDecision())
}

// HandleApprovalJSON processes a JSON-encoded approval input.
func HandleApprovalJSON(broker *agent.ApprovalBroker, data []byte) error {
	input, err := ParseApprovalInput(data)
	if err != nil {
		return err
	}
	return HandleApproval(broker, input)
}
