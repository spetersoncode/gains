package main

import (
	"context"
	"fmt"
	"os"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/workflow"
)

func demoWorkflowClassifier(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│      Workflow Classifier Demo           │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println("\nThis demo uses LLM classification to route support tickets:")
	fmt.Println("  - billing -> Billing handler")
	fmt.Println("  - technical -> Technical handler")
	fmt.Println("  - general -> General handler")

	// Define handlers for each category (concise responses)
	billingHandler := workflow.NewPromptStep("billing-handler", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleSystem, Content: "You are a billing support specialist. Respond in 2-3 sentences max."},
				{Role: ai.RoleUser, Content: s.GetString("ticket")},
			}
		},
		"response",
	)

	technicalHandler := workflow.NewPromptStep("technical-handler", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleSystem, Content: "You are a technical support specialist. Respond in 2-3 sentences max."},
				{Role: ai.RoleUser, Content: s.GetString("ticket")},
			}
		},
		"response",
	)

	generalHandler := workflow.NewPromptStep("general-handler", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleSystem, Content: "You are a general support agent. Respond in 2-3 sentences max."},
				{Role: ai.RoleUser, Content: s.GetString("ticket")},
			}
		},
		"response",
	)

	// Create classifier router
	classifier := workflow.NewClassifierRouter("ticket-classifier", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleSystem, Content: "Classify the following support ticket into exactly one category. Respond with only one word: billing, technical, or general"},
				{Role: ai.RoleUser, Content: s.GetString("ticket")},
			}
		},
		map[string]workflow.Step{
			"billing":   billingHandler,
			"technical": technicalHandler,
			"general":   generalHandler,
		},
		ai.WithMaxTokens(10),
	)

	wf := workflow.New("support-workflow", classifier)

	// Test tickets (billing and technical)
	tickets := []string{
		"I was charged twice for my subscription last month.",
		"The app keeps crashing when I upload large files.",
	}

	for _, ticket := range tickets {
		fmt.Printf("\n--- Ticket ---\n%s\n\n", ticket)
		state := workflow.NewStateFrom(map[string]any{"ticket": ticket})

		events := wf.RunStream(ctx, state, workflow.WithTimeout(1*time.Minute))

		for event := range events {
			switch event.Type {
			case workflow.EventRouteSelected:
				fmt.Printf("[Classified as: %s]\n\n", event.RouteName)
				fmt.Print("Response: ")
			case workflow.EventStreamDelta:
				fmt.Print(event.Delta)
			case workflow.EventStepComplete:
				fmt.Println()
			case workflow.EventError:
				fmt.Fprintf(os.Stderr, "Error: %v\n", event.Error)
			}
		}
		fmt.Println()
	}
}
