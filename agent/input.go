package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InputType specifies the kind of user input expected.
type InputType string

const (
	// InputTypeConfirm requests a yes/no confirmation.
	InputTypeConfirm InputType = "confirm"

	// InputTypeText requests free-form text input.
	InputTypeText InputType = "text"

	// InputTypeChoice requests selection from a list of options.
	InputTypeChoice InputType = "choice"
)

// UserInputRequest represents a request for user input.
// This is used for confirmation dialogs, text prompts, and choice selections.
type UserInputRequest struct {
	ID          string    `json:"id"`                    // Unique identifier for this request
	Type        InputType `json:"type"`                  // Type of input expected
	Title       string    `json:"title,omitempty"`       // Optional title for the dialog
	Message     string    `json:"message"`               // The prompt message
	Choices     []string  `json:"choices,omitempty"`     // Options for InputTypeChoice
	Default     string    `json:"default,omitempty"`     // Default value
	Placeholder string    `json:"placeholder,omitempty"` // Placeholder for text input
}

// UserInputResponse represents the user's response to an input request.
type UserInputResponse struct {
	RequestID string `json:"requestId"`         // ID of the request being responded to
	Value     string `json:"value"`             // The user's input value
	Confirmed bool   `json:"confirmed"`         // For confirm type: true if confirmed
	Cancelled bool   `json:"cancelled"`         // True if user cancelled/dismissed
}

// UserInputBroker manages async user input requests for AG-UI integration.
// It receives responses from the frontend and routes them to waiting requests.
//
// Usage:
//
//	broker := agent.NewUserInputBroker()
//
//	// In your HTTP handler, route frontend responses to the broker
//	go func() {
//	    for response := range frontendResponses {
//	        broker.Respond(response)
//	    }
//	}()
//
//	// Agents can request input using the broker
//	response, err := broker.RequestConfirm(ctx, "Delete file?", "This cannot be undone.")
type UserInputBroker struct {
	mu       sync.Mutex
	pending  map[string]chan UserInputResponse
	timeout  time.Duration
	onSubmit func(req UserInputRequest) // Called when a request is submitted
}

// NewUserInputBroker creates a new UserInputBroker.
// The default timeout for waiting on responses is 5 minutes.
func NewUserInputBroker() *UserInputBroker {
	return &UserInputBroker{
		pending: make(map[string]chan UserInputResponse),
		timeout: 5 * time.Minute,
	}
}

// UserInputBrokerOption configures a UserInputBroker.
type UserInputBrokerOption func(*UserInputBroker)

// WithInputTimeout sets the timeout for waiting on user responses.
func WithInputTimeout(d time.Duration) UserInputBrokerOption {
	return func(b *UserInputBroker) {
		b.timeout = d
	}
}

// WithOnInputSubmit sets a callback that's called when an input request is submitted.
// This is useful for emitting AG-UI activity events.
func WithOnInputSubmit(fn func(req UserInputRequest)) UserInputBrokerOption {
	return func(b *UserInputBroker) {
		b.onSubmit = fn
	}
}

// NewUserInputBrokerWith creates a new UserInputBroker with options.
func NewUserInputBrokerWith(opts ...UserInputBrokerOption) *UserInputBroker {
	b := NewUserInputBroker()
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Respond sends a user response to the broker.
// The response will be routed to the waiting input request.
// Returns an error if there is no pending request for the given ID.
func (b *UserInputBroker) Respond(response UserInputResponse) error {
	b.mu.Lock()
	ch, ok := b.pending[response.RequestID]
	b.mu.Unlock()

	if !ok {
		return fmt.Errorf("no pending input request %q", response.RequestID)
	}

	select {
	case ch <- response:
	default:
	}

	return nil
}

// PendingCount returns the number of pending input requests.
func (b *UserInputBroker) PendingCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pending)
}

// HasPending returns true if there are any pending input requests.
func (b *UserInputBroker) HasPending() bool {
	return b.PendingCount() > 0
}

// Request sends a user input request and waits for a response.
// This is the low-level method; prefer the typed methods like RequestConfirm.
func (b *UserInputBroker) Request(ctx context.Context, req UserInputRequest) (*UserInputResponse, error) {
	// Generate ID if not set
	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	// Create response channel
	ch := make(chan UserInputResponse, 1)

	// Register pending request
	b.mu.Lock()
	b.pending[req.ID] = ch
	b.mu.Unlock()

	// Clean up when done
	defer func() {
		b.mu.Lock()
		delete(b.pending, req.ID)
		close(ch)
		b.mu.Unlock()
	}()

	// Call the onSubmit callback if set
	if b.onSubmit != nil {
		b.onSubmit(req)
	}

	// Wait for response with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, b.timeout)
	defer cancel()

	select {
	case response := <-ch:
		return &response, nil
	case <-timeoutCtx.Done():
		if ctx.Err() != nil {
			return nil, fmt.Errorf("input request cancelled")
		}
		return nil, fmt.Errorf("input request timeout")
	}
}

// RequestConfirm requests a yes/no confirmation from the user.
// Returns true if confirmed, false if rejected or cancelled.
func (b *UserInputBroker) RequestConfirm(ctx context.Context, title, message string) (bool, error) {
	response, err := b.Request(ctx, UserInputRequest{
		Type:    InputTypeConfirm,
		Title:   title,
		Message: message,
	})
	if err != nil {
		return false, err
	}
	if response.Cancelled {
		return false, nil
	}
	return response.Confirmed, nil
}

// RequestText requests text input from the user.
// Returns the entered text, or empty string if cancelled.
func (b *UserInputBroker) RequestText(ctx context.Context, title, message, placeholder, defaultValue string) (string, error) {
	response, err := b.Request(ctx, UserInputRequest{
		Type:        InputTypeText,
		Title:       title,
		Message:     message,
		Placeholder: placeholder,
		Default:     defaultValue,
	})
	if err != nil {
		return "", err
	}
	if response.Cancelled {
		return "", nil
	}
	return response.Value, nil
}

// RequestChoice requests the user to select from a list of choices.
// Returns the selected choice, or empty string if cancelled.
func (b *UserInputBroker) RequestChoice(ctx context.Context, title, message string, choices []string, defaultChoice string) (string, error) {
	response, err := b.Request(ctx, UserInputRequest{
		Type:    InputTypeChoice,
		Title:   title,
		Message: message,
		Choices: choices,
		Default: defaultChoice,
	})
	if err != nil {
		return "", err
	}
	if response.Cancelled {
		return "", nil
	}
	return response.Value, nil
}
