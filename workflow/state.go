package workflow

import "sync"

// State holds workflow execution state passed between steps.
// It is a thread-safe map for in-memory state management.
type State struct {
	mu   sync.RWMutex
	data map[string]any
}

// NewState creates a new empty state.
func NewState() *State {
	return &State{data: make(map[string]any)}
}

// NewStateFrom creates state from an existing map.
func NewStateFrom(data map[string]any) *State {
	s := NewState()
	for k, v := range data {
		s.data[k] = v
	}
	return s
}

// Get retrieves a value from state.
func (s *State) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

// GetString retrieves a string value. Returns empty string if not found or not a string.
func (s *State) GetString(key string) string {
	v, ok := s.Get(key)
	if !ok {
		return ""
	}
	if str, ok := v.(string); ok {
		return str
	}
	return ""
}

// GetInt retrieves an int value. Returns 0 if not found or not an int.
func (s *State) GetInt(key string) int {
	v, ok := s.Get(key)
	if !ok {
		return 0
	}
	if i, ok := v.(int); ok {
		return i
	}
	return 0
}

// GetBool retrieves a bool value. Returns false if not found or not a bool.
func (s *State) GetBool(key string) bool {
	v, ok := s.Get(key)
	if !ok {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// Set stores a value in state.
func (s *State) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// Delete removes a key from state.
func (s *State) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// Has returns true if the key exists in state.
func (s *State) Has(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[key]
	return ok
}

// Keys returns all state keys.
func (s *State) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

// Clone creates a shallow copy of the state for isolation.
func (s *State) Clone() *State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clone := NewState()
	for k, v := range s.data {
		clone.data[k] = v
	}
	return clone
}

// Merge copies values from another state, overwriting existing keys.
func (s *State) Merge(other *State) {
	if other == nil {
		return
	}
	other.mu.RLock()
	defer other.mu.RUnlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range other.data {
		s.data[k] = v
	}
}

// Len returns the number of keys in state.
func (s *State) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}
