// Package model provides model constants for all supported AI providers.
//
// This package exposes typed model constants with pricing information.
// Models know their provider, enabling automatic routing in the client.
//
// # Chat Models
//
// Use chat models with ai.WithModel() or as client defaults:
//
//	import (
//	    ai "github.com/spetersoncode/gains"
//	    "github.com/spetersoncode/gains/client"
//	    "github.com/spetersoncode/gains/model"
//	)
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
//	// Override with different model (routes to OpenAI)
//	resp, err := c.Chat(ctx, messages, ai.WithModel(model.GPT52))
//
// # Image Models
//
// Use image models for generation:
//
//	c := client.New(client.Config{
//	    APIKeys: client.APIKeys{OpenAI: os.Getenv("OPENAI_API_KEY")},
//	    Defaults: client.Defaults{
//	        Image: model.GPTImage1,
//	    },
//	})
//
// # Embedding Models
//
// Use embedding models for vector embeddings:
//
//	c := client.New(client.Config{
//	    APIKeys: client.APIKeys{OpenAI: os.Getenv("OPENAI_API_KEY")},
//	    Defaults: client.Defaults{
//	        Embedding: model.TextEmbedding3Small,
//	    },
//	})
//
// # Pricing Information
//
// All models include pricing methods for cost estimation:
//
//	pricing := model.GPT52.Pricing()
//	inputCost := float64(inputTokens) / 1_000_000 * pricing.InputPerMillion
//	outputCost := float64(outputTokens) / 1_000_000 * pricing.OutputPerMillion
//
// # Provider-Specific Pricing Features
//
// Some pricing fields are provider-specific. Use helper methods to check availability:
//
//	pricing := model.GPT52.Pricing()
//	if pricing.HasCachedPricing() {
//	    // OpenAI models support cached input pricing
//	    cachedCost := float64(cachedTokens) / 1_000_000 * pricing.CachedInputPerMillion
//	}
//
//	pricing := model.Gemini3Pro.Pricing()
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
package model
