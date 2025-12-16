package workflow

import "fmt"

// Key represents a typed state key that associates a name with type T.
// Keys provide compile-time type safety for workflow state access.
//
// Define keys as package-level variables for reuse:
//
//	var (
//	    KeyAnalysis = workflow.NewKey[*AnalysisResult]("analysis")
//	    KeyProjects = workflow.NewKey[[]*ProjectConfig]("projects")
//	    KeyApproved = workflow.NewKey[bool]("approved")
//	)
type Key[T any] struct {
	name string
}

// NewKey creates a typed key with the given name.
// The type parameter T specifies the type of values stored under this key.
func NewKey[T any](name string) Key[T] {
	return Key[T]{name: name}
}

// Name returns the string name of the key.
// This is the underlying key used in state storage.
func (k Key[T]) Name() string {
	return k.name
}

// String implements fmt.Stringer for debugging.
func (k Key[T]) String() string {
	return k.name
}

// Get retrieves a value from state using a typed key.
// Returns the typed value and true if the key exists and the type matches.
// Returns the zero value and false if the key is missing or type mismatches.
//
// Example:
//
//	analysis, ok := workflow.Get(state, KeyAnalysis)
//	if ok {
//	    fmt.Println(analysis.Sentiment)
//	}
func Get[T any](s *State, key Key[T]) (T, bool) {
	var zero T
	v, ok := s.Get(key.name)
	if !ok {
		return zero, false
	}
	typed, ok := v.(T)
	if !ok {
		return zero, false
	}
	return typed, true
}

// Set stores a value in state using a typed key.
// The compiler enforces that value has the correct type for the key.
//
// Example:
//
//	workflow.Set(state, KeyAnalysis, &AnalysisResult{...})
func Set[T any](s *State, key Key[T], value T) {
	s.Set(key.name, value)
}

// MustGet retrieves a value from state using a typed key.
// Panics if the key is missing or the value cannot be asserted to type T.
// Use when you are certain the key exists with the correct type.
//
// Example:
//
//	analysis := workflow.MustGet(state, KeyAnalysis)
//	fmt.Println(analysis.Sentiment)
func MustGet[T any](s *State, key Key[T]) T {
	v, ok := s.Get(key.name)
	if !ok {
		panic(fmt.Sprintf("workflow: state key %q not found", key.name))
	}
	typed, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("workflow: state key %q has type %T, want %T", key.name, v, *new(T)))
	}
	return typed
}

// GetOr retrieves a value from state using a typed key.
// Returns defaultVal if the key is missing or the type mismatches.
//
// Example:
//
//	retries := workflow.GetOr(state, KeyRetryCount, 0)
func GetOr[T any](s *State, key Key[T], defaultVal T) T {
	v, ok := Get(s, key)
	if !ok {
		return defaultVal
	}
	return v
}

// Has returns true if the typed key exists in state.
func Has[T any](s *State, key Key[T]) bool {
	return s.Has(key.name)
}

// Delete removes a typed key from state.
func Delete[T any](s *State, key Key[T]) {
	s.Delete(key.name)
}

// SetIfAbsent stores a value only if the key does not exist.
// Returns true if the value was set, false if the key already existed.
func SetIfAbsent[T any](s *State, key Key[T], value T) bool {
	if s.Has(key.name) {
		return false
	}
	s.Set(key.name, value)
	return true
}

// Update applies a function to the current value and stores the result.
// If the key doesn't exist, applies the function to the zero value.
// Returns the new value.
func Update[T any](s *State, key Key[T], fn func(T) T) T {
	current, _ := Get(s, key)
	newVal := fn(current)
	Set(s, key, newVal)
	return newVal
}

// IntKey creates a Key[int] with the given name.
func IntKey(name string) Key[int] { return NewKey[int](name) }

// StringKey creates a Key[string] with the given name.
func StringKey(name string) Key[string] { return NewKey[string](name) }

// BoolKey creates a Key[bool] with the given name.
func BoolKey(name string) Key[bool] { return NewKey[bool](name) }

// FloatKey creates a Key[float64] with the given name.
func FloatKey(name string) Key[float64] { return NewKey[float64](name) }
