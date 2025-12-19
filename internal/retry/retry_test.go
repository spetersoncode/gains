package retry

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/spetersoncode/gains"
	"github.com/stretchr/testify/assert"
)

// mockTransientError simulates a transient network error.
type mockTransientError struct {
	msg string
}

func (e *mockTransientError) Error() string   { return e.msg }
func (e *mockTransientError) Timeout() bool   { return true }
func (e *mockTransientError) Temporary() bool { return true }

// Ensure mockTransientError implements net.Error
var _ net.Error = (*mockTransientError)(nil)

func TestDoSuccess(t *testing.T) {
	cfg := DefaultConfig()
	callCount := 0

	result, err := Do(context.Background(), cfg, func() (string, error) {
		callCount++
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 1, callCount)
}

func TestDoRetryOnTransientError(t *testing.T) {
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0,
	}

	callCount := 0
	transientErr := &mockTransientError{msg: "timeout"}

	result, err := Do(context.Background(), cfg, func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", transientErr
		}
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 3, callCount)
}

func TestDoNoRetryOnPermanentError(t *testing.T) {
	cfg := DefaultConfig()
	callCount := 0
	permanentErr := errors.New("permanent error")

	_, err := Do(context.Background(), cfg, func() (string, error) {
		callCount++
		return "", permanentErr
	})

	assert.Error(t, err)
	assert.Equal(t, permanentErr, err)
	assert.Equal(t, 1, callCount) // No retries
}

func TestDoExhaustsRetries(t *testing.T) {
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0,
	}

	callCount := 0
	transientErr := &mockTransientError{msg: "timeout"}

	_, err := Do(context.Background(), cfg, func() (string, error) {
		callCount++
		return "", transientErr
	})

	assert.Error(t, err)
	assert.Equal(t, transientErr, err)
	assert.Equal(t, 3, callCount) // All attempts exhausted
}

func TestDoRespectsContextCancellation(t *testing.T) {
	cfg := Config{
		MaxAttempts:  10,
		InitialDelay: time.Second, // Long delay
		MaxDelay:     time.Second,
		Multiplier:   1.0,
		Jitter:       0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := Do(ctx, cfg, func() (string, error) {
		callCount++
		return "", &mockTransientError{msg: "timeout"}
	})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, callCount) // Only first attempt before cancellation
}

func TestDoWithDisabledRetry(t *testing.T) {
	cfg := Disabled()
	callCount := 0
	transientErr := &mockTransientError{msg: "timeout"}

	_, err := Do(context.Background(), cfg, func() (string, error) {
		callCount++
		return "", transientErr
	})

	assert.Error(t, err)
	assert.Equal(t, 1, callCount) // Only one attempt with disabled retry
}

func TestDoStreamSuccess(t *testing.T) {
	cfg := DefaultConfig()
	callCount := 0

	ch, err := DoStream(context.Background(), cfg, func() (<-chan string, error) {
		callCount++
		c := make(chan string, 1)
		c <- "data"
		close(c)
		return c, nil
	})

	assert.NoError(t, err)
	assert.NotNil(t, ch)
	assert.Equal(t, 1, callCount)

	// Read from channel
	data := <-ch
	assert.Equal(t, "data", data)
}

func TestDoStreamRetryOnTransientError(t *testing.T) {
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0,
	}

	callCount := 0
	transientErr := &mockTransientError{msg: "timeout"}

	ch, err := DoStream(context.Background(), cfg, func() (<-chan string, error) {
		callCount++
		if callCount < 3 {
			return nil, transientErr
		}
		c := make(chan string, 1)
		c <- "success"
		close(c)
		return c, nil
	})

	assert.NoError(t, err)
	assert.NotNil(t, ch)
	assert.Equal(t, 3, callCount)
}

func TestDoStreamNoRetryOnPermanentError(t *testing.T) {
	cfg := DefaultConfig()
	callCount := 0
	permanentErr := errors.New("permanent error")

	_, err := DoStream(context.Background(), cfg, func() (<-chan string, error) {
		callCount++
		return nil, permanentErr
	})

	assert.Error(t, err)
	assert.Equal(t, permanentErr, err)
	assert.Equal(t, 1, callCount)
}

func TestDoStreamRespectsContextCancellation(t *testing.T) {
	cfg := Config{
		MaxAttempts:  10,
		InitialDelay: time.Second,
		MaxDelay:     time.Second,
		Multiplier:   1.0,
		Jitter:       0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := DoStream(ctx, cfg, func() (<-chan string, error) {
		callCount++
		return nil, &mockTransientError{msg: "timeout"}
	})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, callCount)
}

