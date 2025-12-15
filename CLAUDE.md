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
├── workflow/         # Composable pipelines: Chain, Parallel, Router
├── models/           # Model constants with pricing information
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
- **agent**: Tool-calling loops with max steps, timeouts, approval workflows, and parallel tool execution.
- **workflow**: Step interface with Chain (sequential), Parallel (concurrent), Router (conditional), and ClassifierRouter (LLM-based routing).
- **models**: Type-safe model selection with pricing data for cost estimation.

### Patterns

- Functional options: `WithModel()`, `WithMaxTokens()`, `WithTemperature()`, etc.
- Channel-based streaming throughout
- Context cancellation respected everywhere
- Internal packages for implementation details (provider, retry, store)

## Coding Conventions

- Always use conventional commits (feat, fix, refactor, docs, test, chore)
- Follow Go idioms: functional options, interfaces, error handling
- Keep provider implementations internal; expose only through client package
- Use `models` package constants for model selection
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
