package main

import (
	"context"
	"fmt"
	"os"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/tool"
)

// Tool argument types with schema tags
type WeatherArgs struct {
	Location string `json:"location" desc:"The city name, e.g. San Francisco" required:"true"`
}

type CalculateArgs struct {
	Expression string `json:"expression" desc:"The mathematical expression to evaluate, e.g. '2 + 2'" required:"true"`
}

type TimeArgs struct{}

func demoAgentStream(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│          Agent Stream Demo              │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Create a registry and register tools with typed handlers
	registry := tool.NewRegistry()

	// Weather tool
	tool.MustRegisterFunc(registry, "get_weather", "Get the current weather for a location",
		func(ctx context.Context, args WeatherArgs) (string, error) {
			// Simulate weather API
			return fmt.Sprintf(`{"location": %q, "temperature": 22, "unit": "celsius", "conditions": "Partly cloudy"}`, args.Location), nil
		},
	)

	// Calculator tool
	tool.MustRegisterFunc(registry, "calculate", "Perform a mathematical calculation",
		func(ctx context.Context, args CalculateArgs) (string, error) {
			// Simulate calculation (in real implementation, use a math parser)
			return fmt.Sprintf(`{"expression": %q, "result": 42}`, args.Expression), nil
		},
	)

	// Create the agent
	a := agent.New(c, registry)

	// Run with streaming events
	fmt.Println("\nUser: What's the weather in Tokyo? Also, what is 21 * 2?")
	fmt.Println("\n--- Agent Execution ---")

	events := a.RunStream(ctx, []ai.Message{
		{Role: ai.RoleUser, Content: "What's the weather in Tokyo? Also, what is 21 * 2?"},
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

	// Create a registry with a single tool using typed handler
	registry := tool.NewRegistry()
	tool.MustRegisterFunc(registry, "get_time", "Get the current time",
		func(ctx context.Context, args TimeArgs) (string, error) {
			return fmt.Sprintf(`{"time": %q}`, time.Now().Format(time.RFC3339)), nil
		},
	)

	a := agent.New(c, registry)

	fmt.Println("\nUser: What time is it?")

	result, err := a.Run(ctx, []ai.Message{
		{Role: ai.RoleUser, Content: "What time is it?"},
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
