# gains

**Go AI Native Scaffold** - A simple, flexible, Go-idiomatic generative AI library.

gains provides a unified interface for chat completions across major LLM providers, inspired by the best ideas from langchain and langgraph but built from the ground up for Go.

## Features

- **Unified Interface** - Single `ChatProvider` interface works with all providers
- **Streaming Support** - First-class support for streaming responses via channels
- **Functional Options** - Go-idiomatic configuration with `WithModel`, `WithTemperature`, etc.
- **Provider Flexibility** - Easy switching between Anthropic, OpenAI, and Google

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
    "github.com/spetersoncode/gains/provider/anthropic"
)

func main() {
    client := anthropic.New()

    resp, err := client.Chat(context.Background(), []gains.Message{
        {Role: gains.RoleUser, Content: "Hello!"},
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Content)
}
```

## Providers

### Anthropic

```go
import "github.com/spetersoncode/gains/provider/anthropic"

// Uses ANTHROPIC_API_KEY environment variable
client := anthropic.New()

// Or with explicit API key
client := anthropic.New(anthropic.WithAPIKey("sk-..."))

// With custom model
client := anthropic.New(anthropic.WithModel("claude-opus-4-20250514"))
```

Default model: `claude-sonnet-4-20250514`

### OpenAI

```go
import "github.com/spetersoncode/gains/provider/openai"

// Uses OPENAI_API_KEY environment variable
client := openai.New()

// Or with explicit API key
client := openai.New(openai.WithAPIKey("sk-..."))
```

Default model: `gpt-4o`

### Google

```go
import "github.com/spetersoncode/gains/provider/google"

// Uses GOOGLE_API_KEY environment variable
client, err := google.New(ctx)

// Or with explicit API key
client, err := google.New(ctx, google.WithAPIKey(ctx, "..."))
```

Default model: `gemini-2.0-flash`

## Streaming

```go
stream, err := client.ChatStream(ctx, []gains.Message{
    {Role: gains.RoleUser, Content: "Tell me a story"},
})
if err != nil {
    panic(err)
}

for event := range stream {
    if event.Err != nil {
        panic(event.Err)
    }
    if event.Done {
        fmt.Printf("\n\nTokens: %d in, %d out\n",
            event.Response.Usage.InputTokens,
            event.Response.Usage.OutputTokens)
        break
    }
    fmt.Print(event.Delta)
}
```

## Request Options

Override settings per-request:

```go
resp, err := client.Chat(ctx, messages,
    gains.WithModel("claude-opus-4-20250514"),
    gains.WithMaxTokens(1000),
    gains.WithTemperature(0.7),
)
```

## Types

### Message

```go
type Message struct {
    Role    Role   // RoleUser, RoleAssistant, RoleSystem
    Content string
}
```

### Response

```go
type Response struct {
    Content      string
    FinishReason string
    Usage        Usage
}

type Usage struct {
    InputTokens  int
    OutputTokens int
}
```

### StreamEvent

```go
type StreamEvent struct {
    Delta    string    // Incremental content
    Done     bool      // True on final event
    Response *Response // Complete response when Done
    Err      error     // Any streaming error
}
```

## Environment Variables

| Provider  | Variable          |
|-----------|-------------------|
| Anthropic | `ANTHROPIC_API_KEY` |
| OpenAI    | `OPENAI_API_KEY`    |
| Google    | `GOOGLE_API_KEY`    |

## License

MIT
