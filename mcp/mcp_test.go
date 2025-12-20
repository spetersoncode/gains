package mcp

import (
	"context"
	"encoding/json"
	"testing"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToMCPTool(t *testing.T) {
	t.Run("converts gains tool to MCP tool", func(t *testing.T) {
		schema := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`)
		gainsTool := ai.Tool{
			Name:        "greet",
			Description: "Greet someone",
			Parameters:  schema,
		}

		mcpTool := ToMCPTool(gainsTool)

		assert.Equal(t, "greet", mcpTool.Name)
		assert.Equal(t, "Greet someone", mcpTool.Description)
		assert.Equal(t, schema, mcpTool.RawInputSchema)
	})

	t.Run("handles nil parameters", func(t *testing.T) {
		gainsTool := ai.Tool{
			Name:        "simple",
			Description: "Simple tool",
			Parameters:  nil,
		}

		mcpTool := ToMCPTool(gainsTool)

		assert.Equal(t, "simple", mcpTool.Name)
		assert.Equal(t, "Simple tool", mcpTool.Description)
	})
}

func TestToMCPTools(t *testing.T) {
	t.Run("converts slice of gains tools", func(t *testing.T) {
		tools := []ai.Tool{
			{Name: "tool1", Description: "First tool"},
			{Name: "tool2", Description: "Second tool"},
		}

		mcpTools := ToMCPTools(tools)

		assert.Len(t, mcpTools, 2)
		assert.Equal(t, "tool1", mcpTools[0].Name)
		assert.Equal(t, "tool2", mcpTools[1].Name)
	})
}

func TestFromMCPTool(t *testing.T) {
	t.Run("converts MCP tool with raw schema", func(t *testing.T) {
		schema := json.RawMessage(`{"type":"object"}`)
		mcpTool := mcp.NewToolWithRawSchema("weather", "Get weather", schema)

		gainsTool := FromMCPTool(mcpTool)

		assert.Equal(t, "weather", gainsTool.Name)
		assert.Equal(t, "Get weather", gainsTool.Description)
		assert.JSONEq(t, `{"type":"object"}`, string(gainsTool.Parameters))
	})

	t.Run("converts MCP tool with structured schema", func(t *testing.T) {
		mcpTool := mcp.NewTool("search",
			mcp.WithDescription("Search the web"),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		)

		gainsTool := FromMCPTool(mcpTool)

		assert.Equal(t, "search", gainsTool.Name)
		assert.Equal(t, "Search the web", gainsTool.Description)
		assert.NotNil(t, gainsTool.Parameters)
	})
}

func TestFromMCPTools(t *testing.T) {
	t.Run("converts slice of MCP tools", func(t *testing.T) {
		mcpTools := []mcp.Tool{
			mcp.NewTool("a", mcp.WithDescription("Tool A")),
			mcp.NewTool("b", mcp.WithDescription("Tool B")),
		}

		gainsTools := FromMCPTools(mcpTools)

		assert.Len(t, gainsTools, 2)
		assert.Equal(t, "a", gainsTools[0].Name)
		assert.Equal(t, "b", gainsTools[1].Name)
	})
}

func TestToMCPCallToolRequest(t *testing.T) {
	t.Run("converts gains tool call to MCP request", func(t *testing.T) {
		call := ai.ToolCall{
			ID:        "call_123",
			Name:      "calculate",
			Arguments: `{"a": 10, "b": 5}`,
		}

		req := ToMCPCallToolRequest(call)

		assert.Equal(t, "calculate", req.Params.Name)
		args, ok := req.Params.Arguments.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(10), args["a"])
		assert.Equal(t, float64(5), args["b"])
	})

	t.Run("handles empty arguments", func(t *testing.T) {
		call := ai.ToolCall{
			ID:        "call_456",
			Name:      "noargs",
			Arguments: "",
		}

		req := ToMCPCallToolRequest(call)

		assert.Equal(t, "noargs", req.Params.Name)
		assert.Nil(t, req.Params.Arguments)
	})
}

func TestFromMCPCallToolResult(t *testing.T) {
	t.Run("converts text result", func(t *testing.T) {
		mcpResult := mcp.NewToolResultText("Hello, World!")

		result := FromMCPCallToolResult("call_123", mcpResult)

		assert.Equal(t, "call_123", result.ToolCallID)
		assert.Equal(t, "Hello, World!", result.Content)
		assert.False(t, result.IsError)
	})

	t.Run("converts error result", func(t *testing.T) {
		mcpResult := mcp.NewToolResultError("something went wrong")

		result := FromMCPCallToolResult("call_456", mcpResult)

		assert.Equal(t, "call_456", result.ToolCallID)
		assert.Equal(t, "something went wrong", result.Content)
		assert.True(t, result.IsError)
	})

	t.Run("handles nil result", func(t *testing.T) {
		result := FromMCPCallToolResult("call_789", nil)

		assert.Equal(t, "call_789", result.ToolCallID)
		assert.Equal(t, "", result.Content)
		assert.True(t, result.IsError)
	})
}

func TestToMCPCallToolResult(t *testing.T) {
	t.Run("converts success result", func(t *testing.T) {
		gainsResult := ai.ToolResult{
			ToolCallID: "call_123",
			Content:    "Success!",
			IsError:    false,
		}

		mcpResult := ToMCPCallToolResult(gainsResult)

		assert.False(t, mcpResult.IsError)
		require.Len(t, mcpResult.Content, 1)
	})

	t.Run("converts error result", func(t *testing.T) {
		gainsResult := ai.ToolResult{
			ToolCallID: "call_456",
			Content:    "Error message",
			IsError:    true,
		}

		mcpResult := ToMCPCallToolResult(gainsResult)

		assert.True(t, mcpResult.IsError)
	})
}

// TestServerIntegration tests the server using an in-process MCP client.
func TestServerIntegration(t *testing.T) {
	t.Run("exposes tools from registry", func(t *testing.T) {
		// Create a registry with test tools
		registry := tool.NewRegistry().Add(
			tool.Func("echo", "Echo text", func(ctx context.Context, args struct {
				Text string `json:"text"`
			}) (string, error) {
				return args.Text, nil
			}),
			tool.Func("add", "Add numbers", func(ctx context.Context, args struct {
				A int `json:"a"`
				B int `json:"b"`
			}) (string, error) {
				data, err := json.Marshal(args.A + args.B)
				return string(data), err
			}),
		)

		// Create MCP server from registry
		server := NewServer(registry,
			WithName("test-server"),
			WithVersion("1.0.0"),
		)

		// Create in-process client
		c, err := client.NewInProcessClient(server)
		require.NoError(t, err)

		ctx := context.Background()

		// Start and initialize client
		err = c.Start(ctx)
		require.NoError(t, err)
		defer c.Close()

		_, err = c.Initialize(ctx, mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				Capabilities:    mcp.ClientCapabilities{},
				ClientInfo: mcp.Implementation{
					Name:    "test-client",
					Version: "1.0.0",
				},
			},
		})
		require.NoError(t, err)

		// List tools
		result, err := c.ListTools(ctx, mcp.ListToolsRequest{})
		require.NoError(t, err)

		assert.Len(t, result.Tools, 2)

		// Find tool names
		names := make([]string, len(result.Tools))
		for i, t := range result.Tools {
			names[i] = t.Name
		}
		assert.Contains(t, names, "echo")
		assert.Contains(t, names, "add")
	})

	t.Run("calls tools and returns results", func(t *testing.T) {
		registry := tool.NewRegistry().Add(
			tool.Func("greet", "Greet someone", func(ctx context.Context, args struct {
				Name string `json:"name"`
			}) (string, error) {
				return "Hello, " + args.Name + "!", nil
			}),
		)

		server := NewServer(registry)
		c, err := client.NewInProcessClient(server)
		require.NoError(t, err)

		ctx := context.Background()
		err = c.Start(ctx)
		require.NoError(t, err)
		defer c.Close()

		_, err = c.Initialize(ctx, mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				Capabilities:    mcp.ClientCapabilities{},
				ClientInfo: mcp.Implementation{
					Name:    "test-client",
					Version: "1.0.0",
				},
			},
		})
		require.NoError(t, err)

		// Call the tool
		result, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "greet",
				Arguments: map[string]any{
					"name": "World",
				},
			},
		})
		require.NoError(t, err)

		assert.False(t, result.IsError)
		require.Len(t, result.Content, 1)
		textContent, ok := result.Content[0].(mcp.TextContent)
		require.True(t, ok)
		assert.Equal(t, "Hello, World!", textContent.Text)
	})

	t.Run("handles tool errors gracefully", func(t *testing.T) {
		registry := tool.NewRegistry().Add(
			tool.Func("fail", "Always fails", func(ctx context.Context, args struct{}) (string, error) {
				return "", assert.AnError
			}),
		)

		server := NewServer(registry)
		c, err := client.NewInProcessClient(server)
		require.NoError(t, err)

		ctx := context.Background()
		err = c.Start(ctx)
		require.NoError(t, err)
		defer c.Close()

		_, err = c.Initialize(ctx, mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				Capabilities:    mcp.ClientCapabilities{},
				ClientInfo: mcp.Implementation{
					Name:    "test-client",
					Version: "1.0.0",
				},
			},
		})
		require.NoError(t, err)

		result, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "fail",
				Arguments: map[string]any{},
			},
		})
		require.NoError(t, err)

		assert.True(t, result.IsError)
	})
}

// TestRemoteRegistryIntegration tests RemoteRegistry using an in-process server.
func TestRemoteRegistryIntegration(t *testing.T) {
	t.Run("creates registry from in-process server", func(t *testing.T) {
		// Create a gains registry with tools
		sourceRegistry := tool.NewRegistry().Add(
			tool.Func("ping", "Ping pong", func(ctx context.Context, args struct{}) (string, error) {
				return "pong", nil
			}),
			tool.Func("echo", "Echo text", func(ctx context.Context, args struct {
				Text string `json:"text"`
			}) (string, error) {
				return args.Text, nil
			}),
		)

		// Create MCP server
		server := NewServer(sourceRegistry)

		// Create in-process client
		c, err := client.NewInProcessClient(server)
		require.NoError(t, err)

		ctx := context.Background()

		// Create RemoteRegistry from the client
		remoteRegistry, err := NewRemoteRegistryFromClient(ctx, c)
		require.NoError(t, err)
		defer remoteRegistry.Close()

		// Check tools are available
		assert.Equal(t, 2, remoteRegistry.Len())
		assert.True(t, remoteRegistry.Has("ping"))
		assert.True(t, remoteRegistry.Has("echo"))

		// Check tool definitions
		pingTool, ok := remoteRegistry.GetTool("ping")
		assert.True(t, ok)
		assert.Equal(t, "ping", pingTool.Name)
		assert.Equal(t, "Ping pong", pingTool.Description)
	})

	t.Run("executes remote tools", func(t *testing.T) {
		sourceRegistry := tool.NewRegistry().Add(
			tool.Func("add", "Add numbers", func(ctx context.Context, args struct {
				A int `json:"a"`
				B int `json:"b"`
			}) (string, error) {
				data, err := json.Marshal(args.A + args.B)
				return string(data), err
			}),
		)

		server := NewServer(sourceRegistry)
		c, err := client.NewInProcessClient(server)
		require.NoError(t, err)

		ctx := context.Background()
		remoteRegistry, err := NewRemoteRegistryFromClient(ctx, c)
		require.NoError(t, err)
		defer remoteRegistry.Close()

		// Execute a tool
		result, err := remoteRegistry.Execute(ctx, ai.ToolCall{
			ID:        "call_123",
			Name:      "add",
			Arguments: `{"a": 10, "b": 5}`,
		})
		require.NoError(t, err)

		assert.Equal(t, "call_123", result.ToolCallID)
		assert.Equal(t, "15", result.Content)
		assert.False(t, result.IsError)
	})

	t.Run("refreshes tool list", func(t *testing.T) {
		sourceRegistry := tool.NewRegistry().Add(
			tool.Func("initial", "Initial tool", func(ctx context.Context, args struct{}) (string, error) {
				return "ok", nil
			}),
		)

		server := NewServer(sourceRegistry)
		c, err := client.NewInProcessClient(server)
		require.NoError(t, err)

		ctx := context.Background()
		remoteRegistry, err := NewRemoteRegistryFromClient(ctx, c)
		require.NoError(t, err)
		defer remoteRegistry.Close()

		assert.Equal(t, 1, remoteRegistry.Len())

		// Refresh should work (even though tools haven't changed in this test)
		err = remoteRegistry.Refresh(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, remoteRegistry.Len())
	})
}
