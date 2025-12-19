package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/tool"
)

// ToolStepOutput holds both the typed arguments and string result
// from a tool execution.
type ToolStepOutput[T any] struct {
	Args   T      // Original typed arguments
	Result string // Tool execution result
}

// ToolStep executes a single tool with typed arguments.
// The typed arguments are preserved in the output, allowing downstream
// steps to access the original structured args.
type ToolStep[S, T any] struct {
	name     string
	registry *tool.Registry
	toolName string
	argsFunc func(*S) (T, error)
	setter   func(*S, *ToolStepOutput[T])
}

// NewToolStep creates a step that executes a tool with typed arguments.
// The setter receives both the typed arguments and the result string.
//
// Parameters:
//   - name: Unique identifier for the step
//   - registry: Tool registry containing the tool handler
//   - toolName: Name of the tool to execute
//   - argsFunc: Function that builds typed tool arguments from state
//   - setter: Function that stores the output in state
//
// Example:
//
//	type SearchArgs struct {
//	    Query string `json:"query"`
//	    Limit int    `json:"limit"`
//	}
//
//	step := NewToolStep[MyState, SearchArgs]("search", registry, "web_search",
//	    func(s *MyState) (SearchArgs, error) {
//	        return SearchArgs{Query: s.Topic, Limit: 10}, nil
//	    },
//	    func(s *MyState, out *ToolStepOutput[SearchArgs]) {
//	        s.SearchResult = out.Result
//	        s.LastQuery = out.Args.Query  // Access typed args
//	    },
//	)
func NewToolStep[S, T any](
	name string,
	registry *tool.Registry,
	toolName string,
	argsFunc func(*S) (T, error),
	setter func(*S, *ToolStepOutput[T]),
) *ToolStep[S, T] {
	return &ToolStep[S, T]{
		name:     name,
		registry: registry,
		toolName: toolName,
		argsFunc: argsFunc,
		setter:   setter,
	}
}

// Name returns the step name.
func (t *ToolStep[S, T]) Name() string { return t.name }

// Run executes the tool.
func (t *ToolStep[S, T]) Run(ctx context.Context, state *S, opts ...Option) error {
	options := ApplyOptions(opts...)

	// Apply step timeout
	if options.StepTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.StepTimeout)
		defer cancel()
	}

	// Build typed arguments
	args, err := t.argsFunc(state)
	if err != nil {
		return &StepError{StepName: t.name, Err: fmt.Errorf("building arguments: %w", err)}
	}

	// Marshal arguments to JSON
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return &StepError{StepName: t.name, Err: fmt.Errorf("marshaling arguments: %w", err)}
	}

	// Create synthetic tool call
	call := ai.ToolCall{
		ID:        fmt.Sprintf("%s-%d", t.name, time.Now().UnixNano()),
		Name:      t.toolName,
		Arguments: string(argsJSON),
	}

	// Execute tool
	result, err := t.registry.Execute(ctx, call)
	if err != nil {
		return &StepError{StepName: t.name, Err: err}
	}

	// Handle tool error
	if result.IsError {
		return &StepError{
			StepName: t.name,
			Err:      &ToolExecutionError{ToolName: t.toolName, Content: result.Content},
		}
	}

	// Store output via setter (includes both args and result)
	if t.setter != nil {
		t.setter(state, &ToolStepOutput[T]{
			Args:   args,
			Result: result.Content,
		})
	}

	return nil
}

// RunStream executes the tool and emits events.
func (t *ToolStep[S, T]) RunStream(ctx context.Context, state *S, opts ...Option) <-chan Event {
	ch := make(chan Event, 10)

	go func() {
		defer close(ch)
		event.Emit(ch, Event{Type: event.StepStart, StepName: t.name})

		err := t.Run(ctx, state, opts...)
		if err != nil {
			event.Emit(ch, Event{Type: event.RunError, StepName: t.name, Error: err})
			return
		}

		event.Emit(ch, Event{Type: event.StepEnd, StepName: t.name})
	}()

	return ch
}
