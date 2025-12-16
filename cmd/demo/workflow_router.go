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

func demoWorkflowRouter(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│         Workflow Router Demo            │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println("\nThis demo routes requests based on input type:")
	fmt.Println("  - Questions -> Answer step")
	fmt.Println("  - Statements -> Expansion step")
	fmt.Println("  - Other -> Default step")

	// Define steps for each route (concise 1-2 sentence responses)
	answerStep := workflow.NewPromptStep("answer", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf("Answer in 1-2 sentences: %s", s.GetString("input"))},
			}
		},
		"response",
	)

	expandStep := workflow.NewPromptStep("expand", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf("Expand on this in 1-2 sentences: %s", s.GetString("input"))},
			}
		},
		"response",
	)

	defaultStep := workflow.NewPromptStep("default", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf("Respond briefly (1-2 sentences): %s", s.GetString("input"))},
			}
		},
		"response",
	)

	// Create router with conditions
	router := workflow.NewRouter("input-router",
		[]workflow.Route{
			{
				Name: "question",
				Condition: func(ctx context.Context, s *workflow.State) bool {
					input := s.GetString("input")
					return strings.Contains(input, "?") || strings.HasPrefix(strings.ToLower(input), "what") ||
						strings.HasPrefix(strings.ToLower(input), "how") || strings.HasPrefix(strings.ToLower(input), "why")
				},
				Step: answerStep,
			},
			{
				Name: "statement",
				Condition: func(ctx context.Context, s *workflow.State) bool {
					input := s.GetString("input")
					return strings.HasSuffix(input, ".") && len(input) > 20
				},
				Step: expandStep,
			},
		},
		defaultStep,
	)

	wf := workflow.New("router-workflow", router)

	// Test cases (one question, one statement)
	testInputs := []string{
		"What is the speed of light?",
		"The ocean covers most of Earth's surface.",
	}

	for _, input := range testInputs {
		fmt.Printf("\n--- Input: %q ---\n", input)
		state := workflow.NewStateFrom(map[string]any{"input": input})

		events := wf.RunStream(ctx, state, workflow.WithTimeout(1*time.Minute))

		for event := range events {
			switch event.Type {
			case workflow.EventRouteSelected:
				fmt.Printf("Route selected: %s\n", event.RouteName)
			case workflow.EventStreamDelta:
				fmt.Print(event.Delta)
			case workflow.EventStepComplete:
				fmt.Println()
			case workflow.EventError:
				fmt.Fprintf(os.Stderr, "Error: %v\n", event.Error)
			}
		}
	}
}
