// Package models provides model constants for all supported AI providers.
//
// This package exposes typed model constants with pricing information
// without requiring users to import provider-specific packages. Use this
// package in conjunction with the client package for a complete solution.
//
// # Chat Models
//
// Use chat models with gains.WithModel():
//
//	import (
//	    "github.com/spetersoncode/gains"
//	    "github.com/spetersoncode/gains/client"
//	    "github.com/spetersoncode/gains/models"
//	)
//
//	c, _ := client.New(ctx, client.Config{
//	    Provider:  client.ProviderOpenAI,
//	    APIKey:    os.Getenv("OPENAI_API_KEY"),
//	    ChatModel: models.GPT52,
//	})
//	resp, err := c.Chat(ctx, messages, gains.WithModel(models.ClaudeSonnet45))
//
// # Image Models
//
// Use image models for generation:
//
//	c, _ := client.New(ctx, client.Config{
//	    Provider:   client.ProviderOpenAI,
//	    APIKey:     os.Getenv("OPENAI_API_KEY"),
//	    ImageModel: models.GPTImage1,
//	})
//
// # Embedding Models
//
// Use embedding models for vector embeddings:
//
//	c, _ := client.New(ctx, client.Config{
//	    Provider:       client.ProviderOpenAI,
//	    APIKey:         os.Getenv("OPENAI_API_KEY"),
//	    EmbeddingModel: models.TextEmbedding3Small,
//	})
//
// # Pricing Information
//
// All models include pricing methods for cost estimation:
//
//	pricing := models.GPT52.Pricing()
//	inputCost := float64(inputTokens) / 1_000_000 * pricing.InputPerMillion
//	outputCost := float64(outputTokens) / 1_000_000 * pricing.OutputPerMillion
//
// # Provider-Specific Pricing Features
//
// Some pricing fields are provider-specific. Use helper methods to check availability:
//
//	pricing := models.GPT52.Pricing()
//	if pricing.HasCachedPricing() {
//	    // OpenAI models support cached input pricing
//	    cachedCost := float64(cachedTokens) / 1_000_000 * pricing.CachedInputPerMillion
//	}
//
//	pricing := models.Gemini3Pro.Pricing()
//	if pricing.HasLongContextPricing() {
//	    // Google models have tiered pricing for >200K token contexts
//	    longInputCost := float64(tokens) / 1_000_000 * pricing.InputPerMillionLong
//	}
//
// # Available Providers
//
// Models are available for three providers:
//
//   - Anthropic: Claude models (chat only)
//   - OpenAI: GPT and O-series models (chat, image, embedding)
//   - Google: Gemini and Imagen models (chat, image, embedding)
package models
