package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test state struct for parallel/aggregator tests
type parallelTestState struct {
	Input   string
	A       string
	B       string
	C       string
	Results []string
	Map     map[string]string
}

func TestParallelWithAggregator(t *testing.T) {
	t.Run("custom aggregator collects results", func(t *testing.T) {
		steps := []Step[parallelTestState]{
			NewFuncStep[parallelTestState]("step1", func(ctx context.Context, state *parallelTestState) error {
				state.A = "from-step1"
				return nil
			}),
			NewFuncStep[parallelTestState]("step2", func(ctx context.Context, state *parallelTestState) error {
				state.B = "from-step2"
				return nil
			}),
		}

		// Aggregator that merges A and B from branch states
		aggregator := func(state *parallelTestState, branches map[string]*parallelTestState, errs map[string]error) error {
			for _, br := range branches {
				if br.A != "" {
					state.A = br.A
				}
				if br.B != "" {
					state.B = br.B
				}
			}
			return nil
		}

		parallel := NewParallel("test", steps, aggregator)
		state := &parallelTestState{}

		err := parallel.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, "from-step1", state.A)
		assert.Equal(t, "from-step2", state.B)
	})

	t.Run("selective merge aggregator", func(t *testing.T) {
		steps := []Step[parallelTestState]{
			NewFuncStep[parallelTestState]("step1", func(ctx context.Context, state *parallelTestState) error {
				state.A = "from-step1"
				state.C = "ignored"
				return nil
			}),
			NewFuncStep[parallelTestState]("step2", func(ctx context.Context, state *parallelTestState) error {
				state.B = "from-step2"
				return nil
			}),
		}

		// Only merge A and B, ignore C
		aggregator := func(state *parallelTestState, branches map[string]*parallelTestState, errs map[string]error) error {
			for _, br := range branches {
				if br.A != "" {
					state.A = br.A
				}
				if br.B != "" {
					state.B = br.B
				}
				// Deliberately not merging C
			}
			return nil
		}

		parallel := NewParallel("test", steps, aggregator)
		state := &parallelTestState{}

		err := parallel.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, "from-step1", state.A)
		assert.Equal(t, "from-step2", state.B)
		assert.Empty(t, state.C, "C should not be merged")
	})

	t.Run("collect into slice", func(t *testing.T) {
		steps := []Step[parallelTestState]{
			NewFuncStep[parallelTestState]("step1", func(ctx context.Context, state *parallelTestState) error {
				state.A = "value1"
				return nil
			}),
			NewFuncStep[parallelTestState]("step2", func(ctx context.Context, state *parallelTestState) error {
				state.A = "value2"
				return nil
			}),
			NewFuncStep[parallelTestState]("step3", func(ctx context.Context, state *parallelTestState) error {
				state.A = "value3"
				return nil
			}),
		}

		// Collect A from all branches into Results slice
		aggregator := func(state *parallelTestState, branches map[string]*parallelTestState, errs map[string]error) error {
			state.Results = make([]string, 0, len(branches))
			for _, br := range branches {
				if br.A != "" {
					state.Results = append(state.Results, br.A)
				}
			}
			return nil
		}

		parallel := NewParallel("test", steps, aggregator)
		state := &parallelTestState{}

		err := parallel.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Len(t, state.Results, 3)
		assert.ElementsMatch(t, []string{"value1", "value2", "value3"}, state.Results)
	})

	t.Run("collect into map by step name", func(t *testing.T) {
		steps := []Step[parallelTestState]{
			NewFuncStep[parallelTestState]("step1", func(ctx context.Context, state *parallelTestState) error {
				state.A = "value1"
				return nil
			}),
			NewFuncStep[parallelTestState]("step2", func(ctx context.Context, state *parallelTestState) error {
				state.A = "value2"
				return nil
			}),
		}

		// Collect A from all branches into Map keyed by step name
		aggregator := func(state *parallelTestState, branches map[string]*parallelTestState, errs map[string]error) error {
			state.Map = make(map[string]string, len(branches))
			for name, br := range branches {
				state.Map[name] = br.A
			}
			return nil
		}

		parallel := NewParallel("test", steps, aggregator)
		state := &parallelTestState{}

		err := parallel.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Len(t, state.Map, 2)
		assert.Equal(t, "value1", state.Map["step1"])
		assert.Equal(t, "value2", state.Map["step2"])
	})

	t.Run("aggregator handles errors with ContinueOnError", func(t *testing.T) {
		steps := []Step[parallelTestState]{
			NewFuncStep[parallelTestState]("step1", func(ctx context.Context, state *parallelTestState) error {
				state.A = "value1"
				return nil
			}),
			NewFuncStep[parallelTestState]("step2", func(ctx context.Context, state *parallelTestState) error {
				return assert.AnError
			}),
			NewFuncStep[parallelTestState]("step3", func(ctx context.Context, state *parallelTestState) error {
				state.A = "value3"
				return nil
			}),
		}

		// Collect from successful steps only
		aggregator := func(state *parallelTestState, branches map[string]*parallelTestState, errs map[string]error) error {
			state.Results = make([]string, 0)
			for _, br := range branches {
				if br.A != "" {
					state.Results = append(state.Results, br.A)
				}
			}
			return nil
		}

		parallel := NewParallel("test", steps, aggregator)
		state := &parallelTestState{}

		err := parallel.Run(context.Background(), state, WithContinueOnError(true))
		require.NoError(t, err)
		assert.Len(t, state.Results, 2)
		assert.ElementsMatch(t, []string{"value1", "value3"}, state.Results)
	})
}
