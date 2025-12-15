<p align="center">
  <img src="mascot.jpeg" alt="gains mascot" width="400">
</p>

# gains

**Go AI Native Scaffold** - A Go-idiomatic generative AI library. Not yet production ready.

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

## Import Convention

We recommend importing with the `ai` alias for cleaner, more readable code:

```go
import ai "github.com/spetersoncode/gains"

// Now you can write:
msg := ai.Message{Role: ai.RoleUser, Content: "Hello"}
```

All examples in this documentation use this convention.

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    ai "github.com/spetersoncode/gains"
    "github.com/spetersoncode/gains/client"
    "github.com/spetersoncode/gains/models"
)

func main() {
    ctx := context.Background()
    c, _ := client.New(ctx, client.Config{
        Provider: client.ProviderAnthropic,
    })

    resp, _ := c.Chat(ctx, []ai.Message{
        {Role: ai.RoleUser, Content: "Hello!"},
    }, ai.WithModel(models.ClaudeSonnet45))

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
ctx := context.Background()

// Anthropic - uses ANTHROPIC_API_KEY
c, _ := client.New(ctx, client.Config{Provider: client.ProviderAnthropic})

// OpenAI - uses OPENAI_API_KEY
c, _ := client.New(ctx, client.Config{Provider: client.ProviderOpenAI})

// Google - uses GOOGLE_API_KEY
c, _ := client.New(ctx, client.Config{Provider: client.ProviderGoogle})
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

Define tools using struct-based schema generation:

```go
type WeatherArgs struct {
    Location string `json:"location"`
    Unit     string `json:"unit"`
}

tools := []ai.Tool{{
    Name:        "get_weather",
    Description: "Get current weather for a location",
    Parameters: ai.SchemaFrom[WeatherArgs]().
        Desc("location", "The city name").Required("location").
        Desc("unit", "Temperature unit").Enum("unit", "celsius", "fahrenheit").
        Build(),
}}

resp, _ := c.Chat(ctx, messages, ai.WithTools(tools))

for _, call := range resp.ToolCalls {
    fmt.Printf("Tool: %s, Args: %s\n", call.Name, call.Arguments)
}
```

## Agent Orchestration

The agent package handles autonomous tool-calling loops with typed handlers:

```go
import "github.com/spetersoncode/gains/agent"

type SearchArgs struct {
    Query string `json:"query"`
}

// Create a tool registry with typed handler
registry := agent.NewRegistry()
agent.MustRegisterFunc(registry, "search", "Search the web",
    ai.SchemaFrom[SearchArgs]().
        Desc("query", "Search query").Required("query").
        Build(),
    func(ctx context.Context, args SearchArgs) (string, error) {
        // args.Query is already parsed
        return results, nil
    },
)

// Create and run agent
a := agent.New(chatProvider, registry)

result, _ := a.Run(ctx, messages,
    agent.WithMaxSteps(10),
    agent.WithTimeout(2*time.Minute),
)
fmt.Println(result.Response.Content)
```

### Human-in-the-Loop

Require approval for sensitive operations:

```go
a := agent.New(provider, registry)

result, _ := a.Run(ctx, messages,
    agent.WithApprovalRequired("delete_file", "send_email"),
    agent.WithApprover(func(ctx context.Context, call ai.ToolCall) (bool, string) {
        fmt.Printf("Allow %s? [y/n]: ", call.Name)
        // Get user input...
        return approved, "" // Return approval and optional rejection reason
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
    ai.WithImageSize(ai.ImageSize1024x1024),
)
fmt.Println(resp.Images[0].URL)
```

## Structured Output

Force JSON output or use schema validation:

```go
// Simple JSON mode
resp, _ := c.Chat(ctx, messages, ai.WithJSONMode())

// With schema validation
resp, _ := c.Chat(ctx, messages,
    ai.WithResponseSchema(ai.ResponseSchema{
        Name:   "result",
        Schema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"score":{"type":"number"}}}`),
    }),
)
```

## Models

The `models` package provides type-safe model selection with pricing info:

```go
import "github.com/spetersoncode/gains/models"

// Auto-updating aliases (recommended)
ai.WithModel(models.ClaudeSonnet45)    // Anthropic
ai.WithModel(models.GPT52)             // OpenAI
ai.WithModel(models.Gemini25Flash)     // Google

// Pinned versions for production stability
ai.WithModel(models.ClaudeSonnet45_20250929)
ai.WithModel(models.Gemini3Pro)
```

## Request Options

```go
resp, _ := c.Chat(ctx, messages,
    ai.WithModel(models.ClaudeOpus45),
    ai.WithMaxTokens(4096),
    ai.WithTemperature(0.7),
    ai.WithTools(tools),
    ai.WithToolChoice(ai.ToolChoiceAuto),
)
```

## Retry Configuration

```go
c, _ := client.New(ctx, client.Config{
    Provider: client.ProviderOpenAI,
    RetryConfig: &client.RetryConfig{
        MaxAttempts:  5,
        InitialDelay: time.Second,
        MaxDelay:     30 * time.Second,
    },
})

// Or disable retries entirely
disabled := client.DisabledRetryConfig()
c, _ := client.New(ctx, client.Config{
    Provider:    client.ProviderOpenAI,
    RetryConfig: &disabled,
})
```

## Events

Observe operations at multiple levels:

```go
// Client events via channel
events := make(chan client.Event, 100)
c, _ := client.New(ctx, client.Config{
    Provider: client.ProviderOpenAI,
    Events:   events,
})

go func() {
    for e := range events {
        fmt.Printf("[%s] %s %v\n", e.Type, e.Operation, e.Duration)
    }
}()

// Agent events via streaming
a := agent.New(provider, registry)
stream := a.RunStream(ctx, messages)
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
