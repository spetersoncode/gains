// Package gains provides a unified interface for interacting with LLM providers.
//
// The gains library abstracts away provider-specific APIs, allowing you to write
// code once and use models from Anthropic (Claude), OpenAI (GPT), and Google (Gemini).
// Models know their provider, so routing happens automatically.
//
// # Import Convention
//
// We recommend importing with the "ai" alias for cleaner code:
//
//	import ai "github.com/spetersoncode/gains"
//
// All examples in this documentation use this convention.
//
// # Core Types
//
// The library defines these key types:
//
//   - [Provider]: Identifies a provider (Anthropic, OpenAI, Google)
//   - [Model]: Interface for models that know their provider
//   - [Message]: Conversation messages with roles and content
//   - [Response]: Chat responses with content, tool calls, and usage
//
// Use the [github.com/spetersoncode/gains/client] package as the entry point
// and the [github.com/spetersoncode/gains/model] package for model selection.
//
// # Basic Usage
//
// Create a client with API keys and default models:
//
//	c := client.New(client.Config{
//	    APIKeys: client.APIKeys{
//	        Anthropic: os.Getenv("ANTHROPIC_API_KEY"),
//	    },
//	    Defaults: client.Defaults{
//	        Chat: model.ClaudeSonnet45,
//	    },
//	})
//
//	messages := []ai.Message{
//	    {Role: ai.RoleUser, Content: "What is the capital of France?"},
//	}
//
//	resp, err := c.Chat(ctx, messages)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Content)
//
// # Model-Centric Routing
//
// Models determine their provider. Override the default with WithModel:
//
//	// Uses default (routes to Anthropic)
//	resp, _ := c.Chat(ctx, messages)
//
//	// Override with GPT (routes to OpenAI)
//	resp, _ := c.Chat(ctx, messages, ai.WithModel(model.GPT52))
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
//	    ai.WithModel(model.ClaudeOpus45),
//	    ai.WithMaxTokens(1000),
//	    ai.WithTemperature(0.7),
//	)
//
// # Tool Calling
//
// Define tools that the model can invoke:
//
//	tools := []ai.Tool{
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
//	resp, err := c.Chat(ctx, messages, ai.WithTools(tools))
//
// # Higher-Level Abstractions
//
// For more complex use cases, see:
//
//   - [github.com/spetersoncode/gains/agent]: Autonomous tool-calling agents
//   - [github.com/spetersoncode/gains/workflow]: Composable AI pipelines
package gains
