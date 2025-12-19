package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testStateStruct struct {
	UserID   string `json:"user_id"`
	Count    int    `json:"count"`
	Score    float64
	Active   bool   `json:"active"`
	Optional string `json:"optional,omitempty"`
	Ignored  string `json:"-"`
}

func TestNewStateFromStruct(t *testing.T) {
	t.Run("populates state from struct fields", func(t *testing.T) {
		input := &testStateStruct{
			UserID: "abc123",
			Count:  42,
			Score:  3.14,
			Active: true,
		}

		state := NewStateFromStruct(input)

		// Check json-tagged fields
		userID, ok := state.Get("user_id")
		assert.True(t, ok)
		assert.Equal(t, "abc123", userID)

		count, ok := state.Get("count")
		assert.True(t, ok)
		assert.Equal(t, 42, count)

		active, ok := state.Get("active")
		assert.True(t, ok)
		assert.Equal(t, true, active)

		// Check field without json tag (uses snake_case)
		score, ok := state.Get("score")
		assert.True(t, ok)
		assert.Equal(t, 3.14, score)
	})

	t.Run("skips zero values", func(t *testing.T) {
		input := &testStateStruct{
			UserID: "abc123",
			// Count is zero
			// Score is zero
			// Active is false
			// Optional is empty
		}

		state := NewStateFromStruct(input)

		_, ok := state.Get("count")
		assert.False(t, ok, "zero int should be skipped")

		_, ok = state.Get("score")
		assert.False(t, ok, "zero float should be skipped")

		_, ok = state.Get("active")
		assert.False(t, ok, "false bool should be skipped")

		_, ok = state.Get("optional")
		assert.False(t, ok, "empty string should be skipped")
	})

	t.Run("skips json:\"-\" fields", func(t *testing.T) {
		input := &testStateStruct{
			UserID:  "abc123",
			Ignored: "should not appear",
		}

		state := NewStateFromStruct(input)

		_, ok := state.Get("Ignored")
		assert.False(t, ok)
		_, ok = state.Get("ignored")
		assert.False(t, ok)
		_, ok = state.Get("-")
		assert.False(t, ok)
	})

	t.Run("handles nil pointer", func(t *testing.T) {
		state := NewStateFromStruct(nil)
		assert.NotNil(t, state)
	})

	t.Run("handles non-struct", func(t *testing.T) {
		s := "not a struct"
		state := NewStateFromStruct(&s)
		assert.NotNil(t, state)
	})

	t.Run("allows dynamic modifications after creation", func(t *testing.T) {
		input := &testStateStruct{UserID: "abc123"}
		state := NewStateFromStruct(input)

		// Can still set additional keys
		state.Set("extra_key", "extra_value")

		extra, ok := state.Get("extra_key")
		assert.True(t, ok)
		assert.Equal(t, "extra_value", extra)
	})
}

type testIncludeZeroStruct struct {
	Count int  `json:"count" state:"include_zero"`
	Flag  bool `json:"flag" state:"include_zero"`
}

func TestNewStateFromStruct_IncludeZero(t *testing.T) {
	t.Run("includes zero values with state:include_zero tag", func(t *testing.T) {
		input := &testIncludeZeroStruct{
			Count: 0,
			Flag:  false,
		}

		state := NewStateFromStruct(input)

		count, ok := state.Get("count")
		assert.True(t, ok, "zero int should be included with include_zero tag")
		assert.Equal(t, 0, count)

		flag, ok := state.Get("flag")
		assert.True(t, ok, "false bool should be included with include_zero tag")
		assert.Equal(t, false, flag)
	})
}

func TestStateInto(t *testing.T) {
	t.Run("populates struct from state", func(t *testing.T) {
		state := NewStateFrom(map[string]any{
			"user_id": "abc123",
			"count":   42,
			"score":   3.14,
			"active":  true,
		})

		var result testStateStruct
		StateInto(state, &result)

		assert.Equal(t, "abc123", result.UserID)
		assert.Equal(t, 42, result.Count)
		assert.Equal(t, 3.14, result.Score)
		assert.Equal(t, true, result.Active)
	})

	t.Run("handles missing keys", func(t *testing.T) {
		state := NewStateFrom(map[string]any{
			"user_id": "abc123",
			// count is missing
		})

		result := testStateStruct{Count: 99} // pre-set value
		StateInto(state, &result)

		assert.Equal(t, "abc123", result.UserID)
		assert.Equal(t, 99, result.Count, "missing key should not overwrite")
	})

	t.Run("handles nil state", func(t *testing.T) {
		var result testStateStruct
		StateInto(nil, &result) // should not panic
	})

	t.Run("handles nil struct", func(t *testing.T) {
		state := NewStateFrom(map[string]any{"user_id": "abc123"})
		StateInto(state, nil) // should not panic
	})
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"UserID", "user_i_d"},
		{"Count", "count"},
		{"MaxRetries", "max_retries"},
		{"HTTPClient", "h_t_t_p_client"},
		{"Simple", "simple"},
		{"ABC", "a_b_c"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := toSnakeCase(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	t.Run("struct to state to struct", func(t *testing.T) {
		original := &testStateStruct{
			UserID: "user123",
			Count:  100,
			Score:  9.5,
			Active: true,
		}

		// Convert to state
		state := NewStateFromStruct(original)

		// Modify state dynamically
		state.Set("count", 200)

		// Convert back to struct
		var result testStateStruct
		StateInto(state, &result)

		assert.Equal(t, "user123", result.UserID)
		assert.Equal(t, 200, result.Count) // modified value
		assert.Equal(t, 9.5, result.Score)
		assert.Equal(t, true, result.Active)
	})
}
