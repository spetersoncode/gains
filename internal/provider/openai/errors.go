package openai

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/openai/openai-go"
	ai "github.com/spetersoncode/gains"
)

// wrapError wraps an OpenAI SDK error with gains error categorization.
// It extracts status codes and Retry-After headers for proper retry handling.
func wrapError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr *openai.Error
	if !errors.As(err, &apiErr) {
		// Not an API error, return as-is (likely network error, handled by heuristics)
		return err
	}

	code := apiErr.StatusCode
	category := categorizeStatusCode(code)
	retryAfter := parseRetryAfter(apiErr.Response)

	msg := err.Error()
	if retryAfter > 0 {
		return ai.NewTransientErrorWithRetry(msg, code, retryAfter, err)
	}

	switch category {
	case ai.ErrorTransient:
		return ai.NewTransientError(msg, code, err)
	case ai.ErrorPermanent:
		return ai.NewPermanentError(msg, code, err)
	case ai.ErrorUserInput:
		return ai.NewUserInputError(msg, code, err)
	default:
		return err
	}
}

// categorizeStatusCode determines the error category from an HTTP status code.
func categorizeStatusCode(code int) ai.ErrorCategory {
	switch {
	case code == 429:
		return ai.ErrorTransient // Rate limited
	case code >= 500 && code < 600:
		return ai.ErrorTransient // Server error
	case code == 401 || code == 403:
		return ai.ErrorPermanent // Authentication/authorization
	case code == 400 || code == 404 || code == 422:
		return ai.ErrorUserInput // Bad request or not found
	default:
		return ai.ErrorPermanent // Default to permanent for unknown codes
	}
}

// parseRetryAfter extracts the Retry-After duration from an HTTP response.
// Returns 0 if the header is not present or cannot be parsed.
func parseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}

	header := resp.Header.Get("Retry-After")
	if header == "" {
		return 0
	}

	// Try parsing as seconds (most common)
	if seconds, err := strconv.Atoi(header); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP-date (RFC 7231)
	if t, err := http.ParseTime(header); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			return delay
		}
	}

	return 0
}
