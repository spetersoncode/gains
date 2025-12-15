package store

import (
	"context"
	"encoding/json"
	"sync"
)

// MemoryAdapter provides thread-safe in-memory storage.
type MemoryAdapter struct {
	mu   sync.RWMutex
	data map[string]json.RawMessage
}

// NewMemoryAdapter creates a new in-memory adapter.
func NewMemoryAdapter() *MemoryAdapter {
	return &MemoryAdapter{
		data: make(map[string]json.RawMessage),
	}
}

// Get retrieves a value by key.
func (m *MemoryAdapter) Get(_ context.Context, key string) (json.RawMessage, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	return v, ok, nil
}

// Set stores a value by key.
func (m *MemoryAdapter) Set(_ context.Context, key string, value json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

// Delete removes a key.
func (m *MemoryAdapter) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

// Has returns true if the key exists.
func (m *MemoryAdapter) Has(_ context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[key]
	return ok, nil
}

// Keys returns all keys.
func (m *MemoryAdapter) Keys(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

// Len returns the number of stored keys.
func (m *MemoryAdapter) Len(_ context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data), nil
}

// Clear removes all data.
func (m *MemoryAdapter) Clear(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]json.RawMessage)
	return nil
}

// Load retrieves all data as a map.
func (m *MemoryAdapter) Load(_ context.Context) (map[string]json.RawMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]json.RawMessage, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result, nil
}

// Save stores all data from a map, replacing existing data.
func (m *MemoryAdapter) Save(_ context.Context, data map[string]json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]json.RawMessage, len(data))
	for k, v := range data {
		m.data[k] = v
	}
	return nil
}
