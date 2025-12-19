package main

import (
	"context"
	"fmt"
	"os"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/event"
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

// ToolStepState is the state struct for the ToolStep workflow demo.
type ToolStepState struct {
	LookupKey     string
	ConstantValue string
	Explanation   string
}

// AgentStepState is the state struct for the AgentStep workflow demo.
type AgentStepState struct {
	Problem     string
	AgentResult *workflow.AgentResult
	Summary     string
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
	step1 := workflow.NewFuncStep[ToolStepState]("setup", func(ctx context.Context, state *ToolStepState) error {
		state.LookupKey = "phi"
		fmt.Println("  Set LookupKey = 'phi'")
		return nil
	})

	// Step 2: Execute the tool directly
	step2 := workflow.NewToolStep[ToolStepState](
		"lookup-constant",
		registry,
		"lookup",
		func(s *ToolStepState) (any, error) {
			return LookupArgs{Key: s.LookupKey}, nil
		},
		func(s *ToolStepState, result string) {
			s.ConstantValue = result
		},
	)

	// Step 3: Use the result
	step3 := workflow.NewPromptStep[ToolStepState]("explain", c,
		func(s *ToolStepState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf(
					"The mathematical constant '%s' has the value %s. In one sentence, explain what this constant represents.",
					s.LookupKey, s.ConstantValue,
				)},
			}
		},
		func(s *ToolStepState, content string) {
			s.Explanation = content
		},
	)

	// Create the chain
	chain := workflow.NewChain("tool-chain", step1, step2, step3)
	wf := workflow.New("tool-workflow", chain)

	// Run with streaming
	fmt.Println("\n--- Executing Chain ---")
	state := &ToolStepState{}
	events := wf.RunStream(ctx, state, workflow.WithTimeout(time.Minute))

	for ev := range events {
		switch ev.Type {
		case event.StepStart:
			fmt.Printf("\n[%s] Starting...\n", ev.StepName)
		case event.MessageDelta:
			fmt.Print(ev.Delta)
		case event.StepEnd:
			if ev.StepName == "lookup-constant" {
				// Tool result is stored in state
				fmt.Printf("  Tool result: %v\n", state.ConstantValue)
			}
		case event.RunError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", ev.Error)
			return
		}
	}

	fmt.Println("\n\n--- Results ---")
	fmt.Printf("Key: %s\n", state.LookupKey)
	fmt.Printf("Value: %s\n", state.ConstantValue)
	fmt.Printf("Explanation: %s\n", state.Explanation)
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
	step1 := workflow.NewFuncStep[AgentStepState]("setup", func(ctx context.Context, state *AgentStepState) error {
		state.Problem = "Calculate the area of a rectangle with width 7.5 and height 12.3"
		return nil
	})

	// Step 2: Run the agent
	step2 := workflow.NewAgentStep[AgentStepState](
		"solver",
		c,
		registry,
		func(s *AgentStepState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf(
					"Solve this problem using the calculator tool:\n\n%s\n\nShow your work by using the calculator, then provide the final answer.",
					s.Problem,
				)},
			}
		},
		func(s *AgentStepState, r *workflow.AgentResult) {
			s.AgentResult = r
		},
		[]agent.Option{agent.WithMaxSteps(3)},
	)

	// Step 3: Summarize
	step3 := workflow.NewPromptStep[AgentStepState]("summarize", c,
		func(s *AgentStepState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf(
					"The agent solved a math problem. It took %d steps and concluded: %s\n\nSummarize what happened in one sentence.",
					s.AgentResult.Steps, s.AgentResult.Response.Content,
				)},
			}
		},
		func(s *AgentStepState, content string) {
			s.Summary = content
		},
	)

	// Create the chain
	chain := workflow.NewChain("agent-chain", step1, step2, step3)
	wf := workflow.New("agent-workflow", chain)

	// Run with streaming
	fmt.Println("\n--- Executing Chain ---")
	state := &AgentStepState{}
	events := wf.RunStream(ctx, state, workflow.WithTimeout(2*time.Minute))

	currentStep := ""
	for ev := range events {
		switch ev.Type {
		case event.StepStart:
			if ev.Step == 0 {
				currentStep = ev.StepName
				fmt.Printf("\n[%s] Starting...\n", currentStep)
			} else {
				fmt.Printf("  Agent iteration %d...\n", ev.Step)
			}
		case event.MessageDelta:
			fmt.Print(ev.Delta)
		case event.ToolCallStart:
			if ev.ToolCall != nil {
				fmt.Printf("\n  Tool call: %s(%s)\n", ev.ToolCall.Name, ev.ToolCall.Arguments)
			}
		case event.ToolCallResult:
			if ev.ToolResult != nil {
				fmt.Printf("  Result: %s\n", ev.ToolResult.Content)
			}
		case event.StepEnd:
			fmt.Println()
		case event.RunError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", ev.Error)
			return
		}
	}

	fmt.Println("\n--- Results ---")
	fmt.Printf("Problem: %s\n", state.Problem)
	if state.AgentResult != nil {
		fmt.Printf("Agent steps: %d\n", state.AgentResult.Steps)
		fmt.Printf("Termination: %s\n", state.AgentResult.Termination)
	}
	fmt.Printf("Summary: %s\n", state.Summary)
}
