package gains

import (
	"errors"
	"fmt"
	"time"
)

// ErrEmptyInput is returned when a required input slice is empty.
var ErrEmptyInput = errors.New("empty input")

// ErrorCategory classifies errors by how they should be handled.
type ErrorCategory string

const (
	// ErrorTransient indicates the error is temporary and the operation can be retried.
	// Examples: rate limits, temporary network issues, server overload.
	ErrorTransient ErrorCategory = "transient"

	// ErrorPermanent indicates the error is not recoverable through retry.
	// Examples: invalid API key, insufficient permissions, model not found.
	ErrorPermanent ErrorCategory = "permanent"

	// ErrorUserInput indicates the user provided invalid input that must be corrected.
	// Examples: malformed request, invalid parameters, content policy violation.
	ErrorUserInput ErrorCategory = "user_input"
)

// CategorizedError is an error that provides information about how it should be handled.
type CategorizedError interface {
	error
	Category() ErrorCategory
	Retryable() bool          // convenience: returns true if Category == ErrorTransient
	StatusCode() int          // HTTP status code if applicable, 0 otherwise
	RetryAfter() time.Duration // suggested retry delay from server, 0 if not available
}

// Error is a categorized error with metadata for error handling decisions.
type Error struct {
	Msg        string
	Cat        ErrorCategory
	Code       int           // HTTP status code, 0 if not applicable
	RetryDelay time.Duration // from Retry-After header, 0 if not available
	Cause      error         // underlying error
}

// Error returns the error message.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Cause)
	}
	return e.Msg
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Cause
}

// Category returns the error category.
func (e *Error) Category() ErrorCategory {
	return e.Cat
}

// Retryable returns true if the error is transient and can be retried.
func (e *Error) Retryable() bool {
	return e.Cat == ErrorTransient
}

// StatusCode returns the HTTP status code, or 0 if not applicable.
func (e *Error) StatusCode() int {
	return e.Code
}

// RetryAfter returns the suggested retry delay, or 0 if not available.
func (e *Error) RetryAfter() time.Duration {
	return e.RetryDelay
}

// NewTransientError creates a transient error that can be retried.
func NewTransientError(msg string, statusCode int, cause error) *Error {
	return &Error{
		Msg:   msg,
		Cat:   ErrorTransient,
		Code:  statusCode,
		Cause: cause,
	}
}

// NewTransientErrorWithRetry creates a transient error with a suggested retry delay.
func NewTransientErrorWithRetry(msg string, statusCode int, retryAfter time.Duration, cause error) *Error {
	return &Error{
		Msg:        msg,
		Cat:        ErrorTransient,
		Code:       statusCode,
		RetryDelay: retryAfter,
		Cause:      cause,
	}
}

// NewPermanentError creates a permanent error that should not be retried.
func NewPermanentError(msg string, statusCode int, cause error) *Error {
	return &Error{
		Msg:   msg,
		Cat:   ErrorPermanent,
		Code:  statusCode,
		Cause: cause,
	}
}

// NewUserInputError creates an error indicating invalid user input.
func NewUserInputError(msg string, statusCode int, cause error) *Error {
	return &Error{
		Msg:   msg,
		Cat:   ErrorUserInput,
		Code:  statusCode,
		Cause: cause,
	}
}

// IsTransient returns true if the error is categorized as transient.
// It checks if the error or any wrapped error implements CategorizedError.
func IsTransient(err error) bool {
	var ce CategorizedError
	if errors.As(err, &ce) {
		return ce.Category() == ErrorTransient
	}
	return false
}

// IsPermanent returns true if the error is categorized as permanent.
// It checks if the error or any wrapped error implements CategorizedError.
func IsPermanent(err error) bool {
	var ce CategorizedError
	if errors.As(err, &ce) {
		return ce.Category() == ErrorPermanent
	}
	return false
}

// IsUserInput returns true if the error is categorized as user input error.
// It checks if the error or any wrapped error implements CategorizedError.
func IsUserInput(err error) bool {
	var ce CategorizedError
	if errors.As(err, &ce) {
		return ce.Category() == ErrorUserInput
	}
	return false
}

// StatusCodeOf returns the HTTP status code from a categorized error, or 0.
func StatusCodeOf(err error) int {
	var ce CategorizedError
	if errors.As(err, &ce) {
		return ce.StatusCode()
	}
	return 0
}

// RetryAfterOf returns the retry delay from a categorized error, or 0.
func RetryAfterOf(err error) time.Duration {
	var ce CategorizedError
	if errors.As(err, &ce) {
		return ce.RetryAfter()
	}
	return 0
}

// ImageError represents an error during image processing.
type ImageError struct {
	Op  string // "decode" or "fetch"
	URL string // the image URL or "base64"
	Err error  // underlying error
}

// Error returns a formatted error message describing the image processing failure.
func (e *ImageError) Error() string {
	return fmt.Sprintf("image %s error for %s: %v", e.Op, e.URL, e.Err)
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *ImageError) Unwrap() error {
	return e.Err
}
