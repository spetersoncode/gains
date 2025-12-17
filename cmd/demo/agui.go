package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/agui"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/tool"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
)

func demoAGUIStream(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│          AG-UI Event Stream Demo        │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("This demo shows how gains events map to AG-UI protocol events.")
	fmt.Println("The output simulates Server-Sent Events (SSE) format.")
	fmt.Println()

	// Create a registry with tools
	registry := tool.NewRegistry()

	tool.MustRegisterFunc(registry, "get_weather", "Get the current weather for a location",
		func(ctx context.Context, args WeatherArgs) (string, error) {
			time.Sleep(100 * time.Millisecond) // Simulate API latency
			return fmt.Sprintf(`{"location": %q, "temperature": 22, "conditions": "Sunny"}`, args.Location), nil
		},
	)

	tool.MustRegisterFunc(registry, "get_time", "Get the current time",
		func(ctx context.Context, args TimeArgs) (string, error) {
			return fmt.Sprintf(`{"time": %q}`, time.Now().Format(time.Kitchen)), nil
		},
	)

	// Create the agent
	a := agent.New(c, registry)

	fmt.Println("User: What's the weather in Paris and what time is it?")
	fmt.Println()
	fmt.Println("─── AG-UI Event Stream (SSE Format) ───")
	fmt.Println()

	// Create AG-UI mapper for this run
	mapper := agui.NewMapper("thread_demo_123", "run_demo_456")

	// Run agent with streaming
	// The agent emits RunStart which maps to RUN_STARTED automatically
	agentEvents := a.RunStream(ctx, []ai.Message{
		{Role: ai.RoleUser, Content: "What's the weather in Paris and what time is it?"},
	},
		agent.WithMaxSteps(5),
		agent.WithTimeout(2*time.Minute),
	)

	// Process and map events
	for ev := range agentEvents {
		aguiEvent := mapper.MapEvent(ev)
		if aguiEvent != nil {
			writeSSE(aguiEvent)
		}
	}

	fmt.Println()
	fmt.Println("─── End of Event Stream ───")
	fmt.Println()
	fmt.Println("In a real AG-UI server, these events would be sent over HTTP/SSE")
	fmt.Println("to an AG-UI-compatible frontend like CopilotKit.")
}

// writeSSE simulates writing an event in SSE format
func writeSSE(event events.Event) {
	data, err := event.ToJSON()
	if err != nil {
		fmt.Printf("error: failed to serialize event: %v\n\n", err)
		return
	}

	// Format as SSE
	fmt.Printf("event: %s\n", event.Type())
	fmt.Printf("data: %s\n\n", string(data))
}

func demoAGUIMessages(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│        AG-UI Message Conversion Demo    │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("This demo shows bidirectional message conversion between")
	fmt.Println("AG-UI format and gains format.")
	fmt.Println()

	// Example AG-UI messages (as they would arrive from a frontend)
	userContent := "What's the weather?"
	aguiMessages := []events.Message{
		{
			ID:      "msg_user_1",
			Role:    "user",
			Content: &userContent,
		},
	}

	fmt.Println("─── AG-UI Messages (from frontend) ───")
	printJSON(aguiMessages)

	// Convert to gains format
	gainsMessages := agui.ToGainsMessages(aguiMessages)

	fmt.Println("─── Converted to gains Messages ───")
	printJSON(gainsMessages)

	// Simulate an assistant response with tool call
	assistantResponse := []ai.Message{
		{
			Role:    ai.RoleUser,
			Content: "What's the weather?",
		},
		{
			Role: ai.RoleAssistant,
			ToolCalls: []ai.ToolCall{
				{
					ID:        "call_abc123",
					Name:      "get_weather",
					Arguments: `{"location": "Paris"}`,
				},
			},
		},
		{
			Role: ai.RoleTool,
			ToolResults: []ai.ToolResult{
				{
					ToolCallID: "call_abc123",
					Content:    `{"temperature": 22, "conditions": "Sunny"}`,
				},
			},
		},
		{
			Role:    ai.RoleAssistant,
			Content: "The weather in Paris is 22°C and sunny!",
		},
	}

	fmt.Println("─── gains Messages (conversation history) ───")
	printJSON(assistantResponse)

	// Convert back to AG-UI format
	aguiConverted := agui.FromGainsMessages(assistantResponse)

	fmt.Println("─── Converted back to AG-UI Messages ───")
	printJSON(aguiConverted)

	fmt.Println()
	fmt.Println("This bidirectional conversion enables:")
	fmt.Println("  • Receiving messages from AG-UI frontends")
	fmt.Println("  • Sending MESSAGES_SNAPSHOT events back to sync state")
}

func printJSON(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
	fmt.Println()
}
