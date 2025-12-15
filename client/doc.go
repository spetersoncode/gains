// Package client provides a unified multi-provider client for AI capabilities.
//
// The Client wraps provider-specific implementations and provides:
//
//   - Provider abstraction: Switch between Anthropic, OpenAI, and Google with config changes
//   - Feature detection: Check provider capabilities before making requests
//   - Automatic retries: Built-in exponential backoff for transient errors
//   - Unified defaults: Configure default models for chat, embeddings, and images
//
// # Basic Usage
//
// Create a client with a specific provider:
//
//	client, err := client.New(ctx, client.Config{
//	    Provider: client.ProviderOpenAI,
//	    APIKey:   os.Getenv("OPENAI_API_KEY"),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	resp, err := client.Chat(ctx, []gains.Message{
//	    {Role: gains.RoleUser, Content: "Hello!"},
//	})
//
// # Provider Configuration
//
// Configure default models for different capabilities:
//
//	client, err := client.New(ctx, client.Config{
//	    Provider:       client.ProviderOpenAI,
//	    APIKey:         os.Getenv("OPENAI_API_KEY"),
//	    ChatModel:      openai.GPT52,
//	    ImageModel:     openai.GPTImage1,
//	    EmbeddingModel: openai.TextEmbedding3Small,
//	})
//
// # Feature Checking
//
// Verify provider capabilities before use:
//
//	if client.SupportsFeature(client.FeatureImage) {
//	    resp, err := client.GenerateImage(ctx, "A sunset over mountains")
//	}
//
// Require features at construction time:
//
//	client, err := client.New(ctx, client.Config{
//	    Provider:         client.ProviderAnthropic,
//	    APIKey:           apiKey,
//	    RequiredFeatures: []client.Feature{client.FeatureChat, client.FeatureEmbedding},
//	})
//	// Returns ErrFeatureNotSupported since Anthropic doesn't support embeddings
//
// # Retry Configuration
//
// The client automatically retries transient errors (rate limits, timeouts, 5xx errors).
// Customize retry behavior:
//
//	client, err := client.New(ctx, client.Config{
//	    Provider:    client.ProviderOpenAI,
//	    APIKey:      apiKey,
//	    RetryConfig: &retry.Config{
//	        MaxAttempts:  5,
//	        InitialDelay: 500 * time.Millisecond,
//	        MaxDelay:     30 * time.Second,
//	    },
//	})
//
// Disable retries entirely:
//
//	cfg := retry.Disabled()
//	client, err := client.New(ctx, client.Config{
//	    Provider:    client.ProviderOpenAI,
//	    APIKey:      apiKey,
//	    RetryConfig: &cfg,
//	})
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
// # Switching Providers
//
// The unified interface makes it easy to switch providers:
//
//	providerName := os.Getenv("AI_PROVIDER")
//
//	var provider client.ProviderName
//	switch providerName {
//	case "anthropic":
//	    provider = client.ProviderAnthropic
//	case "google":
//	    provider = client.ProviderGoogle
//	default:
//	    provider = client.ProviderOpenAI
//	}
//
//	c, err := client.New(ctx, client.Config{
//	    Provider: provider,
//	    APIKey:   os.Getenv("API_KEY"),
//	})
package client
