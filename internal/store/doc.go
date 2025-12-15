// Package store provides pluggable state management for gains workflows and agents.
//
// The package offers three main types:
//   - [Store]: A generic key-value store with map[string]any semantics
//   - [TypedStore]: A type-safe wrapper for managing specific state structs
//   - [MessageStore]: A specialized store for conversation history
//
// All types support pluggable persistence through the [Adapter] interface,
// with a default in-memory implementation provided via [MemoryAdapter].
//
// # Basic Usage
//
// Create a store with the default in-memory adapter:
//
//	s := store.New(nil)
//	s.Set("name", "Alice")
//	s.Set("count", 42)
//
//	name := s.GetString("name")  // "Alice"
//	count := s.GetInt("count")   // 42
//
// # Type-Safe Store
//
// Use TypedStore for structured state:
//
//	type WorkflowState struct {
//	    Input    string
//	    Results  []string
//	    Complete bool
//	}
//
//	s := store.NewTyped(WorkflowState{Input: "hello"}, nil)
//
//	// Read state
//	state := s.Get()
//
//	// Update state
//	s.Update(func(state *WorkflowState) {
//	    state.Results = append(state.Results, "result1")
//	})
//
// # Message Store
//
// Use MessageStore for conversation history:
//
//	history := store.NewMessageStore(nil)
//	history.Append(gains.Message{Role: gains.RoleUser, Content: "Hello"})
//
//	msgs := history.Messages() // Get all messages
//
// # Persistence
//
// Persist state by calling Sync, reload with Reload:
//
//	s := store.New(myAdapter)
//	s.Set("key", "value")
//
//	// Persist to adapter
//	if err := s.Sync(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Later, reload from adapter
//	if err := s.Reload(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
// # Custom Adapters
//
// Implement the Adapter interface for custom persistence:
//
//	type RedisAdapter struct { ... }
//
//	func (r *RedisAdapter) Get(ctx context.Context, key string) (json.RawMessage, bool, error) { ... }
//	func (r *RedisAdapter) Set(ctx context.Context, key string, value json.RawMessage) error { ... }
//	// ... implement remaining methods
//
//	s := store.New(&RedisAdapter{})
package store
