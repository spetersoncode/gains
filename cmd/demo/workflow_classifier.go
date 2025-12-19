package main

import (
	"context"
	"fmt"
	"os"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/workflow"
)

// ClassifierState is the state struct for the classifier workflow demo.
type ClassifierState struct {
	Ticket   string
	Response string
}

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
		func(s *ClassifierState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleSystem, Content: "You are a billing support specialist. Respond in 2-3 sentences max."},
				{Role: ai.RoleUser, Content: s.Ticket},
			}
		},
		nil,
		func(s *ClassifierState) *string { return &s.Response },
	)

	technicalHandler := workflow.NewPromptStep("technical-handler", c,
		func(s *ClassifierState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleSystem, Content: "You are a technical support specialist. Respond in 2-3 sentences max."},
				{Role: ai.RoleUser, Content: s.Ticket},
			}
		},
		nil,
		func(s *ClassifierState) *string { return &s.Response },
	)

	generalHandler := workflow.NewPromptStep("general-handler", c,
		func(s *ClassifierState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleSystem, Content: "You are a general support agent. Respond in 2-3 sentences max."},
				{Role: ai.RoleUser, Content: s.Ticket},
			}
		},
		nil,
		func(s *ClassifierState) *string { return &s.Response },
	)

	// Create classifier router
	classifier := workflow.NewClassifierRouter("ticket-classifier", c,
		func(s *ClassifierState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleSystem, Content: "Classify the following support ticket into exactly one category. Respond with only one word: billing, technical, or general"},
				{Role: ai.RoleUser, Content: s.Ticket},
			}
		},
		map[string]workflow.Step[ClassifierState]{
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
		state := &ClassifierState{Ticket: ticket}

		events := wf.RunStream(ctx, state, workflow.WithTimeout(1*time.Minute))

		for ev := range events {
			switch ev.Type {
			case event.RouteSelected:
				fmt.Printf("[Classified as: %s]\n\n", ev.RouteName)
				fmt.Print("Response: ")
			case event.MessageDelta:
				fmt.Print(ev.Delta)
			case event.StepEnd:
				fmt.Println()
			case event.RunError:
				fmt.Fprintf(os.Stderr, "Error: %v\n", ev.Error)
			}
		}
		fmt.Println()
	}
}
