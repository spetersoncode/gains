package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/chat"
	"github.com/spetersoncode/gains/event"
)

// Step represents a single unit of work in a workflow.
// Steps can be functions, LLM calls, or nested workflows.
type Step interface {
	// Name returns a unique identifier for the step.
	Name() string

	// Run executes the step and returns the result.
	Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error)

	// RunStream executes the step and returns a channel of events.
	RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event
}

// StepFunc is a function signature for simple step implementations.
type StepFunc func(ctx context.Context, state *State) error

// FuncStep wraps a function as a Step.
type FuncStep struct {
	name string
	fn   StepFunc
}

// NewFuncStep creates a step from a function.
func NewFuncStep(name string, fn StepFunc) *FuncStep {
	return &FuncStep{name: name, fn: fn}
}

// Name returns the step name.
func (f *FuncStep) Name() string { return f.name }

// Run executes the function.
func (f *FuncStep) Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error) {
	err := f.fn(ctx, state)
	if err != nil {
		return nil, err
	}
	return &StepResult{
		StepName: f.name,
	}, nil
}

// RunStream executes the function and emits events.
func (f *FuncStep) RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event {
	ch := make(chan Event, 10)
	go func() {
		defer close(ch)
		event.Emit(ch, Event{Type: event.StepStart, StepName: f.name})

		err := f.fn(ctx, state)
		if err != nil {
			event.Emit(ch, Event{Type: event.RunError, StepName: f.name, Error: err})
			return
		}

		event.Emit(ch, Event{
			Type:     event.StepEnd,
			StepName: f.name,
		})
	}()
	return ch
}

// PromptFunc generates messages from state for an LLM call.
type PromptFunc func(state *State) []ai.Message

// PromptStep makes a single LLM call with a dynamic prompt.
type PromptStep struct {
	name       string
	chatClient chat.Client
	prompt     PromptFunc
	outputKey  string
	chatOpts   []ai.Option
}

// NewPromptStep creates a step for a single LLM call.
// The prompt function generates messages from current state.
// If outputKey is non-empty, the response content is stored in state under that key.
func NewPromptStep(name string, c chat.Client, prompt PromptFunc, outputKey string, opts ...ai.Option) *PromptStep {
	return &PromptStep{
		name:       name,
		chatClient: c,
		prompt:     prompt,
		outputKey:  outputKey,
		chatOpts:   opts,
	}
}

// Name returns the step name.
func (p *PromptStep) Name() string { return p.name }

// Run executes the LLM call.
func (p *PromptStep) Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error) {
	options := ApplyOptions(opts...)

	// Merge chat options
	chatOpts := make([]ai.Option, 0, len(p.chatOpts)+len(options.ChatOptions))
	chatOpts = append(chatOpts, p.chatOpts...)
	chatOpts = append(chatOpts, options.ChatOptions...)

	msgs := p.prompt(state)
	resp, err := p.chatClient.Chat(ctx, msgs, chatOpts...)
	if err != nil {
		return nil, err
	}

	if p.outputKey != "" {
		state.Set(p.outputKey, resp.Content)
	}

	return &StepResult{
		StepName: p.name,
		Output:   resp.Content,
		Response: resp,
		Usage:    resp.Usage,
	}, nil
}

// RunStream executes the LLM call with streaming.
func (p *PromptStep) RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		event.Emit(ch, Event{Type: event.StepStart, StepName: p.name})

		options := ApplyOptions(opts...)

		// Merge chat options
		chatOpts := make([]ai.Option, 0, len(p.chatOpts)+len(options.ChatOptions))
		chatOpts = append(chatOpts, p.chatOpts...)
		chatOpts = append(chatOpts, options.ChatOptions...)

		msgs := p.prompt(state)
		streamCh, err := p.chatClient.ChatStream(ctx, msgs, chatOpts...)
		if err != nil {
			event.Emit(ch, Event{Type: event.RunError, StepName: p.name, Error: err})
			return
		}

		var response *ai.Response
		for ev := range streamCh {
			switch ev.Type {
			case event.RunError:
				event.Emit(ch, Event{Type: event.RunError, StepName: p.name, Error: ev.Error})
				return
			case event.MessageStart:
				event.Emit(ch, Event{Type: event.MessageStart, StepName: p.name, MessageID: ev.MessageID})
			case event.MessageDelta:
				event.Emit(ch, Event{Type: event.MessageDelta, StepName: p.name, MessageID: ev.MessageID, Delta: ev.Delta})
			case event.MessageEnd:
				event.Emit(ch, Event{Type: event.MessageEnd, StepName: p.name, MessageID: ev.MessageID, Response: ev.Response})
				response = ev.Response
			}
		}

		if response != nil {
			if p.outputKey != "" {
				state.Set(p.outputKey, response.Content)
			}
			event.Emit(ch, Event{
				Type:     event.StepEnd,
				StepName: p.name,
				Response: response,
			})
		}
	}()

	return ch
}

// TypedPromptStep makes an LLM call with structured output that is automatically
// unmarshaled into type T and stored in state.
type TypedPromptStep[T any] struct {
	name       string
	chatClient chat.Client
	prompt     PromptFunc
	outputKey  string
	chatOpts   []ai.Option
	schema     ai.ResponseSchema
}

