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

func (e *ImageError) Error() string {
	return fmt.Sprintf("image %s error for %s: %v", e.Op, e.URL, e.Err)
}

func (e *ImageError) Unwrap() error {
	return e.Err
}
