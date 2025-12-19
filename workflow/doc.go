// Package workflow provides composable patterns for orchestrating AI-powered pipelines.
//
// The package implements five core workflow patterns:
//   - Chain: Sequential execution where steps share mutable state
//   - Parallel: Concurrent execution with branch state isolation and aggregation
//   - Router: Conditional branching based on predicates or LLM classification
//   - Loop: Iterative execution until a condition is met
//   - Merge: Conditional joining of multiple steps with fan-in aggregation
//
// All workflow types implement the Step[S] interface, enabling arbitrary nesting
// and composition. The generic type parameter S is your user-defined state struct.
//
// # State Model
//
// Define your own state struct to hold workflow data:
//
//	type MyState struct {
//	    Input    string
//	    Analysis SentimentResult
//	    Summary  string
//	}
//
// State is passed by pointer and mutated in place. After workflow completion,
// access results directly from your state fields.
//
// # Basic Usage
//
// Create a simple chain workflow:
//
//	type PipelineState struct {
//	    Topic   string
//	    Content string
//	}
//
//	chain := workflow.NewChain("my-chain",
//	    workflow.NewFuncStep[PipelineState]("setup", func(ctx context.Context, s *PipelineState) error {
//	        s.Topic = "Go generics"
//	        return nil
//	    }),
//	    workflow.NewPromptStep("generate", client,
//	        func(s *PipelineState) []gains.Message {
//	            return []gains.Message{
//	                {Role: gains.RoleUser, Content: "Write about: " + s.Topic},
//	            }
//	        },
//	        nil, // no schema = plain text
//	        func(s *PipelineState) *string { return &s.Content },
//	    ),
//	)
//
//	wf := workflow.New("my-workflow", chain)
//	state := &PipelineState{}
//	result, err := wf.Run(ctx, state)
//	fmt.Println(state.Content) // Access result from state
//
// # Structured Output
//
// Use PromptStep with a schema for automatic JSON unmarshaling:
//
//	type AnalysisState struct {
//	    Input    string
//	    Analysis SentimentResult
//	}
//
//	type SentimentResult struct {
//	    Sentiment string  `json:"sentiment"`
//	    Score     float64 `json:"score"`
//	}
//
//	schema := gains.MustSchemaFor[SentimentResult]()
//
//	step := workflow.NewPromptStep("analyze", client,
//	    func(s *AnalysisState) []gains.Message {
//	        return []gains.Message{{Role: gains.RoleUser, Content: s.Input}}
//	    },
//	    schema, // schema provided = JSON unmarshal
//	    func(s *AnalysisState) *SentimentResult { return &s.Analysis },
//	)
//
// # Parallel Execution
//
// Execute multiple steps concurrently with isolated branch state:
//
//	type ResearchState struct {
//	    Topic       string
//	    Technical   string
//	    Business    string
//	    Combined    string
//	}
//
//	parallel := workflow.NewParallel("research",
//	    []workflow.Step[ResearchState]{technicalStep, businessStep},
//	    func(state *ResearchState, branches map[string]*ResearchState, errors map[string]error) error {
//	        // Each branch ran with a deep copy; merge results back
//	        for name, branch := range branches {
//	            if name == "technical" {
//	                state.Technical = branch.Technical
//	            } else if name == "business" {
//	                state.Business = branch.Business
//	            }
//	        }
//	        state.Combined = state.Technical + "\n\n" + state.Business
//	        return nil
//	    },
//	)
//
// # Conditional Routing
//
// Route based on state conditions:
//
//	type TicketState struct {
//	    Priority string
//	    Response string
//	}
//
//	router := workflow.NewRouter("route",
//	    []workflow.Route[TicketState]{
//	        {
//	            Name: "urgent",
//	            Condition: func(ctx context.Context, s *TicketState) bool {
//	                return s.Priority == "high"
//	            },
//	            Step: urgentHandler,
//	        },
//	    },
//	    normalHandler, // default
//	)
//
// Or use LLM-based classification:
//
//	classifier := workflow.NewClassifierRouter("classify", client,
//	    func(s *TicketState) []gains.Message {
//	        return []gains.Message{
//	            {Role: gains.RoleSystem, Content: "Classify as: billing, technical, general"},
//	            {Role: gains.RoleUser, Content: s.Ticket},
//	        }
//	    },
//	    map[string]workflow.Step[TicketState]{
//	        "billing":   billingStep,
//	        "technical": technicalStep,
//	        "general":   generalStep,
//	    },
//	)
//
// # Iterative Loops
//
// Repeat steps until a condition is met:
//
//	type EditState struct {
//	    Draft    string
//	    Feedback string
//	}
//
//	// Create a writer that revises based on feedback
//	writer := workflow.NewPromptStep("writer", client,
//	    func(s *EditState) []gains.Message {
//	        if s.Feedback == "" {
//	            return []gains.Message{{Role: gains.RoleUser, Content: "Write a poem"}}
//	        }
//	        return []gains.Message{
//	            {Role: gains.RoleUser, Content: fmt.Sprintf(
//	                "Revise this:\n%s\n\nFeedback: %s", s.Draft, s.Feedback,
//	            )},
//	        }
//	    },
//	    nil,
//	    func(s *EditState) *string { return &s.Draft },
//	)
//
//	// Create an editor that approves or provides feedback
//	editor := workflow.NewPromptStep("editor", client,
//	    func(s *EditState) []gains.Message {
//	        return []gains.Message{{
//	            Role:    gains.RoleUser,
//	            Content: "Review this. Reply APPROVED or provide feedback:\n\n" + s.Draft,
//	        }}
//	    },
//	    nil,
//	    func(s *EditState) *string { return &s.Feedback },
//	)
//
//	// Loop until approved
//	loop := workflow.NewLoopUntil("refine",
//	    workflow.NewChain("cycle", writer, editor),
//	    func(s *EditState) bool {
//	        return strings.Contains(strings.ToUpper(s.Feedback), "APPROVED")
//	    },
//	    workflow.WithMaxIterations(5),
//	)
//
// # Streaming Events
//
// Monitor workflow progress in real-time:
//
//	import "github.com/spetersoncode/gains/event"
//
//	state := &MyState{}
//	events := wf.RunStream(ctx, state)
//	for e := range events {
//	    switch e.Type {
//	    case event.StepStart:
//	        fmt.Printf("Starting: %s\n", e.StepName)
//	    case event.MessageDelta:
//	        fmt.Print(e.Delta)
//	    case event.StepEnd:
//	        fmt.Printf("Completed: %s\n", e.StepName)
//	    case event.RunError:
//	        fmt.Printf("Error: %v\n", e.Error)
//	    }
//	}
//	// Access final results from state
//	fmt.Println(state.Summary)
//
// # Composability
//
// Workflows can be nested since all patterns implement Step[S]:
//
//	outer := workflow.NewChain("outer",
//	    setupStep,
//	    workflow.NewParallel("inner-parallel", parallelSteps, aggregator),
//	    workflow.NewRouter("inner-router", routes, defaultStep),
//	    workflow.NewLoopUntil("inner-loop", refinementStep, condition),
//	    finalStep,
//	)
package workflow
