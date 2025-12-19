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

// Test keys for the test suite
var (
	keyString  = NewKey[string]("string")
	keyInt     = NewKey[int]("int")
	keyStruct  = NewKey[*testStruct]("struct")
	keyNilPtr  = NewKey[*testStruct]("nil_ptr")
	keyMissing = NewKey[string]("nonexistent")
	keyCounter = NewKey[int]("counter")
)

func TestKey(t *testing.T) {
	t.Run("Name returns key name", func(t *testing.T) {
		key := NewKey[string]("test_key")
		assert.Equal(t, "test_key", key.Name())
	})

	t.Run("String returns key name", func(t *testing.T) {
		key := NewKey[int]("my_key")
		assert.Equal(t, "my_key", key.String())
	})
}

func TestGet(t *testing.T) {
	state := NewStateFrom(map[string]any{
		"string":  "hello",
		"int":     42,
		"struct":  &testStruct{Name: "test", Value: 100},
		"nil_ptr": (*testStruct)(nil),
	})

	t.Run("returns value when type matches", func(t *testing.T) {
		s, ok := Get(state, keyString)
		require.True(t, ok)
		assert.Equal(t, "hello", s)
	})

	t.Run("returns int when type matches", func(t *testing.T) {
		i, ok := Get(state, keyInt)
		require.True(t, ok)
		assert.Equal(t, 42, i)
	})

	t.Run("returns false when key missing", func(t *testing.T) {
		_, ok := Get(state, keyMissing)
		assert.False(t, ok)
	})

	t.Run("returns false when type mismatches", func(t *testing.T) {
		// Try to get string key as int
		wrongTypeKey := NewKey[int]("string")
		_, ok := Get(state, wrongTypeKey)
		assert.False(t, ok)
	})

	t.Run("works with pointer types", func(t *testing.T) {
		s, ok := Get(state, keyStruct)
		require.True(t, ok)
		assert.Equal(t, "test", s.Name)
		assert.Equal(t, 100, s.Value)
	})

	t.Run("works with nil pointers", func(t *testing.T) {
		s, ok := Get(state, keyNilPtr)
		require.True(t, ok)
		assert.Nil(t, s)
	})
}

func TestSet(t *testing.T) {
	t.Run("sets value with correct type", func(t *testing.T) {
		state := NewStateFrom(nil)
		Set(state, keyString, "world")

		v, ok := state.Get("string")
		require.True(t, ok)
		assert.Equal(t, "world", v)
	})

	t.Run("sets struct pointer", func(t *testing.T) {
		state := NewStateFrom(nil)
		Set(state, keyStruct, &testStruct{Name: "new", Value: 200})

		s, ok := Get(state, keyStruct)
		require.True(t, ok)
		assert.Equal(t, "new", s.Name)
		assert.Equal(t, 200, s.Value)
	})
}

func TestMustGet(t *testing.T) {
	state := NewStateFrom(map[string]any{
		"string": "hello",
		"int":    42,
	})

	t.Run("returns value when type matches", func(t *testing.T) {
		s := MustGet(state, keyString)
		assert.Equal(t, "hello", s)
	})

	t.Run("panics when key missing", func(t *testing.T) {
		assert.PanicsWithValue(t,
			`workflow: state key "nonexistent" not found`,
			func() {
				MustGet(state, keyMissing)
			})
	})

	t.Run("panics when type mismatches", func(t *testing.T) {
		wrongTypeKey := NewKey[int]("string")
		assert.Panics(t, func() {
			MustGet(state, wrongTypeKey)
		})
	})
}

