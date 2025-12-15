package gains

import (
	"errors"
	"fmt"
)

// ErrEmptyInput is returned when a required input slice is empty.
var ErrEmptyInput = errors.New("empty input")

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
