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
	perspectives := []struct {
		name   string
		prompt string
	}{
		{"scientific", "From a scientific perspective, give 2 interesting facts about %s. Be concise (2-3 sentences)."},
		{"historical", "From a historical perspective, share 2 interesting facts about %s. Be concise (2-3 sentences)."},
		{"cultural", "From a cultural perspective, share 2 interesting facts about %s across different societies. Be concise (2-3 sentences)."},
	}

	var steps []workflow.Step
	for _, p := range perspectives {
		perspective := p // capture for closure
		steps = append(steps, workflow.NewPromptStep(
			perspective.name,
			c,
			func(s *workflow.State) []ai.Message {
				return []ai.Message{
					{Role: ai.RoleUser, Content: fmt.Sprintf(perspective.prompt, s.GetString("topic"))},
				}
			},
			perspective.name+"_analysis",
		))
	}

	// Aggregator combines all perspectives
	aggregator := func(state *workflow.State, results map[string]*workflow.StepResult, errors map[string]error) error {
		var combined strings.Builder
		combined.WriteString("## Multi-Perspective Analysis\n\n")
		for _, p := range perspectives {
			if result, ok := results[p.name]; ok {
				combined.WriteString(fmt.Sprintf("### %s\n%s\n\n",
					strings.Title(p.name),
					result.Output))
			} else if err, ok := errors[p.name]; ok {
				combined.WriteString(fmt.Sprintf("### %s\n[Error: %v]\n\n",
					strings.Title(p.name), err))
			}
		}
		state.Set("combined_analysis", combined.String())
		return nil
	}

	parallel := workflow.NewParallel("multi-perspective", steps, aggregator)
	wf := workflow.New("parallel-analysis", parallel)

	// Run
	fmt.Println("\n--- Executing Parallel Analysis ---")
	state := workflow.NewStateFrom(map[string]any{"topic": topic})
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
			fmt.Printf("  [%s] Done (%d/%d)\n", ev.StepName, len(completedSteps), len(perspectives))
		case event.ParallelEnd:
			fmt.Println("All perspectives complete!")
		case event.RunError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", ev.Error)
			return
		}
	}

	fmt.Println("\n--- Combined Results ---")
	fmt.Println(state.GetString("combined_analysis"))
}
