# Workflow System

The workflow package provides composable patterns for orchestrating AI-powered pipelines. It implements four core patterns that can be arbitrarily nested and combined:

- **Chain**: Sequential execution where output flows to the next step
- **Parallel**: Concurrent execution with result aggregation
- **Router**: Conditional branching based on predicates or LLM classification
- **Loop**: Iterative execution until a condition is met

All workflow types implement the `Step` interface, enabling arbitrary nesting and composition.

## Table of Contents

- [Core Concepts](#core-concepts)
  - [The Step Interface](#the-step-interface)
  - [State Management](#state-management)
  - [Type-Safe Keys](#type-safe-keys)
- [Basic Step Types](#basic-step-types)
  - [FuncStep](#funcstep)
  - [PromptStep](#promptstep)
  - [TypedPromptStep](#typedpromptstep)
- [Workflow Patterns](#workflow-patterns)
  - [Chain](#chain)
  - [Parallel](#parallel)
  - [Router](#router)
  - [ClassifierRouter](#classifierrouter)
  - [Loop](#loop)
- [Tool Integration](#tool-integration)
  - [ToolStep](#toolstep)
  - [TypedToolStep](#typedtoolstep)
  - [AgentStep](#agentstep)
- [Advanced Examples](#advanced-examples)
  - [Nested Workflows](#nested-workflows)
  - [Multi-Stage Processing Pipeline](#multi-stage-processing-pipeline)
  - [Research and Analysis System](#research-and-analysis-system)
- [Options and Configuration](#options-and-configuration)
- [Event Handling and Streaming](#event-handling-and-streaming)
- [Error Handling](#error-handling)

---

## Core Concepts

### The Step Interface

Every component in a workflow implements the `Step` interface:

```go
type Step interface {
    Name() string
    Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error)
    RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event
}
```

This uniform interface enables:
- Arbitrary nesting (chains within parallels within routers)
- Consistent execution model across all patterns
- Both synchronous (`Run`) and streaming (`RunStream`) execution

### State Management

State is a thread-safe key-value store that flows through the workflow:

```go
// Create empty state
state := workflow.NewState(nil)

// Create state with initial values
state := workflow.NewStateFrom(map[string]any{
    "input": "Hello, World!",
    "count": 42,
})

// Basic operations
state.Set("key", value)
value, ok := state.Get("key")
state.GetString("key")    // Returns "" if not found
state.GetInt("key")       // Returns 0 if not found
state.GetBool("key")      // Returns false if not found
state.GetFloat("key")     // Returns 0.0 if not found
state.Has("key")          // Check existence
state.Delete("key")       // Remove key
```

### Type-Safe Keys

For compile-time type safety, use `Key[T]`:

```go
// Define typed keys as package-level variables
var (
    KeyInput    = workflow.StringKey("input")
    KeyCount    = workflow.IntKey("count")
    KeyEnabled  = workflow.BoolKey("enabled")
    KeyScore    = workflow.FloatKey("score")
    KeyAnalysis = workflow.NewKey[*AnalysisResult]("analysis")
)

// Type-safe access
workflow.Set(state, KeyInput, "Hello")
input := workflow.MustGet(state, KeyInput)  // Panics if missing
input, ok := workflow.Get(state, KeyInput)  // Returns (value, bool)
input := workflow.GetOr(state, KeyInput, "default")  // With default

// Other typed operations
workflow.Has(state, KeyInput)                         // Check existence
workflow.Delete(state, KeyInput)                      // Remove
workflow.SetIfAbsent(state, KeyInput, "value")        // Set if not exists
workflow.Update(state, KeyCount, func(n int) int {    // Transform in place
    return n + 1
})
```

---

## Basic Step Types

### FuncStep

Wraps arbitrary Go functions as workflow steps:

```go
step := workflow.NewFuncStep("setup", func(ctx context.Context, state *workflow.State) error {
    state.Set("timestamp", time.Now().Unix())
    state.Set("environment", "production")
    return nil
})
```

**Use cases:**
- Initialize state values
- Perform data transformations
- Execute business logic between LLM calls
- Integrate with external services

### PromptStep

Makes a single LLM call with dynamic prompts built from state:

```go
step := workflow.NewPromptStep("summarize", client,
    func(s *workflow.State) []ai.Message {
        content := s.GetString("content")
        return []ai.Message{
            {Role: ai.RoleSystem, Content: "You are a concise summarizer."},
            {Role: ai.RoleUser, Content: "Summarize this:\n\n" + content},
        }
    },
    "summary",  // Output key - response stored here
    ai.WithMaxTokens(500),  // Optional LLM options
)
```

**Features:**
- Dynamic prompt construction from state
- Automatic response storage
- Supports all LLM options (model, temperature, max tokens, etc.)
- Full streaming support

### TypedPromptStep

Returns structured output automatically unmarshaled into a Go type:

```go
// Define the output structure with schema tags
type SentimentAnalysis struct {
    Sentiment  string   `json:"sentiment" desc:"positive, negative, or neutral" enum:"positive,negative,neutral" required:"true"`
    Confidence float64  `json:"confidence" desc:"Confidence score 0-1" min:"0" max:"1" required:"true"`
    Keywords   []string `json:"keywords" desc:"Key phrases" required:"true"`
}

// Define typed key
var KeyAnalysis = workflow.NewKey[*SentimentAnalysis]("analysis")

// Create the step
schema := ai.ResponseSchema{
    Name:        "sentiment_analysis",
    Description: "Sentiment analysis result",
    Schema:      ai.MustSchemaFor[SentimentAnalysis](),
}

step := workflow.NewTypedPromptStepWithKey(
    "analyze",
    client,
    func(s *workflow.State) []ai.Message {
        text := workflow.MustGet(s, KeyInput)
        return []ai.Message{
            {Role: ai.RoleUser, Content: "Analyze sentiment:\n\n" + text},
        }
    },
    schema,
    KeyAnalysis,  // Type-safe output key
)

// Later: type-safe access
analysis := workflow.MustGet(state, KeyAnalysis)
fmt.Printf("Sentiment: %s (%.0f%% confidence)\n",
    analysis.Sentiment, analysis.Confidence*100)
```

**Schema tag reference:**
- `json:"name"` - JSON field name
- `desc:"..."` - Field description for LLM
- `required:"true"` - Mark as required
- `enum:"a,b,c"` - Allowed values
- `min:"0"` / `max:"100"` - Numeric bounds
- `minLength:"1"` / `maxLength:"100"` - String length bounds
- `pattern:"^[A-Z]+"` - Regex pattern
- `default:"value"` - Default value
- `minItems:"1"` / `maxItems:"10"` - Array bounds

---

## Workflow Patterns

### Chain

Sequential execution where each step has access to state set by previous steps:

```go
chain := workflow.NewChain("content-pipeline",
    // Step 1: Generate topic
    workflow.NewPromptStep("generate-topic", client,
        func(s *workflow.State) []ai.Message {
            return []ai.Message{
                {Role: ai.RoleUser, Content: "Give me a random nature topic in 1-2 words."},
            }
        },
        "topic",
    ),

    // Step 2: Write content about the topic
    workflow.NewPromptStep("write-content", client,
        func(s *workflow.State) []ai.Message {
            topic := s.GetString("topic")
            return []ai.Message{
                {Role: ai.RoleUser, Content: "Write a haiku about: " + topic},
            }
        },
        "haiku",
    ),

    // Step 3: Transform the content
    workflow.NewPromptStep("transform", client,
        func(s *workflow.State) []ai.Message {
            haiku := s.GetString("haiku")
            return []ai.Message{
                {Role: ai.RoleUser, Content: "Rewrite in modern style:\n\n" + haiku},
            }
        },
        "transformed",
    ),
)

wf := workflow.New("haiku-workflow", chain)
result, err := wf.Run(ctx, workflow.NewState(nil))
```

**Characteristics:**
- Steps execute in order
- Each step sees state from all previous steps
- First error stops the chain (unless `WithContinueOnError`)
- Token usage accumulated across all steps

### Parallel

Concurrent execution with optional result aggregation:

```go
// Create steps for parallel analysis
steps := []workflow.Step{
    workflow.NewPromptStep("scientific", client,
        func(s *workflow.State) []ai.Message {
            topic := s.GetString("topic")
            return []ai.Message{
                {Role: ai.RoleUser, Content: "Scientific facts about: " + topic},
            }
        },
        "scientific_analysis",
    ),
    workflow.NewPromptStep("historical", client,
        func(s *workflow.State) []ai.Message {
            topic := s.GetString("topic")
            return []ai.Message{
                {Role: ai.RoleUser, Content: "Historical facts about: " + topic},
            }
        },
        "historical_analysis",
    ),
    workflow.NewPromptStep("cultural", client,
        func(s *workflow.State) []ai.Message {
            topic := s.GetString("topic")
            return []ai.Message{
                {Role: ai.RoleUser, Content: "Cultural facts about: " + topic},
            }
        },
        "cultural_analysis",
    ),
}

// With custom aggregator
parallel := workflow.NewParallel("multi-perspective", steps,
    func(state *workflow.State, results map[string]*workflow.StepResult) error {
        var combined strings.Builder
        for name, result := range results {
            combined.WriteString(fmt.Sprintf("## %s\n%s\n\n", name, result.Output))
        }
        state.Set("combined_analysis", combined.String())
        return nil
    },
)

// Without aggregator - branch states auto-merged
parallel := workflow.NewParallel("research", steps, nil)

// Execute with concurrency limit
wf := workflow.New("parallel-workflow", parallel)
result, err := wf.Run(ctx, state, workflow.WithMaxConcurrency(2))
```

**State handling:**
- Each branch receives a cloned copy of the parent state
- Branches can modify their copy independently
- After completion:
  - If aggregator provided: aggregator merges results
  - If no aggregator: all branch states merged back to parent

### Router

Conditional branching based on predicate functions:

```go
// Define route handlers
answerStep := workflow.NewPromptStep("answer", client,
    func(s *workflow.State) []ai.Message {
        return []ai.Message{
            {Role: ai.RoleUser, Content: "Answer: " + s.GetString("input")},
        }
    },
    "response",
)

expandStep := workflow.NewPromptStep("expand", client,
    func(s *workflow.State) []ai.Message {
        return []ai.Message{
            {Role: ai.RoleUser, Content: "Expand on: " + s.GetString("input")},
        }
    },
    "response",
)

defaultStep := workflow.NewPromptStep("default", client,
    func(s *workflow.State) []ai.Message {
        return []ai.Message{
            {Role: ai.RoleUser, Content: "Respond to: " + s.GetString("input")},
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
                return strings.Contains(input, "?")
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
    defaultStep,  // Fallback if no condition matches
)
```

**Built-in condition helpers:**

```go
// Match exact value
workflow.ConditionEquals(KeyPriority, "high")

// Check if key is set
workflow.ConditionSet(KeyResult)

// Custom predicate
workflow.ConditionMatches(KeyScore, func(score float64) bool {
    return score > 0.8
})
```

**Route tracking:**
- Selected route stored in state as `"{router_name}_route"`
- Access via `router.RouteKey()`

### ClassifierRouter

LLM-based classification for intelligent routing:

```go
// Define handlers for each category
handlers := map[string]workflow.Step{
    "billing": workflow.NewPromptStep("billing-handler", client,
        func(s *workflow.State) []ai.Message {
            return []ai.Message{
                {Role: ai.RoleSystem, Content: "You are a billing specialist."},
                {Role: ai.RoleUser, Content: s.GetString("ticket")},
            }
        },
        "response",
    ),
    "technical": workflow.NewPromptStep("technical-handler", client,
        func(s *workflow.State) []ai.Message {
            return []ai.Message{
                {Role: ai.RoleSystem, Content: "You are a technical specialist."},
                {Role: ai.RoleUser, Content: s.GetString("ticket")},
            }
        },
        "response",
    ),
    "general": workflow.NewPromptStep("general-handler", client,
        func(s *workflow.State) []ai.Message {
            return []ai.Message{
                {Role: ai.RoleSystem, Content: "You are a support agent."},
                {Role: ai.RoleUser, Content: s.GetString("ticket")},
            }
        },
        "response",
    ),
}

// Create classifier
classifier := workflow.NewClassifierRouter("ticket-classifier", client,
    func(s *workflow.State) []ai.Message {
        return []ai.Message{
            {Role: ai.RoleSystem, Content: "Classify into: billing, technical, general. Reply with one word."},
            {Role: ai.RoleUser, Content: s.GetString("ticket")},
        }
    },
    handlers,
    ai.WithMaxTokens(10),  // Limit tokens for classification
)
```

**Classification tracking:**
- Raw classification: `classifier.ClassificationKey()` -> `"{name}_classification"`
- Selected route: `classifier.RouteKey()` -> `"{name}_route"`

### Loop

Iterative execution until a condition is met:

```go
// Writer creates or revises content
writer := workflow.NewPromptStep("writer", client,
    func(s *workflow.State) []ai.Message {
        iteration := s.GetInt("content-loop_iteration")
        feedback := s.GetString("feedback")

        if iteration <= 1 || feedback == "" {
            // First iteration
            return []ai.Message{
                {Role: ai.RoleUser, Content: "Write a short poem about sunset."},
            }
        }

        // Subsequent iterations - incorporate feedback
        draft := s.GetString("draft")
        return []ai.Message{
            {Role: ai.RoleUser, Content: fmt.Sprintf(
                "Previous version:\n%s\n\nFeedback: %s\n\nPlease revise.",
                draft, feedback)},
        }
    },
    "draft",
)

// Editor reviews and provides feedback
editor := workflow.NewPromptStep("editor", client,
    func(s *workflow.State) []ai.Message {
        draft := s.GetString("draft")
        return []ai.Message{
            {Role: ai.RoleUser, Content: fmt.Sprintf(
                "Review this:\n\n%s\n\nReply 'APPROVED' if good, or provide feedback.",
                draft)},
        }
    },
    "feedback",
)

// Combine into review cycle
reviewCycle := workflow.NewChain("review-cycle", writer, editor)

// Loop until approved
loop := workflow.NewLoop("content-loop", reviewCycle,
    func(ctx context.Context, s *workflow.State) bool {
        feedback := s.GetString("feedback")
        return strings.Contains(strings.ToUpper(feedback), "APPROVED")
    },
    workflow.WithMaxIterations(5),
)
```

**Loop helpers:**

```go
// Loop until key equals value
loop := workflow.NewLoopUntil("loop", step, "status", "complete")

// Loop while key equals value
loop := workflow.NewLoopWhile("loop", step, "continue", true)

// Loop until key is set
loop := workflow.NewLoopUntilSet("loop", step, "result")

// Type-safe versions
loop := workflow.NewLoopUntilKey("loop", step, KeyStatus, "complete")
loop := workflow.NewLoopWhileKey("loop", step, KeyContinue, true)
loop := workflow.NewLoopUntilKeySet("loop", step, KeyResult)
```

**Iteration tracking:**
- Current iteration stored as `"{loop_name}_iteration"`
- Access via `loop.IterationKey()`
- Returns `ErrMaxIterationsExceeded` if max iterations hit

---

## Tool Integration

### ToolStep

Execute tools directly without LLM involvement:

```go
// Register a tool
registry := tool.NewRegistry()
tool.MustRegisterFunc(registry, "lookup", "Look up data by key",
    func(ctx context.Context, args struct {
        Key string `json:"key" desc:"The key to look up" required:"true"`
    }) (string, error) {
        data := map[string]string{
            "pi": "3.14159",
            "e":  "2.71828",
        }
        if val, ok := data[args.Key]; ok {
            return val, nil
        }
        return "", fmt.Errorf("key not found: %s", args.Key)
    },
)

// Create tool step
toolStep := workflow.NewToolStep(
    "lookup-constant",
    registry,
    "lookup",
    func(s *workflow.State) (any, error) {
        return struct {
            Key string `json:"key"`
        }{Key: s.GetString("lookup_key")}, nil
    },
    "constant_value",  // Output key
)

// Use in chain
chain := workflow.NewChain("tool-chain",
    workflow.NewFuncStep("setup", func(ctx context.Context, state *workflow.State) error {
        state.Set("lookup_key", "pi")
        return nil
    }),
    toolStep,
    workflow.NewPromptStep("explain", client,
        func(s *workflow.State) []ai.Message {
            value := s.GetString("constant_value")
            return []ai.Message{
                {Role: ai.RoleUser, Content: "Explain what pi (" + value + ") represents."},
            }
        },
        "explanation",
    ),
)
```

### TypedToolStep

Type-safe tool execution with typed arguments:

```go
type LookupArgs struct {
    Key string `json:"key" desc:"The key to look up" required:"true"`
}

var KeyToolResult = workflow.StringKey("tool_result")

toolStep := workflow.NewTypedToolStepWithKey(
    "lookup",
    registry,
    "lookup",
    func(s *workflow.State) (LookupArgs, error) {
        return LookupArgs{Key: s.GetString("key")}, nil
    },
    KeyToolResult,
)
```

### AgentStep

Embed autonomous tool-calling agents within workflows:

```go
// Create tool registry
registry := tool.NewRegistry()
tool.MustRegisterFunc(registry, "calculate", "Evaluate math expression",
    func(ctx context.Context, args struct {
        Expression string `json:"expression" required:"true"`
    }) (string, error) {
        // Math evaluation logic
        return result, nil
    },
)

// Create agent step
agentStep := workflow.NewAgentStep(
    "solver",
    client,
    registry,
    func(s *workflow.State) []ai.Message {
        problem := s.GetString("problem")
        return []ai.Message{
            {Role: ai.RoleUser, Content: "Solve: " + problem},
        }
    },
    "agent_result",
    []agent.Option{
        agent.WithMaxSteps(5),
        agent.WithTimeout(time.Minute),
    },
)

// Access results
chain := workflow.NewChain("math-chain",
    agentStep,
    workflow.NewPromptStep("summarize", client,
        func(s *workflow.State) []ai.Message {
            result, _ := s.Get("agent_result")
            agentResult := result.(*workflow.AgentResult)
            return []ai.Message{
                {Role: ai.RoleUser, Content: fmt.Sprintf(
                    "Agent took %d steps and concluded: %s\n\nSummarize.",
                    agentResult.Steps, agentResult.Response.Content)},
            }
        },
        "summary",
    ),
)
```

**AgentResult fields:**
- `Response` - Final AI response
- `Messages` - Full conversation history
- `Steps` - Number of tool-calling iterations
- `Termination` - Why the agent stopped (complete, max_steps, timeout, etc.)

---

## Advanced Examples

### Nested Workflows

Workflows can be arbitrarily nested since all patterns implement `Step`:

```go
// Inner parallel for research
researchParallel := workflow.NewParallel("research",
    []workflow.Step{
        workflow.NewPromptStep("source-a", client, promptA, "research_a"),
        workflow.NewPromptStep("source-b", client, promptB, "research_b"),
        workflow.NewPromptStep("source-c", client, promptC, "research_c"),
    },
    func(state *workflow.State, results map[string]*workflow.StepResult) error {
        // Combine research
        return nil
    },
)

// Inner loop for refinement
refinementLoop := workflow.NewLoop("refine",
    workflow.NewChain("refine-cycle", drafter, reviewer),
    approvalCondition,
    workflow.WithMaxIterations(3),
)

// Outer workflow combining everything
outerWorkflow := workflow.NewChain("full-pipeline",
    workflow.NewFuncStep("setup", setupFunc),
    researchParallel,                    // Parallel nested in chain
    workflow.NewRouter("route",          // Router nested in chain
        []workflow.Route{
            {Name: "detailed", Condition: needsDetail, Step: refinementLoop},  // Loop nested in router
            {Name: "simple", Condition: isSimple, Step: quickStep},
        },
        defaultStep,
    ),
    workflow.NewPromptStep("finalize", client, finalizePrompt, "output"),
)
```

### Multi-Stage Processing Pipeline

Document processing with classification, extraction, and validation:

```go
// Stage 1: Document classification
classifier := workflow.NewClassifierRouter("doc-type", client,
    func(s *workflow.State) []ai.Message {
        return []ai.Message{
            {Role: ai.RoleUser, Content: "Classify document type: invoice, contract, report, other\n\n" + s.GetString("document")},
        }
    },
    map[string]workflow.Step{
        "invoice":  invoiceExtractor,
        "contract": contractExtractor,
        "report":   reportExtractor,
        "other":    genericExtractor,
    },
)

// Stage 2: Parallel validation
validator := workflow.NewParallel("validate",
    []workflow.Step{
        workflow.NewFuncStep("schema-check", schemaValidator),
        workflow.NewPromptStep("completeness", client, completenessPrompt, "completeness_result"),
        workflow.NewPromptStep("accuracy", client, accuracyPrompt, "accuracy_result"),
    },
    func(state *workflow.State, results map[string]*workflow.StepResult) error {
        // Aggregate validation results
        state.Set("validation_passed", allPassed(results))
        return nil
    },
)

// Stage 3: Conditional enrichment loop
enrichmentLoop := workflow.NewLoop("enrich",
    workflow.NewChain("enrich-cycle",
        workflow.NewPromptStep("identify-gaps", client, gapPrompt, "gaps"),
        workflow.NewAgentStep("fill-gaps", client, registry, fillPrompt, "enriched", agentOpts),
    ),
    func(ctx context.Context, s *workflow.State) bool {
        return s.GetString("gaps") == "none" || s.GetInt("enrich_iteration") >= 3
    },
)

// Full pipeline
pipeline := workflow.NewChain("document-pipeline",
    workflow.NewFuncStep("preprocess", preprocessDoc),
    classifier,
    validator,
    workflow.NewRouter("enrich-decision",
        []workflow.Route{
            {
                Name: "needs-enrichment",
                Condition: func(ctx context.Context, s *workflow.State) bool {
                    return !s.GetBool("validation_passed")
                },
                Step: enrichmentLoop,
            },
        },
        workflow.NewFuncStep("skip", func(ctx context.Context, s *workflow.State) error { return nil }),
    ),
    workflow.NewPromptStep("summarize", client, summaryPrompt, "final_summary"),
)
```

### Research and Analysis System

Multi-source research with synthesis:

```go
var (
    KeyTopic     = workflow.StringKey("topic")
    KeySources   = workflow.NewKey[[]string]("sources")
    KeySynthesis = workflow.NewKey[*SynthesisResult]("synthesis")
)

// Research phase - parallel queries
researchSteps := []workflow.Step{
    workflow.NewAgentStep("web-search", client, searchRegistry,
        func(s *workflow.State) []ai.Message {
            topic := workflow.MustGet(s, KeyTopic)
            return []ai.Message{
                {Role: ai.RoleUser, Content: "Search web for: " + topic},
            }
        },
        "web_results",
        []agent.Option{agent.WithMaxSteps(3)},
    ),
    workflow.NewAgentStep("db-query", client, dbRegistry,
        func(s *workflow.State) []ai.Message {
            topic := workflow.MustGet(s, KeyTopic)
            return []ai.Message{
                {Role: ai.RoleUser, Content: "Query database for: " + topic},
            }
        },
        "db_results",
        []agent.Option{agent.WithMaxSteps(2)},
    ),
    workflow.NewPromptStep("expert-opinion", client,
        func(s *workflow.State) []ai.Message {
            topic := workflow.MustGet(s, KeyTopic)
            return []ai.Message{
                {Role: ai.RoleSystem, Content: "You are a domain expert."},
                {Role: ai.RoleUser, Content: "Provide expert analysis on: " + topic},
            }
        },
        "expert_analysis",
    ),
}

research := workflow.NewParallel("research", researchSteps,
    func(state *workflow.State, results map[string]*workflow.StepResult) error {
        sources := make([]string, 0, len(results))
        for name, result := range results {
            sources = append(sources, fmt.Sprintf("[%s]: %v", name, result.Output))
        }
        workflow.Set(state, KeySources, sources)
        return nil
    },
)

// Synthesis phase - iterative refinement
synthesisLoop := workflow.NewLoop("synthesis",
    workflow.NewChain("synthesis-cycle",
        workflow.NewTypedPromptStepWithKey(
            "synthesize",
            client,
            func(s *workflow.State) []ai.Message {
                topic := workflow.MustGet(s, KeyTopic)
                sources := workflow.MustGet(s, KeySources)
                return []ai.Message{
                    {Role: ai.RoleUser, Content: fmt.Sprintf(
                        "Synthesize research on %s:\n\n%s",
                        topic, strings.Join(sources, "\n\n"))},
                }
            },
            synthesisSchema,
            KeySynthesis,
        ),
        workflow.NewPromptStep("critique",
            func(s *workflow.State) []ai.Message {
                synthesis := workflow.MustGet(s, KeySynthesis)
                return []ai.Message{
                    {Role: ai.RoleUser, Content: fmt.Sprintf(
                        "Critique this synthesis (reply APPROVED if good):\n\n%s",
                        synthesis.Summary)},
                }
            },
            "critique",
        ),
    ),
    func(ctx context.Context, s *workflow.State) bool {
        return strings.Contains(strings.ToUpper(s.GetString("critique")), "APPROVED")
    },
    workflow.WithMaxIterations(3),
)

// Full system
system := workflow.NewChain("research-system",
    research,
    synthesisLoop,
    workflow.NewPromptStep("format-output", client, formatPrompt, "final_report"),
)
```

---

## Options and Configuration

### Workflow Options

```go
result, err := wf.Run(ctx, state,
    // Overall workflow timeout
    workflow.WithTimeout(5*time.Minute),

    // Per-step timeout (default: 30s)
    workflow.WithStepTimeout(1*time.Minute),

    // Limit parallel concurrency
    workflow.WithMaxConcurrency(3),

    // Continue past errors
    workflow.WithContinueOnError(true),

    // Custom error handling
    workflow.WithErrorHandler(func(ctx context.Context, stepName string, err error) error {
        log.Printf("Step %s failed: %v", stepName, err)
        return nil  // Return nil to continue, error to stop
    }),

    // Step completion callback
    workflow.WithOnStepComplete(func(ctx context.Context, result *workflow.StepResult) {
        log.Printf("Completed %s: tokens=%d", result.StepName, result.Usage.TotalTokens)
    }),

    // Pass options to all LLM calls
    workflow.WithChatOptions(
        ai.WithModel(model.Claude3Sonnet),
        ai.WithTemperature(0.7),
    ),

    // Convenience shortcuts
    workflow.WithModel(model.Claude3Sonnet),
    workflow.WithMaxTokens(1000),
    workflow.WithTemperature(0.5),
)
```

### Option Inheritance

Options flow down through nested workflows:
- Timeout applies to entire workflow
- Step timeout applies per-step
- Max concurrency applies to Parallel steps
- Chat options passed to all LLM calls

---

## Event Handling and Streaming

### Event Types

```go
import "github.com/spetersoncode/gains/event"

// Workflow lifecycle
event.RunStart      // Workflow started
event.RunEnd        // Workflow completed successfully
event.RunError      // Workflow error

// Step lifecycle
event.StepStart     // Step starting
event.StepEnd       // Step completed
event.StepSkipped   // Step was skipped

// Message streaming
event.MessageStart  // LLM response starting
event.MessageDelta  // Token chunk received
event.MessageEnd    // LLM response complete

// Tool events
event.ToolCallStart  // Tool being called
event.ToolCallArgs   // Tool arguments (streaming)
event.ToolCallEnd    // Tool call complete
event.ToolCallResult // Tool result received

// Pattern-specific
event.ParallelStart  // Parallel block starting
event.ParallelEnd    // Parallel block complete
event.RouteSelected  // Router made selection
event.LoopIteration  // Loop iteration starting
```

### Streaming Execution

```go
events := wf.RunStream(ctx, state, opts...)

for ev := range events {
    switch ev.Type {
    case event.StepStart:
        fmt.Printf("[%s] Starting...\n", ev.StepName)

    case event.MessageDelta:
        fmt.Print(ev.Delta)  // Stream tokens

    case event.StepEnd:
        fmt.Printf("\n[%s] Complete (tokens: %d)\n",
            ev.StepName, ev.Usage.TotalTokens)

    case event.ParallelStart:
        fmt.Println("Starting parallel execution...")

    case event.ParallelEnd:
        fmt.Println("Parallel complete!")

    case event.RouteSelected:
        fmt.Printf("Route selected: %s\n", ev.RouteName)

    case event.LoopIteration:
        fmt.Printf("Loop iteration %d\n", ev.Iteration)

    case event.ToolCallStart:
        fmt.Printf("Calling tool: %s\n", ev.ToolCall.Name)

    case event.ToolCallResult:
        fmt.Printf("Tool result: %s\n", ev.ToolResult.Content)

    case event.RunError:
        fmt.Printf("Error: %v\n", ev.Error)
        return

    case event.RunEnd:
        fmt.Println("Workflow complete!")
    }
}
```

---

## Error Handling

### Error Types

```go
// Step execution failed
var stepErr *workflow.StepError
if errors.As(err, &stepErr) {
    fmt.Printf("Step %s failed: %v\n", stepErr.StepName, stepErr.Err)
}

// Parallel had failures
var parallelErr *workflow.ParallelError
if errors.As(err, &parallelErr) {
    for step, e := range parallelErr.Errors {
        fmt.Printf("Branch %s failed: %v\n", step, e)
    }
}

// Structured output unmarshaling failed
var unmarshalErr *workflow.UnmarshalError
if errors.As(err, &unmarshalErr) {
    fmt.Printf("Failed to parse %s response as %s: %v\nContent: %s\n",
        unmarshalErr.StepName, unmarshalErr.TargetType,
        unmarshalErr.Err, unmarshalErr.Content)
}

// Tool execution failed
var toolErr *workflow.ToolExecutionError
if errors.As(err, &toolErr) {
    fmt.Printf("Tool %s failed: %s\n", toolErr.ToolName, toolErr.Content)
}

// Sentinel errors
errors.Is(err, workflow.ErrWorkflowTimeout)
errors.Is(err, workflow.ErrWorkflowCancelled)
errors.Is(err, workflow.ErrNoRouteMatched)
errors.Is(err, workflow.ErrMaxIterationsExceeded)
```

### Continue on Error

```go
chain := workflow.NewChain("tolerant-chain",
    step1,
    step2,  // May fail
    step3,  // Should still run
)

result, err := chain.Run(ctx, state,
    workflow.WithContinueOnError(true),
    workflow.WithErrorHandler(func(ctx context.Context, stepName string, err error) error {
        // Log error
        log.Printf("Step %s failed: %v", stepName, err)

        // Set error state for downstream steps to check
        state.Set(stepName+"_error", err.Error())

        // Return nil to continue, error to stop
        return nil
    }),
)
```

### Termination Reasons

```go
result, err := wf.Run(ctx, state)

switch result.Termination {
case workflow.TerminationComplete:
    fmt.Println("Success!")
case workflow.TerminationTimeout:
    fmt.Println("Timed out")
case workflow.TerminationCancelled:
    fmt.Println("Cancelled")
case workflow.TerminationError:
    fmt.Printf("Error: %v\n", result.Error)
}
```
