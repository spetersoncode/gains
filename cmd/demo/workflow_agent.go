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
	"github.com/spetersoncode/gains/workflow"
)

// CalculatorArgs defines arguments for the calculator tool.
type CalculatorArgs struct {
	Expression string `json:"expression" desc:"A mathematical expression to evaluate, e.g. '2 + 2' or '10 * 5'" required:"true"`
}

// LookupArgs defines arguments for the lookup tool.
type LookupArgs struct {
	Key string `json:"key" desc:"The key to look up" required:"true"`
}

func demoWorkflowToolStep(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│       Workflow ToolStep Demo            │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println("\nThis demo shows direct tool execution in a workflow:")
	fmt.Println("  1. Store data in state")
	fmt.Println("  2. Execute a tool directly (no LLM)")
	fmt.Println("  3. Use the tool result in subsequent steps")

	// Create a registry with a simple lookup tool
	registry := tool.NewRegistry()
	tool.MustRegisterFunc(registry, "lookup", "Look up a value by key",
		func(ctx context.Context, args LookupArgs) (string, error) {
			// Simulated data store
			data := map[string]string{
				"pi":     "3.14159",
				"e":      "2.71828",
				"phi":    "1.61803",
				"answer": "42",
			}
			if val, ok := data[args.Key]; ok {
				return val, nil
			}
			return "", fmt.Errorf("key not found: %s", args.Key)
		},
	)

	// Step 1: Set which key to look up
	step1 := workflow.NewFuncStep("setup", func(ctx context.Context, state *workflow.State) error {
		state.Set("lookup_key", "phi")
		fmt.Println("  Set lookup_key = 'phi'")
		return nil
	})

	// Step 2: Execute the tool directly
	step2 := workflow.NewToolStep(
		"lookup-constant",
		registry,
		"lookup",
		func(s *workflow.State) (any, error) {
			return LookupArgs{Key: s.GetString("lookup_key")}, nil
		},
		"constant_value",
	)

	// Step 3: Use the result
	step3 := workflow.NewPromptStep("explain", c,
		func(s *workflow.State) []ai.Message {
			key := s.GetString("lookup_key")
			value := s.GetString("constant_value")
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf(
					"The mathematical constant '%s' has the value %s. In one sentence, explain what this constant represents.",
					key, value,
				)},
			}
		},
		"explanation",
	)

	// Create the chain
	chain := workflow.NewChain("tool-chain", step1, step2, step3)
	wf := workflow.New("tool-workflow", chain)

	// Run with streaming
	fmt.Println("\n--- Executing Chain ---")
	state := workflow.NewState(nil)
	events := wf.RunStream(ctx, state, workflow.WithTimeout(time.Minute))

	for event := range events {
		switch event.Type {
		case workflow.EventStepStart:
			fmt.Printf("\n[%s] Starting...\n", event.StepName)
		case workflow.EventStreamDelta:
			fmt.Print(event.Delta)
		case workflow.EventStepComplete:
			if event.Result != nil && event.StepName == "lookup-constant" {
				fmt.Printf("  Tool result: %v\n", event.Result.Output)
			}
		case workflow.EventError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", event.Error)
			return
		}
	}

	fmt.Println("\n\n--- Results ---")
	fmt.Printf("Key: %s\n", state.GetString("lookup_key"))
	fmt.Printf("Value: %s\n", state.GetString("constant_value"))
	fmt.Printf("Explanation: %s\n", state.GetString("explanation"))
}

