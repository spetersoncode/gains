package retry

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

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
