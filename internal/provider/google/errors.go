package google

import (
	"errors"

	ai "github.com/spetersoncode/gains"
	"google.golang.org/genai"
)

// wrapError wraps a Google GenAI error with gains error categorization.
// It extracts status codes for proper retry handling.
// Note: Google's genai.APIError doesn't expose headers, so Retry-After is not available.
func wrapError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr genai.APIError
	if !errors.As(err, &apiErr) {
		// Not an API error, return as-is (likely network error, handled by heuristics)
		return err
	}

	code := apiErr.Code
	category := categorizeStatusCode(code)
	msg := err.Error()

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
