package a2a

import (
	"context"
	"encoding/json"
	"fmt"

	ai "github.com/spetersoncode/gains"
)

// remoteArgs is the input schema for the remote agent tool.
type remoteArgs struct {
	Query string `json:"query" desc:"The query or task to send to the remote agent" required:"true"`
}

// RemoteTool wraps an A2A agent as a gains Tool.
// This allows a gains agent to call remote A2A agents as tools.
type RemoteTool struct {
	name        string
	description string
	client      *Client
	schema      json.RawMessage
}

// RemoteToolOption configures a RemoteTool.
type RemoteToolOption func(*RemoteTool)

// WithToolName sets the tool name (default: "remote_agent").
func WithToolName(name string) RemoteToolOption {
	return func(t *RemoteTool) {
		t.name = name
	}
}

// WithToolDescription sets the tool description.
func WithToolDescription(desc string) RemoteToolOption {
	return func(t *RemoteTool) {
		t.description = desc
	}
}

// WithToolSchema sets a custom schema for the tool arguments.
func WithToolSchema(schema json.RawMessage) RemoteToolOption {
	return func(t *RemoteTool) {
		t.schema = schema
	}
}

// NewRemoteTool creates a new RemoteTool that calls a remote A2A agent.
func NewRemoteTool(client *Client, opts ...RemoteToolOption) *RemoteTool {
	t := &RemoteTool{
		name:        "remote_agent",
		description: "Call a remote AI agent to perform a task",
		client:      client,
		schema:      ai.MustSchemaFor[remoteArgs](),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Tool returns a gains Tool that calls the remote agent.
func (t *RemoteTool) Tool() ai.Tool {
	return ai.Tool{
		Name:        t.name,
		Description: t.description,
		Parameters:  t.schema,
	}
}

// Handler returns a tool handler that calls the remote agent.
func (t *RemoteTool) Handler() func(ctx context.Context, input json.RawMessage) (string, error) {
	return func(ctx context.Context, input json.RawMessage) (string, error) {
		// Parse input
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(input, &args); err != nil {
			return "", fmt.Errorf("invalid input: %w", err)
		}

		// Send to remote agent
		task, err := t.client.SendText(ctx, args.Query)
		if err != nil {
			return "", fmt.Errorf("remote agent call failed: %w", err)
		}

		// Format response
		if task.Status.State == TaskStateFailed {
			if task.Status.Message != nil {
				return fmt.Sprintf("Remote agent failed: %s", task.Status.Message.TextContent()), nil
			}
			return "Remote agent failed", nil
		}

		// Extract response text
		if task.Status.Message != nil {
			return task.Status.Message.TextContent(), nil
		}

		// Fallback to serializing task
		result, _ := json.Marshal(task)
		return string(result), nil
	}
}

// Register registers the remote tool with a tool registry.
func (t *RemoteTool) Register(registry ToolRegistry) {
	registry.RegisterFunc(t.name, t.description, t.schema, t.Handler())
}

// ToolRegistry is the interface for registering tools.
type ToolRegistry interface {
	RegisterFunc(name, description string, schema json.RawMessage, handler func(ctx context.Context, input json.RawMessage) (string, error))
}
