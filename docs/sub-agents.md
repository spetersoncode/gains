# Sub-Agent Patterns and Multi-Agent Architectures

This document describes patterns for building multi-agent systems using gains, including sub-agents as tools, specialist registries, and event forwarding for observability.

## Table of Contents

- [Overview](#overview)
- [Agents as Tools](#agents-as-tools)
  - [Basic Sub-Agent Tool](#basic-sub-agent-tool)
  - [Typed Arguments](#typed-arguments)
  - [Custom Message Mapping](#custom-message-mapping)
- [Specialist Registry](#specialist-registry)
  - [Registering Specialists](#registering-specialists)
  - [Capability-Based Lookup](#capability-based-lookup)
  - [Converting to Tools](#converting-to-tools)
- [Event Forwarding](#event-forwarding)
  - [Enabling Forwarding](#enabling-forwarding)
  - [Observability Benefits](#observability-benefits)
- [Workflow Integration](#workflow-integration)
  - [AgentStep](#agentstep)
  - [Nested Workflows](#nested-workflows)
- [Architecture Patterns](#architecture-patterns)
  - [Orchestrator Pattern](#orchestrator-pattern)
  - [Specialist Delegation](#specialist-delegation)
  - [Pipeline Pattern](#pipeline-pattern)
- [Complete Example](#complete-example)
- [Best Practices](#best-practices)

---

## Overview

Multi-agent architectures enable:

- **Task Delegation**: Main agent delegates specialized tasks to sub-agents
- **Separation of Concerns**: Different agents for different domains
- **Tool Isolation**: Each agent has access to only relevant tools
- **Observability**: Nested agent events flow to the parent stream

---

## Agents as Tools

### Basic Sub-Agent Tool

Wrap an agent as a callable tool using `NewTool`:

```go
import (
    "github.com/spetersoncode/gains/agent"
    "github.com/spetersoncode/gains/tool"
)

// Create a specialized research agent
researchRegistry := tool.NewRegistry()
tool.MustRegisterAll(researchRegistry, tool.WebTools())
researchAgent := agent.New(client, researchRegistry)

// Create main agent's registry
mainRegistry := tool.NewRegistry()

// Add research agent as a tool
mainRegistry.Add(agent.NewTool("research", researchAgent,
    agent.WithToolDescription("Delegate research tasks to the research specialist"),
    agent.WithToolMaxSteps(5),
))

// Main agent can now call 'research' as a tool
mainAgent := agent.New(client, mainRegistry)
```

**Default interface:**

The tool accepts a simple `query` parameter:

```go
// Default ToolArgs
type ToolArgs struct {
    Query string `json:"query" desc:"The query or task for the agent" required:"true"`
}
```

### Typed Arguments

Use `NewToolFunc` for custom typed arguments:

```go
type ResearchArgs struct {
    Topic    string `json:"topic" desc:"Research topic" required:"true"`
    MaxDepth int    `json:"maxDepth" desc:"Maximum research depth" default:"2"`
    Sources  []string `json:"sources" desc:"Preferred sources to check"`
}

mainRegistry.Add(agent.NewToolFunc("research", researchAgent,
    "Research a topic with configurable depth",
    func(args ResearchArgs) []ai.Message {
        prompt := fmt.Sprintf(
            "Research %q to depth %d. Prefer sources: %v",
            args.Topic, args.MaxDepth, args.Sources,
        )
        return []ai.Message{{Role: ai.RoleUser, Content: prompt}}
    },
    agent.WithToolMaxSteps(args.MaxDepth * 2),
))
```

### Custom Message Mapping

For advanced argument-to-message conversion:

```go
mainRegistry.Add(agent.NewTool("analyst", analystAgent,
    agent.WithToolDescription("Analyze data with context"),
    agent.WithToolArgsMapper(func(call ai.ToolCall) ([]ai.Message, error) {
        var args struct {
            Data    string `json:"data"`
            Context string `json:"context"`
        }
        if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
            return nil, err
        }

        return []ai.Message{
            {Role: ai.RoleSystem, Content: "You are a data analyst. Context: " + args.Context},
            {Role: ai.RoleUser, Content: "Analyze this data:\n" + args.Data},
        }, nil
    }),
))
```

---

## Specialist Registry

The `SpecialistRegistry` manages collections of specialized agents:

### Registering Specialists

```go
import "github.com/spetersoncode/gains/agent"

registry := agent.NewSpecialistRegistry()

// Register specialists with descriptions
registry.Register("research", "Research and gather information from the web",
    researchAgent,
    agent.WithCapabilities("web_search", "summarization"),
)

registry.Register("code", "Write, analyze, and debug code",
    codeAgent,
    agent.WithCapabilities("code_generation", "debugging", "analysis"),
)

registry.Register("data", "Analyze and visualize data",
    dataAgent,
    agent.WithCapabilities("analysis", "visualization", "statistics"),
)
```

### Capability-Based Lookup

Find agents by capability:

```go
// Find all agents that can analyze
analysts := registry.ByCapability("analysis")
for _, s := range analysts {
    fmt.Printf("Agent %s can analyze\n", s.Name)
}

// Check if a specific specialist exists
if registry.Has("research") {
    agent := registry.GetAgent("research")
    // Use the agent...
}

// Get all registered names
names := registry.Names() // ["research", "code", "data"]
```

### Converting to Tools

Add all specialists to a tool registry:

```go
// Method 1: RegisterTo (convenience)
toolRegistry := tool.NewRegistry()
specialists.RegisterTo(toolRegistry,
    agent.WithToolEventForwarding(), // Forward events from all sub-agents
)

// Method 2: AsTools (more control)
for _, t := range specialists.AsTools() {
    toolRegistry.Add(t)
}

// Method 3: AsToolsWith (per-specialist options)
tools := specialists.AsToolsWith(func(s *agent.Specialist) []agent.ToolOption {
    opts := []agent.ToolOption{
        agent.WithToolEventForwarding(),
    }
    // Custom max steps based on specialist
    if s.Name == "research" {
        opts = append(opts, agent.WithToolMaxSteps(10))
    }
    return opts
})
```

---

## Event Forwarding

### Enabling Forwarding

Enable event forwarding to observe sub-agent activity:

```go
mainRegistry.Add(agent.NewTool("research", researchAgent,
    agent.WithToolEventForwarding(), // Key option
    agent.WithToolDescription("Research with observable progress"),
))
```

**How it works:**

1. Parent agent's tool execution sets a forwarding channel in context
2. Sub-agent sends events to this channel via `event.ForwardChannelFromContext`
3. Parent receives sub-agent events (message deltas, tool calls, etc.)
4. AG-UI mapper handles nested run depth automatically

### Observability Benefits

With forwarding enabled, you get:

- **Streaming Tokens**: Sub-agent's response streams to parent
- **Tool Call Visibility**: See what tools sub-agents are using
- **Progress Tracking**: Step-by-step visibility into sub-agent work
- **AG-UI Integration**: Nested events properly mapped for frontend

```go
// Parent agent stream includes:
// - RunStart (parent)
// - RunStart (sub-agent) - filtered by mapper's depth tracking
// - MessageDelta from sub-agent
// - ToolCallStart from sub-agent
// - ToolCallResult from sub-agent
// - MessageEnd from sub-agent
// - RunEnd (sub-agent) - filtered
// - ToolCallResult (parent's perspective)
// - RunEnd (parent)
```

---

## Workflow Integration

### AgentStep

Embed agents in workflows using `AgentStep`:

```go
import "github.com/spetersoncode/gains/workflow"

type MyState struct {
    Topic          string
    ResearchResult *workflow.AgentResult
    Summary        string
}

// Create agent step
researchStep := workflow.NewAgentStep[MyState](
    "research",
    client,
    researchRegistry,
    func(s *MyState) []ai.Message {
        return []ai.Message{{
            Role:    ai.RoleUser,
            Content: fmt.Sprintf("Research %s thoroughly", s.Topic),
        }}
    },
    func(s *MyState, r *workflow.AgentResult) {
        s.ResearchResult = r
    },
    []agent.Option{
        agent.WithMaxSteps(5),
        agent.WithTimeout(2 * time.Minute),
    },
    ai.WithModel(model.Claude35Sonnet),
)

// Use in a chain
chain := workflow.NewChain[MyState]("research-pipeline",
    researchStep,
    workflow.NewPromptStep[MyState]("summarize", client,
        func(s *MyState) []ai.Message {
            return []ai.Message{{
                Role:    ai.RoleUser,
                Content: "Summarize: " + s.ResearchResult.Response.Content,
            }}
        },
        func(s *MyState, resp string) { s.Summary = resp },
    ),
)
```

### Nested Workflows

Workflows can contain agents, which can contain sub-agents:

```go
// Level 1: Main workflow
mainWorkflow := workflow.NewChain[State]("main",
    // Level 2: Agent step
    workflow.NewAgentStep[State]("orchestrator", client, registry, ...),
    // The orchestrator agent can call sub-agent tools (Level 3)
)

// Event depth is tracked automatically:
// - mainWorkflow emits RunStart (depth=1, emitted)
// - orchestrator agent emits RunStart (depth=2, filtered)
// - sub-agent tool emits RunStart (depth=3, filtered)
// - Only outermost lifecycle events reach AG-UI
```

---

## Architecture Patterns

### Orchestrator Pattern

A coordinator agent delegates to specialists:

```go
// Create specialist agents
specialists := agent.NewSpecialistRegistry()
specialists.Register("researcher", "Research topics", researchAgent)
specialists.Register("writer", "Write content", writerAgent)
specialists.Register("editor", "Edit and improve content", editorAgent)

// Create orchestrator with access to all specialists
orchestratorRegistry := tool.NewRegistry()
specialists.RegisterTo(orchestratorRegistry, agent.WithToolEventForwarding())

orchestrator := agent.New(client, orchestratorRegistry)

// Orchestrator prompt
messages := []ai.Message{{
    Role: ai.RoleSystem,
    Content: `You are an orchestrator agent. You have access to:
- researcher: For gathering information
- writer: For creating content
- editor: For improving content

Coordinate these specialists to complete the user's request.`,
}}
```

### Specialist Delegation

Direct delegation based on task type:

```go
// Router selects the right specialist
router := workflow.NewClassifierRouter[State]("route", client,
    func(s *State) []ai.Message {
        return []ai.Message{{
            Role:    ai.RoleUser,
            Content: "Classify task: research, coding, or analysis\n\n" + s.Task,
        }}
    },
    map[string]workflow.Step[State]{
        "research": workflow.NewAgentStep[State]("research", client, researchReg, ...),
        "coding":   workflow.NewAgentStep[State]("coding", client, codeReg, ...),
        "analysis": workflow.NewAgentStep[State]("analysis", client, dataReg, ...),
    },
)
```

### Pipeline Pattern

Sequential processing through multiple agents:

```go
pipeline := workflow.NewChain[State]("content-pipeline",
    // Stage 1: Research agent gathers information
    workflow.NewAgentStep[State]("research", client, researchReg,
        func(s *State) []ai.Message { return research(s.Topic) },
        func(s *State, r *workflow.AgentResult) { s.Research = r.Response.Content },
        []agent.Option{agent.WithMaxSteps(5)},
    ),

    // Stage 2: Writer agent creates content
    workflow.NewAgentStep[State]("write", client, writerReg,
        func(s *State) []ai.Message { return write(s.Research) },
        func(s *State, r *workflow.AgentResult) { s.Draft = r.Response.Content },
        []agent.Option{agent.WithMaxSteps(3)},
    ),

    // Stage 3: Editor agent polishes
    workflow.NewAgentStep[State]("edit", client, editorReg,
        func(s *State) []ai.Message { return edit(s.Draft) },
        func(s *State, r *workflow.AgentResult) { s.Final = r.Response.Content },
        []agent.Option{agent.WithMaxSteps(3)},
    ),
)
```

---

## Complete Example

Full multi-agent system:

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"

    ai "github.com/spetersoncode/gains"
    "github.com/spetersoncode/gains/agent"
    "github.com/spetersoncode/gains/client"
    "github.com/spetersoncode/gains/event"
    "github.com/spetersoncode/gains/tool"
)

func main() {
    c := client.New(client.WithAnthropic(os.Getenv("ANTHROPIC_API_KEY")))

    // Create specialized agents
    researchAgent := createResearchAgent(c)
    codeAgent := createCodeAgent(c)

    // Create specialist registry
    specialists := agent.NewSpecialistRegistry()
    specialists.Register("research",
        "Research topics and gather information from the web",
        researchAgent,
    )
    specialists.Register("code",
        "Write, analyze, and debug code",
        codeAgent,
    )

    // Create orchestrator
    orchestratorRegistry := tool.NewRegistry()
    specialists.RegisterTo(orchestratorRegistry,
        agent.WithToolEventForwarding(),
        agent.WithToolMaxSteps(5),
    )

    orchestrator := agent.New(c, orchestratorRegistry)

    // Run with streaming
    ctx := context.Background()
    events := orchestrator.RunStream(ctx, []ai.Message{
        {Role: ai.RoleSystem, Content: `You coordinate specialist agents.
Available specialists: research, code.
Delegate appropriately based on the task.`},
        {Role: ai.RoleUser, Content: "Research Go's error handling best practices and write an example."},
    },
        agent.WithMaxSteps(10),
        agent.WithTimeout(5*time.Minute),
    )

    // Process events
    for ev := range events {
        switch ev.Type {
        case event.MessageDelta:
            fmt.Print(ev.Delta)
        case event.ToolCallStart:
            fmt.Printf("\n[Calling %s]\n", ev.ToolCall.Name)
        case event.ToolCallResult:
            fmt.Printf("\n[%s completed]\n", ev.ToolCall.Name)
        case event.RunEnd:
            fmt.Println("\n\nDone!")
        case event.RunError:
            fmt.Printf("\nError: %v\n", ev.Error)
        }
    }
}

func createResearchAgent(c *client.Client) *agent.Agent {
    reg := tool.NewRegistry()
    tool.MustRegisterAll(reg, tool.WebTools(
        tool.WithAllowedHosts("github.com", "go.dev", "pkg.go.dev"),
    ))
    return agent.New(c, reg)
}

func createCodeAgent(c *client.Client) *agent.Agent {
    reg := tool.NewRegistry()
    tool.MustRegisterFunc(reg, "run_go", "Execute Go code",
        func(ctx context.Context, args struct {
            Code string `json:"code" required:"true"`
        }) (string, error) {
            // In production: safely sandbox and execute
            return "// Code would execute here", nil
        },
    )
    return agent.New(c, reg)
}
```

---

## Best Practices

### 1. Limit Sub-Agent Steps

Prevent runaway sub-agents:

```go
agent.NewTool("research", researchAgent,
    agent.WithToolMaxSteps(5), // Reasonable limit
)
```

### 2. Use Descriptive Tool Names

Help the main agent choose correctly:

```go
// Good: Clear purpose
specialists.Register("web_researcher", "Search the web and summarize findings", ...)
specialists.Register("code_reviewer", "Review code for bugs and improvements", ...)

// Avoid: Vague names
specialists.Register("agent1", "Does stuff", ...)
```

### 3. Enable Event Forwarding for Debugging

Always enable in development:

```go
specialists.RegisterTo(registry,
    agent.WithToolEventForwarding(), // See sub-agent progress
)
```

### 4. Isolate Tool Access

Each agent should only have relevant tools:

```go
// Research agent: web tools only
researchRegistry := tool.NewRegistry()
tool.MustRegisterAll(researchRegistry, tool.WebTools())

// Code agent: file tools only
codeRegistry := tool.NewRegistry()
tool.MustRegisterAll(codeRegistry, tool.FileTools())
```

### 5. Set Appropriate Timeouts

Sub-agents need their own timeouts:

```go
agent.NewTool("research", researchAgent,
    agent.WithToolAgentOptions(
        agent.WithTimeout(2*time.Minute),
        agent.WithHandlerTimeout(30*time.Second),
    ),
)
```

### 6. Document Specialist Capabilities

Make it clear what each specialist can do:

```go
specialists.Register("data_analyst",
    "Analyze datasets: compute statistics, find patterns, generate visualizations. "+
        "Supports CSV, JSON, and SQL data sources. "+
        "Cannot modify data, only read and analyze.",
    dataAgent,
    agent.WithCapabilities("statistics", "visualization", "pattern_detection"),
)
```

---

## Related Documentation

- [AG-UI Event Sequences](agui-events.md) - Nested run event handling
- [Workflows](workflows.md) - AgentStep and workflow patterns
