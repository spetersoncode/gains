package openai

// Model pricing last verified: December 14, 2025
// Source: https://platform.openai.com/docs/pricing

// ChatModel represents an OpenAI chat/completion model.
type ChatModel string

const (
	// GPT-5.2 Series (Latest - December 2025)
	GPT52    ChatModel = "gpt-5.2"     // Flagship model
	GPT52Pro ChatModel = "gpt-5.2-pro" // Enhanced reasoning

	// GPT-5.1 Series
	GPT51      ChatModel = "gpt-5.1"
	GPT51Mini  ChatModel = "gpt-5.1-mini"
	GPT51Codex ChatModel = "gpt-5.1-codex" // Optimized for code

	// GPT-5 Series
	GPT5     ChatModel = "gpt-5"
	GPT5Mini ChatModel = "gpt-5-mini"
	GPT5Nano ChatModel = "gpt-5-nano"
	GPT5Pro  ChatModel = "gpt-5-pro"

	// O-Series Reasoning Models
	O3     ChatModel = "o3"
	O3Mini ChatModel = "o3-mini"
	O4Mini ChatModel = "o4-mini"

	// DefaultChatModel is the recommended default model.
	DefaultChatModel ChatModel = GPT52
)

// ChatModelPricing contains pricing per million tokens (USD).
type ChatModelPricing struct {
	InputPerMillion       float64
	OutputPerMillion      float64
	CachedInputPerMillion float64 // For cached prompts
}

// Pricing returns the pricing for this model.
func (m ChatModel) Pricing() ChatModelPricing {
	switch m {
	case GPT52:
		return ChatModelPricing{InputPerMillion: 1.75, OutputPerMillion: 14.00, CachedInputPerMillion: 0.175}
	case GPT52Pro:
		return ChatModelPricing{InputPerMillion: 3.50, OutputPerMillion: 28.00, CachedInputPerMillion: 0.35}
	case GPT51:
		return ChatModelPricing{InputPerMillion: 1.25, OutputPerMillion: 10.00, CachedInputPerMillion: 0.125}
	case GPT51Mini:
		return ChatModelPricing{InputPerMillion: 0.30, OutputPerMillion: 1.25, CachedInputPerMillion: 0.03}
	case GPT51Codex:
		return ChatModelPricing{InputPerMillion: 1.25, OutputPerMillion: 10.00, CachedInputPerMillion: 0.125}
	case GPT5:
		return ChatModelPricing{InputPerMillion: 1.25, OutputPerMillion: 10.00, CachedInputPerMillion: 0.125}
	case GPT5Mini:
		return ChatModelPricing{InputPerMillion: 0.25, OutputPerMillion: 1.00, CachedInputPerMillion: 0.025}
	case GPT5Nano:
		return ChatModelPricing{InputPerMillion: 0.10, OutputPerMillion: 0.40, CachedInputPerMillion: 0.01}
	case GPT5Pro:
		return ChatModelPricing{InputPerMillion: 2.50, OutputPerMillion: 20.00, CachedInputPerMillion: 0.25}
	case O3:
		return ChatModelPricing{InputPerMillion: 2.00, OutputPerMillion: 16.00, CachedInputPerMillion: 0.20}
	case O3Mini:
		return ChatModelPricing{InputPerMillion: 0.50, OutputPerMillion: 2.00, CachedInputPerMillion: 0.05}
	case O4Mini:
		return ChatModelPricing{InputPerMillion: 0.50, OutputPerMillion: 2.00, CachedInputPerMillion: 0.05}
	default:
		return ChatModelPricing{}
	}
}

// String returns the model identifier string.
func (m ChatModel) String() string { return string(m) }

// ImageModel represents an OpenAI image generation model.
type ImageModel string

const (
	GPTImage1     ImageModel = "gpt-image-1"      // Flagship
	GPTImage1Mini ImageModel = "gpt-image-1-mini" // Budget option

	// DefaultImageModel is the recommended default image model.
	DefaultImageModel ImageModel = GPTImage1
)

// ImageModelPricing contains per-image pricing by quality (USD).
type ImageModelPricing struct {
	LowQuality    float64
	MediumQuality float64
	HighQuality   float64
}

// Pricing returns the pricing for this image model.
func (m ImageModel) Pricing() ImageModelPricing {
	switch m {
	case GPTImage1:
		return ImageModelPricing{LowQuality: 0.011, MediumQuality: 0.042, HighQuality: 0.167}
	case GPTImage1Mini:
		return ImageModelPricing{LowQuality: 0.005, MediumQuality: 0.013, HighQuality: 0.052}
	default:
		return ImageModelPricing{}
	}
}

// String returns the model identifier string.
func (m ImageModel) String() string { return string(m) }

// EmbeddingModel represents an OpenAI embedding model.
type EmbeddingModel string

const (
	TextEmbedding3Large EmbeddingModel = "text-embedding-3-large" // 3072 dimensions
	TextEmbedding3Small EmbeddingModel = "text-embedding-3-small" // 1536 dimensions

	// DefaultEmbeddingModel is the recommended default embedding model.
	DefaultEmbeddingModel EmbeddingModel = TextEmbedding3Small
)

// EmbeddingModelPricing contains per million token pricing (USD).
type EmbeddingModelPricing struct {
	PerMillion float64
}

// Pricing returns the pricing for this embedding model.
func (m EmbeddingModel) Pricing() EmbeddingModelPricing {
	switch m {
	case TextEmbedding3Large:
		return EmbeddingModelPricing{PerMillion: 0.13}
	case TextEmbedding3Small:
		return EmbeddingModelPricing{PerMillion: 0.02}
	default:
		return EmbeddingModelPricing{}
	}
}

// String returns the model identifier string.
func (m EmbeddingModel) String() string { return string(m) }
