package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/spetersoncode/gains"
)

// Condition determines if a route should be taken.
type Condition func(ctx context.Context, state *State) bool

// Route represents a conditional path in a router.
type Route struct {
	Name      string
	Condition Condition
	Step      Step
}

// Router selects and executes a step based on conditions.
type Router struct {
	name         string
	routes       []Route
	defaultRoute Step
}

// NewRouter creates a conditional router.
// Routes are evaluated in order; first match wins.
// Default route is used if no conditions match (can be nil).
func NewRouter(name string, routes []Route, defaultRoute Step) *Router {
	return &Router{
		name:         name,
		routes:       routes,
		defaultRoute: defaultRoute,
	}
}

// Name returns the router name.
func (r *Router) Name() string { return r.name }

// Run evaluates conditions and executes the matching step.
func (r *Router) Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error) {
	options := ApplyOptions(opts...)

	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// Find matching route
	var selectedStep Step
	var selectedName string

	for _, route := range r.routes {
		if route.Condition(ctx, state) {
			selectedStep = route.Step
			selectedName = route.Name
			break
		}
	}

	if selectedStep == nil {
		if r.defaultRoute != nil {
			selectedStep = r.defaultRoute
			selectedName = "default"
		} else {
			return nil, ErrNoRouteMatched
		}
	}

	// Store selected route in state
	state.Set(r.name+"_route", selectedName)

	return selectedStep.Run(ctx, state, opts...)
}

// RunStream evaluates conditions and streams the matching step's events.
func (r *Router) RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		options := ApplyOptions(opts...)

		if options.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, options.Timeout)
			defer cancel()
		}

		emit(ch, Event{Type: EventStepStart, StepName: r.name})

		// Find matching route
		var selectedStep Step
		var selectedName string

		for _, route := range r.routes {
			if route.Condition(ctx, state) {
				selectedStep = route.Step
				selectedName = route.Name
				break
			}
		}

		if selectedStep == nil {
			if r.defaultRoute != nil {
				selectedStep = r.defaultRoute
				selectedName = "default"
			} else {
				emit(ch, Event{Type: EventError, StepName: r.name, Error: ErrNoRouteMatched})
				return
			}
		}

		emit(ch, Event{
			Type:      EventRouteSelected,
			StepName:  r.name,
			RouteName: selectedName,
		})

		state.Set(r.name+"_route", selectedName)

		// Forward events from selected step
		stepEvents := selectedStep.RunStream(ctx, state, opts...)
		for event := range stepEvents {
			ch <- event
		}
	}()

	return ch
}

// ClassifierRouter uses an LLM to classify input and route accordingly.
type ClassifierRouter struct {
	name     string
	provider gains.ChatProvider
	prompt   PromptFunc
	routes   map[string]Step
	chatOpts []gains.Option
}

// NewClassifierRouter creates a router that uses LLM classification.
// The LLM response should match one of the route keys.
func NewClassifierRouter(
	name string,
	provider gains.ChatProvider,
	prompt PromptFunc,
	routes map[string]Step,
	opts ...gains.Option,
) *ClassifierRouter {
	return &ClassifierRouter{
		name:     name,
		provider: provider,
		prompt:   prompt,
		routes:   routes,
		chatOpts: opts,
	}
}

// Name returns the router name.
func (c *ClassifierRouter) Name() string { return c.name }

// Run classifies input and executes the matching route.
func (c *ClassifierRouter) Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error) {
	options := ApplyOptions(opts...)

	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// Merge chat options
	chatOpts := make([]gains.Option, 0, len(c.chatOpts)+len(options.ChatOptions))
	chatOpts = append(chatOpts, c.chatOpts...)
	chatOpts = append(chatOpts, options.ChatOptions...)

	// Get classification from LLM
	msgs := c.prompt(state)
	resp, err := c.provider.Chat(ctx, msgs, chatOpts...)
	if err != nil {
		return nil, &StepError{StepName: c.name, Err: err}
	}

	classification := strings.TrimSpace(resp.Content)
	state.Set(c.name+"_classification", classification)

	// Find matching route
	selectedStep, ok := c.routes[classification]
	if !ok {
		return nil, &StepError{
			StepName: c.name,
			Err:      fmt.Errorf("unknown classification: %s", classification),
		}
	}

	state.Set(c.name+"_route", classification)

	return selectedStep.Run(ctx, state, opts...)
}

// RunStream classifies input with streaming and executes the matching route.
func (c *ClassifierRouter) RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		options := ApplyOptions(opts...)

		if options.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, options.Timeout)
			defer cancel()
		}

		emit(ch, Event{Type: EventStepStart, StepName: c.name})

		// Merge chat options
		chatOpts := make([]gains.Option, 0, len(c.chatOpts)+len(options.ChatOptions))
		chatOpts = append(chatOpts, c.chatOpts...)
		chatOpts = append(chatOpts, options.ChatOptions...)

		// Get classification with streaming
		msgs := c.prompt(state)
		streamCh, err := c.provider.ChatStream(ctx, msgs, chatOpts...)
		if err != nil {
			emit(ch, Event{Type: EventError, StepName: c.name, Error: err})
			return
		}

		var classification string
		for event := range streamCh {
			if event.Err != nil {
				emit(ch, Event{Type: EventError, StepName: c.name, Error: event.Err})
				return
			}
			if event.Delta != "" {
				emit(ch, Event{Type: EventStreamDelta, StepName: c.name, Delta: event.Delta})
			}
			if event.Done && event.Response != nil {
				classification = strings.TrimSpace(event.Response.Content)
			}
		}

		state.Set(c.name+"_classification", classification)

		selectedStep, ok := c.routes[classification]
		if !ok {
			emit(ch, Event{
				Type:     EventError,
				StepName: c.name,
				Error:    fmt.Errorf("unknown classification: %s", classification),
			})
			return
		}

		emit(ch, Event{
			Type:      EventRouteSelected,
			StepName:  c.name,
			RouteName: classification,
		})

		state.Set(c.name+"_route", classification)

		// Forward events from selected step
		stepEvents := selectedStep.RunStream(ctx, state, opts...)
		for event := range stepEvents {
			ch <- event
		}
	}()

	return ch
}
