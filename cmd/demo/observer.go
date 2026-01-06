package main

import (
	"context"
	"fmt"
	"os"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/model"
	"github.com/spetersoncode/gains/observer"
	"github.com/spetersoncode/gains/tool"
)

func demoObserver(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│     Observer Demo - Console Output      │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Create tools
	registry := tool.NewRegistry()
	tool.MustRegisterFunc(registry, "get_weather", "Get the current weather for a location",
		func(ctx context.Context, args WeatherArgs) (string, error) {
			time.Sleep(100 * time.Millisecond) // Simulate API call
			return fmt.Sprintf(`{"location": %q, "temperature": 18, "unit": "celsius", "conditions": "Sunny"}`, args.Location), nil
		},
	)
	tool.MustRegisterFunc(registry, "calculate", "Perform a mathematical calculation",
		func(ctx context.Context, args CalculateArgs) (string, error) {
			return fmt.Sprintf(`{"expression": %q, "result": 42}`, args.Expression), nil
		},
	)

	// Create Console observer with cost tracking and verbose output
	// Using Claude Sonnet pricing for demonstration (adjust based on actual model)
	obs := observer.NewStdoutConsole(
		observer.WithCostTracking(model.ClaudeSonnet45),
		observer.WithVerbose(),
	)
	defer obs.Close()

	fmt.Println("\nThis demo shows the Console observer with cost tracking.")
	fmt.Println("The observer captures all events and displays them in real-time.")
	fmt.Println("\n─── Observer Output ───")

	// Create agent with observer
	a := agent.New(c, registry)

	// Run agent with observer option
	result, err := a.Run(ctx, []ai.Message{
		{Role: ai.RoleUser, Content: "What's the weather in Paris? Also calculate 21 * 2."},
	},
		agent.WithObserver(obs),
		agent.WithMaxSteps(5),
		agent.WithTimeout(2*time.Minute),
	)

	fmt.Println("───────────────────────")

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		return
	}

	// Show final results
	fmt.Printf("\nAgent response: %s\n", result.Response.Content)
	fmt.Printf("Steps completed: %d\n", result.Steps)
	fmt.Printf("Termination: %s\n", result.Termination)

	// Show cost summary from observer
	if summary := obs.CostSummary(); summary != nil {
		fmt.Printf("\n─── Cost Summary ───\n")
		fmt.Printf("Input tokens:  %d\n", summary.InputTokens)
		fmt.Printf("Output tokens: %d\n", summary.OutputTokens)
		fmt.Printf("Total tokens:  %d\n", summary.TotalTokens)
		fmt.Printf("Estimated cost: $%.6f\n", summary.Cost)
	}
}

func demoObserverFile(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│    Observer Demo - Multi + File Output  │")
	fmt.Println("└─────────────────────────────────────────┘")

	// Create tools
	registry := tool.NewRegistry()
	tool.MustRegisterFunc(registry, "get_time", "Get the current time",
		func(ctx context.Context, args TimeArgs) (string, error) {
			return fmt.Sprintf(`{"time": %q}`, time.Now().Format(time.RFC3339)), nil
		},
	)

	// Create a temp file for JSON output
	tmpFile, err := os.CreateTemp("", "gains-observer-*.jsonl")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp file: %v\n", err)
		return
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Create File observer
	fileObs, err := observer.NewFile(tmpPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file observer: %v\n", err)
		return
	}

	// Create Multi observer combining Console (stderr) and File
	consoleObs := observer.NewConsole(os.Stderr)
	multiObs := observer.NewMulti(consoleObs, fileObs)
	defer multiObs.Close()

	fmt.Println("\nThis demo shows the Multi observer combining:")
	fmt.Println("  1. Console observer (human-readable to stderr)")
	fmt.Println("  2. File observer (JSON lines to temp file)")
	fmt.Printf("\nJSON output will be written to: %s\n", tmpPath)
	fmt.Println("\n─── Observer Output (stderr) ───")

	// Create agent with observer
	a := agent.New(c, registry)

	// Run agent
	result, err := a.Run(ctx, []ai.Message{
		{Role: ai.RoleUser, Content: "What time is it right now?"},
	},
		agent.WithObserver(multiObs),
		agent.WithMaxSteps(3),
	)

	fmt.Println("────────────────────────────────")

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		return
	}

	fmt.Printf("\nAgent response: %s\n", result.Response.Content)

	// Read and display the JSON file contents
	fmt.Println("\n─── JSON File Contents ───")
	jsonContent, err := os.ReadFile(tmpPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading JSON file: %v\n", err)
		return
	}
	fmt.Println(string(jsonContent))

	fmt.Println("Note: Each line is a JSON object with trace context fields (trace_id, span_id)")
}
