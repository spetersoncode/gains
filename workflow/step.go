package workflow

import (
	"context"

	ai "github.com/spetersoncode/gains"
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
		emit(ch, Event{Type: EventStepStart, StepName: f.name})

		err := f.fn(ctx, state)
		if err != nil {
			emit(ch, Event{Type: EventError, StepName: f.name, Error: err})
			return
		}

		emit(ch, Event{
			Type:     EventStepComplete,
			StepName: f.name,
			Result:   &StepResult{StepName: f.name},
		})
	}()
	return ch
}

// PromptFunc generates messages from state for an LLM call.
type PromptFunc func(state *State) []ai.Message

// PromptStep makes a single LLM call with a dynamic prompt.
type PromptStep struct {
	name      string
	provider  ai.ChatProvider
	prompt    PromptFunc
	outputKey string
	chatOpts  []ai.Option
}

// NewPromptStep creates a step for a single LLM call.
// The prompt function generates messages from current state.
// If outputKey is non-empty, the response content is stored in state under that key.
func NewPromptStep(name string, provider ai.ChatProvider, prompt PromptFunc, outputKey string, opts ...ai.Option) *PromptStep {
	return &PromptStep{
		name:      name,
		provider:  provider,
		prompt:    prompt,
		outputKey: outputKey,
		chatOpts:  opts,
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
	resp, err := p.provider.Chat(ctx, msgs, chatOpts...)
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
		emit(ch, Event{Type: EventStepStart, StepName: p.name})

		options := ApplyOptions(opts...)

		// Merge chat options
		chatOpts := make([]ai.Option, 0, len(p.chatOpts)+len(options.ChatOptions))
		chatOpts = append(chatOpts, p.chatOpts...)
		chatOpts = append(chatOpts, options.ChatOptions...)

		msgs := p.prompt(state)
		streamCh, err := p.provider.ChatStream(ctx, msgs, chatOpts...)
		if err != nil {
			emit(ch, Event{Type: EventError, StepName: p.name, Error: err})
			return
		}

		var response *ai.Response
		for event := range streamCh {
			if event.Err != nil {
				emit(ch, Event{Type: EventError, StepName: p.name, Error: event.Err})
				return
			}
			if event.Delta != "" {
				emit(ch, Event{Type: EventStreamDelta, StepName: p.name, Delta: event.Delta})
			}
			if event.Done && event.Response != nil {
				response = event.Response
			}
		}

		if response != nil {
			if p.outputKey != "" {
				state.Set(p.outputKey, response.Content)
			}
			emit(ch, Event{
				Type:     EventStepComplete,
				StepName: p.name,
				Result: &StepResult{
					StepName: p.name,
					Output:   response.Content,
					Response: response,
					Usage:    response.Usage,
				},
			})
		}
	}()

	return ch
}
