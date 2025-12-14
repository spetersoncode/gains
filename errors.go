package gains

import "fmt"

// APIError represents an error returned by a provider's API.
type APIError struct {
	StatusCode int
	Message    string
	Provider   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s API error (status %d): %s", e.Provider, e.StatusCode, e.Message)
}

// AuthError represents an authentication failure.
type AuthError struct {
	Provider string
	Message  string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("%s authentication error: %s", e.Provider, e.Message)
}

// RateLimitError represents a rate limiting error.
type RateLimitError struct {
	Provider   string
	Message    string
	RetryAfter int // seconds until retry is allowed, if provided
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("%s rate limit exceeded: %s (retry after %ds)", e.Provider, e.Message, e.RetryAfter)
	}
	return fmt.Sprintf("%s rate limit exceeded: %s", e.Provider, e.Message)
}
