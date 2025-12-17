// Package client provides a unified multi-provider client for AI capabilities.
//
// The Client wraps provider-specific implementations and provides:
//
//   - Model-centric routing: Models know their provider; switching is automatic
//   - Multi-provider support: Configure all providers at once, use any model
//   - Automatic retries: Built-in exponential backoff for transient errors
//   - Event emission: Observable operations via channel
//
// # Basic Usage
//
// Create a client with API keys and default models:
//
//	c := client.New(client.Config{
//	    APIKeys: client.APIKeys{
//	        Anthropic: os.Getenv("ANTHROPIC_API_KEY"),
//	        OpenAI:    os.Getenv("OPENAI_API_KEY"),
//	    },
//	    Defaults: client.Defaults{
//	        Chat: model.ClaudeSonnet45,
//	    },
//	})
//
//	resp, err := c.Chat(ctx, []ai.Message{
//	    {Role: ai.RoleUser, Content: "Hello!"},
//	})
//
// # Model-Centric Routing
//
// Models determine their provider. The client routes automatically:
//
//	// Uses default model (routes to Anthropic)
//	resp, _ := c.Chat(ctx, messages)
//
//	// Override with GPT-5.2 (routes to OpenAI)
//	resp, _ := c.Chat(ctx, messages, ai.WithModel(model.GPT52))
//
//	// Override with Gemini (routes to Google)
//	resp, _ := c.Chat(ctx, messages, ai.WithModel(model.Gemini25Flash))
//
// # Feature Detection
//
// Check provider capabilities before use:
//
//	if c.SupportsFeature(client.FeatureImage) {
//	    resp, err := c.GenerateImage(ctx, "A sunset over mountains")
//	}
//
// # Provider Capabilities
//
// Feature support by provider:
//
//	| Provider  | Chat | Embeddings | Images |
//	|-----------|------|------------|--------|
//	| Anthropic | Yes  | No         | No     |
//	| OpenAI    | Yes  | Yes        | Yes    |
//	| Google    | Yes  | Yes        | Yes    |
//
// # Retry Configuration
//
// The client automatically retries transient errors (rate limits, timeouts, 5xx errors).
// Customize retry behavior:
//
//	c := client.New(client.Config{
//	    APIKeys: client.APIKeys{OpenAI: os.Getenv("OPENAI_API_KEY")},
//	    RetryConfig: &retry.Config{
//	        MaxAttempts:  5,
//	        InitialDelay: 500 * time.Millisecond,
//	        MaxDelay:     30 * time.Second,
//	    },
//	})
//
// # Events
//
// Observe operations via an event channel:
//
//	events := make(chan client.Event, 100)
//	c := client.New(client.Config{
//	    APIKeys: client.APIKeys{OpenAI: os.Getenv("OPENAI_API_KEY")},
//	    Events:  events,
//	})
//
//	go func() {
//	    for e := range events {
//	        fmt.Printf("[%s] %s took %v\n", e.Type, e.Operation, e.Duration)
//	    }
//	}()
package client
