package retry

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/spetersoncode/gains"
	"github.com/stretchr/testify/assert"
)

// mockAPIError simulates an API error with a status code.
type mockAPIError struct {
	code int
	msg  string
}

func (e *mockAPIError) Error() string    { return e.msg }
func (e *mockAPIError) StatusCode() int { return e.code }

// mockNetError simulates a network error with timeout/temporary flags.
type mockNetError struct {
	msg       string
	timeout   bool
	temporary bool
}

func (e *mockNetError) Error() string   { return e.msg }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temporary }

var _ net.Error = (*mockNetError)(nil)

func TestIsTransientStatusCode(t *testing.T) {
	tests := []struct {
		code     int
		expected bool
	}{
		{200, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true},  // Rate limit
		{500, true},  // Internal server error
		{502, true},  // Bad gateway
		{503, true},  // Service unavailable
		{504, true},  // Gateway timeout
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.code), func(t *testing.T) {
			assert.Equal(t, tt.expected, isTransientStatusCode(tt.code))
		})
	}
}

func TestIsTransientWithAPIError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "rate limit 429",
			err:      &mockAPIError{code: 429, msg: "rate limited"},
			expected: true,
		},
		{
			name:     "server error 500",
			err:      &mockAPIError{code: 500, msg: "internal server error"},
			expected: true,
		},
		{
			name:     "bad gateway 502",
			err:      &mockAPIError{code: 502, msg: "bad gateway"},
			expected: true,
		},
		{
			name:     "service unavailable 503",
			err:      &mockAPIError{code: 503, msg: "service unavailable"},
			expected: true,
		},
		{
			name:     "gateway timeout 504",
			err:      &mockAPIError{code: 504, msg: "gateway timeout"},
			expected: true,
		},
		{
			name:     "bad request 400",
			err:      &mockAPIError{code: 400, msg: "bad request"},
			expected: false,
		},
		{
			name:     "unauthorized 401",
			err:      &mockAPIError{code: 401, msg: "unauthorized"},
			expected: false,
		},
		{
			name:     "forbidden 403",
			err:      &mockAPIError{code: 403, msg: "forbidden"},
			expected: false,
		},
		{
			name:     "not found 404",
			err:      &mockAPIError{code: 404, msg: "not found"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsTransient(tt.err))
		})
	}
}

func TestIsTransientWithNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "timeout error",
			err:      &mockNetError{msg: "connection timeout", timeout: true},
			expected: true,
		},
		{
			name:     "temporary error",
			err:      &mockNetError{msg: "temporary failure", temporary: true},
			expected: true, // matched via error string pattern
		},
		{
			name:     "non-transient network error",
			err:      &mockNetError{msg: "invalid address", timeout: false, temporary: false},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsTransient(tt.err))
		})
	}
}

func TestIsTransientWithStringPatterns(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "connection reset",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "connection refused",
			err:      errors.New("dial tcp: connection refused"),
			expected: true,
		},
		{
			name:     "timeout in message",
			err:      errors.New("request timeout"),
			expected: true,
		},
		{
			name:     "rate limit in message",
			err:      errors.New("rate limit exceeded"),
			expected: true,
		},
		{
			name:     "too many requests",
			err:      errors.New("too many requests"),
			expected: true,
		},
		{
			name:     "service unavailable",
			err:      errors.New("service unavailable"),
			expected: true,
		},
		{
			name:     "bad gateway in message",
			err:      errors.New("502 bad gateway"),
			expected: true,
		},
		{
			name:     "gateway timeout in message",
			err:      errors.New("504 gateway timeout"),
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("invalid input"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsTransient(tt.err))
		})
	}
}

func TestIsTransientWithGoogleAPIErrorPatterns(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "google api 429",
			err:      errors.New("googleapi: Error 429: Rate Limit Exceeded"),
			expected: true,
		},
		{
			name:     "google api 500",
			err:      errors.New("googleapi: Error 500: Internal Server Error"),
			expected: true,
		},
		{
			name:     "google api 502",
			err:      errors.New("googleapi: Error 502: Bad Gateway"),
			expected: true,
		},
		{
			name:     "google api 503",
			err:      errors.New("googleapi: Error 503: Service Unavailable"),
			expected: true,
		},
		{
			name:     "google api 504",
			err:      errors.New("googleapi: Error 504: Gateway Timeout"),
			expected: true,
		},
		{
			name:     "google api 400",
			err:      errors.New("googleapi: Error 400: Bad Request"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsTransient(tt.err))
		})
	}
}

func TestIsTransientWithWrappedError(t *testing.T) {
	innerErr := &mockAPIError{code: 429, msg: "rate limited"}
	wrappedErr := fmt.Errorf("operation failed: %w", innerErr)

	assert.True(t, IsTransient(wrappedErr))
}

func TestIsTransientWithCategorizedError(t *testing.T) {
	t.Run("transient categorized error", func(t *testing.T) {
		err := gains.NewTransientError("rate limited", 429, nil)
		assert.True(t, IsTransient(err))
	})

	t.Run("permanent categorized error", func(t *testing.T) {
		err := gains.NewPermanentError("unauthorized", 401, nil)
		assert.False(t, IsTransient(err))
	})

	t.Run("user input categorized error", func(t *testing.T) {
		err := gains.NewUserInputError("bad request", 400, nil)
		assert.False(t, IsTransient(err))
	})

	t.Run("wrapped categorized error", func(t *testing.T) {
		inner := gains.NewTransientError("rate limited", 429, nil)
		wrapped := fmt.Errorf("failed: %w", inner)
		assert.True(t, IsTransient(wrapped))
	})

	t.Run("categorized error takes precedence over status code heuristics", func(t *testing.T) {
		// A 429 status code would normally be transient via heuristics,
		// but explicit permanent categorization should override that
		err := gains.NewPermanentError("rate limit but don't retry", 429, nil)
		assert.False(t, IsTransient(err))
	})
}
