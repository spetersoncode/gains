package client

import (
	"time"

	"github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/retry"
)

// EventType identifies the kind of event occurring during client operations.
type EventType string

const (
	// EventRequestStart fires before an API request begins.
	EventRequestStart EventType = "request_start"

	// EventRequestComplete fires after an API request completes successfully.
	EventRequestComplete EventType = "request_complete"

	// EventRequestError fires when an API request fails.
	EventRequestError EventType = "request_error"

	// EventRetry fires when a retry event occurs (forwarded from retry package).
	EventRetry EventType = "retry"
)

// Event represents an observable occurrence during client operations.
type Event struct {
	// Type identifies the kind of event.
	Type EventType

	// Operation identifies the API operation ("chat", "chat_stream", "embed", "image").
	Operation string

	// Provider identifies which AI provider is being used.
	Provider ProviderName

	// Model is the model name being used (if known).
	Model string

	// Duration is the elapsed time for completed requests.
	Duration time.Duration

	// Usage contains token usage information (for chat operations).
	Usage *gains.Usage

	// Error contains the error for EventRequestError.
	Error error

	// RetryEvent contains the underlying retry event for EventRetry.
	RetryEvent *retry.Event

	// Timestamp is when the event occurred.
	Timestamp time.Time
}

// emit sends an event with timestamp to the channel without blocking.
func emit(ch chan<- Event, event Event) {
	if ch == nil {
		return
	}
	event.Timestamp = time.Now()
	select {
	case ch <- event:
	default:
		// Channel full - don't block
	}
}
