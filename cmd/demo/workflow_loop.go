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

// LoopState is the state struct for the loop workflow demo.
type LoopState struct {
	Iteration int
	Draft     string
	Feedback  string
}

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
		func(s *LoopState) []ai.Message {
			if s.Iteration <= 1 || s.Feedback == "" {
				// First iteration - create initial content
				return []ai.Message{
					{Role: ai.RoleUser, Content: "Write a very short 2-line poem about a sunset. Be creative but keep it brief."},
				}
			}

			// Subsequent iterations - revise based on feedback
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf(
					"Here's your previous poem:\n\n%s\n\nEditor feedback: %s\n\nPlease revise the poem based on this feedback. Keep it to 2 lines.",
					s.Draft, s.Feedback)},
			}
		},
		nil,
		func(s *LoopState) *string { return &s.Draft },
	)

	// Step 2: Editor reviews and either approves or provides feedback
	editor := workflow.NewPromptStep("editor", c,
		func(s *LoopState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf(
					`Review this poem:

%s

If the poem is creative and well-crafted, respond with exactly "APPROVED" followed by brief praise.
If it needs improvement, provide specific constructive feedback (1-2 sentences) to make it better.
Be a tough but fair editor - only approve truly good work.`, s.Draft)},
			}
		},
		nil,
		func(s *LoopState) *string { return &s.Feedback },
	)

	// Create the review cycle chain
	reviewCycle := workflow.NewChain("review-cycle", writer, editor)

	// Create the loop that exits when approved
	loop := workflow.NewLoopUntil("content-loop", reviewCycle,
		func(s *LoopState) bool {
			return strings.Contains(strings.ToUpper(s.Feedback), "APPROVED")
		},
		workflow.WithMaxIterations(3),
	)

	wf := workflow.New("loop-workflow", loop)

	// Run with streaming
	fmt.Println("\n--- Starting Content Loop ---")
	state := &LoopState{}
	events := wf.RunStream(ctx, state, workflow.WithTimeout(3*time.Minute))

	currentStep := ""
	for ev := range events {
		switch ev.Type {
		case event.LoopIteration:
			state.Iteration = ev.Iteration
			fmt.Printf("\n═══ Iteration %d ═══\n", ev.Iteration)
		case event.StepStart:
			currentStep = ev.StepName
			if currentStep == "writer" {
				fmt.Print("\n[Writer] ")
			} else if currentStep == "editor" {
				fmt.Print("\n[Editor] ")
			}
		case event.MessageDelta:
			fmt.Print(ev.Delta)
		case event.StepEnd:
			fmt.Println()
		case event.RunEnd:
			// Loop completed successfully
		case event.RunError:
			if ev.Error == workflow.ErrMaxIterationsExceeded {
				fmt.Println("\n⚠ Max iterations reached - editor never fully approved!")
			} else {
				fmt.Fprintf(os.Stderr, "\nError: %v\n", ev.Error)
			}
			return
		}
	}

	fmt.Println("\n--- Final Results ---")
	fmt.Printf("Total iterations: %d\n", state.Iteration)
	fmt.Printf("\nFinal poem:\n%s\n", state.Draft)
	fmt.Printf("\nFinal feedback:\n%s\n", state.Feedback)
}
