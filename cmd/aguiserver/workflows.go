package main

import (
	"context"

	"github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/workflow"
)

// GreetingState is the state for the greeting demo workflow.
type GreetingState struct {
	Name     string `json:"name"`
	Style    string `json:"style"` // e.g., "formal", "casual", "pirate"
	Greeting string `json:"greeting"`
}

// SetupDemoWorkflows creates a workflow registry with demo workflows.
func SetupDemoWorkflows(c *client.Client) *workflow.Registry {
	registry := workflow.NewRegistry()

	// Register the greeting workflow
	greetingWorkflow := createGreetingWorkflow(c)
	registry.Register(workflow.NewRunnerJSON[GreetingState]("greeting", greetingWorkflow))

	return registry
}

// createGreetingWorkflow creates a simple workflow that generates a styled greeting.
func createGreetingWorkflow(c *client.Client) workflow.Step[GreetingState] {
	// Step 1: Validate input and set defaults
	validate := workflow.NewFuncStep("validate", func(ctx context.Context, state *GreetingState) error {
		if state.Name == "" {
			state.Name = "friend"
		}
		if state.Style == "" {
			state.Style = "casual"
		}
		return nil
	})

	// Step 2: Generate greeting using LLM
	generateGreeting := workflow.NewPromptStep[GreetingState, string](
		"generate",
		c,
		func(state *GreetingState) []gains.Message {
			return []gains.Message{
				{
					Role:    "user",
					Content: "Generate a " + state.Style + " greeting for " + state.Name + ". Keep it to one sentence.",
				},
			}
		},
		nil, // No schema - plain text response
		func(state *GreetingState) *string { return &state.Greeting },
	)

	// Chain the steps
	return workflow.NewChain("greeting-workflow", validate, generateGreeting)
}