func demoWorkflowAgentStep(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│       Workflow AgentStep Demo           │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println("\nThis demo shows an autonomous agent within a workflow:")
	fmt.Println("  1. Set up a math problem")
	fmt.Println("  2. Run an agent that can use a calculator tool")
	fmt.Println("  3. Summarize the agent's work")

	// Create a registry with a calculator tool
	registry := tool.NewRegistry()
	tool.MustRegisterFunc(registry, "calculate", "Evaluate a mathematical expression",
		func(ctx context.Context, args CalculatorArgs) (string, error) {
			// Simple expression evaluator (in production, use a proper parser)
			var result float64
			_, err := fmt.Sscanf(args.Expression, "%f", &result)
			if err != nil {
				// Try basic operations
				var a, b float64
				var op string
				if _, err := fmt.Sscanf(args.Expression, "%f %s %f", &a, &op, &b); err == nil {
					switch op {
					case "+":
						result = a + b
					case "-":
						result = a - b
					case "*":
						result = a * b
					case "/":
						if b != 0 {
							result = a / b
						} else {
							return "Error: division by zero", nil
						}
					default:
						return fmt.Sprintf("Unknown operator: %s", op), nil
					}
				} else {
					return fmt.Sprintf("Could not parse expression: %s", args.Expression), nil
				}
			}
			return fmt.Sprintf("%.4f", result), nil
		},
	)

	// Step 1: Set up the problem
	step1 := workflow.NewFuncStep("setup", func(ctx context.Context, state *workflow.State) error {
		state.Set("problem", "Calculate the area of a rectangle with width 7.5 and height 12.3")
		return nil
	})

	// Step 2: Run the agent
	step2 := workflow.NewAgentStep(
		"solver",
		c,
		registry,
		func(s *workflow.State) []ai.Message {
			problem := s.GetString("problem")
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf(
					"Solve this problem using the calculator tool:\n\n%s\n\nShow your work by using the calculator, then provide the final answer.",
					problem,
				)},
			}
		},
		"agent_result",
		[]agent.Option{agent.WithMaxSteps(3)},
	)

	// Step 3: Summarize
	step3 := workflow.NewPromptStep("summarize", c,
		func(s *workflow.State) []ai.Message {
			result, _ := s.Get("agent_result")
			agentResult := result.(*workflow.AgentResult)
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf(
					"The agent solved a math problem. It took %d steps and concluded: %s\n\nSummarize what happened in one sentence.",
					agentResult.Steps, agentResult.Response.Content,
				)},
			}
		},
		"summary",
	)

	// Create the chain
	chain := workflow.NewChain("agent-chain", step1, step2, step3)
	wf := workflow.New("agent-workflow", chain)

	// Run with streaming
	fmt.Println("\n--- Executing Chain ---")
	state := workflow.NewState(nil)
	events := wf.RunStream(ctx, state, workflow.WithTimeout(2*time.Minute))

	currentStep := ""
	for event := range events {
		switch event.Type {
		case workflow.EventStepStart:
			if event.AgentStep == 0 {
				currentStep = event.StepName
				fmt.Printf("\n[%s] Starting...\n", currentStep)
			} else {
				fmt.Printf("  Agent iteration %d...\n", event.AgentStep)
			}
		case workflow.EventStreamDelta:
			fmt.Print(event.Delta)
		case workflow.EventToolCall:
			if event.Message == "requested" && event.ToolCall != nil {
				fmt.Printf("\n  📞 Tool call: %s(%s)\n", event.ToolCall.Name, event.ToolCall.Arguments)
			}
		case workflow.EventToolResult:
			if event.ToolResult != nil {
				fmt.Printf("  📋 Result: %s\n", event.ToolResult.Content)
			}
		case workflow.EventStepComplete:
			fmt.Println()
		case workflow.EventError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", event.Error)
			return
		}
	}

	fmt.Println("\n--- Results ---")
	fmt.Printf("Problem: %s\n", state.GetString("problem"))
	if result, ok := state.Get("agent_result"); ok {
		agentResult := result.(*workflow.AgentResult)
		fmt.Printf("Agent steps: %d\n", agentResult.Steps)
		fmt.Printf("Termination: %s\n", agentResult.Termination)
	}
	fmt.Printf("Summary: %s\n", state.GetString("summary"))
}
