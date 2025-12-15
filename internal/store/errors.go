package store

import (
	"errors"
	"fmt"
)

var (
	// ErrKeyNotFound indicates the requested key does not exist.
	ErrKeyNotFound = errors.New("store: key not found")

	// ErrAdapterClosed indicates the adapter has been closed.
	ErrAdapterClosed = errors.New("store: adapter closed")
)

// SerializationError wraps JSON marshaling/unmarshaling errors with context.
type SerializationError struct {
	Key string
	Err error
}

func (e *SerializationError) Error() string {
	return fmt.Sprintf("store: serialization error for key %q: %v", e.Key, e.Err)
}

func (e *SerializationError) Unwrap() error {
	return e.Err
}
