// Package mcp provides MCP (Model Context Protocol) integration for gains.
//
// MCP is a protocol that enables AI assistants to access external tools and data.
// This package provides bidirectional integration:
//
//   - Server: Expose a gains [tool.Registry] as an MCP server, allowing MCP clients
//     like Claude Desktop to discover and use your tools.
//   - Client: Connect to MCP servers and use their tools through [RemoteRegistry],
//     which can be used with gains agents.
//
// # Exposing Tools as an MCP Server
//
// To expose your gains tools to MCP clients:
//
//	registry := tool.NewRegistry().Add(
//	    tool.Func("weather", "Get weather", weatherHandler),
//	    tool.Func("search", "Search web", searchHandler),
//	)
//
//	// Serve over stdio (for subprocess-based MCP clients)
//	if err := mcp.ServeStdio(registry); err != nil {
//	    log.Fatal(err)
//	}
//
// # Consuming MCP Servers
//
// To use tools from an MCP server with a gains agent:
//
//	// Connect to an MCP server
//	remote, err := mcp.NewRemoteRegistry(ctx, "./my-mcp-server", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer remote.Close()
//
//	// Use the remote tools with an agent
//	agent := agent.New(client)
//	for _, t := range remote.Tools() {
//	    agent.RegisterTool(t, remote.Execute)
//	}
package mcp

import (
	"encoding/json"
	"strings"

	ai "github.com/spetersoncode/gains"
	"github.com/mark3labs/mcp-go/mcp"
)

// ToMCPTool converts a gains Tool to an MCP Tool.
// The gains Tool.Parameters JSON schema is used as the MCP Tool's RawInputSchema.
func ToMCPTool(t ai.Tool) mcp.Tool {
	return mcp.NewToolWithRawSchema(t.Name, t.Description, t.Parameters)
}

// ToMCPTools converts a slice of gains Tools to MCP Tools.
func ToMCPTools(tools []ai.Tool) []mcp.Tool {
	result := make([]mcp.Tool, len(tools))
	for i, t := range tools {
		result[i] = ToMCPTool(t)
	}
	return result
}

// FromMCPTool converts an MCP Tool to a gains Tool.
// It extracts the JSON schema from either RawInputSchema or InputSchema.
func FromMCPTool(t mcp.Tool) ai.Tool {
	var schema json.RawMessage

	// Prefer raw schema if available
	if len(t.RawInputSchema) > 0 {
		schema = t.RawInputSchema
	} else {
		// Marshal the structured schema
		data, err := json.Marshal(t.InputSchema)
		if err == nil {
			schema = data
		}
	}

	return ai.Tool{
		Name:        t.Name,
		Description: t.Description,
		Parameters:  schema,
	}
}

// FromMCPTools converts a slice of MCP Tools to gains Tools.
func FromMCPTools(tools []mcp.Tool) []ai.Tool {
	result := make([]ai.Tool, len(tools))
	for i, t := range tools {
		result[i] = FromMCPTool(t)
	}
	return result
}

// ToMCPCallToolRequest converts a gains ToolCall to an MCP CallToolRequest.
func ToMCPCallToolRequest(call ai.ToolCall) mcp.CallToolRequest {
	var args any
	if call.Arguments != "" {
		// Try to parse as JSON
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			// If not valid JSON, use the string directly
			args = call.Arguments
		}
	}

	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      call.Name,
			Arguments: args,
		},
	}
}

// FromMCPCallToolResult converts an MCP CallToolResult to a gains ToolResult.
// The result content is extracted and concatenated as text.
func FromMCPCallToolResult(callID string, result *mcp.CallToolResult) ai.ToolResult {
	if result == nil {
		return ai.ToolResult{
			ToolCallID: callID,
			Content:    "",
			IsError:    true,
		}
	}

	// Extract text content from result
	var textParts []string
	for _, c := range result.Content {
		switch content := c.(type) {
		case mcp.TextContent:
			textParts = append(textParts, content.Text)
		case *mcp.TextContent:
			textParts = append(textParts, content.Text)
		default:
			// For non-text content, try to marshal as JSON
			if data, err := json.Marshal(content); err == nil {
				textParts = append(textParts, string(data))
			}
		}
	}

	// If there's structured content, include it
	if result.StructuredContent != nil {
		if data, err := json.Marshal(result.StructuredContent); err == nil {
			textParts = append(textParts, string(data))
		}
	}

	return ai.ToolResult{
		ToolCallID: callID,
		Content:    strings.Join(textParts, "\n"),
		IsError:    result.IsError,
	}
}

// ToMCPCallToolResult converts a gains ToolResult to an MCP CallToolResult.
func ToMCPCallToolResult(result ai.ToolResult) *mcp.CallToolResult {
	if result.IsError {
		return mcp.NewToolResultError(result.Content)
	}
	return mcp.NewToolResultText(result.Content)
}
