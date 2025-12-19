package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Define typed keys for tests
var (
	keyDone       = NewKey[bool]("done")
	keyStatus     = NewKey[string]("status")
	keyRetry      = NewKey[bool]("retry")
	keyLoopResult = NewKey[string]("loop_result")
	keyReady      = NewKey[bool]("ready")
	keyCount      = NewKey[int]("count")
	keyItems      = NewKey[[]string]("items")
)

func TestNewLoopUntil(t *testing.T) {
	t.Run("exits when key equals value", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("increment", func(ctx context.Context, state *State) error {
			iterations++
			if iterations >= 3 {
				Set(state, keyDone, true)
			}
			return nil
		})

		loop := NewLoopUntil("test-loop", step, keyDone, true)
		state := NewState(nil)

		_, err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 3, iterations)
	})

	t.Run("continues when key has different value", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("increment", func(ctx context.Context, state *State) error {
			iterations++
			Set(state, keyStatus, "pending")
			if iterations >= 2 {
				Set(state, keyStatus, "complete")
			}
			return nil
		})

		loop := NewLoopUntil("test-loop", step, keyStatus, "complete")
		state := NewState(nil)

		_, err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 2, iterations)
	})

	t.Run("respects max iterations", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("never-done", func(ctx context.Context, state *State) error {
			iterations++
			return nil
		})

		loop := NewLoopUntil("test-loop", step, keyDone, true, WithMaxIterations(5))
		state := NewState(nil)

		_, err := loop.Run(context.Background(), state)
		assert.ErrorIs(t, err, ErrMaxIterationsExceeded)
		assert.Equal(t, 5, iterations)
	})
}

func TestNewLoopWhile(t *testing.T) {
	t.Run("continues while key equals value", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("increment", func(ctx context.Context, state *State) error {
			iterations++
			if iterations >= 3 {
				Set(state, keyRetry, false)
			}
			return nil
		})

		loop := NewLoopWhile("test-loop", step, keyRetry, true)
		state := NewState(nil)
		Set(state, keyRetry, true)

		_, err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 3, iterations)
	})

	t.Run("exits immediately when key not set", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("increment", func(ctx context.Context, state *State) error {
			iterations++
			return nil
		})

		// Use a key that's not set - loop should exit after first iteration
		nonexistent := NewKey[string]("nonexistent")
		loop := NewLoopWhile("test-loop", step, nonexistent, "value")
		state := NewState(nil)

		_, err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 1, iterations) // runs once, then condition checked
	})

	t.Run("exits when key changes to different value", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("change-status", func(ctx context.Context, state *State) error {
			iterations++
			if iterations >= 2 {
				Set(state, keyStatus, "done")
			}
			return nil
		})

		loop := NewLoopWhile("test-loop", step, keyStatus, "running")
		state := NewState(nil)
		Set(state, keyStatus, "running")

		_, err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 2, iterations)
	})
}

func TestNewLoopUntilSet(t *testing.T) {
	t.Run("exits when key becomes truthy string", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("set-result", func(ctx context.Context, state *State) error {
			iterations++
			if iterations >= 2 {
				Set(state, keyLoopResult, "success")
			}
			return nil
		})

		loop := NewLoopUntilSet("test-loop", step, keyLoopResult)
		state := NewState(nil)

		_, err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 2, iterations)
	})

	t.Run("continues when key is empty string", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("set-result", func(ctx context.Context, state *State) error {
			iterations++
			if iterations < 3 {
				Set(state, keyLoopResult, "")
			} else {
				Set(state, keyLoopResult, "done")
			}
			return nil
		})

		loop := NewLoopUntilSet("test-loop", step, keyLoopResult)
		state := NewState(nil)

		_, err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 3, iterations)
	})

	t.Run("exits when key becomes true bool", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("set-flag", func(ctx context.Context, state *State) error {
			iterations++
			if iterations >= 2 {
				Set(state, keyReady, true)
			}
			return nil
		})

		loop := NewLoopUntilSet("test-loop", step, keyReady)
		state := NewState(nil)

		_, err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 2, iterations)
	})

	t.Run("continues when key is false bool", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("set-flag", func(ctx context.Context, state *State) error {
			iterations++
			Set(state, keyReady, iterations >= 3)
			return nil
		})

		loop := NewLoopUntilSet("test-loop", step, keyReady)
		state := NewState(nil)

		_, err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 3, iterations)
	})

	t.Run("exits when key becomes non-zero int", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("set-count", func(ctx context.Context, state *State) error {
			iterations++
			if iterations >= 2 {
				Set(state, keyCount, 42)
			}
			return nil
		})

		loop := NewLoopUntilSet("test-loop", step, keyCount)
		state := NewState(nil)

		_, err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 2, iterations)
	})

	t.Run("continues when key is zero int", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("set-count", func(ctx context.Context, state *State) error {
			iterations++
			if iterations < 3 {
				Set(state, keyCount, 0)
			} else {
				Set(state, keyCount, 1)
			}
			return nil
		})

		loop := NewLoopUntilSet("test-loop", step, keyCount)
		state := NewState(nil)

		_, err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 3, iterations)
	})

	t.Run("exits when key becomes non-empty slice", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("set-items", func(ctx context.Context, state *State) error {
			iterations++
			if iterations >= 2 {
				Set(state, keyItems, []string{"a", "b"})
			}
			return nil
		})

		loop := NewLoopUntilSet("test-loop", step, keyItems)
		state := NewState(nil)

		_, err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 2, iterations)
	})

	t.Run("continues when key is empty slice", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep("set-items", func(ctx context.Context, state *State) error {
			iterations++
			if iterations < 3 {
				Set(state, keyItems, []string{})
			} else {
				Set(state, keyItems, []string{"done"})
			}
			return nil
		})

		loop := NewLoopUntilSet("test-loop", step, keyItems)
		state := NewState(nil)

		_, err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 3, iterations)
	})
}

func TestIsTruthy(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		// Falsy values
		{"nil", nil, false},
		{"empty string", "", false},
		{"false bool", false, false},
		{"zero int", 0, false},
		{"zero int64", int64(0), false},
		{"zero float64", 0.0, false},
		{"empty slice", []string{}, false},
		{"empty map", map[string]int{}, false},

		// Truthy values
		{"non-empty string", "hello", true},
		{"true bool", true, true},
		{"positive int", 42, true},
		{"negative int", -1, true},
		{"positive int64", int64(100), true},
		{"positive float64", 3.14, true},
		{"negative float64", -2.5, true},
		{"non-empty slice", []string{"a"}, true},
		{"non-empty map", map[string]int{"a": 1}, true},
		{"struct", struct{ X int }{1}, true},
		{"pointer to zero", new(int), true}, // pointer exists, truthy
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := NewState(nil)
			if tt.value != nil {
				state.Set("key", tt.value)
			}
			result := isTruthy(state, "key")
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("missing key", func(t *testing.T) {
		state := NewState(nil)
		result := isTruthy(state, "nonexistent")
		assert.False(t, result)
	})
}
