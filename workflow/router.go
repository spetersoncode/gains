package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/chat"
	"github.com/spetersoncode/gains/event"
)

// RouteResult can be stored in state to record which route was taken.
// Store this via a setter if you need to access route information.
type RouteResult struct {
	RouteName      string // Name of the route that was taken
	Classification string // For ClassifierRouter: the LLM classification
}

// Condition determines if a route should be taken.
type Condition[S any] func(ctx context.Context, state *S) bool

// Route represents a conditional path in a router.
type Route[S any] struct {
	Name      string
	Condition Condition[S]
	Step      Step[S]
}

// Router selects and executes a step based on conditions.
type Router[S any] struct {
	name         string
	routes       []Route[S]
	defaultRoute Step[S]
}

// NewRouter creates a conditional router.
// Routes are evaluated in order; first match wins.
// Default route is used if no conditions match (can be nil).
func NewRouter[S any](name string, routes []Route[S], defaultRoute Step[S]) *Router[S] {
	return &Router[S]{
		name:         name,
		routes:       routes,
		defaultRoute: defaultRoute,
	}
}

// Name returns the router name.
func (r *Router[S]) Name() string { return r.name }

// Run evaluates conditions and executes the matching step.
func (r *Router[S]) Run(ctx context.Context, state *S, opts ...Option) error {
	options := ApplyOptions(opts...)

	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// Find matching route
	var selectedStep Step[S]

	for _, route := range r.routes {
		if route.Condition(ctx, state) {
			selectedStep = route.Step
			break
		}
	}

	if selectedStep == nil {
		if r.defaultRoute != nil {
			selectedStep = r.defaultRoute
		} else {
			return ErrNoRouteMatched
		}
	}

	return selectedStep.Run(ctx, state, opts...)
}

// RunStream evaluates conditions and streams the matching step's events.
func (r *Router[S]) RunStream(ctx context.Context, state *S, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		options := ApplyOptions(opts...)

		if options.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, options.Timeout)
			defer cancel()
		}

		event.Emit(ch, Event{Type: event.StepStart, StepName: r.name})

		// Find matching route
		var selectedStep Step[S]
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
				event.Emit(ch, Event{Type: event.RunError, StepName: r.name, Error: ErrNoRouteMatched})
				return
			}
		}

		event.Emit(ch, Event{
			Type:      event.RouteSelected,
			StepName:  r.name,
			RouteName: selectedName,
		})

		// Forward events from selected step
		stepEvents := selectedStep.RunStream(ctx, state, opts...)
		for ev := range stepEvents {
			ch <- ev
		}
	}()

	return ch
}

// ClassifierRouter uses an LLM to classify input and route accordingly.
type ClassifierRouter[S any] struct {
	name       string
	chatClient chat.Client
	prompt     PromptFunc[S]
	routes     map[string]Step[S]
	chatOpts   []ai.Option
}

// NewClassifierRouter creates a router that uses LLM classification.
// The LLM response should match one of the route keys (case-insensitive).
// For more reliable classification, use ClassifierSchema().
func NewClassifierRouter[S any](
	name string,
	c chat.Client,
	prompt PromptFunc[S],
	routes map[string]Step[S],
	opts ...ai.Option,
) *ClassifierRouter[S] {
	return &ClassifierRouter[S]{
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
func ClassifierSchema[S any](routes map[string]Step[S]) ai.Option {
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
func (c *ClassifierRouter[S]) Name() string { return c.name }

// Run classifies input and executes the matching route.
func (c *ClassifierRouter[S]) Run(ctx context.Context, state *S, opts ...Option) error {
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
		return &StepError{StepName: c.name, Err: err}
	}

	classification, err := extractClassification(resp.Content)
	if err != nil {
		return &StepError{StepName: c.name, Err: err}
	}

	// Find matching route (case-insensitive)
	var selectedStep Step[S]
	for routeName, step := range c.routes {
		if strings.EqualFold(routeName, classification) {
			selectedStep = step
			break
		}
	}
	if selectedStep == nil {
		return &StepError{
			StepName: c.name,
			Err:      fmt.Errorf("unknown classification: %q", classification),
		}
	}

	return selectedStep.Run(ctx, state, opts...)
}

// RunStream classifies input with streaming and executes the matching route.
func (c *ClassifierRouter[S]) RunStream(ctx context.Context, state *S, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		options := ApplyOptions(opts...)

		if options.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, options.Timeout)
			defer cancel()
		}

		event.Emit(ch, Event{Type: event.StepStart, StepName: c.name})

		// Merge chat options
		chatOpts := make([]ai.Option, 0, len(c.chatOpts)+len(options.ChatOptions))
		chatOpts = append(chatOpts, c.chatOpts...)
		chatOpts = append(chatOpts, options.ChatOptions...)

		// Get classification with streaming
		msgs := c.prompt(state)
		streamCh, err := c.chatClient.ChatStream(ctx, msgs, chatOpts...)
		if err != nil {
			event.Emit(ch, Event{Type: event.RunError, StepName: c.name, Error: err})
			return
		}

		var classification string
		for ev := range streamCh {
			switch ev.Type {
			case event.RunError:
				event.Emit(ch, Event{Type: event.RunError, StepName: c.name, Error: ev.Error})
				return
			case event.MessageDelta:
				event.Emit(ch, Event{Type: event.MessageDelta, StepName: c.name, Delta: ev.Delta})
			case event.MessageEnd:
				if ev.Response != nil {
					var err error
					classification, err = extractClassification(ev.Response.Content)
					if err != nil {
						event.Emit(ch, Event{Type: event.RunError, StepName: c.name, Error: err})
						return
					}
				}
			}
		}

		// Find matching route (case-insensitive)
		var selectedStep Step[S]
		var matchedRoute string
		for routeName, step := range c.routes {
			if strings.EqualFold(routeName, classification) {
				selectedStep = step
				matchedRoute = routeName
				break
			}
		}
		if selectedStep == nil {
			event.Emit(ch, Event{
				Type:     event.RunError,
				StepName: c.name,
				Error:    fmt.Errorf("unknown classification: %q", classification),
			})
			return
		}

		event.Emit(ch, Event{
			Type:      event.RouteSelected,
			StepName:  c.name,
			RouteName: matchedRoute,
		})

		// Forward events from selected step
		stepEvents := selectedStep.RunStream(ctx, state, opts...)
		for ev := range stepEvents {
			ch <- ev
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
