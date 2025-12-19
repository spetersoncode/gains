package workflow

import (
	"reflect"
	"strings"

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

// NewStateFromStruct creates a new State initialized from a struct's fields.
// Field names are derived from json tags, falling back to the field name in snake_case.
// Zero values are skipped unless the field has `state:"include_zero"` tag.
//
// Example:
//
//	type MyState struct {
//	    UserID   string `json:"user_id"`
//	    Count    int    `json:"count"`
//	    Optional string `json:"optional,omitempty"`
//	}
//
//	state := workflow.NewStateFromStruct(&MyState{
//	    UserID: "abc123",
//	    Count:  42,
//	})
//	// State now contains: {"user_id": "abc123", "count": 42}
//
// The struct pointer can be nil, in which case an empty state is returned.
func NewStateFromStruct(structPtr any) *State {
	data := structToMap(structPtr)
	return store.NewFrom(data)
}

// StateInto copies state values into a struct's fields.
// Field names are matched using json tags, falling back to field name in snake_case.
// Only exported fields with matching state keys are populated.
//
// Example:
//
//	var result MyState
//	workflow.StateInto(state, &result)
//	fmt.Println(result.UserID) // "abc123"
func StateInto(state *State, structPtr any) {
	if structPtr == nil || state == nil {
		return
	}

	v := reflect.ValueOf(structPtr)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		key := fieldToKey(field)
		if key == "" || key == "-" {
			continue
		}

		val, ok := state.Get(key)
		if !ok {
			continue
		}

		fieldVal := v.Field(i)
		if !fieldVal.CanSet() {
			continue
		}

		// Try to set the value if types are compatible
		valReflect := reflect.ValueOf(val)
		if valReflect.Type().AssignableTo(fieldVal.Type()) {
			fieldVal.Set(valReflect)
		} else if valReflect.Type().ConvertibleTo(fieldVal.Type()) {
			fieldVal.Set(valReflect.Convert(fieldVal.Type()))
		}
	}
}

// structToMap converts a struct to a map using json tags for keys.
func structToMap(structPtr any) map[string]any {
	if structPtr == nil {
		return nil
	}

	v := reflect.ValueOf(structPtr)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	result := make(map[string]any)
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		key := fieldToKey(field)
		if key == "" || key == "-" {
			continue
		}

		fieldVal := v.Field(i)
		val := fieldVal.Interface()

		// Skip zero values unless explicitly included
		if isZero(fieldVal) && !hasIncludeZero(field) {
			continue
		}

		result[key] = val
	}

	return result
}

// fieldToKey extracts the state key from a struct field.
// Uses json tag if present, otherwise converts field name to snake_case.
func fieldToKey(field reflect.StructField) string {
	// Check json tag first
	if tag := field.Tag.Get("json"); tag != "" {
		parts := strings.Split(tag, ",")
		if parts[0] != "" {
			return parts[0]
		}
	}

	// Fall back to snake_case of field name
	return toSnakeCase(field.Name)
}

// hasIncludeZero checks if a field has the state:"include_zero" tag.
func hasIncludeZero(field reflect.StructField) bool {
	tag := field.Tag.Get("state")
	return strings.Contains(tag, "include_zero")
}

// isZero checks if a reflect.Value is the zero value for its type.
func isZero(v reflect.Value) bool {
	return v.IsZero()
}

// toSnakeCase converts CamelCase to snake_case.
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			result.WriteByte(byte(r) + 32) // lowercase
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// NewMemoryAdapter creates a new in-memory adapter for State.
func NewMemoryAdapter() *MemoryAdapter {
	return store.NewMemoryAdapter()
}
