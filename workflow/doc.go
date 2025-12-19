// Package workflow provides composable patterns for orchestrating AI-powered pipelines.
//
// The package implements four core workflow patterns:
//   - Chain: Sequential execution where output flows to the next step
//   - Parallel: Concurrent execution with result aggregation
//   - Router: Conditional branching based on predicates or LLM classification
//   - Loop: Iterative execution until a condition is met
//
// All workflow types implement the Step interface, enabling arbitrary nesting
// and composition of patterns.
//
// # Basic Usage
//
// Create a simple chain workflow:
//
//	chain := workflow.NewChain("my-chain",
//		workflow.NewFuncStep("setup", func(ctx context.Context, state *workflow.State) error {
//			state.Set("input", "Hello, World!")
//			return nil
//		}),
//		workflow.NewPromptStep("process", provider,
//			func(s *workflow.State) []gains.Message {
//				return []gains.Message{
//					{Role: gains.RoleUser, Content: s.GetString("input")},
//				}
//			},
//			"output",
//		),
//	)
//
//	wf := workflow.New("my-workflow", chain)
//	result, err := wf.Run(context.Background(), workflow.NewState())
//
// # Parallel Execution
//
// Execute multiple steps concurrently:
//
//	parallel := workflow.NewParallel("research", steps,
//		func(state *workflow.State, results map[string]*workflow.StepResult, errors map[string]error) error {
//			// Aggregate results (errors contains any failed steps when ContinueOnError=true)
//			var combined string
//			for name, result := range results {
//				combined += fmt.Sprintf("%s: %v\n", name, result.Output)
//			}
//			state.Set("combined", combined)
//			return nil
//		},
//	)
//
// Access branch state values with typed helpers:
//
//	var KeyAnalysis = workflow.NewKey[*AnalysisResult]("analysis")
//
//	parallel := workflow.NewParallel("analyze", steps,
//		func(state *workflow.State, results map[string]*workflow.StepResult, errors map[string]error) error {
//			for name, result := range results {
//				// Type-safe access to branch state
//				analysis := workflow.MustGetFromBranch(result, KeyAnalysis)
//				fmt.Printf("%s score: %d\n", name, analysis.Score)
//			}
//			return nil
//		},
//	)
//
// # Conditional Routing
//
// Route based on state conditions:
//
//	router := workflow.NewRouter("route",
//		[]workflow.Route{
//			{
//				Name: "high-priority",
//				Condition: func(ctx context.Context, s *workflow.State) bool {
//					return s.GetString("priority") == "high"
//				},
//				Step: highPriorityStep,
//			},
//		},
//		defaultStep,
//	)
//
// Or use LLM-based classification:
//
//	classifier := workflow.NewClassifierRouter("classify", provider,
//		func(s *workflow.State) []gains.Message {
//			return []gains.Message{
//				{Role: gains.RoleSystem, Content: "Classify as: billing, technical, general"},
//				{Role: gains.RoleUser, Content: s.GetString("ticket")},
//			}
//		},
//		map[string]workflow.Step{
//			"billing":   billingStep,
//			"technical": technicalStep,
//			"general":   generalStep,
//		},
//	)
//
// # Iterative Loops
//
// Repeat steps until a condition is met:
//
//	// Create a content creator that reads feedback on subsequent iterations
//	creator := workflow.NewPromptStep("creator", provider,
//		func(s *workflow.State) []gains.Message {
//			feedback := s.GetString("feedback")
//			if feedback == "" {
//				return []gains.Message{{Role: gains.RoleUser, Content: "Write a blog post about Go"}}
//			}
//			return []gains.Message{
//				{Role: gains.RoleUser, Content: "Write a blog post about Go"},
//				{Role: gains.RoleAssistant, Content: s.GetString("draft")},
//				{Role: gains.RoleUser, Content: "Revise based on: " + feedback},
//			}
//		},
//		workflow.WithOutputKey("draft"),
//	)
//
//	// Create an editor that approves or provides feedback
//	editor := workflow.NewPromptStep("editor", provider,
//		func(s *workflow.State) []gains.Message {
//			return []gains.Message{{
//				Role: gains.RoleUser,
//				Content: "Review this draft. Reply APPROVED or provide feedback:\n\n" +
//					s.GetString("draft"),
//			}}
//		},
//		workflow.WithOutputKey("feedback"),
//	)
//
//	// Combine into a chain and loop until approved
//	reviewCycle := workflow.NewChain("review-cycle", creator, editor)
//	loop := workflow.NewLoop("content-loop", reviewCycle,
//		func(ctx context.Context, s *workflow.State) bool {
//			return strings.Contains(s.GetString("feedback"), "APPROVED")
//		},
//		workflow.WithMaxIterations(5),
//	)
//
// # Streaming Events
//
// Monitor workflow progress in real-time:
//
//	import "github.com/spetersoncode/gains/event"
//
//	events := wf.RunStream(ctx, state)
//	for e := range events {
//		switch e.Type {
//		case event.StepStart:
//			fmt.Printf("Starting: %s\n", e.StepName)
//		case event.MessageDelta:
//			fmt.Print(e.Delta)
//		case event.StepEnd:
//			fmt.Printf("Completed: %s\n", e.StepName)
//		case event.RunError:
//			fmt.Printf("Error: %v\n", e.Error)
//		}
//	}
//
// # Composability
//
// Workflows can be nested since all patterns implement Step:
//
//	outer := workflow.NewChain("outer",
//		setupStep,
//		workflow.NewParallel("inner-parallel", parallelSteps, nil),
//		workflow.NewRouter("inner-router", routes, nil),
//		workflow.NewLoop("inner-loop", refinementStep, condition),
//		finalStep,
//	)
package workflow
