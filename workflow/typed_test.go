package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	Name  string
	Value int
}

func TestGetTyped(t *testing.T) {
	state := NewStateFrom(map[string]any{
		"string":  "hello",
		"int":     42,
		"struct":  &testStruct{Name: "test", Value: 100},
		"nil_ptr": (*testStruct)(nil),
	})

	t.Run("returns value when type matches", func(t *testing.T) {
		s, ok := GetTyped[string](state, "string")
		require.True(t, ok)
		assert.Equal(t, "hello", s)
	})

	t.Run("returns int when type matches", func(t *testing.T) {
		i, ok := GetTyped[int](state, "int")
		require.True(t, ok)
		assert.Equal(t, 42, i)
	})

	t.Run("returns false when key missing", func(t *testing.T) {
		_, ok := GetTyped[string](state, "nonexistent")
		assert.False(t, ok)
	})

	t.Run("returns false when type mismatches", func(t *testing.T) {
		_, ok := GetTyped[int](state, "string")
		assert.False(t, ok)
	})

	t.Run("works with pointer types", func(t *testing.T) {
		s, ok := GetTyped[*testStruct](state, "struct")
		require.True(t, ok)
		assert.Equal(t, "test", s.Name)
		assert.Equal(t, 100, s.Value)
	})

	t.Run("works with nil pointers", func(t *testing.T) {
		s, ok := GetTyped[*testStruct](state, "nil_ptr")
		require.True(t, ok)
		assert.Nil(t, s)
	})
}

func TestMustGet(t *testing.T) {
	state := NewStateFrom(map[string]any{
		"string": "hello",
		"int":    42,
	})

	t.Run("returns value when type matches", func(t *testing.T) {
		s := MustGet[string](state, "string")
		assert.Equal(t, "hello", s)
	})

	t.Run("panics when key missing", func(t *testing.T) {
		assert.PanicsWithValue(t,
			`workflow: state key "nonexistent" not found`,
			func() {
				MustGet[string](state, "nonexistent")
			})
	})

	t.Run("panics when type mismatches", func(t *testing.T) {
		assert.Panics(t, func() {
			MustGet[int](state, "string")
		})
	})
}

func TestGetTypedOr(t *testing.T) {
	state := NewStateFrom(map[string]any{
		"string": "hello",
		"int":    42,
	})

	t.Run("returns value when present", func(t *testing.T) {
		s := GetTypedOr(state, "string", "default")
		assert.Equal(t, "hello", s)
	})

	t.Run("returns default when missing", func(t *testing.T) {
		s := GetTypedOr(state, "missing", "default")
		assert.Equal(t, "default", s)
	})

	t.Run("returns default when type mismatches", func(t *testing.T) {
		i := GetTypedOr(state, "string", 999)
		assert.Equal(t, 999, i)
	})
}
