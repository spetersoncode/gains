package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ai "github.com/spetersoncode/gains"
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
	name       string
	chatClient ChatClient
	prompt     PromptFunc
	routes     map[string]Step
	chatOpts   []ai.Option
}

// NewClassifierRouter creates a router that uses LLM classification.
// The LLM response should match one of the route keys (case-insensitive).
// For more reliable classification, use ClassifierSchema().
func NewClassifierRouter(
	name string,
	c ChatClient,
	prompt PromptFunc,
	routes map[string]Step,
	opts ...ai.Option,
) *ClassifierRouter {
	return &ClassifierRouter{
		name:       name,
		chatClient: c,
		prompt:     prompt,
		routes:     routes,
		chatOpts:   opts,
	}
}

// ClassifierSchema returns a ai.Option that enforces structured output
// for classification. Use with providers that support JSON schema (OpenAI, Anthropic).
// Note: May not work with streaming on all providers.
func ClassifierSchema(routes map[string]Step) ai.Option {
	routeKeys := make([]any, 0, len(routes))
	for key := range routes {
		routeKeys = append(routeKeys, key)
	}

	schemaMap := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"classification": map[string]any{
				"type": "string",
				"enum": routeKeys,
			},
		},
		"required": []string{"classification"},
	}
	schemaJSON, _ := json.Marshal(schemaMap)

	return ai.WithResponseSchema(ai.ResponseSchema{
		Name:        "classification",
		Description: "Classification result",
		Schema:      schemaJSON,
	})
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
	chatOpts := make([]ai.Option, 0, len(c.chatOpts)+len(options.ChatOptions))
	chatOpts = append(chatOpts, c.chatOpts...)
	chatOpts = append(chatOpts, options.ChatOptions...)

	// Get classification from LLM
	msgs := c.prompt(state)
	resp, err := c.chatClient.Chat(ctx, msgs, chatOpts...)
	if err != nil {
		return nil, &StepError{StepName: c.name, Err: err}
	}

	classification, err := extractClassification(resp.Content)
	if err != nil {
		return nil, &StepError{StepName: c.name, Err: err}
	}
	state.Set(c.name+"_classification", classification)

	// Find matching route (case-insensitive)
	var selectedStep Step
	var matchedRoute string
	for routeName, step := range c.routes {
		if strings.EqualFold(routeName, classification) {
			selectedStep = step
			matchedRoute = routeName
			break
		}
	}
	if selectedStep == nil {
		return nil, &StepError{
			StepName: c.name,
			Err:      fmt.Errorf("unknown classification: %q", classification),
		}
	}

	state.Set(c.name+"_route", matchedRoute)

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
		chatOpts := make([]ai.Option, 0, len(c.chatOpts)+len(options.ChatOptions))
		chatOpts = append(chatOpts, c.chatOpts...)
		chatOpts = append(chatOpts, options.ChatOptions...)

		// Get classification with streaming
		msgs := c.prompt(state)
		streamCh, err := c.chatClient.ChatStream(ctx, msgs, chatOpts...)
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
				var err error
				classification, err = extractClassification(event.Response.Content)
				if err != nil {
					emit(ch, Event{Type: EventError, StepName: c.name, Error: err})
					return
				}
			}
		}

		state.Set(c.name+"_classification", classification)

		// Find matching route (case-insensitive)
		var selectedStep Step
		var matchedRoute string
		for routeName, step := range c.routes {
			if strings.EqualFold(routeName, classification) {
				selectedStep = step
				matchedRoute = routeName
				break
			}
		}
		if selectedStep == nil {
			emit(ch, Event{
				Type:     EventError,
				StepName: c.name,
				Error:    fmt.Errorf("unknown classification: %q", classification),
			})
			return
		}

		emit(ch, Event{
			Type:      EventRouteSelected,
			StepName:  c.name,
			RouteName: matchedRoute,
		})

		state.Set(c.name+"_route", matchedRoute)

		// Forward events from selected step
		stepEvents := selectedStep.RunStream(ctx, state, opts...)
		for event := range stepEvents {
			ch <- event
		}
	}()

	return ch
}

// extractClassification parses the LLM response to get the classification.
// Handles both JSON structured output and plain text responses.
func extractClassification(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("empty classification response")
	}

	// Try to parse as JSON first (structured output)
	if strings.HasPrefix(content, "{") {
		var result struct {
			Classification string `json:"classification"`
		}
		if err := json.Unmarshal([]byte(content), &result); err == nil && result.Classification != "" {
			return strings.ToLower(result.Classification), nil
		}
	}

	// Fall back to plain text parsing
	classification := strings.ToLower(content)
	// Strip trailing punctuation that models sometimes add
	classification = strings.TrimRight(classification, ".,!?;:")
	return classification, nil
}

// RouteKey returns a typed key for the selected route name.
// The key name follows the pattern "{routerName}_route".
func (r *Router) RouteKey() Key[string] {
	return NewKey[string](r.name + "_route")
}

// RouteKey returns a typed key for the selected route name.
// The key name follows the pattern "{routerName}_route".
func (c *ClassifierRouter) RouteKey() Key[string] {
	return NewKey[string](c.name + "_route")
}

// ClassificationKey returns a typed key for the raw classification result.
// The key name follows the pattern "{routerName}_classification".
func (c *ClassifierRouter) ClassificationKey() Key[string] {
	return NewKey[string](c.name + "_classification")
}

// ConditionEquals returns a condition that matches when the key equals value.
func ConditionEquals[T comparable](key Key[T], value T) Condition {
	return func(ctx context.Context, state *State) bool {
		v, ok := Get(state, key)
		return ok && v == value
	}
}

// ConditionSet returns a condition that matches when the key exists in state.
func ConditionSet[T any](key Key[T]) Condition {
	return func(ctx context.Context, state *State) bool {
		return Has(state, key)
	}
}

// ConditionMatches returns a condition using a predicate function on the key value.
func ConditionMatches[T any](key Key[T], pred func(T) bool) Condition {
	return func(ctx context.Context, state *State) bool {
		v, ok := Get(state, key)
		return ok && pred(v)
	}
}
