package store

import (
	"context"
	"encoding/json"
)

// Adapter defines the interface for persistence backends.
// Implementations must be thread-safe.
type Adapter interface {
	// Get retrieves a value by key. Returns nil, false, nil if not found.
	Get(ctx context.Context, key string) (json.RawMessage, bool, error)

	// Set stores a value by key.
	Set(ctx context.Context, key string, value json.RawMessage) error

	// Delete removes a key. No error if key doesn't exist.
	Delete(ctx context.Context, key string) error

	// Has returns true if the key exists.
	Has(ctx context.Context, key string) (bool, error)

	// Keys returns all keys.
	Keys(ctx context.Context) ([]string, error)

	// Len returns the number of stored keys.
	Len(ctx context.Context) (int, error)

	// Clear removes all data.
	Clear(ctx context.Context) error

	// Load retrieves all data as a map.
	Load(ctx context.Context) (map[string]json.RawMessage, error)

	// Save stores all data from a map, replacing existing data.
	Save(ctx context.Context, data map[string]json.RawMessage) error
}
