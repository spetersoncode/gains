package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/workflow"
)

func demoWorkflowChain(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│         Workflow Chain Demo             │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println("\nThis demo shows a sequential chain workflow:")
	fmt.Println("  1. Generate a topic")
	fmt.Println("  2. Write a haiku about that topic")
	fmt.Println("  3. Translate the haiku to another style")

	// Step 1: Generate a random topic
	step1 := workflow.NewPromptStep("generate-topic", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: "Give me one random nature topic in 1-2 words only. Just the topic, nothing else."},
			}
		},
		"topic",
	)

	// Step 2: Write a haiku about the topic
	step2 := workflow.NewPromptStep("write-haiku", c,
		func(s *workflow.State) []ai.Message {
			topic := s.GetString("topic")
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf("Write a haiku about: %s\n\nJust the haiku, no explanation.", topic)},
			}
		},
		"haiku",
	)

	// Step 3: Transform the haiku
	step3 := workflow.NewPromptStep("transform", c,
		func(s *workflow.State) []ai.Message {
			haiku := s.GetString("haiku")
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf("Take this haiku and rewrite it in a modern, humorous style while keeping the same theme:\n\n%s\n\nJust the new version, no explanation.", haiku)},
			}
		},
		"transformed",
	)

	// Create the chain
	chain := workflow.NewChain("haiku-pipeline", step1, step2, step3)
	wf := workflow.New("haiku-workflow", chain)

	// Run with streaming
	fmt.Println("\n--- Executing Chain ---")
	state := workflow.NewState(nil)
	events := wf.RunStream(ctx, state, workflow.WithTimeout(2*time.Minute))

	currentStep := ""
	for event := range events {
		switch event.Type {
		case workflow.EventStepStart:
			currentStep = event.StepName
			fmt.Printf("\n[%s] Starting...\n", currentStep)
		case workflow.EventStreamDelta:
			fmt.Print(event.Delta)
		case workflow.EventStepComplete:
			fmt.Println()
		case workflow.EventError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", event.Error)
			return
		}
	}

	fmt.Println("\n--- Results ---")
	fmt.Printf("Topic: %s\n", strings.TrimSpace(state.GetString("topic")))
	fmt.Printf("\nOriginal Haiku:\n%s\n", state.GetString("haiku"))
	fmt.Printf("\nModern Version:\n%s\n", state.GetString("transformed"))
}
