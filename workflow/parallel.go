package workflow

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/spetersoncode/gains/event"
)

// Aggregator combines results from parallel steps into the shared state.
// Each branch runs with a deep copy of state; aggregator merges branch states back.
// The errors map contains any step failures when ContinueOnError is true.
type Aggregator[S any] func(state *S, branches map[string]*S, errors map[string]error) error

// Parallel executes steps concurrently and aggregates results.
type Parallel[S any] struct {
	name       string
	steps      []Step[S]
	aggregator Aggregator[S]
}

// NewParallel creates a parallel workflow.
// The aggregator is called with all results after all steps complete.
// If aggregator is nil, no automatic merging occurs (user handles via aggregator).
func NewParallel[S any](name string, steps []Step[S], aggregator Aggregator[S]) *Parallel[S] {
	return &Parallel[S]{
		name:       name,
		steps:      steps,
		aggregator: aggregator,
	}
}

// DeepClone creates a deep copy of a struct using JSON serialization.
// This is safe for concurrent use and handles nested structures.
// For performance-critical code, implement a custom clone method.
func DeepClone[S any](src *S) (*S, error) {
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	var dst S
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil, err
	}
	return &dst, nil
}

// Name returns the parallel workflow name.
func (p *Parallel[S]) Name() string { return p.name }

// Run executes steps concurrently.
func (p *Parallel[S]) Run(ctx context.Context, state *S, opts ...Option) error {
	options := ApplyOptions(opts...)

	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	branches := make(map[string]*S)
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
		go func(s Step[S]) {
			defer wg.Done()

			if sem != nil {
				sem <- struct{}{}
				defer func() { <-sem }()
			}

			// Each parallel branch gets a deep-cloned state
			branchState, err := DeepClone(state)
			if err != nil {
				mu.Lock()
				errors[s.Name()] = &StepError{StepName: s.Name(), Err: err}
				mu.Unlock()
				return
			}

			stepCtx := ctx
			if options.StepTimeout > 0 {
				var cancel context.CancelFunc
				stepCtx, cancel = context.WithTimeout(ctx, options.StepTimeout)
				defer cancel()
			}

			err = s.Run(stepCtx, branchState, opts...)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errors[s.Name()] = err
			} else {
				branches[s.Name()] = branchState
			}
		}(step)
	}

	wg.Wait()

	// Handle errors
	if len(errors) > 0 && !options.ContinueOnError {
		return &ParallelError{Errors: errors}
	}

	// Aggregate results
	if p.aggregator != nil {
		if err := p.aggregator(state, branches, errors); err != nil {
			return err
		}
	}

	return nil
}

// RunStream executes steps concurrently and emits events.
func (p *Parallel[S]) RunStream(ctx context.Context, state *S, opts ...Option) <-chan Event {
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

		branches := make(map[string]*S)
		errors := make(map[string]error)
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
			go func(s Step[S]) {
				defer wg.Done()

				if sem != nil {
					sem <- struct{}{}
					defer func() { <-sem }()
				}

				// Deep clone state for this branch
				branchState, err := DeepClone(state)
				if err != nil {
					mu.Lock()
					errors[s.Name()] = &StepError{StepName: s.Name(), Err: err}
					mu.Unlock()
					eventCh <- Event{Type: event.RunError, StepName: s.Name(), Error: err}
					return
				}

				stepEvents := s.RunStream(ctx, branchState, opts...)

				for ev := range stepEvents {
					mu.Lock()
					if ev.Type == event.StepEnd {
						branches[s.Name()] = branchState
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
			if err := p.aggregator(state, branches, errors); err != nil {
				event.Emit(ch, Event{Type: event.RunError, StepName: p.name, Error: err})
				return
			}
		}

		event.Emit(ch, Event{
			Type:     event.ParallelEnd,
			StepName: p.name,
		})
	}()

	return ch
}
