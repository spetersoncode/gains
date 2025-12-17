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

// ToolArgsFunc generates tool arguments from state.
// Returns a value that will be JSON-marshaled as the tool arguments.
type ToolArgsFunc func(state *State) (any, error)

// ToolStep executes a single tool without LLM involvement.
// Use for direct tool invocations where the arguments are known.
type ToolStep struct {
	name      string
	registry  *tool.Registry
	toolName  string
	argsFunc  ToolArgsFunc
	outputKey string
}

// NewToolStep creates a step that executes a single tool.
//
// Parameters:
//   - name: Unique identifier for the step
//   - registry: Tool registry containing the tool handler
//   - toolName: Name of the tool to execute
//   - argsFunc: Function that builds tool arguments from state
//   - outputKey: State key for storing tool result (empty to skip storage)
//
// Example:
//
//	step := workflow.NewToolStep(
//	    "fetch_data",
//	    registry,
//	    "http_get",
//	    func(s *workflow.State) (any, error) {
//	        return map[string]string{"url": s.GetString("api_endpoint")}, nil
//	    },
//	    "api_response",
//	)
func NewToolStep(
	name string,
	registry *tool.Registry,
	toolName string,
	argsFunc ToolArgsFunc,
	outputKey string,
) *ToolStep {
	return &ToolStep{
		name:      name,
		registry:  registry,
		toolName:  toolName,
		argsFunc:  argsFunc,
		outputKey: outputKey,
	}
}

// Name returns the step name.
func (t *ToolStep) Name() string { return t.name }

// Run executes the tool.
func (t *ToolStep) Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error) {
	options := ApplyOptions(opts...)

	// Apply step timeout
	if options.StepTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.StepTimeout)
		defer cancel()
	}

	// Build arguments
	args, err := t.argsFunc(state)
	if err != nil {
		return nil, &StepError{StepName: t.name, Err: fmt.Errorf("building arguments: %w", err)}
	}

	// Marshal arguments to JSON
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, &StepError{StepName: t.name, Err: fmt.Errorf("marshaling arguments: %w", err)}
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
		return nil, &StepError{StepName: t.name, Err: err}
	}

	// Handle tool error
	if result.IsError {
		return nil, &StepError{
			StepName: t.name,
			Err:      &ToolExecutionError{ToolName: t.toolName, Content: result.Content},
		}
	}

	// Store result
	if t.outputKey != "" {
		state.Set(t.outputKey, result.Content)
	}

	return &StepResult{
		StepName: t.name,
		Output:   result.Content,
		Metadata: map[string]any{
			"tool_name":    t.toolName,
			"tool_call_id": call.ID,
		},
	}, nil
}

// RunStream executes the tool and emits events.
func (t *ToolStep) RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event {
	ch := make(chan Event, 10)

	go func() {
		defer close(ch)
		event.Emit(ch, Event{Type: event.StepStart, StepName: t.name})

		result, err := t.Run(ctx, state, opts...)
		if err != nil {
			event.Emit(ch, Event{Type: event.RunError, StepName: t.name, Error: err})
			return
		}

		event.Emit(ch, Event{Type: event.StepEnd, StepName: t.name})
		_ = result
	}()

	return ch
}

// TypedToolStep executes a single tool with typed arguments.
type TypedToolStep[T any] struct {
	name      string
	registry  *tool.Registry
	toolName  string
	argsFunc  func(state *State) (T, error)
	outputKey string
}

// NewTypedToolStep creates a step that executes a tool with typed arguments.
// The type T should match the tool's expected argument structure.
//
// Example:
//
//	type SearchArgs struct {
//	    Query string `json:"query"`
//	    Limit int    `json:"limit"`
//	}
//
//	step := workflow.NewTypedToolStep[SearchArgs](
//	    "search",
//	    registry,
//	    "web_search",
//	    func(s *workflow.State) (SearchArgs, error) {
//	        return SearchArgs{
//	            Query: s.GetString("search_query"),
//	            Limit: 10,
//	        }, nil
//	    },
//	    "search_results",
//	)
func NewTypedToolStep[T any](
	name string,
	registry *tool.Registry,
	toolName string,
	argsFunc func(state *State) (T, error),
	outputKey string,
) *TypedToolStep[T] {
	return &TypedToolStep[T]{
		name:      name,
		registry:  registry,
		toolName:  toolName,
		argsFunc:  argsFunc,
		outputKey: outputKey,
	}
}

// Name returns the step name.
func (t *TypedToolStep[T]) Name() string { return t.name }

// Run executes the tool.
func (t *TypedToolStep[T]) Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error) {
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
		return nil, &StepError{StepName: t.name, Err: fmt.Errorf("building arguments: %w", err)}
	}

	// Marshal arguments to JSON
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, &StepError{StepName: t.name, Err: fmt.Errorf("marshaling arguments: %w", err)}
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
		return nil, &StepError{StepName: t.name, Err: err}
	}

	// Handle tool error
	if result.IsError {
		return nil, &StepError{
			StepName: t.name,
			Err:      &ToolExecutionError{ToolName: t.toolName, Content: result.Content},
		}
	}

	// Store result
	if t.outputKey != "" {
		state.Set(t.outputKey, result.Content)
	}

	return &StepResult{
		StepName: t.name,
		Output:   result.Content,
		Metadata: map[string]any{
			"tool_name":    t.toolName,
			"tool_call_id": call.ID,
		},
	}, nil
}

// RunStream executes the tool and emits events.
func (t *TypedToolStep[T]) RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event {
	ch := make(chan Event, 10)

	go func() {
		defer close(ch)
		event.Emit(ch, Event{Type: event.StepStart, StepName: t.name})

		result, err := t.Run(ctx, state, opts...)
		if err != nil {
			event.Emit(ch, Event{Type: event.RunError, StepName: t.name, Error: err})
			return
		}

		event.Emit(ch, Event{Type: event.StepEnd, StepName: t.name})
		_ = result
	}()

	return ch
}

// NewTypedToolStepWithKey creates a TypedToolStep that stores output using a typed key.
func NewTypedToolStepWithKey[T any](
	name string,
	registry *tool.Registry,
	toolName string,
	argsFunc func(state *State) (T, error),
	outputKey Key[string],
) *TypedToolStep[T] {
	return &TypedToolStep[T]{
		name:      name,
		registry:  registry,
		toolName:  toolName,
		argsFunc:  argsFunc,
		outputKey: outputKey.Name(),
	}
}
