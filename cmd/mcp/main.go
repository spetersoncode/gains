// Command mcp is a reference MCP server that exposes gains tools over stdio.
//
// This server demonstrates how to expose a gains tool.Registry as an MCP server,
// allowing MCP clients (like Claude Desktop or other AI assistants) to discover
// and use the tools.
//
// Usage:
//
//	go run ./cmd/mcp
//
// Configuration for Claude Desktop (~/Library/Application Support/Claude/claude_desktop_config.json):
//
//	{
//	    "mcpServers": {
//	        "gains-tools": {
//	            "command": "go",
//	            "args": ["run", "./cmd/mcp"],
//	            "cwd": "/path/to/gains"
//	        }
//	    }
//	}
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spetersoncode/gains/mcp"
	"github.com/spetersoncode/gains/tool"
)

func main() {
	// Create a registry with example tools
	registry := tool.NewRegistry().Add(
		tool.Func("echo", "Echo back the input text", echoHandler),
		tool.Func("time", "Get the current time", timeHandler),
		tool.Func("calculate", "Perform basic arithmetic", calculateHandler),
	)

	// Serve the tools over MCP stdio
	if err := mcp.ServeStdio(registry,
		mcp.WithName("gains-mcp-example"),
		mcp.WithVersion("1.0.0"),
	); err != nil {
		log.Fatal(err)
	}
}

// EchoArgs are the arguments for the echo tool.
type EchoArgs struct {
	Text string `json:"text" desc:"The text to echo back" required:"true"`
}

func echoHandler(ctx context.Context, args EchoArgs) (string, error) {
	return args.Text, nil
}

// TimeArgs are the arguments for the time tool.
type TimeArgs struct {
	Format string `json:"format" desc:"Time format (optional): 'rfc3339', 'unix', or 'human'" default:"human"`
}

func timeHandler(ctx context.Context, args TimeArgs) (string, error) {
	now := time.Now()

	switch strings.ToLower(args.Format) {
	case "rfc3339":
		return now.Format(time.RFC3339), nil
	case "unix":
		return fmt.Sprintf("%d", now.Unix()), nil
	default:
		return now.Format("Monday, January 2, 2006 at 3:04 PM MST"), nil
	}
}

// CalculateArgs are the arguments for the calculate tool.
type CalculateArgs struct {
	Operation string  `json:"operation" desc:"The operation to perform" enum:"add,subtract,multiply,divide" required:"true"`
	A         float64 `json:"a" desc:"First number" required:"true"`
	B         float64 `json:"b" desc:"Second number" required:"true"`
}

func calculateHandler(ctx context.Context, args CalculateArgs) (string, error) {
	var result float64

	switch args.Operation {
	case "add":
		result = args.A + args.B
	case "subtract":
		result = args.A - args.B
	case "multiply":
		result = args.A * args.B
	case "divide":
		if args.B == 0 {
			return "", fmt.Errorf("cannot divide by zero")
		}
		result = args.A / args.B
	default:
		return "", fmt.Errorf("unknown operation: %s", args.Operation)
	}

	return fmt.Sprintf("%.6g", result), nil
}
