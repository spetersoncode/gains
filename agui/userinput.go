package agui

import (
	"encoding/json"

	"github.com/spetersoncode/gains/agent"
)

// UserInputInput represents a user input response from the AG-UI frontend.
// This corresponds to a user action on a user_input activity.
type UserInputInput struct {
	RequestID string `json:"requestId"`
	Value     string `json:"value,omitempty"`
	Confirmed bool   `json:"confirmed,omitempty"`
	Cancelled bool   `json:"cancelled,omitempty"`
}

// ParseUserInputInput parses a user input response from JSON.
func ParseUserInputInput(data []byte) (*UserInputInput, error) {
	var input UserInputInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, err
	}
	return &input, nil
}

// ToResponse converts a UserInputInput to an agent.UserInputResponse.
func (u *UserInputInput) ToResponse() agent.UserInputResponse {
	return agent.UserInputResponse{
		RequestID: u.RequestID,
		Value:     u.Value,
		Confirmed: u.Confirmed,
		Cancelled: u.Cancelled,
	}
}

// HandleUserInput processes a user input and sends it to the broker.
// This is a convenience function for AG-UI server handlers.
func HandleUserInput(broker *agent.UserInputBroker, input *UserInputInput) error {
	return broker.Respond(input.ToResponse())
}

// HandleUserInputJSON processes a JSON-encoded user input response.
func HandleUserInputJSON(broker *agent.UserInputBroker, data []byte) error {
	input, err := ParseUserInputInput(data)
	if err != nil {
		return err
	}
	return HandleUserInput(broker, input)
}
