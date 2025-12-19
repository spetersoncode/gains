package workflow

import (
	"context"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/chat"
	"github.com/spetersoncode/gains/event"
)

// Step represents a single unit of work in a workflow.
// Steps are generic over the state type S, providing compile-time type safety.
// State is passed by pointer and mutated in place - the caller retains access to the final state.
type Step[S any] interface {
	// Name returns a unique identifier for the step.
	Name() string

	// Run executes the step and mutates state in place.
	Run(ctx context.Context, state *S, opts ...Option) error

	// RunStream executes the step and returns a channel of events.
	// State is mutated in place during streaming.
	RunStream(ctx context.Context, state *S, opts ...Option) <-chan Event
}

// StepFunc is a function signature for simple step implementations.
type StepFunc[S any] func(ctx context.Context, state *S) error

// FuncStep wraps a function as a Step.
type FuncStep[S any] struct {
	name string
	fn   StepFunc[S]
}

// NewFuncStep creates a step from a function.
func NewFuncStep[S any](name string, fn StepFunc[S]) *FuncStep[S] {
	return &FuncStep[S]{name: name, fn: fn}
}

// Name returns the step name.
func (f *FuncStep[S]) Name() string { return f.name }

// Run executes the function.
func (f *FuncStep[S]) Run(ctx context.Context, state *S, opts ...Option) error {
	return f.fn(ctx, state)
}

// RunStream executes the function and emits events.
func (f *FuncStep[S]) RunStream(ctx context.Context, state *S, opts ...Option) <-chan Event {
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
type PromptFunc[S any] func(state *S) []ai.Message

// PromptStep makes a single LLM call with a dynamic prompt.
// The setter receives the raw response content as a string.
// For structured output, add WithResponseSchema to opts and unmarshal in setter.
type PromptStep[S any] struct {
	name       string
	chatClient chat.Client
	prompt     PromptFunc[S]
	setter     func(*S, string)
	chatOpts   []ai.Option
}

// NewPromptStep creates a step for a single LLM call.
// The setter receives the response content as a string.
//
// For plain text output:
//
//	step := NewPromptStep[MyState]("summarize", client, promptFn,
//	    func(s *MyState, content string) { s.Summary = content },
//	)
//
// For structured output, add WithResponseSchema and unmarshal in setter:
//
//	step := NewPromptStep[MyState]("analyze", client, promptFn,
//	    func(s *MyState, content string) {
//	        json.Unmarshal([]byte(content), &s.Analysis)
//	    },
//	    ai.WithResponseSchema(analysisSchema),
//	)
func NewPromptStep[S any](
	name string,
	c chat.Client,
	prompt PromptFunc[S],
	setter func(*S, string),
	opts ...ai.Option,
) *PromptStep[S] {
	return &PromptStep[S]{
		name:       name,
		chatClient: c,
		prompt:     prompt,
		setter:     setter,
		chatOpts:   opts,
	}
}

// Name returns the step name.
func (p *PromptStep[S]) Name() string { return p.name }

// Run executes the LLM call.
func (p *PromptStep[S]) Run(ctx context.Context, state *S, opts ...Option) error {
	options := ApplyOptions(opts...)

	// Merge chat options: constructor opts first, then runtime opts
	chatOpts := make([]ai.Option, 0, len(p.chatOpts)+len(options.ChatOptions))
	chatOpts = append(chatOpts, p.chatOpts...)
	chatOpts = append(chatOpts, options.ChatOptions...)

	msgs := p.prompt(state)
	resp, err := p.chatClient.Chat(ctx, msgs, chatOpts...)
	if err != nil {
		return err
	}

	if p.setter != nil {
		p.setter(state, resp.Content)
	}

	return nil
}

// RunStream executes the LLM call with streaming.
func (p *PromptStep[S]) RunStream(ctx context.Context, state *S, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		event.Emit(ch, Event{Type: event.StepStart, StepName: p.name})

		options := ApplyOptions(opts...)

		// Merge chat options: constructor opts first, then runtime opts
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
			if p.setter != nil {
				p.setter(state, response.Content)
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
