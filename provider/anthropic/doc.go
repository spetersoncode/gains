// Package anthropic provides an Anthropic Claude API client implementing [gains.ChatProvider].
//
// This package wraps the official Anthropic Go SDK to provide Claude model access
// through the gains unified interface.
//
// # Supported Features
//
//   - Chat completions (streaming and non-streaming)
//   - Tool/function calling
//   - Multimodal inputs (images)
//   - Structured JSON output
//
// Note: Anthropic does not currently support embeddings or image generation.
//
// # Available Models
//
// Claude 4.5 family (latest):
//
//   - [ClaudeOpus45]: Most capable model for complex tasks
//   - [ClaudeSonnet45]: Balanced performance and cost (recommended default)
//   - [ClaudeHaiku45]: Fast and cost-effective for simpler tasks
//
// Use pinned versions (e.g., [ClaudeOpus45_20251101]) for production stability.
//
// # Basic Usage
//
//	client := anthropic.New(os.Getenv("ANTHROPIC_API_KEY"))
//
//	messages := []gains.Message{
//	    {Role: gains.RoleUser, Content: "Explain quantum computing briefly."},
//	}
//
//	resp, err := client.Chat(ctx, messages)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Content)
//
// # Model Selection
//
// Set a default model at client creation:
//
//	client := anthropic.New(apiKey, anthropic.WithModel(anthropic.ClaudeOpus45))
//
// Or per-request:
//
//	resp, err := client.Chat(ctx, messages, gains.WithModel(anthropic.ClaudeHaiku45))
//
// # Streaming
//
//	stream, err := client.ChatStream(ctx, messages)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for event := range stream {
//	    if event.Err != nil {
//	        log.Fatal(event.Err)
//	    }
//	    if event.Done {
//	        fmt.Printf("\nTokens: %d in, %d out\n",
//	            event.Response.Usage.InputTokens,
//	            event.Response.Usage.OutputTokens)
//	    } else {
//	        fmt.Print(event.Delta)
//	    }
//	}
//
// # Pricing
//
// Get model pricing programmatically:
//
//	pricing := anthropic.ClaudeSonnet45.Pricing()
//	fmt.Printf("Input: $%.2f/M tokens, Output: $%.2f/M tokens\n",
//	    pricing.InputPerMillion, pricing.OutputPerMillion)
package anthropic
