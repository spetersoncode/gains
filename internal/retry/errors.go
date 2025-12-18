package retry

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"syscall"

	"github.com/spetersoncode/gains"
)

// statusCoder is an interface for errors that have an HTTP status code.
// Both Anthropic and OpenAI SDK errors implement this interface.
type statusCoder interface {
	StatusCode() int
}

// IsTransient determines if an error is transient and should be retried.
// It first checks if the error implements gains.CategorizedError for explicit
// categorization. If not, it falls back to heuristic detection:
// - Rate limits (HTTP 429)
// - Server errors (HTTP 5xx)
// - Network timeouts
// - Connection resets
// - DNS failures
func IsTransient(err error) bool {
	if err == nil {
		return false
	}

	// First, check if error implements CategorizedError for explicit categorization
	var ce gains.CategorizedError
	if errors.As(err, &ce) {
		return ce.Category() == gains.ErrorTransient
	}

	// Fall back to heuristic detection for uncategorized errors

	// Check for API errors with status codes (works with Anthropic/OpenAI SDKs)
	var sc statusCoder
	if errors.As(err, &sc) {
		if isTransientStatusCode(sc.StatusCode()) {
			return true
		}
	}

	// Check for Google API errors (googleapi.Error has Code field, not StatusCode method)
	if code := extractGoogleAPIErrorCode(err); code > 0 {
		if isTransientStatusCode(code) {
			return true
		}
	}

	// Check network-level errors
	if isTransientNetworkError(err) {
		return true
	}

	return false
}

// isTransientStatusCode checks if an HTTP status code indicates a transient error.
func isTransientStatusCode(code int) bool {
	// 429 = Rate Limited
	if code == 429 {
		return true
	}
	// 5xx = Server Errors
	if code >= 500 && code < 600 {
		return true
	}
	return false
}

// extractGoogleAPIErrorCode extracts the status code from a Google API error.
// Google's googleapi.Error has a Code field instead of StatusCode() method.
func extractGoogleAPIErrorCode(err error) int {
	// Use reflection-free approach: check if error has a Code field via interface
	type coder interface {
		Error() string
	}

	// Check error message for common Google API error patterns
	errStr := err.Error()
	if strings.Contains(errStr, "googleapi:") {
		// Try to extract status code from error message pattern "googleapi: Error 429:"
		if strings.Contains(errStr, "Error 429") {
			return 429
		}
		if strings.Contains(errStr, "Error 500") {
			return 500
		}
		if strings.Contains(errStr, "Error 502") {
			return 502
		}
		if strings.Contains(errStr, "Error 503") {
			return 503
		}
		if strings.Contains(errStr, "Error 504") {
			return 504
		}
	}
	return 0
}

// isTransientNetworkError checks for network-level transient errors.
func isTransientNetworkError(err error) bool {
	// Check for timeout errors
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Check for URL errors (wrapping network errors)
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		// Check wrapped error
		if urlErr.Err != nil && isTransientNetworkError(urlErr.Err) {
			return true
		}
	}

	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		// Temporary DNS failures are retryable
		return dnsErr.Temporary()
	}

	// Check for syscall errors (connection reset, etc.)
	var syscallErr syscall.Errno
	if errors.As(err, &syscallErr) {
		switch syscallErr {
		case syscall.ECONNRESET,   // Connection reset by peer
			syscall.ECONNREFUSED, // Connection refused
			syscall.ETIMEDOUT:    // Connection timed out
			return true
		}
		// Platform-specific: ENETUNREACH and EHOSTUNREACH may not exist on Windows
	}

	// Check for common error message patterns (fallback)
	errMsg := strings.ToLower(err.Error())
	transientPatterns := []string{
		"connection reset",
		"connection refused",
		"timeout",
		"temporary failure",
		"service unavailable",
		"too many requests",
		"rate limit",
		"server error",
		"bad gateway",
		"gateway timeout",
	}
	for _, pattern := range transientPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}
