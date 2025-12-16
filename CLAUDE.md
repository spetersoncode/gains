# CLAUDE.md

**gains** - Go AI Native Scaffold. A production-ready, Go-idiomatic generative AI library providing unified interfaces for Anthropic, OpenAI, and Google.

## Commands

```bash
go build ./...   # Build all packages
go vet ./...     # Run static analysis
go test ./...    # Run all tests
```

## Project Structure

```
gains/
├── *.go              # Core types: Message, Response, Tool, Options, interfaces
├── client/           # Unified multi-provider client with retry & events
├── agent/            # Autonomous tool-calling agent orchestration
├── workflow/         # Composable pipelines: Chain, Parallel, Router, TypedPromptStep
├── tool/             # Tool infrastructure: Registry, binding, built-in tools
├── model/            # Model constants with pricing information
├── internal/
│   ├── provider/     # Provider implementations (anthropic, openai, google)
│   ├── retry/        # Exponential backoff with jitter
│   └── store/        # State management (Store, TypedStore, MessageStore)
└── cmd/demo/         # Example implementations
```

## Architecture

### Core Interfaces

- `ChatProvider` - Chat/ChatStream for all providers
- `EmbeddingProvider` - Text embeddings (OpenAI, Google)
- `ImageProvider` - Image generation (OpenAI, Google)

### Key Packages

- **client**: Entry point for most users. Unified access to all provider capabilities with automatic retry and event emission.
- **agent**: Tool-calling loops with max steps, timeouts, approval workflows, and parallel tool execution. Uses `tool.Registry` for tool management.
- **workflow**: Step interface with Chain (sequential), Parallel (concurrent), Router (conditional), ClassifierRouter (LLM-based routing), Loop (iterative), and TypedPromptStep (auto-unmarshaling structured output). Uses `Key[T]` for type-safe state access: `Get`, `Set`, `MustGet`, `GetOr`.
- **tool**: Tool infrastructure including Registry, Handler types, function binding with auto schema generation from struct tags, and built-in tools (file, HTTP, search, client tools).
- **model**: Type-safe model selection with pricing data for cost estimation.

### Patterns

- Functional options: `WithModel()`, `WithMaxTokens()`, `WithTemperature()`, etc.
- Channel-based streaming throughout
- Context cancellation respected everywhere
- Internal packages for implementation details (provider, retry, store)

### Struct Tags for Schema Generation

Use struct tags to define JSON schemas for tool arguments and structured output:

```go
type Args struct {
    Location string  `json:"location" desc:"City name" required:"true"`
    Unit     string  `json:"unit" enum:"celsius,fahrenheit"`
    Days     int     `json:"days" min:"1" max:"7"`
}
schema := gains.MustSchemaFor[Args]()
```

Supported tags: `json`, `desc`, `required`, `enum`, `min`, `max`, `minLength`, `maxLength`, `pattern`, `default`, `minItems`, `maxItems`

## Coding Conventions

- Always use conventional commits (feat, fix, refactor, docs, test, chore)
- Follow Go idioms: functional options, interfaces, error handling
- Keep provider implementations internal; expose only through client package
- Use `model` package constants for model selection
- Streaming methods return `<-chan EventType`
- All public APIs should have godoc comments

## Provider Capabilities

| Provider  | Chat | Images | Embeddings |
|-----------|:----:|:------:|:----------:|
| Anthropic | ✓    | -      | -          |
| OpenAI    | ✓    | ✓      | ✓          |
| Google    | ✓    | ✓      | ✓          |

## Environment Variables

- `ANTHROPIC_API_KEY` - Anthropic Claude API
- `OPENAI_API_KEY` - OpenAI API
- `GOOGLE_API_KEY` - Google Gemini API
