package gains

import "time"

// RetryConfig holds retry configuration parameters.
// Use DefaultRetryConfig() for sensible defaults or create custom configs.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (default: 10).
	// The initial request counts as attempt 1.
	MaxAttempts int

	// InitialDelay is the base delay before the first retry (default: 1s).
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries (default: 60s).
	MaxDelay time.Duration

	// Multiplier is the exponential backoff multiplier (default: 2.0).
	Multiplier float64

	// Jitter adds randomness to prevent thundering herd (default: 0.1 = 10%).
	// Delay is multiplied by (1 + random(-jitter, +jitter)).
	Jitter float64
}

// DefaultRetryConfig returns the default retry configuration.
//   - 10 max attempts
//   - 1 second initial delay
//   - 60 second max delay
//   - 2x exponential multiplier
//   - 10% jitter
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 1 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}
}

// DisabledRetryConfig returns a configuration that disables retries (single attempt).
func DisabledRetryConfig() RetryConfig {
	return RetryConfig{MaxAttempts: 1}
}

// NewRetryConfig creates a custom retry configuration.
// Parameters:
//   - maxAttempts: maximum number of attempts (including initial request)
//   - initialDelay: base delay before first retry
//   - maxDelay: maximum delay between retries
//   - multiplier: exponential backoff multiplier (e.g., 2.0 doubles delay each retry)
//   - jitter: random jitter factor (e.g., 0.1 adds ±10% randomness)
func NewRetryConfig(maxAttempts int, initialDelay, maxDelay time.Duration, multiplier, jitter float64) RetryConfig {
	return RetryConfig{
		MaxAttempts:  maxAttempts,
		InitialDelay: initialDelay,
		MaxDelay:     maxDelay,
		Multiplier:   multiplier,
		Jitter:       jitter,
	}
}
