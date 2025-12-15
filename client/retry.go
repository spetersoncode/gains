package client

import (
	"time"

	"github.com/spetersoncode/gains/internal/retry"
)

// RetryConfig holds retry configuration parameters.
type RetryConfig = retry.Config

// RetryEvent represents an observable occurrence during retry execution.
type RetryEvent = retry.Event

// RetryEventType identifies the kind of event occurring during retry execution.
type RetryEventType = retry.EventType

// Retry event type constants.
const (
	RetryEventAttemptStart  = retry.EventAttemptStart
	RetryEventAttemptFailed = retry.EventAttemptFailed
	RetryEventRetrying      = retry.EventRetrying
	RetryEventSuccess       = retry.EventSuccess
	RetryEventExhausted     = retry.EventExhausted
)

// DefaultRetryConfig returns the default retry configuration.
//   - 10 max attempts
//   - 1 second initial delay
//   - 60 second max delay
//   - 2x exponential multiplier
//   - 10% jitter
func DefaultRetryConfig() RetryConfig {
	return retry.DefaultConfig()
}

// DisabledRetryConfig returns a configuration that disables retries (single attempt).
func DisabledRetryConfig() RetryConfig {
	return retry.Disabled()
}

// IsTransientError determines if an error is transient and should be retried.
// It checks for rate limits, server errors, network timeouts, and connection issues.
func IsTransientError(err error) bool {
	return retry.IsTransient(err)
}

// Ensure time import is used (for documentation examples).
var _ time.Duration
