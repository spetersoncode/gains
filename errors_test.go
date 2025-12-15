package gains

import (
	"errors"
	"testing"

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
