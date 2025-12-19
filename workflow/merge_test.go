package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	keyA      = NewKey[string]("a")
	keyB      = NewKey[string]("b")
	keyC      = NewKey[string]("c")
	keyResult = NewKey[[]string]("result")
	keyMap    = NewKey[map[string]string]("map")
)

func TestMergeAll(t *testing.T) {
	steps := []Step{
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			Set(state, keyA, "from-step1")
			return nil
		}),
		NewFuncStep("step2", func(ctx context.Context, state *State) error {
			Set(state, keyB, "from-step2")
			return nil
		}),
	}

	parallel := NewParallel("test", steps, MergeAll())
	state := NewState(nil)

	_, err := parallel.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, "from-step1", MustGet(state, keyA))
	assert.Equal(t, "from-step2", MustGet(state, keyB))
}

func TestMergeKeys(t *testing.T) {
	steps := []Step{
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			Set(state, keyA, "from-step1")
			Set(state, keyC, "ignored")
			return nil
		}),
		NewFuncStep("step2", func(ctx context.Context, state *State) error {
			Set(state, keyB, "from-step2")
			return nil
		}),
	}

	// Only merge keyA and keyB, ignore keyC
	parallel := NewParallel("test", steps, MergeKeys("a", "b"))
	state := NewState(nil)

	_, err := parallel.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, "from-step1", MustGet(state, keyA))
	assert.Equal(t, "from-step2", MustGet(state, keyB))
	assert.False(t, Has(state, keyC), "keyC should not be merged")
}

func TestMergeTypedKey(t *testing.T) {
	steps := []Step{
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			Set(state, keyA, "first")
			Set(state, keyB, "should-be-ignored")
			return nil
		}),
		NewFuncStep("step2", func(ctx context.Context, state *State) error {
			Set(state, keyA, "second")
			return nil
		}),
	}

	// Only merge keyA
	parallel := NewParallel("test", steps, MergeTypedKey(keyA))
	state := NewState(nil)

	_, err := parallel.Run(context.Background(), state)

	require.NoError(t, err)
	// One of "first" or "second" will be present (non-deterministic)
	val, ok := Get(state, keyA)
	assert.True(t, ok)
	assert.Contains(t, []string{"first", "second"}, val)
	assert.False(t, Has(state, keyB), "keyB should not be merged")
}

func TestCollectInto(t *testing.T) {
	steps := []Step{
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			Set(state, keyA, "value1")
			return nil
		}),
		NewFuncStep("step2", func(ctx context.Context, state *State) error {
			Set(state, keyA, "value2")
			return nil
		}),
		NewFuncStep("step3", func(ctx context.Context, state *State) error {
			Set(state, keyA, "value3")
			return nil
		}),
	}

	parallel := NewParallel("test", steps, CollectInto(keyA, keyResult))
	state := NewState(nil)

	_, err := parallel.Run(context.Background(), state)

	require.NoError(t, err)
	results := MustGet(state, keyResult)
	assert.Len(t, results, 3)
	assert.ElementsMatch(t, []string{"value1", "value2", "value3"}, results)
}

func TestCollectMap(t *testing.T) {
	steps := []Step{
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			Set(state, keyA, "value1")
			return nil
		}),
		NewFuncStep("step2", func(ctx context.Context, state *State) error {
			Set(state, keyA, "value2")
			return nil
		}),
	}

	parallel := NewParallel("test", steps, CollectMap(keyA, keyMap))
	state := NewState(nil)

	_, err := parallel.Run(context.Background(), state)

	require.NoError(t, err)
	results := MustGet(state, keyMap)
	assert.Len(t, results, 2)
	assert.Equal(t, "value1", results["step1"])
	assert.Equal(t, "value2", results["step2"])
}

func TestCollectInto_WithErrors(t *testing.T) {
	steps := []Step{
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			Set(state, keyA, "value1")
			return nil
		}),
		NewFuncStep("step2", func(ctx context.Context, state *State) error {
			return assert.AnError
		}),
		NewFuncStep("step3", func(ctx context.Context, state *State) error {
			Set(state, keyA, "value3")
			return nil
		}),
	}

	parallel := NewParallel("test", steps, CollectInto(keyA, keyResult))
	state := NewState(nil)

	// With ContinueOnError, should collect from successful steps
	_, err := parallel.Run(context.Background(), state, WithContinueOnError(true))

	require.NoError(t, err)
	results := MustGet(state, keyResult)
	assert.Len(t, results, 2)
	assert.ElementsMatch(t, []string{"value1", "value3"}, results)
}
