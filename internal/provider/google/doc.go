// Package google provides a Google Gemini API client implementing gains provider interfaces.
//
// This package wraps the Google GenAI SDK to provide Gemini model access
// through the gains unified interface.
//
// # Supported Features
//
//   - Chat completions via [gains.ChatProvider]
//   - Text embeddings via [gains.EmbeddingProvider]
//   - Image generation via [gains.ImageProvider] (Imagen models)
//   - Tool/function calling
//   - Multimodal inputs (images)
//   - Structured JSON output with schema validation
//
// # Available Models
//
// Chat models:
//
//   - [Gemini3Pro]: Latest flagship model
//   - [Gemini3DeepThink]: Enhanced reasoning capabilities
//   - [Gemini25Pro]: Previous generation flagship
//   - [Gemini25Flash]: Fast and cost-effective (recommended default)
//   - [Gemini25FlashLite]: Most cost-effective option
//
// Image models:
//
//   - [Imagen4]: Standard image generation
//   - [Imagen4Fast]: Faster generation
//   - [Imagen4Ultra]: Highest quality
//
// Embedding models:
//
//   - [GeminiEmbedding001]: 3072 dimensions (recommended)
//
// # Basic Usage
//
// Note: Google client requires a context for initialization:
//
//	client, err := google.New(ctx, os.Getenv("GOOGLE_API_KEY"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	messages := []gains.Message{
//	    {Role: gains.RoleUser, Content: "What's the weather like on Mars?"},
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
//	client, err := google.New(ctx, apiKey, google.WithModel(google.Gemini3Pro))
//
// Or override per-request:
//
//	resp, err := client.Chat(ctx, messages, gains.WithModel(google.Gemini25FlashLite))
//
// # Embeddings
//
// Google embeddings support task type hints for better results:
//
//	resp, err := client.Embed(ctx, []string{"search query"},
//	    gains.WithEmbeddingModel(google.GeminiEmbedding001),
//	    gains.WithEmbeddingTaskType(gains.EmbeddingTaskTypeRetrievalQuery),
//	)
//
// # Image Generation
//
//	resp, err := client.GenerateImage(ctx, "A serene Japanese garden",
//	    gains.WithImageModel(google.Imagen4Ultra),
//	    gains.WithImageCount(2),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, img := range resp.Images {
//	    fmt.Println(img.URL)
//	}
//
// # Pricing
//
// Google has tiered pricing based on context length:
//
//	pricing := google.Gemini25Pro.Pricing()
//	fmt.Printf("Standard: $%.2f/M in, $%.2f/M out\n",
//	    pricing.InputPerMillion, pricing.OutputPerMillion)
//	fmt.Printf("Long context (>200K): $%.2f/M in, $%.2f/M out\n",
//	    pricing.InputPerMillionLong, pricing.OutputPerMillionLong)
package google