func TestGetOr(t *testing.T) {
	state := NewStateFrom(map[string]any{
		"string": "hello",
		"int":    42,
	})

	t.Run("returns value when present", func(t *testing.T) {
		s := GetOr(state, keyString, "default")
		assert.Equal(t, "hello", s)
	})

	t.Run("returns default when missing", func(t *testing.T) {
		s := GetOr(state, keyMissing, "default")
		assert.Equal(t, "default", s)
	})

	t.Run("returns default when type mismatches", func(t *testing.T) {
		wrongTypeKey := NewKey[int]("string")
		i := GetOr(state, wrongTypeKey, 999)
		assert.Equal(t, 999, i)
	})
}

func TestHas(t *testing.T) {
	state := NewStateFrom(map[string]any{
		"string": "hello",
	})

	t.Run("returns true when key exists", func(t *testing.T) {
		assert.True(t, Has(state, keyString))
	})

	t.Run("returns false when key missing", func(t *testing.T) {
		assert.False(t, Has(state, keyMissing))
	})
}

func TestDelete(t *testing.T) {
	state := NewStateFrom(map[string]any{
		"string": "hello",
	})

	Delete(state, keyString)
	assert.False(t, state.Has("string"))
}

func TestSetIfAbsent(t *testing.T) {
	t.Run("sets when key absent", func(t *testing.T) {
		state := NewStateFrom(nil)
		ok := SetIfAbsent(state, keyString, "hello")
		assert.True(t, ok)

		s, _ := Get(state, keyString)
		assert.Equal(t, "hello", s)
	})

	t.Run("does not set when key present", func(t *testing.T) {
		state := NewStateFrom(map[string]any{
			"string": "original",
		})
		ok := SetIfAbsent(state, keyString, "new")
		assert.False(t, ok)

		s, _ := Get(state, keyString)
		assert.Equal(t, "original", s)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("updates existing value", func(t *testing.T) {
		state := NewStateFrom(map[string]any{
			"counter": 10,
		})

		result := Update(state, keyCounter, func(v int) int { return v + 5 })
		assert.Equal(t, 15, result)

		v, _ := Get(state, keyCounter)
		assert.Equal(t, 15, v)
	})

	t.Run("updates from zero when missing", func(t *testing.T) {
		state := NewStateFrom(nil)

		result := Update(state, keyCounter, func(v int) int { return v + 1 })
		assert.Equal(t, 1, result)

		v, _ := Get(state, keyCounter)
		assert.Equal(t, 1, v)
	})
}

func TestConvenienceConstructors(t *testing.T) {
	t.Run("IntKey", func(t *testing.T) {
		key := IntKey("count")
		assert.Equal(t, "count", key.Name())

		state := NewStateFrom(nil)
		Set(state, key, 42)

		v, ok := Get(state, key)
		require.True(t, ok)
		assert.Equal(t, 42, v)
	})

	t.Run("StringKey", func(t *testing.T) {
		key := StringKey("name")
		assert.Equal(t, "name", key.Name())

		state := NewStateFrom(nil)
		Set(state, key, "test")

		v, ok := Get(state, key)
		require.True(t, ok)
		assert.Equal(t, "test", v)
	})

	t.Run("BoolKey", func(t *testing.T) {
		key := BoolKey("enabled")
		assert.Equal(t, "enabled", key.Name())

		state := NewStateFrom(nil)
		Set(state, key, true)

		v, ok := Get(state, key)
		require.True(t, ok)
		assert.True(t, v)
	})

	t.Run("FloatKey", func(t *testing.T) {
		key := FloatKey("score")
		assert.Equal(t, "score", key.Name())

		state := NewStateFrom(nil)
		Set(state, key, 3.14)

		v, ok := Get(state, key)
		require.True(t, ok)
		assert.Equal(t, 3.14, v)
	})
}

func TestGetBranchState(t *testing.T) {
	t.Run("returns state when present in metadata", func(t *testing.T) {
		branchState := NewStateFrom(map[string]any{"key": "value"})
		result := &StepResult{
			Metadata: map[string]any{"branch_state": branchState},
		}

		state, ok := GetBranchState(result)
		require.True(t, ok)
		assert.Same(t, branchState, state)
	})

	t.Run("returns false when nil result", func(t *testing.T) {
		state, ok := GetBranchState(nil)
		assert.False(t, ok)
		assert.Nil(t, state)
	})

	t.Run("returns false when nil metadata", func(t *testing.T) {
		result := &StepResult{Metadata: nil}

		state, ok := GetBranchState(result)
		assert.False(t, ok)
		assert.Nil(t, state)
	})

	t.Run("returns false when branch_state missing", func(t *testing.T) {
		result := &StepResult{
			Metadata: map[string]any{"other": "data"},
		}

		state, ok := GetBranchState(result)
		assert.False(t, ok)
		assert.Nil(t, state)
	})

	t.Run("returns false when branch_state wrong type", func(t *testing.T) {
		result := &StepResult{
			Metadata: map[string]any{"branch_state": "not a state"},
		}

		state, ok := GetBranchState(result)
		assert.False(t, ok)
		assert.Nil(t, state)
	})
}

func TestGetFromBranch(t *testing.T) {
	t.Run("returns typed value from branch state", func(t *testing.T) {
		branchState := NewStateFrom(nil)
		Set(branchState, keyString, "hello from branch")
		Set(branchState, keyInt, 42)

		result := &StepResult{
			Metadata: map[string]any{"branch_state": branchState},
		}

		s, ok := GetFromBranch(result, keyString)
		require.True(t, ok)
		assert.Equal(t, "hello from branch", s)

		i, ok := GetFromBranch(result, keyInt)
		require.True(t, ok)
		assert.Equal(t, 42, i)
	})

	t.Run("returns false when key missing in branch state", func(t *testing.T) {
		branchState := NewStateFrom(nil)
		result := &StepResult{
			Metadata: map[string]any{"branch_state": branchState},
		}

		_, ok := GetFromBranch(result, keyMissing)
		assert.False(t, ok)
	})

	t.Run("returns false when no branch state", func(t *testing.T) {
		result := &StepResult{Metadata: nil}

		_, ok := GetFromBranch(result, keyString)
		assert.False(t, ok)
	})

	t.Run("returns false when type mismatches", func(t *testing.T) {
		branchState := NewStateFrom(map[string]any{"string": "hello"})
		result := &StepResult{
			Metadata: map[string]any{"branch_state": branchState},
		}

		wrongTypeKey := NewKey[int]("string")
		_, ok := GetFromBranch(result, wrongTypeKey)
		assert.False(t, ok)
	})
}

func TestMustGetFromBranch(t *testing.T) {
	t.Run("returns typed value from branch state", func(t *testing.T) {
		branchState := NewStateFrom(nil)
		Set(branchState, keyString, "hello from branch")

		result := &StepResult{
			Metadata: map[string]any{"branch_state": branchState},
		}

		s := MustGetFromBranch(result, keyString)
		assert.Equal(t, "hello from branch", s)
	})

	t.Run("panics when no branch state", func(t *testing.T) {
		result := &StepResult{Metadata: nil}

		assert.PanicsWithValue(t,
			"workflow: StepResult has no branch_state metadata",
			func() {
				MustGetFromBranch(result, keyString)
			})
	})

	t.Run("panics when key missing", func(t *testing.T) {
		branchState := NewStateFrom(nil)
		result := &StepResult{
			Metadata: map[string]any{"branch_state": branchState},
		}

		assert.PanicsWithValue(t,
			`workflow: state key "nonexistent" not found`,
			func() {
				MustGetFromBranch(result, keyMissing)
			})
	})

	t.Run("panics when type mismatches", func(t *testing.T) {
		branchState := NewStateFrom(map[string]any{"string": "hello"})
		result := &StepResult{
			Metadata: map[string]any{"branch_state": branchState},
		}

		wrongTypeKey := NewKey[int]("string")
		assert.Panics(t, func() {
			MustGetFromBranch(result, wrongTypeKey)
		})
	})
}
