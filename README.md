<p align="center">
  <img src="mascot.jpeg" alt="gains mascot" width="400">
</p>

# gains

**Go AI Native Scaffold** - A production-ready, Go-idiomatic generative AI library.

gains provides a unified interface for building AI applications across Anthropic, OpenAI, and Google. Inspired by langchain and langgraph, but built from the ground up for Go with first-class streaming, tool orchestration, and composable workflows.

## Features

- **Unified Client** - Single interface for chat, embeddings, and image generation across providers
- **Agent Orchestration** - Autonomous tool-calling loops with approval workflows
- **Composable Workflows** - Chain, Parallel, and Router patterns for complex pipelines
- **Streaming First** - Channel-based streaming throughout the entire API
- **Functional Options** - Go-idiomatic configuration with type-safe model selection
- **Automatic Retry** - Exponential backoff with jitter for transient errors
- **Event System** - Observable operations at client, agent, and workflow levels

## Installation

```bash
go get github.com/spetersoncode/gains
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    "github.com/spetersoncode/gains"
    "github.com/spetersoncode/gains/client"
    "github.com/spetersoncode/gains/models"
)

func main() {
    c, _ := client.New(client.Config{
        Provider: client.ProviderAnthropic,
    })

    resp, _ := c.Chat(context.Background(), []gains.Message{
        {Role: gains.RoleUser, Content: "Hello!"},
    }, gains.WithModel(models.ClaudeSonnet4))

    fmt.Println(resp.Content)
}
```

## Providers

The unified client supports all three providers with feature detection:

| Provider  | Chat | Images | Embeddings |
|-----------|:----:|:------:|:----------:|
| Anthropic | ✓    | -      | -          |
| OpenAI    | ✓    | ✓      | ✓          |
| Google    | ✓    | ✓      | ✓          |

```go
// Anthropic - uses ANTHROPIC_API_KEY
c, _ := client.New(client.Config{Provider: client.ProviderAnthropic})

// OpenAI - uses OPENAI_API_KEY
c, _ := client.New(client.Config{Provider: client.ProviderOpenAI})

// Google - uses GOOGLE_API_KEY
c, _ := client.New(client.Config{Provider: client.ProviderGoogle})
```

## Chat & Streaming

```go
// Basic chat
resp, _ := c.Chat(ctx, messages)
fmt.Println(resp.Content)

// Streaming
stream, _ := c.ChatStream(ctx, messages)
for event := range stream {
    if event.Err != nil {
        break
    }
    fmt.Print(event.Delta)
}
```

## Tool Calling

Define tools and let the model invoke them:

```go
tools := []gains.Tool{{
    Name:        "get_weather",
    Description: "Get current weather for a location",
    Parameters: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "location": map[string]any{"type": "string"},
        },
        "required": []string{"location"},
    },
}}

resp, _ := c.Chat(ctx, messages, gains.WithTools(tools))

for _, call := range resp.ToolCalls {
    fmt.Printf("Tool: %s, Args: %s\n", call.Name, call.Arguments)
}
```

## Agent Orchestration

The agent package handles autonomous tool-calling loops:

```go
import "github.com/spetersoncode/gains/agent"

// Create a tool registry
registry := agent.NewRegistry()
registry.MustRegister(gains.Tool{
    Name:        "search",
    Description: "Search the web",
    Parameters:  searchParams,
}, func(ctx context.Context, args string) (string, error) {
    // Execute search...
    return results, nil
})

// Create and run agent
a := agent.New(chatProvider, registry,
    agent.WithMaxSteps(10),
    agent.WithTimeout(2*time.Minute),
)

result, _ := a.Run(ctx, messages)
fmt.Println(result.Response.Content)
```

### Human-in-the-Loop

Require approval for sensitive operations:

```go
a := agent.New(provider, registry,
    agent.WithApprovalRequired("delete_file", "send_email"),
    agent.WithApprover(func(ctx context.Context, call gains.ToolCall) bool {
        fmt.Printf("Allow %s? [y/n]: ", call.Name)
        // Get user input...
        return approved
    }),
)
```

