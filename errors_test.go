package gains

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestErrEmptyInput(t *testing.T) {
	t.Run("is a sentinel error", func(t *testing.T) {
		assert.Error(t, ErrEmptyInput)
		assert.Equal(t, "empty input", ErrEmptyInput.Error())
	})

	t.Run("can be compared with errors.Is", func(t *testing.T) {
		err := ErrEmptyInput
		assert.True(t, errors.Is(err, ErrEmptyInput))
	})
}

func TestImageError(t *testing.T) {
	t.Run("Error returns formatted message", func(t *testing.T) {
		tests := []struct {
			name     string
			op       string
			url      string
			err      error
			expected string
		}{
			{
				name:     "decode error",
				op:       "decode",
				url:      "base64",
				err:      errors.New("invalid encoding"),
				expected: "image decode error for base64: invalid encoding",
			},
			{
				name:     "fetch error",
				op:       "fetch",
				url:      "https://example.com/image.png",
				err:      errors.New("connection refused"),
				expected: "image fetch error for https://example.com/image.png: connection refused",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				imgErr := &ImageError{
					Op:  tt.op,
					URL: tt.url,
					Err: tt.err,
				}
				assert.Equal(t, tt.expected, imgErr.Error())
			})
		}
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		underlyingErr := errors.New("underlying error")
		imgErr := &ImageError{
			Op:  "decode",
			URL: "test.png",
			Err: underlyingErr,
		}

		assert.Equal(t, underlyingErr, imgErr.Unwrap())
		assert.True(t, errors.Is(imgErr, underlyingErr))
	})

	t.Run("Unwrap returns nil when no underlying error", func(t *testing.T) {
		imgErr := &ImageError{
			Op:  "decode",
			URL: "test.png",
			Err: nil,
		}

		assert.Nil(t, imgErr.Unwrap())
	})
}

func TestUnmarshalError(t *testing.T) {
	t.Run("Error without context", func(t *testing.T) {
		err := &UnmarshalError{
			Content:    `{"invalid": json}`,
			TargetType: "BookInfo",
			Err:        errors.New("unexpected end of JSON input"),
		}
		expected := "failed to unmarshal response into BookInfo: unexpected end of JSON input"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Error with context", func(t *testing.T) {
		err := &UnmarshalError{
			Context:    "workflow: step \"parse\"",
			Content:    `{"invalid": json}`,
			TargetType: "BookInfo",
			Err:        errors.New("unexpected end of JSON input"),
		}
		expected := "workflow: step \"parse\": failed to unmarshal response into BookInfo: unexpected end of JSON input"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		underlying := errors.New("parse error")
		err := &UnmarshalError{
			Content:    "invalid",
			TargetType: "TestType",
			Err:        underlying,
		}
		assert.Equal(t, underlying, err.Unwrap())
		assert.True(t, errors.Is(err, underlying))
	})
}

func TestErrorCategory(t *testing.T) {
	t.Run("constants have expected values", func(t *testing.T) {
		assert.Equal(t, ErrorCategory("transient"), ErrorTransient)
		assert.Equal(t, ErrorCategory("permanent"), ErrorPermanent)
		assert.Equal(t, ErrorCategory("user_input"), ErrorUserInput)
	})
}

func TestError(t *testing.T) {
	t.Run("Error message without cause", func(t *testing.T) {
		err := &Error{Msg: "rate limited", Cat: ErrorTransient, Code: 429}
		assert.Equal(t, "rate limited", err.Error())
	})

	t.Run("Error message with cause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := &Error{Msg: "rate limited", Cat: ErrorTransient, Code: 429, Cause: cause}
		assert.Equal(t, "rate limited: underlying error", err.Error())
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := &Error{Msg: "failed", Cause: cause}
		assert.Equal(t, cause, err.Unwrap())
		assert.True(t, errors.Is(err, cause))
	})

	t.Run("implements CategorizedError", func(t *testing.T) {
		err := &Error{Msg: "test", Cat: ErrorTransient, Code: 429, RetryDelay: 5 * time.Second}

		var ce CategorizedError
		assert.True(t, errors.As(err, &ce))
		assert.Equal(t, ErrorTransient, ce.Category())
		assert.True(t, ce.Retryable())
		assert.Equal(t, 429, ce.StatusCode())
		assert.Equal(t, 5*time.Second, ce.RetryAfter())
	})

	t.Run("Retryable returns true only for transient", func(t *testing.T) {
		transient := &Error{Cat: ErrorTransient}
		permanent := &Error{Cat: ErrorPermanent}
		userInput := &Error{Cat: ErrorUserInput}

		assert.True(t, transient.Retryable())
		assert.False(t, permanent.Retryable())
		assert.False(t, userInput.Retryable())
	})
}

