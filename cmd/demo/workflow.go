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
	aggregator := func(state *workflow.State, results map[string]*workflow.StepResult) error {
		var combined strings.Builder
		combined.WriteString("## Multi-Perspective Analysis\n\n")
		for _, p := range perspectives {
			if result, ok := results[p.name]; ok {
				combined.WriteString(fmt.Sprintf("### %s\n%s\n\n",
					strings.Title(p.name),
					result.Output))
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
	for event := range events {
		switch event.Type {
		case workflow.EventParallelStart:
			fmt.Println("Starting parallel execution...")
		case workflow.EventStepStart:
			fmt.Printf("  [%s] Analyzing...\n", event.StepName)
		case workflow.EventStepComplete:
			completedSteps[event.StepName] = true
			fmt.Printf("  [%s] Done (%d/%d)\n", event.StepName, len(completedSteps), len(perspectives))
		case workflow.EventParallelComplete:
			fmt.Println("All perspectives complete!")
		case workflow.EventError:
			fmt.Fprintf(os.Stderr, "\nError: %v\n", event.Error)
			return
		}
	}

	fmt.Println("\n--- Combined Results ---")
	fmt.Println(state.GetString("combined_analysis"))
}

func demoWorkflowRouter(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│         Workflow Router Demo            │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println("\nThis demo routes requests based on input type:")
	fmt.Println("  - Questions -> Answer step")
	fmt.Println("  - Statements -> Expansion step")
	fmt.Println("  - Other -> Default step")

	// Define steps for each route
	answerStep := workflow.NewPromptStep("answer", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf("Please answer this question concisely: %s", s.GetString("input"))},
			}
		},
		"response",
	)

	expandStep := workflow.NewPromptStep("expand", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf("Please expand on this statement with additional context: %s", s.GetString("input"))},
			}
		},
		"response",
	)

	defaultStep := workflow.NewPromptStep("default", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: fmt.Sprintf("Please respond appropriately to: %s", s.GetString("input"))},
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

	// Test cases
	testInputs := []string{
		"What is the speed of light?",
		"The ocean covers most of Earth's surface.",
		"Hello there!",
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

func demoWorkflowClassifier(ctx context.Context, c *client.Client) {
	fmt.Println("\n┌─────────────────────────────────────────┐")
	fmt.Println("│      Workflow Classifier Demo           │")
	fmt.Println("└─────────────────────────────────────────┘")
	fmt.Println("\nThis demo uses LLM classification to route support tickets:")
	fmt.Println("  - billing -> Billing handler")
	fmt.Println("  - technical -> Technical handler")
	fmt.Println("  - general -> General handler")

	// Define handlers for each category
	billingHandler := workflow.NewPromptStep("billing-handler", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleSystem, Content: "You are a billing support specialist. Be helpful and mention payment options if relevant."},
				{Role: ai.RoleUser, Content: s.GetString("ticket")},
			}
		},
		"response",
	)

	technicalHandler := workflow.NewPromptStep("technical-handler", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleSystem, Content: "You are a technical support specialist. Provide clear troubleshooting steps."},
				{Role: ai.RoleUser, Content: s.GetString("ticket")},
			}
		},
		"response",
	)

	generalHandler := workflow.NewPromptStep("general-handler", c,
		func(s *workflow.State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleSystem, Content: "You are a general support agent. Be friendly and helpful."},
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

	// Test tickets
	tickets := []string{
		"I was charged twice for my subscription last month. Can you help me get a refund?",
		"The app keeps crashing when I try to upload files larger than 10MB.",
		"I love your product! Just wanted to say thanks to the team.",
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
