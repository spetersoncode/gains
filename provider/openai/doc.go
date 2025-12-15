// Package openai provides an OpenAI API client implementing gains provider interfaces.
//
// This package wraps the official OpenAI Go SDK to provide GPT model access
// through the gains unified interface.
//
// # Supported Features
//
//   - Chat completions via [gains.ChatProvider]
//   - Text embeddings via [gains.EmbeddingProvider]
//   - Image generation via [gains.ImageProvider]
//   - Tool/function calling
//   - Multimodal inputs (images)
//   - Structured JSON output with schema validation
//
// # Available Models
//
// Chat models:
//
//   - [GPT52]: Flagship model (recommended default)
//   - [GPT52Pro]: Enhanced reasoning capabilities
//   - [GPT51], [GPT51Mini], [GPT51Codex]: Previous generation
//   - [O3], [O3Mini], [O4Mini]: Reasoning-optimized models
//
// Image models:
//
//   - [GPTImage1]: High-quality image generation
//   - [GPTImage1Mini]: Cost-effective option
//
// Embedding models:
//
//   - [TextEmbedding3Large]: 3072 dimensions
//   - [TextEmbedding3Small]: 1536 dimensions (recommended default)
//
// # Basic Usage
//
//	client := openai.New(os.Getenv("OPENAI_API_KEY"))
//
//	messages := []gains.Message{
//	    {Role: gains.RoleSystem, Content: "You are a helpful assistant."},
//	    {Role: gains.RoleUser, Content: "Hello!"},
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
//	client := openai.New(apiKey, openai.WithModel(openai.GPT52Pro))
//
// Or override per-request:
//
//	resp, err := client.Chat(ctx, messages, gains.WithModel(openai.GPT51Mini))
//
// # Embeddings
//
//	texts := []string{"Hello world", "Goodbye world"}
//	resp, err := client.Embed(ctx, texts,
//	    gains.WithEmbeddingModel(openai.TextEmbedding3Large),
//	    gains.WithEmbeddingDimensions(1024),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Embedding dimensions: %d\n", len(resp.Embeddings[0]))
//
// # Image Generation
//
//	resp, err := client.GenerateImage(ctx, "A futuristic city at sunset",
//	    gains.WithImageModel(openai.GPTImage1),
//	    gains.WithImageSize(gains.ImageSize1024x1024),
//	    gains.WithImageQuality(gains.ImageQualityHD),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(resp.Images[0].URL)
//
// # Pricing
//
// Get model pricing programmatically:
//
//	chatPricing := openai.GPT52.Pricing()
//	fmt.Printf("Input: $%.2f/M, Output: $%.2f/M, Cached: $%.3f/M\n",
//	    chatPricing.InputPerMillion,
//	    chatPricing.OutputPerMillion,
//	    chatPricing.CachedInputPerMillion)
//
//	embedPricing := openai.TextEmbedding3Small.Pricing()
//	fmt.Printf("Embeddings: $%.2f/M tokens\n", embedPricing.PerMillion)
package openai
