// Package gains provides a unified interface for interacting with LLM providers.
//
// The gains library abstracts away provider-specific APIs, allowing you to write
// code once and switch between Anthropic (Claude), OpenAI (GPT), and Google (Gemini)
// with minimal changes.
//
// # Core Interfaces
//
// The library defines three main provider interfaces:
//
//   - [ChatProvider]: Send conversations and receive responses (text, streaming, tool calls)
//   - [EmbeddingProvider]: Generate vector embeddings for text
//   - [ImageProvider]: Generate images from text prompts
//
// Use the [github.com/spetersoncode/gains/client] package as the entry point
// for provider access, and the [github.com/spetersoncode/gains/models] package
// for model selection.
//
// # Basic Usage
//
// Send a simple chat message:
//
//	c, err := client.New(ctx, client.Config{
//	    Provider: client.ProviderAnthropic,
//	    APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	messages := []gains.Message{
//	    {Role: gains.RoleUser, Content: "What is the capital of France?"},
//	}
//
//	resp, err := c.Chat(ctx, messages)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Content)
//
// # Streaming Responses
//
// For real-time output, use ChatStream:
//
//	stream, err := c.ChatStream(ctx, messages)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for event := range stream {
//	    if event.Err != nil {
//	        log.Fatal(event.Err)
//	    }
//	    fmt.Print(event.Delta)
//	}
//
// # Configuration Options
//
// Customize requests with functional options:
//
//	resp, err := c.Chat(ctx, messages,
//	    gains.WithModel(models.ClaudeOpus45),
//	    gains.WithMaxTokens(1000),
//	    gains.WithTemperature(0.7),
//	)
//
// # Tool Calling
//
// Define tools that the model can invoke:
//
//	tools := []gains.Tool{
//	    {
//	        Name:        "get_weather",
//	        Description: "Get current weather for a location",
//	        Parameters:  json.RawMessage(`{
//	            "type": "object",
//	            "properties": {
//	                "location": {"type": "string", "description": "City name"}
//	            },
//	            "required": ["location"]
//	        }`),
//	    },
//	}
//
//	resp, err := c.Chat(ctx, messages, gains.WithTools(tools))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Handle tool calls
//	for _, call := range resp.ToolCalls {
//	    fmt.Printf("Tool: %s, Args: %s\n", call.Name, call.Arguments)
//	}
//
// # Structured Output
//
// Request JSON responses with schema validation:
//
//	schema := &gains.ResponseSchema{
//	    Name:   "answer",
//	    Schema: json.RawMessage(`{"type":"object","properties":{"answer":{"type":"string"}}}`),
//	}
//
//	resp, err := c.Chat(ctx, messages, gains.WithResponseSchema(schema))
//
// # Multimodal Messages
//
// Send images alongside text:
//
//	messages := []gains.Message{
//	    {
//	        Role: gains.RoleUser,
//	        Parts: []gains.ContentPart{
//	            gains.NewTextPart("What's in this image?"),
//	            gains.NewImageURLPart("https://example.com/image.jpg"),
//	        },
//	    },
//	}
//
// # Higher-Level Abstractions
//
// For more complex use cases, see:
//
//   - [github.com/spetersoncode/gains/agent]: Autonomous tool-calling agents
//   - [github.com/spetersoncode/gains/workflow]: Composable AI pipelines
//   - [github.com/spetersoncode/gains/retry]: Retry logic with exponential backoff
package gains
