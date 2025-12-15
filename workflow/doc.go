// Package workflow provides composable patterns for orchestrating AI-powered pipelines.
//
// The package implements three core workflow patterns:
//   - Chain: Sequential execution where output flows to the next step
//   - Parallel: Concurrent execution with result aggregation
//   - Router: Conditional branching based on predicates or LLM classification
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
//		func(state *workflow.State, results map[string]*workflow.StepResult) error {
//			// Aggregate results
//			var combined string
//			for name, result := range results {
//				combined += fmt.Sprintf("%s: %v\n", name, result.Output)
//			}
//			state.Set("combined", combined)
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
// # Streaming Events
//
// Monitor workflow progress in real-time:
//
//	events := wf.RunStream(ctx, state)
//	for event := range events {
//		switch event.Type {
//		case workflow.EventStepStart:
//			fmt.Printf("Starting: %s\n", event.StepName)
//		case workflow.EventStreamDelta:
//			fmt.Print(event.Delta)
//		case workflow.EventStepComplete:
//			fmt.Printf("Completed: %s\n", event.StepName)
//		case workflow.EventError:
//			fmt.Printf("Error: %v\n", event.Error)
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
//		finalStep,
//	)
package workflow
