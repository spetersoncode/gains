package workflow

import (
	"context"
)

// TypedParallel executes steps concurrently where each branch produces the same type T.
// Results are automatically collected into a slice.
type TypedParallel[T any] struct {
	name      string
	steps     []Step
	inputKey  Key[T]
	outputKey Key[[]T]
}

// NewTypedParallel creates a parallel workflow for homogeneous branches.
// Each branch should write its result to inputKey; all results are collected
// into outputKey as []T. Order of results is non-deterministic.
//
// Example:
//
//	var KeyChunkAnalysis = workflow.NewKey[*Analysis]("chunk_analysis")
//	var KeyAllAnalyses = workflow.NewKey[[]*Analysis]("all_analyses")
//
//	parallel := workflow.NewTypedParallel(
//	    "analyze-chunks",
//	    chunkSteps,
//	    KeyChunkAnalysis,  // Each branch writes here
//	    KeyAllAnalyses,    // Collected results go here
//	)
func NewTypedParallel[T any](name string, steps []Step, inputKey Key[T], outputKey Key[[]T]) *TypedParallel[T] {
	return &TypedParallel[T]{
		name:      name,
		steps:     steps,
		inputKey:  inputKey,
		outputKey: outputKey,
	}
}

// Name returns the parallel workflow name.
func (p *TypedParallel[T]) Name() string { return p.name }

// Run executes steps concurrently and collects results.
func (p *TypedParallel[T]) Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error) {
	inner := NewParallel(p.name, p.steps, CollectInto(p.inputKey, p.outputKey))
	return inner.Run(ctx, state, opts...)
}

// RunStream executes steps concurrently with streaming events.
func (p *TypedParallel[T]) RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event {
	inner := NewParallel(p.name, p.steps, CollectInto(p.inputKey, p.outputKey))
	return inner.RunStream(ctx, state, opts...)
}

// TypedAggregator combines typed results from parallel branches.
type TypedAggregator[T, U any] func(results []T) U

// TypedParallelWithAggregator executes steps concurrently with custom typed aggregation.
type TypedParallelWithAggregator[T, U any] struct {
	name       string
	steps      []Step
	inputKey   Key[T]
	aggregator TypedAggregator[T, U]
	outputKey  Key[U]
}

// NewTypedParallelWithAggregator creates a parallel workflow with custom typed aggregation.
// Each branch writes to inputKey; the aggregator combines all T values into U;
// the result is stored in outputKey.
//
// Example:
//
//	var KeyScore = workflow.NewKey[int]("score")
//	var KeyTotal = workflow.NewKey[int]("total")
//
//	parallel := workflow.NewTypedParallelWithAggregator(
//	    "sum-scores",
//	    scoreSteps,
//	    KeyScore,
//	    func(scores []int) int {
//	        sum := 0
//	        for _, s := range scores { sum += s }
//	        return sum
//	    },
//	    KeyTotal,
//	)
func NewTypedParallelWithAggregator[T, U any](
	name string,
	steps []Step,
	inputKey Key[T],
	aggregator TypedAggregator[T, U],
	outputKey Key[U],
) *TypedParallelWithAggregator[T, U] {
	return &TypedParallelWithAggregator[T, U]{
		name:       name,
		steps:      steps,
		inputKey:   inputKey,
		aggregator: aggregator,
		outputKey:  outputKey,
	}
}

// Name returns the parallel workflow name.
func (p *TypedParallelWithAggregator[T, U]) Name() string { return p.name }

// Run executes steps concurrently and applies the typed aggregator.
func (p *TypedParallelWithAggregator[T, U]) Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error) {
	aggregator := func(state *State, results map[string]*StepResult, errors map[string]error) error {
		var collected []T
		for _, result := range results {
			if val, ok := GetFromBranch(result, p.inputKey); ok {
				collected = append(collected, val)
			}
		}
		combined := p.aggregator(collected)
		Set(state, p.outputKey, combined)
		return nil
	}

	inner := NewParallel(p.name, p.steps, aggregator)
	return inner.Run(ctx, state, opts...)
}

// RunStream executes steps concurrently with streaming events.
func (p *TypedParallelWithAggregator[T, U]) RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event {
	aggregator := func(state *State, results map[string]*StepResult, errors map[string]error) error {
		var collected []T
		for _, result := range results {
			if val, ok := GetFromBranch(result, p.inputKey); ok {
				collected = append(collected, val)
			}
		}
		combined := p.aggregator(collected)
		Set(state, p.outputKey, combined)
		return nil
	}

	inner := NewParallel(p.name, p.steps, aggregator)
	return inner.RunStream(ctx, state, opts...)
}
