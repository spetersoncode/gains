package mcp

import (
	"context"
	"fmt"
	"sync"

	ai "github.com/spetersoncode/gains"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// RemoteRegistry provides access to tools from an MCP server.
// It implements a similar interface to [tool.Registry] but proxies
// tool calls to the remote MCP server.
//
// RemoteRegistry is safe for concurrent use. The tool list is cached
// locally and can be refreshed with [RemoteRegistry.Refresh].
type RemoteRegistry struct {
	client *client.Client
	mu     sync.RWMutex
	tools  map[string]ai.Tool
}

// NewRemoteRegistry creates a RemoteRegistry connected to an MCP server via stdio.
// The command is the path to the MCP server executable, and args are passed to it.
//
// Example:
//
//	registry, err := mcp.NewRemoteRegistry(ctx, "./my-mcp-server", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer registry.Close()
//
//	// Use with an agent
//	agent := agent.New(client, agent.WithToolRegistry(registry))
func NewRemoteRegistry(ctx context.Context, command string, env []string, args ...string) (*RemoteRegistry, error) {
	c, err := client.NewStdioMCPClient(command, env, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	return newRemoteRegistryFromClient(ctx, c)
}

// NewRemoteRegistrySSE creates a RemoteRegistry connected to an MCP server via SSE.
//
// Example:
//
//	registry, err := mcp.NewRemoteRegistrySSE(ctx, "http://localhost:8080/mcp")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer registry.Close()
func NewRemoteRegistrySSE(ctx context.Context, baseURL string) (*RemoteRegistry, error) {
	c, err := client.NewSSEMCPClient(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE MCP client: %w", err)
	}

	return newRemoteRegistryFromClient(ctx, c)
}

// NewRemoteRegistryFromClient creates a RemoteRegistry from an existing MCP client.
// The client must already be started. This function will initialize it and fetch tools.
func NewRemoteRegistryFromClient(ctx context.Context, c *client.Client) (*RemoteRegistry, error) {
	return newRemoteRegistryFromClient(ctx, c)
}

func newRemoteRegistryFromClient(ctx context.Context, c *client.Client) (*RemoteRegistry, error) {
	// Start the client connection
	if err := c.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start MCP client: %w", err)
	}

	// Initialize the MCP session
	_, err := c.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo: mcp.Implementation{
				Name:    "gains-mcp-client",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to initialize MCP session: %w", err)
	}

	r := &RemoteRegistry{
		client: c,
		tools:  make(map[string]ai.Tool),
	}

	// Fetch available tools
	if err := r.Refresh(ctx); err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	return r, nil
}

// Close closes the connection to the MCP server.
func (r *RemoteRegistry) Close() error {
	return r.client.Close()
}

// Refresh fetches the current list of tools from the MCP server.
// This can be called to update the tool list if the server's tools change.
func (r *RemoteRegistry) Refresh(ctx context.Context) error {
	result, err := r.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]ai.Tool, len(result.Tools))
	for _, t := range result.Tools {
		r.tools[t.Name] = FromMCPTool(t)
	}

	return nil
}

// Tools returns all tools available from the MCP server.
func (r *RemoteRegistry) Tools() []ai.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]ai.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// GetTool retrieves a tool definition by name.
func (r *RemoteRegistry) GetTool(name string) (ai.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tools[name]
	return t, ok
}

// Names returns the names of all available tools.
func (r *RemoteRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Len returns the number of available tools.
func (r *RemoteRegistry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// Execute calls a tool on the remote MCP server.
func (r *RemoteRegistry) Execute(ctx context.Context, call ai.ToolCall) (ai.ToolResult, error) {
	req := ToMCPCallToolRequest(call)

	result, err := r.client.CallTool(ctx, req)
	if err != nil {
		return ai.ToolResult{
			ToolCallID: call.ID,
			Content:    err.Error(),
			IsError:    true,
		}, nil
	}

	return FromMCPCallToolResult(call.ID, result), nil
}

// Has returns true if the registry has a tool with the given name.
func (r *RemoteRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[name]
	return ok
}
