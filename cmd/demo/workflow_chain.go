package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/workflow"
)

// ChainState is the state struct for the chain workflow demo.
type ChainState struct {
	Topic       string
	Haiku       string
	Transformed string
}

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
		func(s *ChainState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: "Give me one random nature topic in 1-2 words only. Just the topic, nothing else."},
			}
		},
		nil,
		func(s *ChainState) *string { return &s.Topic },
	)

	// Step 2: Write a haiku about the topic
	step2 := workflow.NewPromptStep("write-haiku", c,
		func(s *ChainState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf("Write a haiku about: %s\n\nJust the haiku, no explanation.", strings.TrimSpace(s.Topic))},
			}
		},
		nil,
		func(s *ChainState) *string { return &s.Haiku },
	)

	// Step 3: Transform the haiku
	step3 := workflow.NewPromptStep("transform", c,
		func(s *ChainState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf("Take this haiku and rewrite it in a modern, humorous style while keeping the same theme:\n\n%s\n\nJust the new version, no explanation.", s.Haiku)},
			}
		},
		nil,
		func(s *ChainState) *string { return &s.Transformed },
	)

	// Create the chain
	chain := workflow.NewChain("haiku-pipeline", step1, step2, step3)
	wf := workflow.New("haiku-workflow", chain)

	// Run with streaming
	fmt.Println("\n--- Executing Chain ---")
	state := &ChainState{}
	events := wf.RunStream(ctx, state, workflow.WithTimeout(2*time.Minute))

	currentStep := ""
	for ev := range events {
		switch ev.Type {
		case event.StepStart:
			currentStep = ev.StepName
			fmt.Printf("\n[%s] Starting...\n", currentStep)
		case event.MessageDelta:
			fmt.Print(ev.Delta)
		case event.StepEnd:
			fmt.Println()
		case event.RunError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", ev.Error)
			return
		}
	}

	fmt.Println("\n--- Results ---")
	fmt.Printf("Topic: %s\n", state.Topic)
	fmt.Printf("\nOriginal Haiku:\n%s\n", state.Haiku)
	fmt.Printf("\nModern Version:\n%s\n", state.Transformed)
}
