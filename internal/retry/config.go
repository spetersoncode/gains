// Package retry provides retry logic with exponential backoff for transient errors.
package retry

import (
	"math"
	"math/rand"
	"time"
)

// Config holds retry configuration parameters.
type Config struct {
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

// DefaultConfig returns the default retry configuration.
// - 10 max attempts
// - 1 second initial delay
// - 60 second max delay
// - 2x exponential multiplier
// - 10% jitter
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  10,
		InitialDelay: 1 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}
}

// Disabled returns a configuration that disables retries (single attempt).
func Disabled() Config {
	return Config{MaxAttempts: 1}
}

// Delay calculates the delay for a given attempt number (0-indexed).
// Formula: min(maxDelay, initialDelay * multiplier^attempt) * (1 + jitter)
func (c Config) Delay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	delay := float64(c.InitialDelay) * math.Pow(c.Multiplier, float64(attempt))
	if delay > float64(c.MaxDelay) {
		delay = float64(c.MaxDelay)
	}

	// Apply jitter: random value in range [-jitter, +jitter]
	if c.Jitter > 0 {
		jitterFactor := 1.0 + (rand.Float64()*2-1)*c.Jitter
		delay *= jitterFactor
	}

	return time.Duration(delay)
}
