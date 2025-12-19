package agent

import (
	"context"
	"encoding/json"
	"fmt"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/tool"
)

// ToolArgs is the default argument type for agent tools.
// It provides a simple query-based interface for invoking sub-agents.
type ToolArgs struct {
	Query string `json:"query" desc:"The query or task for the agent" required:"true"`
}

// ToolOption configures an agent tool.
type ToolOption func(*toolConfig)

type toolConfig struct {
	description    string
	maxSteps       int
	agentOptions   []Option
	schema         json.RawMessage
	argsMapper     func(call ai.ToolCall) ([]ai.Message, error)
	forwardEvents  bool
}

// WithToolDescription sets a custom description for the agent tool.
func WithToolDescription(desc string) ToolOption {
	return func(c *toolConfig) {
		c.description = desc
	}
}

// WithToolMaxSteps sets the maximum steps for the sub-agent.
func WithToolMaxSteps(n int) ToolOption {
	return func(c *toolConfig) {
		c.maxSteps = n
	}
}

// WithToolAgentOptions passes options through to the agent.
func WithToolAgentOptions(opts ...Option) ToolOption {
	return func(c *toolConfig) {
		c.agentOptions = append(c.agentOptions, opts...)
	}
}

// WithToolSchema sets a custom parameter schema for the agent tool.
// Use this when you want different arguments than the default ToolArgs.
func WithToolSchema(schema json.RawMessage) ToolOption {
	return func(c *toolConfig) {
		c.schema = schema
	}
}

// WithToolArgsMapper sets a custom function to convert tool arguments to messages.
// The function receives the raw tool call and should return the messages to send to the agent.
func WithToolArgsMapper(mapper func(call ai.ToolCall) ([]ai.Message, error)) ToolOption {
	return func(c *toolConfig) {
		c.argsMapper = mapper
	}
}

// WithToolEventForwarding enables event forwarding from the sub-agent to the parent.
// When enabled and a forwarding channel is available in the context, all sub-agent
// events are forwarded to the parent event stream for observability.
//
// This allows the parent agent (or AG-UI frontend) to observe sub-agent progress:
// - Streaming message deltas from the sub-agent
// - Tool calls made by the sub-agent
// - Step progress within the sub-agent
//
// Example:
//
//	mainRegistry.Add(agent.NewTool("research", researchAgent,
//	    agent.WithToolEventForwarding(),
//	    agent.WithToolDescription("Research a topic with observable progress"),
//	))
func WithToolEventForwarding() ToolOption {
	return func(c *toolConfig) {
		c.forwardEvents = true
	}
}

// NewTool creates a tool registration that wraps an agent as a callable tool.
// This enables sub-agent patterns where one agent can delegate tasks to specialized agents.
//
// By default, the tool accepts ToolArgs (a simple query field) and runs the agent
// with a user message containing the query. The final response content is returned.
//
// Example:
//
//	// Create a specialized research agent
//	researchAgent := agent.New(client, researchTools)
//
//	// Register it as a tool in the main agent's registry
//	mainRegistry.Add(agent.NewTool("research", researchAgent,
//	    agent.WithToolDescription("Delegate complex research tasks to a specialized research agent"),
//	    agent.WithToolMaxSteps(5),
//	))
func NewTool(name string, a *Agent, opts ...ToolOption) tool.Registration {
	cfg := &toolConfig{
		description: fmt.Sprintf("Invoke the %s agent", name),
		maxSteps:    5,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Use default schema if not provided
	schema := cfg.schema
	if schema == nil {
		schema = tool.MustSchemaFor[ToolArgs]()
	}

	// Use default args mapper if not provided
	mapper := cfg.argsMapper
	if mapper == nil {
		mapper = func(call ai.ToolCall) ([]ai.Message, error) {
			var args ToolArgs
			if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
				return nil, fmt.Errorf("failed to parse arguments: %w", err)
			}
			return []ai.Message{{Role: ai.RoleUser, Content: args.Query}}, nil
		}
	}

	// Build agent options
	agentOpts := []Option{WithMaxSteps(cfg.maxSteps)}
	agentOpts = append(agentOpts, cfg.agentOptions...)

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		messages, err := mapper(call)
		if err != nil {
			return "", err
		}

		// Check for event forwarding channel in context
		forwardCh := event.ForwardChannelFromContext(ctx)

		// Use streaming mode if forwarding is enabled and channel is available
		if cfg.forwardEvents && forwardCh != nil {
			return runWithEventForwarding(ctx, a, messages, agentOpts, forwardCh)
		}

		// Standard blocking execution
		result, err := a.Run(ctx, messages, agentOpts...)
		if err != nil {
			return "", fmt.Errorf("agent execution failed: %w", err)
		}

		if result.Response == nil {
			return "", fmt.Errorf("agent returned no response")
		}

		return result.Response.Content, nil
	}

	return tool.Registration{
		Tool: ai.Tool{
			Name:        name,
			Description: cfg.description,
			Parameters:  schema,
		},
		Handler: handler,
	}
}

