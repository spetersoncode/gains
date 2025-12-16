package workflow

import "fmt"

// GetTyped retrieves a value from state and asserts it to type T.
// Returns the typed value and true if the key exists and the type matches.
// Returns the zero value and false if the key is missing or the value
// cannot be asserted to type T.
//
// Example:
//
//	analysis, ok := workflow.GetTyped[*AnalysisResult](state, "analysis")
//	if !ok {
//	    // Handle missing or wrong type
//	}
func GetTyped[T any](s *State, key string) (T, bool) {
	var zero T
	v, ok := s.Get(key)
	if !ok {
		return zero, false
	}
	typed, ok := v.(T)
	if !ok {
		return zero, false
	}
	return typed, true
}

// MustGet retrieves a value from state and asserts it to type T.
// Panics if the key is missing or the value cannot be asserted to type T.
// Use this when you are certain the key exists with the correct type,
// and a mismatch indicates a programming error.
//
// Example:
//
//	analysis := workflow.MustGet[*AnalysisResult](state, "analysis")
func MustGet[T any](s *State, key string) T {
	v, ok := s.Get(key)
	if !ok {
		panic(fmt.Sprintf("workflow: state key %q not found", key))
	}
	typed, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("workflow: state key %q has type %T, want %T", key, v, *new(T)))
	}
	return typed
}

// GetTypedOr retrieves a value from state and asserts it to type T.
// Returns defaultVal if the key is missing or the value cannot be
// asserted to type T.
//
// Example:
//
//	count := workflow.GetTypedOr(state, "retry_count", 0)
func GetTypedOr[T any](s *State, key string, defaultVal T) T {
	v, ok := GetTyped[T](s, key)
	if !ok {
		return defaultVal
	}
	return v
}
