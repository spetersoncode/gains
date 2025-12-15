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