func TestErrorFactories(t *testing.T) {
	cause := errors.New("original error")

	t.Run("NewTransientError", func(t *testing.T) {
		err := NewTransientError("rate limited", 429, cause)
		assert.Equal(t, ErrorTransient, err.Category())
		assert.Equal(t, 429, err.StatusCode())
		assert.Equal(t, time.Duration(0), err.RetryAfter())
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("NewTransientErrorWithRetry", func(t *testing.T) {
		err := NewTransientErrorWithRetry("rate limited", 429, 30*time.Second, cause)
		assert.Equal(t, ErrorTransient, err.Category())
		assert.Equal(t, 429, err.StatusCode())
		assert.Equal(t, 30*time.Second, err.RetryAfter())
		assert.Equal(t, cause, err.Unwrap())
	})

	t.Run("NewPermanentError", func(t *testing.T) {
		err := NewPermanentError("unauthorized", 401, cause)
		assert.Equal(t, ErrorPermanent, err.Category())
		assert.Equal(t, 401, err.StatusCode())
		assert.False(t, err.Retryable())
	})

	t.Run("NewUserInputError", func(t *testing.T) {
		err := NewUserInputError("invalid request", 400, cause)
		assert.Equal(t, ErrorUserInput, err.Category())
		assert.Equal(t, 400, err.StatusCode())
		assert.False(t, err.Retryable())
	})
}

func TestCategoryHelpers(t *testing.T) {
	transient := NewTransientError("rate limited", 429, nil)
	permanent := NewPermanentError("unauthorized", 401, nil)
	userInput := NewUserInputError("bad request", 400, nil)
	plain := errors.New("plain error")

	t.Run("IsTransient", func(t *testing.T) {
		assert.True(t, IsTransient(transient))
		assert.False(t, IsTransient(permanent))
		assert.False(t, IsTransient(userInput))
		assert.False(t, IsTransient(plain))
		assert.False(t, IsTransient(nil))
	})

	t.Run("IsPermanent", func(t *testing.T) {
		assert.False(t, IsPermanent(transient))
		assert.True(t, IsPermanent(permanent))
		assert.False(t, IsPermanent(userInput))
		assert.False(t, IsPermanent(plain))
		assert.False(t, IsPermanent(nil))
	})

	t.Run("IsUserInput", func(t *testing.T) {
		assert.False(t, IsUserInput(transient))
		assert.False(t, IsUserInput(permanent))
		assert.True(t, IsUserInput(userInput))
		assert.False(t, IsUserInput(plain))
		assert.False(t, IsUserInput(nil))
	})

	t.Run("works with wrapped errors", func(t *testing.T) {
		wrapped := fmt.Errorf("wrapped: %w", transient)
		assert.True(t, IsTransient(wrapped))

		doubleWrapped := fmt.Errorf("outer: %w", wrapped)
		assert.True(t, IsTransient(doubleWrapped))
	})
}

func TestStatusCodeOf(t *testing.T) {
	t.Run("returns status code from categorized error", func(t *testing.T) {
		err := NewTransientError("rate limited", 429, nil)
		assert.Equal(t, 429, StatusCodeOf(err))
	})

	t.Run("returns 0 for plain error", func(t *testing.T) {
		err := errors.New("plain error")
		assert.Equal(t, 0, StatusCodeOf(err))
	})

	t.Run("returns 0 for nil", func(t *testing.T) {
		assert.Equal(t, 0, StatusCodeOf(nil))
	})

	t.Run("works with wrapped error", func(t *testing.T) {
		inner := NewTransientError("rate limited", 429, nil)
		wrapped := fmt.Errorf("wrapped: %w", inner)
		assert.Equal(t, 429, StatusCodeOf(wrapped))
	})
}

func TestRetryAfterOf(t *testing.T) {
	t.Run("returns retry delay from categorized error", func(t *testing.T) {
		err := NewTransientErrorWithRetry("rate limited", 429, 30*time.Second, nil)
		assert.Equal(t, 30*time.Second, RetryAfterOf(err))
	})

	t.Run("returns 0 for error without retry delay", func(t *testing.T) {
		err := NewTransientError("rate limited", 429, nil)
		assert.Equal(t, time.Duration(0), RetryAfterOf(err))
	})

	t.Run("returns 0 for plain error", func(t *testing.T) {
		err := errors.New("plain error")
		assert.Equal(t, time.Duration(0), RetryAfterOf(err))
	})

	t.Run("returns 0 for nil", func(t *testing.T) {
		assert.Equal(t, time.Duration(0), RetryAfterOf(nil))
	})
}
