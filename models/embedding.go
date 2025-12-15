package models

// EmbeddingModel represents an embedding model from any provider.
type EmbeddingModel struct {
	id         string
	provider   Provider
	dimensions int
	pricing    EmbeddingPricing
}

// String returns the API identifier for this model.
func (m EmbeddingModel) String() string { return m.id }

// Provider returns which provider this model belongs to.
func (m EmbeddingModel) Provider() Provider { return m.provider }

// Dimensions returns the output vector dimensions for this model.
func (m EmbeddingModel) Dimensions() int { return m.dimensions }

// Pricing returns the pricing for this model.
func (m EmbeddingModel) Pricing() EmbeddingPricing { return m.pricing }

// OpenAI Embedding Models
// Model pricing last verified: December 14, 2025
var (
	// Text Embedding 3 Series
	TextEmbedding3Large = EmbeddingModel{id: "text-embedding-3-large", provider: ProviderOpenAI, dimensions: 3072, pricing: EmbeddingPricing{PerMillion: 0.13}}
	TextEmbedding3Small = EmbeddingModel{id: "text-embedding-3-small", provider: ProviderOpenAI, dimensions: 1536, pricing: EmbeddingPricing{PerMillion: 0.02}}

	// DefaultOpenAIEmbeddingModel is the recommended default OpenAI embedding model.
	DefaultOpenAIEmbeddingModel = TextEmbedding3Small
)

// Google Embedding Models
// Model pricing last verified: December 14, 2025
var (
	// Gemini Embedding
	GeminiEmbedding001 = EmbeddingModel{id: "gemini-embedding-001", provider: ProviderGoogle, dimensions: 3072, pricing: EmbeddingPricing{PerMillion: 0.15}}

	// DefaultGoogleEmbeddingModel is the recommended default Google embedding model.
	DefaultGoogleEmbeddingModel = GeminiEmbedding001
)
