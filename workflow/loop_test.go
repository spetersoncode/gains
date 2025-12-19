package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test state struct for loop tests
type loopTestState struct {
	Done       bool
	Status     string
	Retry      bool
	LoopResult string
	Ready      bool
	Count      int
	Items      []string
}

func TestNewLoopUntil(t *testing.T) {
	t.Run("exits when predicate returns true", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep[loopTestState]("increment", func(ctx context.Context, state *loopTestState) error {
			iterations++
			if iterations >= 3 {
				state.Done = true
			}
			return nil
		})

		loop := NewLoopUntil("test-loop", step, func(s *loopTestState) bool {
			return s.Done
		})
		state := &loopTestState{}

		err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 3, iterations)
	})

	t.Run("continues when predicate returns false", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep[loopTestState]("increment", func(ctx context.Context, state *loopTestState) error {
			iterations++
			state.Status = "pending"
			if iterations >= 2 {
				state.Status = "complete"
			}
			return nil
		})

		loop := NewLoopUntil("test-loop", step, func(s *loopTestState) bool {
			return s.Status == "complete"
		})
		state := &loopTestState{}

		err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 2, iterations)
	})

	t.Run("respects max iterations", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep[loopTestState]("never-done", func(ctx context.Context, state *loopTestState) error {
			iterations++
			return nil
		})

		loop := NewLoopUntil("test-loop", step,
			func(s *loopTestState) bool { return s.Done },
			WithMaxIterations(5),
		)
		state := &loopTestState{}

		err := loop.Run(context.Background(), state)
		assert.ErrorIs(t, err, ErrMaxIterationsExceeded)
		assert.Equal(t, 5, iterations)
	})
}

func TestNewLoopWhile(t *testing.T) {
	t.Run("continues while predicate returns true", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep[loopTestState]("increment", func(ctx context.Context, state *loopTestState) error {
			iterations++
			if iterations >= 3 {
				state.Retry = false
			}
			return nil
		})

		loop := NewLoopWhile("test-loop", step, func(s *loopTestState) bool {
			return s.Retry
		})
		state := &loopTestState{Retry: true}

		err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 3, iterations)
	})

	t.Run("exits immediately when predicate returns false", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep[loopTestState]("increment", func(ctx context.Context, state *loopTestState) error {
			iterations++
			return nil
		})

		loop := NewLoopWhile("test-loop", step, func(s *loopTestState) bool {
			return s.Status == "running" // Status is empty, so false
		})
		state := &loopTestState{}

		err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 1, iterations) // runs once, then condition checked
	})

	t.Run("exits when predicate changes to false", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep[loopTestState]("change-status", func(ctx context.Context, state *loopTestState) error {
			iterations++
			if iterations >= 2 {
				state.Status = "done"
			}
			return nil
		})

		loop := NewLoopWhile("test-loop", step, func(s *loopTestState) bool {
			return s.Status == "running"
		})
		state := &loopTestState{Status: "running"}

		err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 2, iterations)
	})
}

func TestNewLoopN(t *testing.T) {
	t.Run("executes exactly n times", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep[loopTestState]("increment", func(ctx context.Context, state *loopTestState) error {
			iterations++
			state.Count = iterations
			return nil
		})

		loop := NewLoopN("test-loop", step, 5)
		state := &loopTestState{}

		err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 5, iterations)
		assert.Equal(t, 5, state.Count)
	})

	t.Run("executes once for n=1", func(t *testing.T) {
		iterations := 0
		step := NewFuncStep[loopTestState]("increment", func(ctx context.Context, state *loopTestState) error {
			iterations++
			return nil
		})

		loop := NewLoopN("test-loop", step, 1)
		state := &loopTestState{}

		err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, 1, iterations)
	})
}

func TestNewLoopWithExitCondition(t *testing.T) {
	t.Run("provides iteration count to condition", func(t *testing.T) {
		var seenIterations []int
		step := NewFuncStep[loopTestState]("track", func(ctx context.Context, state *loopTestState) error {
			return nil
		})

		loop := NewLoopWithExitCondition("test-loop", step,
			func(ctx context.Context, s *loopTestState, iter int) bool {
				seenIterations = append(seenIterations, iter)
				return iter >= 3
			},
		)
		state := &loopTestState{}

		err := loop.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, seenIterations)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		step := NewFuncStep[loopTestState]("slow", func(ctx context.Context, state *loopTestState) error {
			return nil
		})

		ctx, cancel := context.WithCancel(context.Background())
		iterations := 0

		loop := NewLoopWithExitCondition("test-loop", step,
			func(ctx context.Context, s *loopTestState, iter int) bool {
				iterations++
				if iterations >= 2 {
					cancel()
				}
				return false
			},
		)
		state := &loopTestState{}

		err := loop.Run(ctx, state)
		assert.Error(t, err) // Should error due to context cancellation
	})
}
