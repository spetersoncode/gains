package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/tool"
)

// ToolOption configures a workflow tool.
type ToolOption func(*toolConfig)

type toolConfig struct {
	description  string
	resultMapper func(events []event.Event) (string, error)
	workflowOpts []Option
}

// WithToolDescription sets a custom description for the workflow tool.
func WithToolDescription(desc string) ToolOption {
	return func(c *toolConfig) {
		c.description = desc
	}
}

// WithToolWorkflowOptions passes options through to the workflow.
func WithToolWorkflowOptions(opts ...Option) ToolOption {
	return func(c *toolConfig) {
		c.workflowOpts = append(c.workflowOpts, opts...)
	}
}

// WithToolResultMapper sets a custom function to convert workflow events to a tool result.
// By default, the mapper returns the last non-error message content.
func WithToolResultMapper(mapper func(events []event.Event) (string, error)) ToolOption {
	return func(c *toolConfig) {
		c.resultMapper = mapper
	}
}

// NewTool creates a tool registration that wraps a workflow.Runner as a callable tool.
// This enables workflow composition where agents can invoke workflows as sub-tasks.
//
// The tool arguments are passed directly to the workflow as initial state.
// The workflow runs to completion and the final state or message content is returned.
//
// Example:
//
//	type AnalysisState struct {
//	    Query  string `json:"query" desc:"Analysis query" required:"true"`
//	    Result string `json:"result"`
//	}
//
//	runner := workflow.NewRunnerJSON[AnalysisState]("analysis", analysisWorkflow)
//	registry.Add(workflow.NewTool(runner,
//	    workflow.WithToolDescription("Run complex data analysis"),
//	))
func NewTool(runner Runner, opts ...ToolOption) tool.Registration {
	cfg := &toolConfig{
		description: fmt.Sprintf("Invoke the %s workflow", runner.Name()),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Default result mapper: return last message content or final state
	resultMapper := cfg.resultMapper
	if resultMapper == nil {
		resultMapper = defaultToolResultMapper
	}

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		// Parse arguments as state input
		var input any
		if call.Arguments != "" && call.Arguments != "{}" {
			if err := json.Unmarshal([]byte(call.Arguments), &input); err != nil {
				return "", fmt.Errorf("failed to parse arguments: %w", err)
			}
		}

		// Run workflow and collect events
		var events []event.Event
		for ev := range runner.RunStream(ctx, input, cfg.workflowOpts...) {
			events = append(events, ev)

			// Check for errors
			if ev.Type == event.RunError && ev.Error != nil {
				return "", fmt.Errorf("workflow error: %w", ev.Error)
			}
		}

		return resultMapper(events)
	}

	return tool.Registration{
		Tool: ai.Tool{
			Name:        runner.Name(),
			Description: cfg.description,
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`), // Accept any JSON
		},
		Handler: handler,
	}
}

// NewToolWithSchema creates a workflow tool with a typed schema.
// Use this when you want compile-time type safety for workflow inputs.
//
// Example:
//
//	type SearchInput struct {
//	    Query   string `json:"query" desc:"Search query" required:"true"`
//	    Limit   int    `json:"limit" desc:"Max results"`
//	}
//
//	registry.Add(workflow.NewToolWithSchema[SearchInput](searchRunner,
//	    workflow.WithToolDescription("Search for information"),
//	))
func NewToolWithSchema[T any](runner Runner, opts ...ToolOption) tool.Registration {
	reg := NewTool(runner, opts...)
	reg.Tool.Parameters = tool.MustSchemaFor[T]()
	return reg
}

// defaultToolResultMapper extracts a result from workflow events.
// It returns the last message content if available, otherwise a success message.
func defaultToolResultMapper(events []event.Event) (string, error) {
	// Look for message content in reverse order (last message first)
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		if ev.Type == event.MessageEnd && ev.Response != nil && ev.Response.Content != "" {
			return ev.Response.Content, nil
		}
		if ev.Type == event.StepEnd && ev.Response != nil && ev.Response.Content != "" {
			return ev.Response.Content, nil
		}
	}

	// Check for state snapshot with content
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type == event.StateSnapshot && events[i].State != nil {
			// Try to extract a "result" or "output" field from state
			if state, ok := events[i].State.(map[string]any); ok {
				if result, ok := state["result"].(string); ok && result != "" {
					return result, nil
				}
				if output, ok := state["output"].(string); ok && output != "" {
					return output, nil
				}
			}
			// Return JSON representation of state
			data, err := json.Marshal(events[i].State)
			if err == nil {
				return string(data), nil
			}
		}
	}

	return "Workflow completed successfully", nil
}
