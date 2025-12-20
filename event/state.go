package event

import (
	"context"
	"encoding/json"
	"sync"
)

// sharedStateKey is the context key for shared state.
type sharedStateKey struct{}

// SharedState holds the current shared state and provides thread-safe access.
// It tracks the current state and can emit updates via the event channel.
type SharedState struct {
	mu    sync.RWMutex
	state map[string]any
}

// NewSharedState creates a new SharedState from the initial state.
// If initial is nil, creates an empty state.
func NewSharedState(initial any) *SharedState {
	ss := &SharedState{
		state: make(map[string]any),
	}
	if initial != nil {
		// Convert to map[string]any via JSON round-trip
		if m, ok := initial.(map[string]any); ok {
			ss.state = m
		} else {
			data, err := json.Marshal(initial)
			if err == nil {
				json.Unmarshal(data, &ss.state)
			}
		}
	}
	return ss
}

// Get returns the current state as a map.
func (s *SharedState) Get() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Return a copy to prevent external mutation
	result := make(map[string]any, len(s.state))
	for k, v := range s.state {
		result[k] = v
	}
	return result
}

// GetField returns a specific field from the state.
func (s *SharedState) GetField(path string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Simple single-level lookup for now
	// TODO: support nested paths like "/foo/bar"
	key := path
	if len(path) > 0 && path[0] == '/' {
		key = path[1:]
	}
	return s.state[key]
}

// Set replaces the entire state and emits a STATE_SNAPSHOT event.
func (s *SharedState) Set(ctx context.Context, newState map[string]any) {
	s.mu.Lock()
	s.state = newState
	s.mu.Unlock()

	// Emit snapshot
	if ch := ForwardChannelFromContext(ctx); ch != nil {
		EmitSnapshot(ch, newState)
	}
}

// Update updates specific fields and emits a STATE_DELTA event.
func (s *SharedState) Update(ctx context.Context, patches ...JSONPatch) {
	s.mu.Lock()
	// Apply patches to local state
	for _, p := range patches {
		key := p.Path
		if len(key) > 0 && key[0] == '/' {
			key = key[1:]
		}
		switch p.Op {
		case PatchAdd, PatchReplace:
			s.state[key] = p.Value
		case PatchRemove:
			delete(s.state, key)
		}
	}
	s.mu.Unlock()

	// Emit delta
	if ch := ForwardChannelFromContext(ctx); ch != nil {
		EmitDelta(ch, patches...)
	}
}

// UpdateField is a convenience method to update a single field.
func (s *SharedState) UpdateField(ctx context.Context, path string, value any) {
	s.Update(ctx, Replace(path, value))
}

// WithSharedState returns a new context with the shared state attached.
func WithSharedState(ctx context.Context, state *SharedState) context.Context {
	return context.WithValue(ctx, sharedStateKey{}, state)
}

// SharedStateFromContext retrieves the shared state from the context.
// Returns nil if no shared state is attached.
func SharedStateFromContext(ctx context.Context) *SharedState {
	if v := ctx.Value(sharedStateKey{}); v != nil {
		return v.(*SharedState)
	}
	return nil
}