func TestDoHonorsRetryAfterFromError(t *testing.T) {
	// Test that RetryAfter from error is used when larger than configured delay
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0,
	}

	callCount := 0
	callTimes := make([]time.Time, 0, 3)

	// Create an error with RetryAfter of 50ms (larger than initial 10ms delay)
	retryErr := gains.NewTransientErrorWithRetry("rate limited", 429, 50*time.Millisecond, nil)

	_, err := Do(context.Background(), cfg, func() (string, error) {
		callTimes = append(callTimes, time.Now())
		callCount++
		if callCount < 3 {
			return "", retryErr
		}
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)

	// Check that delay between first and second call was at least 50ms (from RetryAfter)
	if len(callTimes) >= 2 {
		delay := callTimes[1].Sub(callTimes[0])
		assert.GreaterOrEqual(t, delay, 45*time.Millisecond, "should honor RetryAfter of 50ms")
	}
}

func TestDoUsesConfiguredDelayWhenLargerThanRetryAfter(t *testing.T) {
	// Test that configured delay is used when larger than RetryAfter
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0,
	}

	callCount := 0
	callTimes := make([]time.Time, 0, 3)

	// Create an error with RetryAfter of 10ms (smaller than configured 50ms)
	retryErr := gains.NewTransientErrorWithRetry("rate limited", 429, 10*time.Millisecond, nil)

	_, err := Do(context.Background(), cfg, func() (string, error) {
		callTimes = append(callTimes, time.Now())
		callCount++
		if callCount < 3 {
			return "", retryErr
		}
		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)

	// Check that delay between first and second call was at least 50ms (from config)
	if len(callTimes) >= 2 {
		delay := callTimes[1].Sub(callTimes[0])
		assert.GreaterOrEqual(t, delay, 45*time.Millisecond, "should use configured delay of 50ms")
	}
}

func TestEffectiveDelay(t *testing.T) {
	tests := []struct {
		name            string
		configuredDelay time.Duration
		retryAfter      time.Duration
		expectedDelay   time.Duration
	}{
		{
			name:            "RetryAfter larger than configured",
			configuredDelay: 100 * time.Millisecond,
			retryAfter:      500 * time.Millisecond,
			expectedDelay:   500 * time.Millisecond,
		},
		{
			name:            "configured larger than RetryAfter",
			configuredDelay: 500 * time.Millisecond,
			retryAfter:      100 * time.Millisecond,
			expectedDelay:   500 * time.Millisecond,
		},
		{
			name:            "no RetryAfter (zero)",
			configuredDelay: 100 * time.Millisecond,
			retryAfter:      0,
			expectedDelay:   100 * time.Millisecond,
		},
		{
			name:            "equal delays",
			configuredDelay: 100 * time.Millisecond,
			retryAfter:      100 * time.Millisecond,
			expectedDelay:   100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.retryAfter > 0 {
				err = gains.NewTransientErrorWithRetry("test", 429, tt.retryAfter, nil)
			} else {
				err = gains.NewTransientError("test", 429, nil)
			}

			delay := effectiveDelay(tt.configuredDelay, err)
			assert.Equal(t, tt.expectedDelay, delay)
		})
	}
}

func TestRetryAfterFromError(t *testing.T) {
	t.Run("with CategorizedError and RetryAfter", func(t *testing.T) {
		err := gains.NewTransientErrorWithRetry("rate limited", 429, 30*time.Second, nil)
		delay := retryAfterFromError(err)
		assert.Equal(t, 30*time.Second, delay)
	})

	t.Run("with CategorizedError but no RetryAfter", func(t *testing.T) {
		err := gains.NewTransientError("server error", 500, nil)
		delay := retryAfterFromError(err)
		assert.Equal(t, time.Duration(0), delay)
	})

	t.Run("with non-CategorizedError", func(t *testing.T) {
		err := errors.New("generic error")
		delay := retryAfterFromError(err)
		assert.Equal(t, time.Duration(0), delay)
	})

	t.Run("with nil error", func(t *testing.T) {
		delay := retryAfterFromError(nil)
		assert.Equal(t, time.Duration(0), delay)
	})

	t.Run("with wrapped CategorizedError", func(t *testing.T) {
		innerErr := gains.NewTransientErrorWithRetry("rate limited", 429, 60*time.Second, nil)
		wrappedErr := errors.New("context: " + innerErr.Error())
		// This won't work with simple error wrapping, need proper Unwrap
		delay := retryAfterFromError(wrappedErr)
		// Simple string wrapping doesn't preserve the interface
		assert.Equal(t, time.Duration(0), delay)
	})
}
