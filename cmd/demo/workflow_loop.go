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

func demoWorkflowLoop(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│          Workflow Loop Demo             │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println("\nThis demo shows an iterative content refinement loop:")
	fmt.Println("  1. Writer creates a short poem")
	fmt.Println("  2. Editor reviews and provides feedback or approves")
	fmt.Println("  3. Loop continues until approved (max 3 iterations)")

	// Step 1: Writer creates/revises content based on feedback
	writer := workflow.NewPromptStep("writer", c,
		func(s *workflow.State) []ai.Message {
			iteration := s.GetInt("content-loop_iteration")
			feedback := s.GetString("feedback")

			if iteration <= 1 || feedback == "" {
				// First iteration - create initial content
				return []ai.Message{
					{Role: ai.RoleUser, Content: "Write a very short 2-line poem about a sunset. Be creative but keep it brief."},
				}
			}

			// Subsequent iterations - revise based on feedback
			draft := s.GetString("draft")
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf(
					"Here's your previous poem:\n\n%s\n\nEditor feedback: %s\n\nPlease revise the poem based on this feedback. Keep it to 2 lines.",
					draft, feedback)},
			}
		},
		"draft",
	)

	// Step 2: Editor reviews and either approves or provides feedback
	editor := workflow.NewPromptStep("editor", c,
		func(s *workflow.State) []ai.Message {
			draft := s.GetString("draft")
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf(
					`Review this poem:

%s

If the poem is creative and well-crafted, respond with exactly "APPROVED" followed by brief praise.
If it needs improvement, provide specific constructive feedback (1-2 sentences) to make it better.
Be a tough but fair editor - only approve truly good work.`, draft)},
			}
		},
		"feedback",
	)

	// Create the review cycle chain
	reviewCycle := workflow.NewChain("review-cycle", writer, editor)

	// Create the loop with exit condition
	loop := workflow.NewLoop("content-loop", reviewCycle,
		func(ctx context.Context, s *workflow.State) bool {
			feedback := s.GetString("feedback")
			return strings.Contains(strings.ToUpper(feedback), "APPROVED")
		},
		workflow.WithMaxIterations(3),
	)

	wf := workflow.New("loop-workflow", loop)

	// Run with streaming
	fmt.Println("\n--- Starting Content Loop ---")
	state := workflow.NewState(nil)
	events := wf.RunStream(ctx, state, workflow.WithTimeout(3*time.Minute))

	currentStep := ""
	for event := range events {
		switch event.Type {
		case workflow.EventLoopIteration:
			fmt.Printf("\n═══ Iteration %d ═══\n", event.Iteration)
		case workflow.EventStepStart:
			currentStep = event.StepName
			if currentStep == "writer" {
				fmt.Print("\n[Writer] ")
			} else if currentStep == "editor" {
				fmt.Print("\n[Editor] ")
			}
		case workflow.EventStreamDelta:
			fmt.Print(event.Delta)
		case workflow.EventStepComplete:
			fmt.Println()
		case workflow.EventWorkflowComplete:
			// Loop completed successfully
		case workflow.EventError:
			if event.Error == workflow.ErrMaxIterationsExceeded {
				fmt.Println("\n⚠ Max iterations reached - editor never fully approved!")
			} else {
				fmt.Fprintf(os.Stderr, "\nError: %v\n", event.Error)
			}
			return
		}
	}

	fmt.Println("\n--- Final Results ---")
	fmt.Printf("Total iterations: %d\n", state.GetInt("content-loop_iteration"))
	fmt.Printf("\nFinal poem:\n%s\n", state.GetString("draft"))
	fmt.Printf("\nFinal feedback:\n%s\n", state.GetString("feedback"))
}