// NewToolFunc creates a tool registration with a custom typed argument handler.
// This provides type-safe argument handling for agent tools.
//
// Example:
//
//	type ResearchArgs struct {
//	    Topic     string `json:"topic" desc:"Research topic" required:"true"`
//	    MaxDepth  int    `json:"maxDepth" desc:"Maximum research depth" default:"2"`
//	}
//
//	mainRegistry.Add(agent.NewToolFunc("research", researchAgent, "Research a topic",
//	    func(args ResearchArgs) []ai.Message {
//	        return []ai.Message{
//	            ai.NewUserMessage(fmt.Sprintf("Research %s with depth %d", args.Topic, args.MaxDepth)),
//	        }
//	    },
//	))
func NewToolFunc[T any](name string, a *Agent, description string, toMessages func(args T) []ai.Message, opts ...ToolOption) tool.Registration {
	cfg := &toolConfig{
		description: description,
		maxSteps:    5,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	schema := cfg.schema
	if schema == nil {
		schema = tool.MustSchemaFor[T]()
	}

	agentOpts := []Option{WithMaxSteps(cfg.maxSteps)}
	agentOpts = append(agentOpts, cfg.agentOptions...)

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args T
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", fmt.Errorf("failed to parse arguments: %w", err)
		}

		messages := toMessages(args)

		// Check for event forwarding channel in context
		forwardCh := event.ForwardChannelFromContext(ctx)

		// Use streaming mode if forwarding is enabled and channel is available
		if cfg.forwardEvents && forwardCh != nil {
			return runWithEventForwarding(ctx, a, messages, agentOpts, forwardCh)
		}

		result, err := a.Run(ctx, messages, agentOpts...)
		if err != nil {
			return "", fmt.Errorf("agent execution failed: %w", err)
		}

		if result.Response == nil {
			return "", fmt.Errorf("agent returned no response")
		}

		return result.Response.Content, nil
	}

	return tool.Registration{
		Tool: ai.Tool{
			Name:        name,
			Description: cfg.description,
			Parameters:  schema,
		},
		Handler: handler,
	}
}

// runWithEventForwarding runs an agent using RunStream and forwards all events
// to the parent's event channel. Returns the final response content.
func runWithEventForwarding(ctx context.Context, a *Agent, messages []ai.Message, opts []Option, forwardCh chan<- Event) (string, error) {
	eventCh := a.RunStream(ctx, messages, opts...)

	var lastResponse *ai.Response
	var runError error

	for ev := range eventCh {
		// Forward the event to the parent channel
		select {
		case forwardCh <- ev:
		case <-ctx.Done():
			return "", ctx.Err()
		}

		// Track final response and errors
		switch ev.Type {
		case event.RunEnd:
			if ev.Response != nil {
				lastResponse = ev.Response
			}
		case event.RunError:
			runError = ev.Error
		case event.StepEnd:
			// Track last response from steps as fallback
			if ev.Response != nil {
				lastResponse = ev.Response
			}
		}
	}

	if runError != nil {
		return "", fmt.Errorf("agent execution failed: %w", runError)
	}

	if lastResponse == nil {
		return "", fmt.Errorf("agent returned no response")
	}

	return lastResponse.Content, nil
}
