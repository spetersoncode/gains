package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/tool"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ServerOption configures a Server.
type ServerOption func(*serverConfig)

type serverConfig struct {
	name    string
	version string
}

// WithName sets the server name reported to MCP clients.
func WithName(name string) ServerOption {
	return func(c *serverConfig) {
		c.name = name
	}
}

// WithVersion sets the server version reported to MCP clients.
func WithVersion(version string) ServerOption {
	return func(c *serverConfig) {
		c.version = version
	}
}

// NewServer creates an MCP server that exposes tools from a gains tool.Registry.
// Each tool in the registry is registered with the MCP server, allowing MCP clients
// to discover and call the tools.
//
// Example:
//
//	registry := tool.NewRegistry().Add(
//	    tool.Func("weather", "Get weather", weatherHandler),
//	    tool.Func("search", "Search web", searchHandler),
//	)
//
//	mcpServer := mcp.NewServer(registry,
//	    mcp.WithName("my-tools"),
//	    mcp.WithVersion("1.0.0"),
//	)
//
//	server.ServeStdio(mcpServer)
func NewServer(registry *tool.Registry, opts ...ServerOption) *server.MCPServer {
	cfg := &serverConfig{
		name:    "gains-mcp-server",
		version: "1.0.0",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	s := server.NewMCPServer(
		cfg.name,
		cfg.version,
		server.WithToolCapabilities(true),
	)

	// Register each tool from the gains registry with the MCP server
	for _, t := range registry.Tools() {
		mcpTool := ToMCPTool(t)
		toolName := t.Name // capture for closure

		handler, ok := registry.Get(toolName)
		if !ok || handler == nil {
			// Skip client-side tools that have no handler
			continue
		}

		s.AddTool(mcpTool, createMCPHandler(toolName, handler))
	}

	return s
}

// createMCPHandler wraps a gains tool.Handler as an MCP tool handler.
func createMCPHandler(toolName string, handler tool.Handler) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Convert arguments to JSON string
		var argsJSON string
		if req.Params.Arguments != nil {
			data, err := json.Marshal(req.Params.Arguments)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to marshal arguments: %v", err)), nil
			}
			argsJSON = string(data)
		} else {
			argsJSON = "{}"
		}

		// Create a gains ToolCall
		call := ai.ToolCall{
			ID:        "", // MCP doesn't provide call IDs in the same way
			Name:      toolName,
			Arguments: argsJSON,
		}

		// Execute the handler
		result, err := handler(ctx, call)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

// ServeStdio starts an MCP server that communicates over stdin/stdout.
// This is the standard transport for MCP servers invoked as subprocesses.
//
// Example:
//
//	registry := tool.NewRegistry().Add(
//	    tool.Func("hello", "Say hello", helloHandler),
//	)
//
//	if err := mcp.ServeStdio(registry); err != nil {
//	    log.Fatal(err)
//	}
func ServeStdio(registry *tool.Registry, opts ...ServerOption) error {
	s := NewServer(registry, opts...)
	return server.ServeStdio(s)
}
