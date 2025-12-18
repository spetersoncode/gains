package workflow

import (
	"github.com/spetersoncode/gains/internal/store"
)

// State provides thread-safe key-value state management for workflows.
// This is the primary state container passed through workflow steps.
//
// For type-safe state access, use the typed Key[T] API:
//
//	var KeyCount = workflow.NewKey[int]("count")
//	var KeyName = workflow.NewKey[string]("name")
//
//	workflow.Set(state, KeyCount, 42)
//	workflow.Set(state, KeyName, "example")
//
//	count, ok := workflow.Get(state, KeyCount)  // 42, true
//	name := workflow.MustGet(state, KeyName)    // "example"
//
// The typed API provides compile-time type checking. See [Key], [Get], [Set],
// [MustGet], and [GetOr] for the full typed API.
type State = store.Store

// StateAdapter defines the interface for persistence backends.
// Implementations must be thread-safe.
type StateAdapter = store.Adapter

// MemoryAdapter is an in-memory implementation of StateAdapter.
type MemoryAdapter = store.MemoryAdapter

// NewState creates a new State with the given adapter.
// If adapter is nil, a default in-memory adapter is used.
func NewState(adapter StateAdapter) *State {
	return store.New(adapter)
}

// NewStateFrom creates a new State initialized with the given data.
func NewStateFrom(data map[string]any) *State {
	return store.NewFrom(data)
}

// NewMemoryAdapter creates a new in-memory adapter for State.
func NewMemoryAdapter() *MemoryAdapter {
	return store.NewMemoryAdapter()
}
