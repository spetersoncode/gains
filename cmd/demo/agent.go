package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/client"
)

func demoAgentStream(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│          Agent Stream Demo              │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Create a registry and register tools
	registry := agent.NewRegistry()

	// Weather tool
	registry.MustRegister(
		gains.Tool{
			Name:        "get_weather",
			Description: "Get the current weather for a location",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"location": {
						"type": "string",
						"description": "The city name, e.g. San Francisco"
					}
				},
				"required": ["location"]
			}`),
		},
		func(ctx context.Context, call gains.ToolCall) (string, error) {
			var args struct {
				Location string `json:"location"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
				return "", err
			}
			// Simulate weather API
			return fmt.Sprintf(`{"location": %q, "temperature": 22, "unit": "celsius", "conditions": "Partly cloudy"}`, args.Location), nil
		},
	)

	// Calculator tool
	registry.MustRegister(
		gains.Tool{
			Name:        "calculate",
			Description: "Perform a mathematical calculation",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"expression": {
						"type": "string",
						"description": "The mathematical expression to evaluate, e.g. '2 + 2'"
					}
				},
				"required": ["expression"]
			}`),
		},
		func(ctx context.Context, call gains.ToolCall) (string, error) {
			var args struct {
				Expression string `json:"expression"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
				return "", err
			}
			// Simulate calculation (in real implementation, use a math parser)
			return fmt.Sprintf(`{"expression": %q, "result": 42}`, args.Expression), nil
		},
	)

	// Create the agent
	a := agent.New(c, registry)

	// Run with streaming events
	fmt.Println("\nUser: What's the weather in Tokyo? Also, what is 21 * 2?")
	fmt.Println("\n--- Agent Execution ---")

	events := a.RunStream(ctx, []gains.Message{
		{Role: gains.RoleUser, Content: "What's the weather in Tokyo? Also, what is 21 * 2?"},
	},
		agent.WithMaxSteps(5),
		agent.WithTimeout(2*time.Minute),
	)

	// Process events
	for event := range events {
		switch event.Type {
		case agent.EventStepStart:
			fmt.Printf("\n[Step %d]\n", event.Step)

		case agent.EventStreamDelta:
			fmt.Print(event.Delta)

		case agent.EventToolCallRequested:
			fmt.Printf("\n  -> Tool requested: %s(%s)\n", event.ToolCall.Name, event.ToolCall.Arguments)

		case agent.EventToolResult:
			status := "success"
			if event.ToolResult.IsError {
				status = "error"
			}
			fmt.Printf("  <- Tool result [%s]: %s\n", status, truncate(event.ToolResult.Content, 80))

		case agent.EventStepComplete:
			if event.Response != nil {
				fmt.Printf("\n  [Tokens: %d in, %d out]\n",
					event.Response.Usage.InputTokens,
					event.Response.Usage.OutputTokens)
			}

		case agent.EventAgentComplete:
			fmt.Printf("\n\n--- Agent Complete ---\n")
			fmt.Printf("Termination: %s\n", event.Message)
			if event.Response != nil {
				fmt.Printf("Final response: %s\n", event.Response.Content)
			}

		case agent.EventError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", event.Error)
		}
	}
}

func demoAgent(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│              Agent Demo                 │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Create a registry with a single tool
	registry := agent.NewRegistry()
	registry.MustRegister(
		gains.Tool{
			Name:        "get_time",
			Description: "Get the current time",
			Parameters:  json.RawMessage(`{"type": "object", "properties": {}}`),
		},
		func(ctx context.Context, call gains.ToolCall) (string, error) {
			return fmt.Sprintf(`{"time": %q}`, time.Now().Format(time.RFC3339)), nil
		},
	)

	a := agent.New(c, registry)

	fmt.Println("\nUser: What time is it?")

	result, err := a.Run(ctx, []gains.Message{
		{Role: gains.RoleUser, Content: "What time is it?"},
	}, agent.WithMaxSteps(3))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Printf("\nAgent response: %s\n", result.Response.Content)
	fmt.Printf("Steps: %d\n", result.Steps)
	fmt.Printf("Termination: %s\n", result.Termination)
	fmt.Printf("Total tokens: %d in, %d out\n", result.TotalUsage.InputTokens, result.TotalUsage.OutputTokens)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
