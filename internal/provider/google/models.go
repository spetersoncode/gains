package google

// Model pricing last verified: December 14, 2025
// Source: https://ai.google.dev/gemini-api/docs/pricing

// ChatModel represents a Google Gemini chat model.
type ChatModel string

const (
	// Gemini 3.0 (Latest - November 2025)
	Gemini3Pro       ChatModel = "gemini-3.0-pro"
	Gemini3DeepThink ChatModel = "gemini-3.0-deep-think"

	// Gemini 2.5 Series
	Gemini25Pro       ChatModel = "gemini-2.5-pro"
	Gemini25Flash     ChatModel = "gemini-2.5-flash"
	Gemini25FlashLite ChatModel = "gemini-2.5-flash-lite"

	// DefaultChatModel is the recommended default model.
	DefaultChatModel ChatModel = Gemini25Flash
)

// ChatModelPricing contains pricing per million tokens (USD).
// Some models have tiered pricing based on context length.
type ChatModelPricing struct {
	InputPerMillion      float64 // Standard (<=200K tokens)
	OutputPerMillion     float64 // Standard
	InputPerMillionLong  float64 // Long context (>200K tokens)
	OutputPerMillionLong float64 // Long context
}

// Pricing returns the pricing for this model.
func (m ChatModel) Pricing() ChatModelPricing {
	switch m {
	case Gemini3Pro:
		return ChatModelPricing{
			InputPerMillion: 2.00, OutputPerMillion: 12.00,
			InputPerMillionLong: 4.00, OutputPerMillionLong: 18.00,
		}
	case Gemini3DeepThink:
		return ChatModelPricing{
			InputPerMillion: 4.00, OutputPerMillion: 24.00,
			InputPerMillionLong: 8.00, OutputPerMillionLong: 36.00,
		}
	case Gemini25Pro:
		return ChatModelPricing{
			InputPerMillion: 1.25, OutputPerMillion: 10.00,
			InputPerMillionLong: 2.50, OutputPerMillionLong: 15.00,
		}
	case Gemini25Flash:
		return ChatModelPricing{
			InputPerMillion: 0.15, OutputPerMillion: 0.60,
			InputPerMillionLong: 0.15, OutputPerMillionLong: 0.60,
		}
	case Gemini25FlashLite:
		return ChatModelPricing{
			InputPerMillion: 0.075, OutputPerMillion: 0.30,
			InputPerMillionLong: 0.075, OutputPerMillionLong: 0.30,
		}
	default:
		return ChatModelPricing{}
	}
}

// String returns the model identifier string.
func (m ChatModel) String() string { return string(m) }

// ImageModel represents a Google Imagen model.
type ImageModel string

const (
	Imagen4      ImageModel = "imagen-4.0-generate-001"
	Imagen4Fast  ImageModel = "imagen-4.0-fast-generate-001"
	Imagen4Ultra ImageModel = "imagen-4.0-ultra-generate-001"

	// DefaultImageModel is the recommended default image model.
	DefaultImageModel ImageModel = Imagen4
)

// ImageModelPricing contains per-image pricing (USD).
type ImageModelPricing struct {
	PerImage float64
}

// Pricing returns the pricing for this image model.
func (m ImageModel) Pricing() ImageModelPricing {
	switch m {
	case Imagen4, Imagen4Fast:
		return ImageModelPricing{PerImage: 0.04}
	case Imagen4Ultra:
		return ImageModelPricing{PerImage: 0.06}
	default:
		return ImageModelPricing{}
	}
}

// String returns the model identifier string.
func (m ImageModel) String() string { return string(m) }

// EmbeddingModel represents a Google text embedding model.
type EmbeddingModel string

const (
	GeminiEmbedding001 EmbeddingModel = "gemini-embedding-001" // 3072 dimensions, recommended

	// DefaultEmbeddingModel is the recommended default embedding model.
	DefaultEmbeddingModel EmbeddingModel = GeminiEmbedding001
)

// EmbeddingModelPricing contains per million token pricing (USD).
type EmbeddingModelPricing struct {
	PerMillion float64
}

// Pricing returns the pricing for this embedding model.
func (m EmbeddingModel) Pricing() EmbeddingModelPricing {
	switch m {
	case GeminiEmbedding001:
		return EmbeddingModelPricing{PerMillion: 0.15}
	default:
		return EmbeddingModelPricing{}
	}
}

// String returns the model identifier string.
func (m EmbeddingModel) String() string { return string(m) }
