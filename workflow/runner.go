package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/spetersoncode/gains/event"
)

// Runner is a type-erased interface for executing workflows.
// It allows workflows with different state types to be stored and executed
// uniformly without knowing the concrete state type at compile time.
//
// This interface is designed for AG-UI server integration where workflows
// need to be dispatched by name with untyped input.
type Runner interface {
	// Name returns the workflow's unique identifier.
	Name() string

	// RunStream executes the workflow with the given input state and returns an event stream.
	// The input is decoded into the workflow's state type.
	// The state parameter may be nil if no initial state is provided.
	RunStream(ctx context.Context, state any, opts ...Option) <-chan Event
}

// RunnerFunc wraps a Step[S] as a Runner using a state factory function.
// The factory creates a new state instance and optionally initializes it from input.
type RunnerFunc[S any] struct {
	name    string
	step    Step[S]
	factory func(input any) (*S, error)
}

// NewRunner creates a Runner from a Step[S] and a state factory.
//
// The factory function receives the untyped input (typically map[string]any from JSON)
// and should return an initialized state pointer. If input is nil, the factory
// should return a zero-initialized state.
//
// Example:
//
//	type MyState struct {
//	    Query   string `json:"query"`
//	    Results []string
//	}
//
//	runner := workflow.NewRunner("search", myWorkflow, func(input any) (*MyState, error) {
//	    state := &MyState{}
//	    if input != nil {
//	        data, _ := json.Marshal(input)
//	        json.Unmarshal(data, state)
//	    }
//	    return state, nil
//	})
func NewRunner[S any](name string, step Step[S], factory func(input any) (*S, error)) Runner {
	return &RunnerFunc[S]{
		name:    name,
		step:    step,
		factory: factory,
	}
}

// NewRunnerJSON creates a Runner that decodes JSON input into the state type.
// This is a convenience function for the common case of JSON-encoded input.
//
// Example:
//
//	type MyState struct {
//	    Query string `json:"query"`
//	}
//
//	runner := workflow.NewRunnerJSON[MyState]("search", myWorkflow)
func NewRunnerJSON[S any](name string, step Step[S]) Runner {
	return NewRunner(name, step, func(input any) (*S, error) {
		state := new(S)
		if input == nil {
			return state, nil
		}

		// If input is already the correct type, use it directly
		if typed, ok := input.(*S); ok {
			return typed, nil
		}

		// Otherwise, marshal to JSON and unmarshal to the target type
		data, err := json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal input: %w", err)
		}

		if err := json.Unmarshal(data, state); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input to state: %w", err)
		}

		return state, nil
	})
}

// Name returns the workflow name.
func (r *RunnerFunc[S]) Name() string {
	return r.name
}

// RunStream executes the workflow and returns an event stream.
func (r *RunnerFunc[S]) RunStream(ctx context.Context, input any, opts ...Option) <-chan Event {
	ch := make(chan event.Event, 100)

	go func() {
		defer close(ch)

		// Create state from input
		state, err := r.factory(input)
		if err != nil {
			event.Emit(ch, Event{Type: event.RunError, Error: err})
			return
		}

		// Emit run start
		event.Emit(ch, Event{Type: event.RunStart})

		// Run the workflow
		for ev := range r.step.RunStream(ctx, state, opts...) {
			event.Emit(ch, ev)
		}

		// Emit run end
		event.Emit(ch, Event{Type: event.RunEnd})
	}()

	return ch
}

// Registry stores and retrieves Runners by name.
// It provides named workflow dispatch for AG-UI server integration.
type Registry struct {
	mu      sync.RWMutex
	runners map[string]Runner
}

// NewRegistry creates a new workflow registry.
func NewRegistry() *Registry {
	return &Registry{
		runners: make(map[string]Runner),
	}
}

// Register adds a Runner to the registry.
// If a runner with the same name already exists, it is replaced.
func (r *Registry) Register(runner Runner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runners[runner.Name()] = runner
}

// Get retrieves a Runner by name.
// Returns nil if no runner with the given name exists.
func (r *Registry) Get(name string) Runner {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.runners[name]
}

// Has returns true if a runner with the given name exists.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.runners[name]
	return ok
}

// Unregister removes a Runner from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.runners, name)
}

// Names returns all registered workflow names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.runners))
	for name := range r.runners {
		names = append(names, name)
	}
	return names
}

// Len returns the number of registered runners.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.runners)
}

// RunStream executes the named workflow and returns an event stream.
// Returns an error channel if the workflow is not found.
func (r *Registry) RunStream(ctx context.Context, name string, input any, opts ...Option) <-chan Event {
	runner := r.Get(name)
	if runner == nil {
		ch := make(chan event.Event, 1)
		event.Emit(ch, Event{
			Type:  event.RunError,
			Error: fmt.Errorf("workflow not found: %s", name),
		})
		close(ch)
		return ch
	}

	return runner.RunStream(ctx, input, opts...)
}
