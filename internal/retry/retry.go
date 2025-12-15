package retry

import (
	"context"
	"time"
)

// Do executes the given function with retry logic.
// It respects context cancellation during backoff waits.
// Returns the result on success, or the last error if all attempts fail.
func Do[T any](ctx context.Context, cfg Config, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !IsTransient(err) {
			return zero, err
		}

		// Don't sleep after the last attempt
		if attempt < cfg.MaxAttempts-1 {
			delay := cfg.Delay(attempt)

			// Respect context cancellation during sleep
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	return zero, lastErr
}

// DoStream is like Do but for functions that return a channel.
// It retries the stream connection establishment, not individual chunks.
func DoStream[T any](ctx context.Context, cfg Config, fn func() (<-chan T, error)) (<-chan T, error) {
	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		ch, err := fn()
		if err == nil {
			return ch, nil
		}

		lastErr = err

		// Check if error is retryable
		if !IsTransient(err) {
			return nil, err
		}

		// Don't sleep after the last attempt
		if attempt < cfg.MaxAttempts-1 {
			delay := cfg.Delay(attempt)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	return nil, lastErr
}

// DoWithEvents is like Do but emits events for observability.
// Events are sent non-blocking; if the channel is full, events are dropped.
// Pass nil for events to disable event emission (equivalent to Do).
func DoWithEvents[T any](ctx context.Context, cfg Config, events chan<- Event, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		emit(events, Event{
			Type:        EventAttemptStart,
			Attempt:     attempt + 1,
			MaxAttempts: cfg.MaxAttempts,
		})

		result, err := fn()
		if err == nil {
			emit(events, Event{
				Type:        EventSuccess,
				Attempt:     attempt + 1,
				MaxAttempts: cfg.MaxAttempts,
			})
			return result, nil
		}

		lastErr = err
		retryable := IsTransient(err)

		emit(events, Event{
			Type:        EventAttemptFailed,
			Attempt:     attempt + 1,
			MaxAttempts: cfg.MaxAttempts,
			Error:       err,
			Retryable:   retryable,
		})

		// Check if error is retryable
		if !retryable {
			return zero, err
		}

		// Don't sleep after the last attempt
		if attempt < cfg.MaxAttempts-1 {
			delay := cfg.Delay(attempt)

			emit(events, Event{
				Type:        EventRetrying,
				Attempt:     attempt + 1,
				MaxAttempts: cfg.MaxAttempts,
				Delay:       delay,
			})

			// Respect context cancellation during sleep
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	emit(events, Event{
		Type:        EventExhausted,
		Attempt:     cfg.MaxAttempts,
		MaxAttempts: cfg.MaxAttempts,
		Error:       lastErr,
	})

	return zero, lastErr
}

// DoStreamWithEvents is like DoStream but emits events for observability.
// Events are sent non-blocking; if the channel is full, events are dropped.
// Pass nil for events to disable event emission (equivalent to DoStream).
func DoStreamWithEvents[T any](ctx context.Context, cfg Config, events chan<- Event, fn func() (<-chan T, error)) (<-chan T, error) {
	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		emit(events, Event{
			Type:        EventAttemptStart,
			Attempt:     attempt + 1,
			MaxAttempts: cfg.MaxAttempts,
		})

		ch, err := fn()
		if err == nil {
			emit(events, Event{
				Type:        EventSuccess,
				Attempt:     attempt + 1,
				MaxAttempts: cfg.MaxAttempts,
			})
			return ch, nil
		}

		lastErr = err
		retryable := IsTransient(err)

		emit(events, Event{
			Type:        EventAttemptFailed,
			Attempt:     attempt + 1,
			MaxAttempts: cfg.MaxAttempts,
			Error:       err,
			Retryable:   retryable,
		})

		// Check if error is retryable
		if !retryable {
			return nil, err
		}

		// Don't sleep after the last attempt
		if attempt < cfg.MaxAttempts-1 {
			delay := cfg.Delay(attempt)

			emit(events, Event{
				Type:        EventRetrying,
				Attempt:     attempt + 1,
				MaxAttempts: cfg.MaxAttempts,
				Delay:       delay,
			})

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	emit(events, Event{
		Type:        EventExhausted,
		Attempt:     cfg.MaxAttempts,
		MaxAttempts: cfg.MaxAttempts,
		Error:       lastErr,
	})

	return nil, lastErr
}
