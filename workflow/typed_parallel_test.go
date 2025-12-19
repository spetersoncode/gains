package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	keyScore     = NewKey[int]("score")
	keyScores    = NewKey[[]int]("scores")
	keyTotalSum  = NewKey[int]("total")
	keyAnalysis  = NewKey[string]("analysis")
	keyAnalyses  = NewKey[[]string]("analyses")
)

func TestTypedParallel(t *testing.T) {
	steps := []Step{
		NewFuncStep("scorer1", func(ctx context.Context, state *State) error {
			Set(state, keyScore, 10)
			return nil
		}),
		NewFuncStep("scorer2", func(ctx context.Context, state *State) error {
			Set(state, keyScore, 20)
			return nil
		}),
		NewFuncStep("scorer3", func(ctx context.Context, state *State) error {
			Set(state, keyScore, 30)
			return nil
		}),
	}

	parallel := NewTypedParallel("collect-scores", steps, keyScore, keyScores)
	state := NewState(nil)

	_, err := parallel.Run(context.Background(), state)

	require.NoError(t, err)
	scores := MustGet(state, keyScores)
	assert.Len(t, scores, 3)
	assert.ElementsMatch(t, []int{10, 20, 30}, scores)
}

func TestTypedParallel_Name(t *testing.T) {
	parallel := NewTypedParallel("my-parallel", nil, keyScore, keyScores)
	assert.Equal(t, "my-parallel", parallel.Name())
}

func TestTypedParallel_RunStream(t *testing.T) {
	steps := []Step{
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			Set(state, keyAnalysis, "result1")
			return nil
		}),
		NewFuncStep("step2", func(ctx context.Context, state *State) error {
			Set(state, keyAnalysis, "result2")
			return nil
		}),
	}

	parallel := NewTypedParallel("stream-test", steps, keyAnalysis, keyAnalyses)
	state := NewState(nil)

	events := parallel.RunStream(context.Background(), state)

	// Drain events
	for range events {
	}

	analyses := MustGet(state, keyAnalyses)
	assert.Len(t, analyses, 2)
	assert.ElementsMatch(t, []string{"result1", "result2"}, analyses)
}

func TestTypedParallelWithAggregator(t *testing.T) {
	steps := []Step{
		NewFuncStep("scorer1", func(ctx context.Context, state *State) error {
			Set(state, keyScore, 10)
			return nil
		}),
		NewFuncStep("scorer2", func(ctx context.Context, state *State) error {
			Set(state, keyScore, 20)
			return nil
		}),
		NewFuncStep("scorer3", func(ctx context.Context, state *State) error {
			Set(state, keyScore, 30)
			return nil
		}),
	}

	// Sum all scores
	parallel := NewTypedParallelWithAggregator(
		"sum-scores",
		steps,
		keyScore,
		func(scores []int) int {
			sum := 0
			for _, s := range scores {
				sum += s
			}
			return sum
		},
		keyTotalSum,
	)

	state := NewState(nil)
	_, err := parallel.Run(context.Background(), state)

	require.NoError(t, err)
	total := MustGet(state, keyTotalSum)
	assert.Equal(t, 60, total)
}

func TestTypedParallelWithAggregator_Name(t *testing.T) {
	parallel := NewTypedParallelWithAggregator(
		"my-aggregator",
		nil,
		keyScore,
		func(scores []int) int { return 0 },
		keyTotalSum,
	)
	assert.Equal(t, "my-aggregator", parallel.Name())
}

func TestTypedParallelWithAggregator_RunStream(t *testing.T) {
	steps := []Step{
		NewFuncStep("scorer1", func(ctx context.Context, state *State) error {
			Set(state, keyScore, 5)
			return nil
		}),
		NewFuncStep("scorer2", func(ctx context.Context, state *State) error {
			Set(state, keyScore, 15)
			return nil
		}),
	}

	parallel := NewTypedParallelWithAggregator(
		"stream-sum",
		steps,
		keyScore,
		func(scores []int) int {
			sum := 0
			for _, s := range scores {
				sum += s
			}
			return sum
		},
		keyTotalSum,
	)

	state := NewState(nil)
	events := parallel.RunStream(context.Background(), state)

	// Drain events
	for range events {
	}

	total := MustGet(state, keyTotalSum)
	assert.Equal(t, 20, total)
}

func TestTypedParallel_WithContinueOnError(t *testing.T) {
	steps := []Step{
		NewFuncStep("scorer1", func(ctx context.Context, state *State) error {
			Set(state, keyScore, 10)
			return nil
		}),
		NewFuncStep("scorer2", func(ctx context.Context, state *State) error {
			return assert.AnError
		}),
		NewFuncStep("scorer3", func(ctx context.Context, state *State) error {
			Set(state, keyScore, 30)
			return nil
		}),
	}

	parallel := NewTypedParallel("with-errors", steps, keyScore, keyScores)
	state := NewState(nil)

	_, err := parallel.Run(context.Background(), state, WithContinueOnError(true))

	require.NoError(t, err)
	scores := MustGet(state, keyScores)
	assert.Len(t, scores, 2)
	assert.ElementsMatch(t, []int{10, 30}, scores)
}