## Workflows

Build complex pipelines with composable patterns:

```go
import "github.com/spetersoncode/gains/workflow"

// Chain - sequential execution
chain := workflow.NewChain("pipeline",
    workflow.NewPromptStep("analyze", provider, "Analyze: {{.input}}"),
    workflow.NewPromptStep("summarize", provider, "Summarize: {{.analyze}}"),
)

// Parallel - concurrent execution
parallel := workflow.NewParallel("multi-analysis",
    []workflow.Step{step1, step2, step3},
    aggregator,
)

// Router - conditional branching
router := workflow.NewRouter("classifier",
    workflow.Route{
        Condition: func(s *workflow.State) bool { return s.GetString("type") == "question" },
        Step:      questionHandler,
    },
    workflow.Route{
        Condition: func(s *workflow.State) bool { return s.GetString("type") == "task" },
        Step:      taskHandler,
    },
)

// Execute workflow
wf := workflow.New("my-workflow", chain)
result, _ := wf.Run(ctx, initialState)
```

## Embeddings

```go
resp, _ := c.Embed(ctx, []string{"Hello world"})
fmt.Printf("Dimensions: %d\n", len(resp.Embeddings[0]))
```

## Image Generation

```go
resp, _ := c.GenerateImage(ctx, "A sunset over mountains",
    gains.WithImageSize(gains.ImageSize1024x1024),
)
fmt.Println(resp.Images[0].URL)
```

## Structured Output

Force JSON output with schema validation:

```go
resp, _ := c.Chat(ctx, messages,
    gains.WithJSONMode(true),
    gains.WithResponseSchema(map[string]any{
        "type": "object",
        "properties": map[string]any{
            "name":  map[string]any{"type": "string"},
            "score": map[string]any{"type": "number"},
        },
    }),
)
```

## Models

The `models` package provides type-safe model selection with pricing info:

```go
import "github.com/spetersoncode/gains/models"

// Auto-updating aliases (recommended)
gains.WithModel(models.ClaudeSonnet4)
gains.WithModel(models.GPT4o)
gains.WithModel(models.Gemini25Flash)

// Pinned versions
gains.WithModel(models.ClaudeSonnet4_20250514)
```

## Request Options

```go
resp, _ := c.Chat(ctx, messages,
    gains.WithModel(models.ClaudeOpus4),
    gains.WithMaxTokens(4096),
    gains.WithTemperature(0.7),
    gains.WithTools(tools),
    gains.WithToolChoice(gains.ToolChoiceAuto),
)
```

## Retry Configuration

```go
c, _ := client.New(client.Config{
    Provider: client.ProviderOpenAI,
    Retry: &client.RetryConfig{
        MaxAttempts:  5,
        InitialDelay: time.Second,
        MaxDelay:     30 * time.Second,
    },
})
```

## Events

Observe operations at multiple levels:

```go
// Client events
c.OnEvent(func(e client.Event) {
    fmt.Printf("[%s] %s %s %v\n", e.Type, e.Provider, e.Model, e.Duration)
})

// Agent events (via streaming)
stream := agent.RunStream(ctx, messages)
for event := range stream {
    switch event.Type {
    case agent.EventToolCallStarted:
        fmt.Printf("Calling tool: %s\n", event.ToolCall.Name)
    case agent.EventStreamDelta:
        fmt.Print(event.Delta)
    }
}
```

## Environment Variables

| Provider  | Variable            |
|-----------|---------------------|
| Anthropic | `ANTHROPIC_API_KEY` |
| OpenAI    | `OPENAI_API_KEY`    |
| Google    | `GOOGLE_API_KEY`    |

## Examples

See the [`cmd/demo`](cmd/demo) directory for complete examples:

- `chat.go` - Basic chat
- `chat_stream.go` - Streaming responses
- `vision.go` - Image input/vision
- `tools.go` - Tool calling
- `agent.go` - Agent orchestration
- `structured.go` - JSON mode / structured output
- `embeddings.go` - Text embeddings
- `images.go` - Image generation
- `workflow.go` - Workflow patterns

## License

MIT
