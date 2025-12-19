package workflow

import (
	"context"
	"sync"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
)

// Aggregator combines results from parallel steps into the shared state.
// The errors map contains any step failures when ContinueOnError is true,
// giving full visibility into which steps succeeded and which failed.
type Aggregator func(state *State, results map[string]*StepResult, errors map[string]error) error

// Parallel executes steps concurrently and aggregates results.
type Parallel struct {
	name       string
	steps      []Step
	aggregator Aggregator
}

// NewParallel creates a parallel workflow.
// The aggregator is called with all results after all steps complete.
// If aggregator is nil, each branch's state changes are merged back.
func NewParallel(name string, steps []Step, aggregator Aggregator) *Parallel {
	return &Parallel{
		name:       name,
		steps:      steps,
		aggregator: aggregator,
	}
}

// Name returns the parallel workflow name.
func (p *Parallel) Name() string { return p.name }

// Run executes steps concurrently.
func (p *Parallel) Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error) {
	options := ApplyOptions(opts...)

	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	results := make(map[string]*StepResult)
	errors := make(map[string]error)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Semaphore for concurrency limiting
	var sem chan struct{}
	if options.MaxConcurrency > 0 {
		sem = make(chan struct{}, options.MaxConcurrency)
	}

	for _, step := range p.steps {
		wg.Add(1)
		go func(s Step) {
			defer wg.Done()

			if sem != nil {
				sem <- struct{}{}
				defer func() { <-sem }()
			}

			// Each parallel branch gets a cloned state
			branchState := state.Clone()

			stepCtx := ctx
			if options.StepTimeout > 0 {
				var cancel context.CancelFunc
				stepCtx, cancel = context.WithTimeout(ctx, options.StepTimeout)
				defer cancel()
			}

			result, err := s.Run(stepCtx, branchState, opts...)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errors[s.Name()] = err
			} else {
				// Store the branch state in metadata for potential merging
				if result.Metadata == nil {
					result.Metadata = make(map[string]any)
				}
				result.Metadata["branch_state"] = branchState
				results[s.Name()] = result
			}
		}(step)
	}

	wg.Wait()

	// Handle errors
	if len(errors) > 0 && !options.ContinueOnError {
		return nil, &ParallelError{Errors: errors}
	}

	// Aggregate results
	if p.aggregator != nil {
		if err := p.aggregator(state, results, errors); err != nil {
			return nil, err
		}
	} else {
		// Default: merge all branch states back
		for _, result := range results {
			if branchState, ok := result.Metadata["branch_state"].(*State); ok {
				state.Merge(branchState)
			}
		}
	}

	// Calculate total usage
	var totalUsage ai.Usage
	for _, result := range results {
		totalUsage.InputTokens += result.Usage.InputTokens
		totalUsage.OutputTokens += result.Usage.OutputTokens
	}

	return &StepResult{
		StepName: p.name,
		Usage:    totalUsage,
		Metadata: map[string]any{"parallel_results": results},
	}, nil
}

// RunStream executes steps concurrently and emits events.
func (p *Parallel) RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		options := ApplyOptions(opts...)

		if options.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, options.Timeout)
			defer cancel()
		}

		event.Emit(ch, Event{Type: event.ParallelStart, StepName: p.name})

		results := make(map[string]*StepResult)
		errors := make(map[string]error)
		branchStates := make(map[string]*State)
		var mu sync.Mutex
		var wg sync.WaitGroup

		// Create a merged event channel
		eventCh := make(chan Event, len(p.steps)*100)

		// Semaphore for concurrency limiting
		var sem chan struct{}
		if options.MaxConcurrency > 0 {
			sem = make(chan struct{}, options.MaxConcurrency)
		}

		for _, step := range p.steps {
			wg.Add(1)
			go func(s Step) {
				defer wg.Done()

				if sem != nil {
					sem <- struct{}{}
					defer func() { <-sem }()
				}

				branchState := state.Clone()
				stepEvents := s.RunStream(ctx, branchState, opts...)

				for ev := range stepEvents {
					mu.Lock()
					if ev.Type == event.StepEnd && ev.Response != nil {
						results[s.Name()] = &StepResult{
							StepName: s.Name(),
							Response: ev.Response,
							Usage:    ev.Response.Usage,
						}
						branchStates[s.Name()] = branchState
					}
					if ev.Type == event.RunError {
						errors[s.Name()] = ev.Error
						// In ContinueOnError mode, emit StepSkipped instead of RunError
						if options.ContinueOnError {
							eventCh <- Event{
								Type:     event.StepSkipped,
								StepName: s.Name(),
								Error:    ev.Error,
								Message:  "step failed, continuing",
							}
							mu.Unlock()
							continue
						}
					}
					mu.Unlock()
					eventCh <- ev
				}
			}(step)
		}

		// Wait for all steps and close event channel
		go func() {
			wg.Wait()
			close(eventCh)
		}()

		// Forward all events
		for ev := range eventCh {
			ch <- ev
		}

		// Handle errors
		if len(errors) > 0 && !options.ContinueOnError {
			event.Emit(ch, Event{Type: event.RunError, StepName: p.name, Error: &ParallelError{Errors: errors}})
			return
		}

		// Aggregate
		if p.aggregator != nil {
			if err := p.aggregator(state, results, errors); err != nil {
				event.Emit(ch, Event{Type: event.RunError, StepName: p.name, Error: err})
				return
			}
		} else {
			for name := range results {
				if branchState, ok := branchStates[name]; ok {
					state.Merge(branchState)
				}
			}
		}

		// Calculate total usage
		var totalUsage ai.Usage
		for _, result := range results {
			totalUsage.InputTokens += result.Usage.InputTokens
			totalUsage.OutputTokens += result.Usage.OutputTokens
		}

		event.Emit(ch, Event{
			Type:     event.ParallelEnd,
			StepName: p.name,
		})
	}()

	return ch
}
