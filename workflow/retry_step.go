package workflow

import (
	"context"

	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/internal/retry"
)

// RetryStep wraps a step with retry logic.
// Transient errors are retried with exponential backoff.
type RetryStep[S any] struct {
	name   string
	step   Step[S]
	config retry.Config
}

// NewRetryStep creates a step that retries on transient errors.
// Uses the default retry configuration (10 attempts, exponential backoff).
//
// Example:
//
//	step := NewRetryStep("fetch-with-retry", fetchStep)
func NewRetryStep[S any](name string, step Step[S]) *RetryStep[S] {
	return &RetryStep[S]{
		name:   name,
		step:   step,
		config: retry.DefaultConfig(),
	}
}

// NewRetryStepWithConfig creates a step with custom retry configuration.
//
// Example:
//
//	cfg := retry.Config{
//	    MaxAttempts:  3,
//	    InitialDelay: 500 * time.Millisecond,
//	    MaxDelay:     5 * time.Second,
//	    Multiplier:   2.0,
//	    Jitter:       0.1,
//	}
//	step := NewRetryStepWithConfig("fetch", fetchStep, cfg)
func NewRetryStepWithConfig[S any](name string, step Step[S], config retry.Config) *RetryStep[S] {
	return &RetryStep[S]{
		name:   name,
		step:   step,
		config: config,
	}
}

// Name returns the step name.
func (r *RetryStep[S]) Name() string { return r.name }

// Run executes the wrapped step with retry logic.
func (r *RetryStep[S]) Run(ctx context.Context, state *S, opts ...Option) error {
	_, err := retry.Do(ctx, r.config, func() (struct{}, error) {
		err := r.step.Run(ctx, state, opts...)
		return struct{}{}, err
	})
	return err
}

// RunStream executes the wrapped step with retry logic and emits events.
// Retry events are emitted to provide observability into retry attempts.
func (r *RetryStep[S]) RunStream(ctx context.Context, state *S, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		event.Emit(ch, Event{Type: event.StepStart, StepName: r.name})

		// Create event channel for retry observability
		retryEvents := make(chan retry.Event, 10)

		// Run with events in background, closing retryEvents when done
		var runErr error
		go func() {
			defer close(retryEvents)
			_, runErr = retry.DoWithEvents(ctx, r.config, retryEvents, func() (struct{}, error) {
				err := r.step.Run(ctx, state, opts...)
				return struct{}{}, err
			})
		}()

		// Forward retry events as workflow events
		for re := range retryEvents {
			ch <- Event{
				Type:     mapRetryEventType(re.Type),
				StepName: r.name,
				Error:    re.Error,
				Attempt:  re.Attempt,
				Message:  formatRetryMessage(re),
			}
		}

		// Channel is closed, emit final event
		if runErr != nil {
			event.Emit(ch, Event{Type: event.RunError, StepName: r.name, Error: runErr})
			return
		}

		event.Emit(ch, Event{Type: event.StepEnd, StepName: r.name})
	}()

	return ch
}

// mapRetryEventType maps retry event types to workflow event types.
func mapRetryEventType(t retry.EventType) event.Type {
	switch t {
	case retry.EventAttemptStart:
		return event.RetryAttempt
	case retry.EventAttemptFailed:
		return event.RetryFailed
	case retry.EventRetrying:
		return event.RetryScheduled
	case retry.EventSuccess:
		return event.RetrySuccess
	case retry.EventExhausted:
		return event.RetryExhausted
	default:
		return event.StepStart
	}
}

// formatRetryMessage creates a human-readable message for retry events.
func formatRetryMessage(e retry.Event) string {
	switch e.Type {
	case retry.EventAttemptStart:
		return ""
	case retry.EventAttemptFailed:
		if e.Retryable {
			return "attempt failed, will retry"
		}
		return "attempt failed, not retryable"
	case retry.EventRetrying:
		return e.Delay.String()
	case retry.EventSuccess:
		return "succeeded"
	case retry.EventExhausted:
		return "all attempts exhausted"
	default:
		return ""
	}
}
