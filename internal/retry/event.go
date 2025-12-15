package retry

import "time"

// EventType identifies the kind of event occurring during retry execution.
type EventType string

const (
	// EventAttemptStart fires before each attempt.
	EventAttemptStart EventType = "attempt_start"

	// EventAttemptFailed fires after a failed attempt.
	EventAttemptFailed EventType = "attempt_failed"

	// EventRetrying fires before sleeping between attempts.
	EventRetrying EventType = "retrying"

	// EventSuccess fires when an attempt succeeds.
	EventSuccess EventType = "success"

	// EventExhausted fires when all retry attempts are exhausted.
	EventExhausted EventType = "exhausted"
)

// Event represents an observable occurrence during retry execution.
type Event struct {
	// Type identifies the kind of event.
	Type EventType

	// Attempt is the current attempt number (1-indexed).
	Attempt int

	// MaxAttempts is the total number of attempts allowed.
	MaxAttempts int

	// Error contains the error from a failed attempt.
	Error error

	// Delay is the duration before the next attempt (for EventRetrying).
	Delay time.Duration

	// Retryable indicates whether the error was classified as transient.
	Retryable bool

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
