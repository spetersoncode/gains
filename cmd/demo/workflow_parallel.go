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

// ParallelState is the state struct for the parallel workflow demo.
type ParallelState struct {
	Topic            string
	Scientific       string
	Historical       string
	Cultural         string
	CombinedAnalysis string
}

func demoWorkflowParallel(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│        Workflow Parallel Demo           │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println("\nThis demo analyzes a topic from 3 perspectives in parallel:")
	fmt.Println("  - Scientific perspective")
	fmt.Println("  - Historical perspective")
	fmt.Println("  - Cultural perspective")

	topic := "The Moon"
	fmt.Printf("\nTopic: %s\n", topic)

	// Create parallel analysis steps
	scientificStep := workflow.NewPromptStep("scientific", c,
		func(s *ParallelState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf("From a scientific perspective, give 2 interesting facts about %s. Be concise (2-3 sentences).", s.Topic)},
			}
		},
		nil,
		func(s *ParallelState) *string { return &s.Scientific },
	)

	historicalStep := workflow.NewPromptStep("historical", c,
		func(s *ParallelState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf("From a historical perspective, share 2 interesting facts about %s. Be concise (2-3 sentences).", s.Topic)},
			}
		},
		nil,
		func(s *ParallelState) *string { return &s.Historical },
	)

	culturalStep := workflow.NewPromptStep("cultural", c,
		func(s *ParallelState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf("From a cultural perspective, share 2 interesting facts about %s across different societies. Be concise (2-3 sentences).", s.Topic)},
			}
		},
		nil,
		func(s *ParallelState) *string { return &s.Cultural },
	)

	steps := []workflow.Step[ParallelState]{scientificStep, historicalStep, culturalStep}

	// Aggregator combines all perspectives
	aggregator := func(state *ParallelState, branches map[string]*ParallelState, errors map[string]error) error {
		var combined strings.Builder
		combined.WriteString("## Multi-Perspective Analysis\n\n")

		perspectives := []struct {
			name  string
			field *string
		}{
			{"scientific", &state.Scientific},
			{"historical", &state.Historical},
			{"cultural", &state.Cultural},
		}

		for _, p := range perspectives {
			if br, ok := branches[p.name]; ok {
				// Copy result from branch state to main state
				switch p.name {
				case "scientific":
					state.Scientific = br.Scientific
				case "historical":
					state.Historical = br.Historical
				case "cultural":
					state.Cultural = br.Cultural
				}
				combined.WriteString(fmt.Sprintf("### %s\n%s\n\n",
					strings.Title(p.name), *p.field))
			} else if err, ok := errors[p.name]; ok {
				combined.WriteString(fmt.Sprintf("### %s\n[Error: %v]\n\n",
					strings.Title(p.name), err))
			}
		}
		state.CombinedAnalysis = combined.String()
		return nil
	}

	parallel := workflow.NewParallel("multi-perspective", steps, aggregator)
	wf := workflow.New("parallel-analysis", parallel)

	// Run
	fmt.Println("\n--- Executing Parallel Analysis ---")
	state := &ParallelState{Topic: topic}
	events := wf.RunStream(ctx, state,
		workflow.WithTimeout(2*time.Minute),
		workflow.WithMaxConcurrency(3),
	)

	completedSteps := make(map[string]bool)
	for ev := range events {
		switch ev.Type {
		case event.ParallelStart:
			fmt.Println("Starting parallel execution...")
		case event.StepStart:
			fmt.Printf("  [%s] Analyzing...\n", ev.StepName)
		case event.StepEnd:
			completedSteps[ev.StepName] = true
			fmt.Printf("  [%s] Done (%d/%d)\n", ev.StepName, len(completedSteps), 3)
		case event.ParallelEnd:
			fmt.Println("All perspectives complete!")
		case event.RunError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", ev.Error)
			return
		}
	}

	fmt.Println("\n--- Combined Results ---")
	fmt.Println(state.CombinedAnalysis)
}
