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

// PromptStep makes a single LLM call and stores the result in a state field.
// Generic over state type S and field type T.
//
// When schema is non-nil, the response is unmarshaled as JSON into the field.
// When schema is nil, T must be string and the response is assigned directly.
// Run() returns error only - results are stored in state via the field getter.
type PromptStep[S, T any] struct {
	name       string
	chatClient chat.Client
	prompt     PromptFunc[S]
	schema     *ai.ResponseSchema
	field      func(*S) *T
	chatOpts   []ai.Option
}

// NewPromptStep creates a step for a single LLM call.
// The field getter returns a pointer to where the result should be stored.
// Type parameters are inferred from the function arguments.
//
// For plain text (schema = nil):
//
//	step := NewPromptStep("summarize", client, promptFn, nil,
//	    func(s *MyState) *string { return &s.Summary },
//	)
//
// For structured JSON (schema required):
//
//	step := NewPromptStep("analyze", client, promptFn, schema,
//	    func(s *MyState) *Analysis { return &s.Analysis },
//	)
func NewPromptStep[S, T any](
	name string,
	c chat.Client,
	prompt PromptFunc[S],
	schema *ai.ResponseSchema,
	field func(*S) *T,
	opts ...ai.Option,
) *PromptStep[S, T] {
	return &PromptStep[S, T]{
		name:       name,
		chatClient: c,
		prompt:     prompt,
		schema:     schema,
		field:      field,
		chatOpts:   opts,
	}
}

// Name returns the step name.
func (p *PromptStep[S, T]) Name() string { return p.name }

// Run executes the LLM call.
func (p *PromptStep[S, T]) Run(ctx context.Context, state *S, opts ...Option) error {
	options := ApplyOptions(opts...)

	// Merge chat options: constructor opts first, then runtime opts
	chatOpts := make([]ai.Option, 0, len(p.chatOpts)+len(options.ChatOptions)+1)
	chatOpts = append(chatOpts, p.chatOpts...)
	chatOpts = append(chatOpts, options.ChatOptions...)

	// Add schema if provided
	if p.schema != nil {
		chatOpts = append(chatOpts, ai.WithResponseSchema(*p.schema))
	}

	msgs := p.prompt(state)
	resp, err := p.chatClient.Chat(ctx, msgs, chatOpts...)
	if err != nil {
		return err
	}

	if p.field != nil {
		if err := p.storeResult(state, resp.Content); err != nil {
			return err
		}
	}

	return nil
}

// storeResult stores the response content into the field.
func (p *PromptStep[S, T]) storeResult(state *S, content string) error {
	fieldPtr := p.field(state)
	if p.schema != nil {
		// Structured output: unmarshal JSON
		if err := json.Unmarshal([]byte(content), fieldPtr); err != nil {
			return &ai.UnmarshalError{
				Context:    fmt.Sprintf("workflow step %q", p.name),
				Content:    content,
				TargetType: fmt.Sprintf("%T", *fieldPtr),
				Err:        err,
			}
		}
	} else {
		// Plain text: T must be string
		if strPtr, ok := any(fieldPtr).(*string); ok {
			*strPtr = content
		} else {
			return fmt.Errorf("workflow step %q: nil schema requires string field, got %T", p.name, fieldPtr)
		}
	}
	return nil
}

// RunStream executes the LLM call with streaming.
func (p *PromptStep[S, T]) RunStream(ctx context.Context, state *S, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)
		event.Emit(ch, Event{Type: event.StepStart, StepName: p.name})

		options := ApplyOptions(opts...)

		// Merge chat options: constructor opts first, then runtime opts
		chatOpts := make([]ai.Option, 0, len(p.chatOpts)+len(options.ChatOptions)+1)
		chatOpts = append(chatOpts, p.chatOpts...)
		chatOpts = append(chatOpts, options.ChatOptions...)

		// Add schema if provided
		if p.schema != nil {
			chatOpts = append(chatOpts, ai.WithResponseSchema(*p.schema))
		}

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
			if p.field != nil {
				if err := p.storeResult(state, response.Content); err != nil {
					event.Emit(ch, Event{Type: event.RunError, StepName: p.name, Error: err})
					return
				}
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