// NewTypedPromptStep creates a step that returns structured output of type T.
// The schema parameter defines the JSON schema for the response.
// The unmarshaled *T is stored in state under outputKey.
//
// Example:
//
//	type Analysis struct {
//	    Sentiment  string   `json:"sentiment"`
//	    Keywords   []string `json:"keywords"`
//	}
//
//	analysisSchema := ai.ResponseSchema{
//	    Name: "analysis",
//	    Schema: schema.Object().
//	        Field("sentiment", schema.String().Required()).
//	        Field("keywords", schema.Array(schema.String()).Required()).
//	        MustBuild(),
//	}
//
//	step := workflow.NewTypedPromptStep[Analysis](
//	    "analyze",
//	    client,
//	    func(s *State) []ai.Message {
//	        return []ai.Message{{Role: ai.RoleUser, Content: s.GetString("text")}}
//	    },
//	    analysisSchema,
//	    "analysis",
//	    ai.WithModel(model.Claude35Sonnet),
//	)
//
//	// After execution, state.Get("analysis") returns *Analysis
func NewTypedPromptStep[T any](
	name string,
	c chat.Client,
	prompt PromptFunc,
	schema ai.ResponseSchema,
	outputKey string,
	opts ...ai.Option,
) *TypedPromptStep[T] {
	return &TypedPromptStep[T]{
		name:       name,
		chatClient: c,
		prompt:     prompt,
		outputKey:  outputKey,
		chatOpts:   opts,
		schema:     schema,
	}
}

// Name returns the step name.
func (p *TypedPromptStep[T]) Name() string { return p.name }

// Run executes the LLM call and unmarshals the response into T.
func (p *TypedPromptStep[T]) Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error) {
	options := ApplyOptions(opts...)

	// Merge chat options, adding our response schema
	chatOpts := make([]ai.Option, 0, len(p.chatOpts)+len(options.ChatOptions)+1)
	chatOpts = append(chatOpts, p.chatOpts...)
	chatOpts = append(chatOpts, options.ChatOptions...)
	chatOpts = append(chatOpts, ai.WithResponseSchema(p.schema))

	msgs := p.prompt(state)
	resp, err := p.chatClient.Chat(ctx, msgs, chatOpts...)
	if err != nil {
		return nil, err
	}

	// Unmarshal response content into T
	var result T
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		return nil, &ai.UnmarshalError{
			Context:    fmt.Sprintf("workflow: step %q", p.name),
			Content:    resp.Content,
			TargetType: fmt.Sprintf("%T", result),
			Err:        err,
		}
	}

	// Store pointer to result in state
	if p.outputKey != "" {
		state.Set(p.outputKey, &result)
	}

	return &StepResult{
		StepName: p.name,
		Output:   &result,
		Response: resp,
		Usage:    resp.Usage,
	}, nil
}

// RunStream executes the LLM call with streaming and unmarshals the final response.
func (p *TypedPromptStep[T]) RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		event.Emit(ch, Event{Type: event.StepStart, StepName: p.name})

		options := ApplyOptions(opts...)

		// Merge chat options, adding our response schema
		chatOpts := make([]ai.Option, 0, len(p.chatOpts)+len(options.ChatOptions)+1)
		chatOpts = append(chatOpts, p.chatOpts...)
		chatOpts = append(chatOpts, options.ChatOptions...)
		chatOpts = append(chatOpts, ai.WithResponseSchema(p.schema))

		msgs := p.prompt(state)
		streamCh, err := p.chatClient.ChatStream(ctx, msgs, chatOpts...)
		if err != nil {
			event.Emit(ch, Event{Type: event.RunError, StepName: p.name, Error: err})
			return
		}

		var response *ai.Response
		for ev := range streamCh {
			switch ev.Type {
			case event.RunError:
				event.Emit(ch, Event{Type: event.RunError, StepName: p.name, Error: ev.Error})
				return
			case event.MessageStart:
				event.Emit(ch, Event{Type: event.MessageStart, StepName: p.name, MessageID: ev.MessageID})
			case event.MessageDelta:
				event.Emit(ch, Event{Type: event.MessageDelta, StepName: p.name, MessageID: ev.MessageID, Delta: ev.Delta})
			case event.MessageEnd:
				event.Emit(ch, Event{Type: event.MessageEnd, StepName: p.name, MessageID: ev.MessageID, Response: ev.Response})
				response = ev.Response
			}
		}

		if response != nil {
			var result T
			if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
				event.Emit(ch, Event{
					Type:     event.RunError,
					StepName: p.name,
					Error: &ai.UnmarshalError{
						Context:    fmt.Sprintf("workflow: step %q", p.name),
						Content:    response.Content,
						TargetType: fmt.Sprintf("%T", result),
						Err:        err,
					},
				})
				return
			}

			if p.outputKey != "" {
				state.Set(p.outputKey, &result)
			}

			event.Emit(ch, Event{
				Type:     event.StepEnd,
				StepName: p.name,
				Response: response,
			})
		}
	}()

	return ch
}

// NewTypedPromptStepWithKey creates a step that stores output using a typed key.
// This provides stronger type guarantees than the string-based version.
// The key type must be a pointer (*T) since TypedPromptStep stores pointers.
//
// Example:
//
//	var KeyAnalysis = workflow.NewKey[*SentimentAnalysis]("analysis")
//
//	step := workflow.NewTypedPromptStepWithKey(
//	    "analyze",
//	    client,
//	    promptFunc,
//	    sentimentSchema,
//	    KeyAnalysis,
//	)
func NewTypedPromptStepWithKey[T any](
	name string,
	c chat.Client,
	prompt PromptFunc,
	schema ai.ResponseSchema,
	outputKey Key[*T],
	opts ...ai.Option,
) *TypedPromptStep[T] {
	return &TypedPromptStep[T]{
		name:       name,
		chatClient: c,
		prompt:     prompt,
		outputKey:  outputKey.Name(),
		chatOpts:   opts,
		schema:     schema,
	}
}
