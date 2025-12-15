package store

import (
	"context"
	"encoding/json"
	"sync"
)

// TypedStore provides type-safe state management for a specific struct type.
type TypedStore[T any] struct {
	mu      sync.RWMutex
	data    T
	adapter Adapter
}

// NewTyped creates a new TypedStore with the given initial value and adapter.
// If adapter is nil, a default in-memory adapter is used.
func NewTyped[T any](initial T, adapter Adapter) *TypedStore[T] {
	if adapter == nil {
		adapter = NewMemoryAdapter()
	}
	return &TypedStore[T]{
		data:    initial,
		adapter: adapter,
	}
}

// Get returns the current state value.
func (s *TypedStore[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

// Set replaces the entire state value.
func (s *TypedStore[T]) Set(value T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = value
}

// Update applies a function to modify the state.
// The function receives a pointer to the state for in-place modification.
func (s *TypedStore[T]) Update(fn func(*T)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.data)
}

// Clone creates a deep copy of the TypedStore via JSON serialization.
// Note: This only works correctly if T is JSON-serializable.
func (s *TypedStore[T]) Clone() (*TypedStore[T], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	raw, err := json.Marshal(s.data)
	if err != nil {
		return nil, err
	}

	var cloned T
	if err := json.Unmarshal(raw, &cloned); err != nil {
		return nil, err
	}

	return NewTyped(cloned, nil), nil
}

// Sync persists the state to the adapter under the given key.
func (s *TypedStore[T]) Sync(ctx context.Context, key string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	raw, err := json.Marshal(s.data)
	if err != nil {
		return &SerializationError{Key: key, Err: err}
	}
	return s.adapter.Set(ctx, key, raw)
}

// Reload loads state from the adapter using the given key.
func (s *TypedStore[T]) Reload(ctx context.Context, key string) error {
	raw, ok, err := s.adapter.Get(ctx, key)
	if err != nil {
		return err
	}
	if !ok {
		return ErrKeyNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Unmarshal(raw, &s.data)
}

// Adapter returns the underlying adapter.
func (s *TypedStore[T]) Adapter() Adapter {
	return s.adapter
}
