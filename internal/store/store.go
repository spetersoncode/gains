package store

import (
	"context"
	"encoding/json"
	"sync"
)

// Store provides thread-safe key-value state management with pluggable persistence.
type Store struct {
	mu      sync.RWMutex
	adapter Adapter
	cache   map[string]any
}

// New creates a new Store with the given adapter.
// If adapter is nil, a default in-memory adapter is used.
func New(adapter Adapter) *Store {
	if adapter == nil {
		adapter = NewMemoryAdapter()
	}
	return &Store{
		adapter: adapter,
		cache:   make(map[string]any),
	}
}

// NewFrom creates a new Store initialized with the given data.
func NewFrom(data map[string]any) *Store {
	s := New(nil)
	for k, v := range data {
		s.cache[k] = v
	}
	return s
}

// Get retrieves a value from the store.
func (s *Store) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.cache[key]
	return v, ok
}

// GetString retrieves a string value. Returns empty string if not found or wrong type.
func (s *Store) GetString(key string) string {
	v, ok := s.Get(key)
	if !ok {
		return ""
	}
	if str, ok := v.(string); ok {
		return str
	}
	return ""
}

// GetInt retrieves an int value. Returns 0 if not found or wrong type.
// Handles float64 from JSON unmarshaling.
func (s *Store) GetInt(key string) int {
	v, ok := s.Get(key)
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case int32:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	}
	return 0
}

// GetBool retrieves a bool value. Returns false if not found or wrong type.
func (s *Store) GetBool(key string) bool {
	v, ok := s.Get(key)
	if !ok {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// GetFloat retrieves a float64 value. Returns 0.0 if not found or wrong type.
func (s *Store) GetFloat(key string) float64 {
	v, ok := s.Get(key)
	if !ok {
		return 0.0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0.0
}

// Set stores a value in the store.
func (s *Store) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[key] = value
}

// Delete removes a key from the store.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, key)
}

// Has returns true if the key exists.
func (s *Store) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.cache[key]
	return ok
}

// Keys returns all keys in the store.
func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.cache))
	for k := range s.cache {
		keys = append(keys, k)
	}
	return keys
}

// Len returns the number of keys in the store.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.cache)
}

// Clone creates a shallow copy of the store with a new in-memory adapter.
func (s *Store) Clone() *Store {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clone := New(nil)
	for k, v := range s.cache {
		clone.cache[k] = v
	}
	return clone
}

// Merge copies values from another store, overwriting existing keys.
func (s *Store) Merge(other *Store) {
	if other == nil {
		return
	}
	other.mu.RLock()
	defer other.mu.RUnlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range other.cache {
		s.cache[k] = v
	}
}

// Sync persists the current cache to the adapter.
func (s *Store) Sync(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := make(map[string]json.RawMessage, len(s.cache))
	for k, v := range s.cache {
		raw, err := json.Marshal(v)
		if err != nil {
			return &SerializationError{Key: k, Err: err}
		}
		data[k] = raw
	}
	return s.adapter.Save(ctx, data)
}

// Reload loads data from the adapter into the cache.
func (s *Store) Reload(ctx context.Context) error {
	data, err := s.adapter.Load(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache = make(map[string]any, len(data))
	for k, raw := range data {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return &SerializationError{Key: k, Err: err}
		}
		s.cache[k] = v
	}
	return nil
}

// Data returns a shallow copy of the internal cache map.
func (s *Store) Data() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data := make(map[string]any, len(s.cache))
	for k, v := range s.cache {
		data[k] = v
	}
	return data
}

// Adapter returns the underlying adapter.
func (s *Store) Adapter() Adapter {
	return s.adapter
}
