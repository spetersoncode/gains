package main

import (
	"context"
	"fmt"
	"os"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
)

// ToolWeatherArgs defines the parameters for the weather tool
type ToolWeatherArgs struct {
	Location string `json:"location"`
	Unit     string `json:"unit"`
}

func demoToolCalling(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│          Tool Calling Demo              │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Define a weather tool using struct-based schema generation
	tools := []ai.Tool{
		{
			Name:        "get_weather",
			Description: "Get the current weather for a location",
			Parameters: ai.SchemaFrom[ToolWeatherArgs]().
				Desc("location", "The city name, e.g. San Francisco").
				Required("location").
				Desc("unit", "The temperature unit").
				Enum("unit", "celsius", "fahrenheit").
				Build(),
		},
	}

	messages := []ai.Message{
		{Role: ai.RoleUser, Content: "What's the weather like in Tokyo?"},
	}

	fmt.Println("User: What's the weather like in Tokyo?")
	fmt.Println("Tools available: get_weather(location, unit)")
	fmt.Println()

	// First call: model should request tool use
	resp, err := c.Chat(ctx, messages, ai.WithTools(tools))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	if len(resp.ToolCalls) == 0 {
		fmt.Println("Model response (no tool call):", resp.Content)
		return
	}

	// Show the tool call
	tc := resp.ToolCalls[0]
	fmt.Printf("Model requested tool: %s\n", tc.Name)
	fmt.Printf("Arguments: %s\n", tc.Arguments)

	// Simulate tool execution
	toolResult := `{"temperature": 22, "unit": "celsius", "conditions": "Partly cloudy"}`
	fmt.Printf("Tool result: %s\n", toolResult)

	// Continue conversation with tool result
	messages = append(messages,
		ai.Message{
			Role:      ai.RoleAssistant,
			ToolCalls: resp.ToolCalls,
		},
		ai.NewToolResultMessage(ai.ToolResult{
			ToolCallID: tc.ID,
			Content:    toolResult,
		}),
	)

	// Second call: model should use the tool result
	finalResp, err := c.Chat(ctx, messages, ai.WithTools(tools))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return
	}

	fmt.Printf("Final response: %s\n", finalResp.Content)
	fmt.Printf("[Tokens: %d in, %d out]\n", finalResp.Usage.InputTokens, finalResp.Usage.OutputTokens)
}
