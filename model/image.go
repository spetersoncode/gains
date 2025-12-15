package model

// ImageModel represents an image generation model from any provider.
type ImageModel struct {
	id       string
	provider Provider
	pricing  ImagePricing
}

// String returns the API identifier for this model.
func (m ImageModel) String() string { return m.id }

// Provider returns which provider this model belongs to.
func (m ImageModel) Provider() Provider { return m.provider }

// Pricing returns the pricing for this model.
func (m ImageModel) Pricing() ImagePricing { return m.pricing }

// OpenAI Image Models
// Model pricing last verified: December 14, 2025
var (
	// GPT Image 1 Series
	GPTImage1     = ImageModel{id: "gpt-image-1", provider: ProviderOpenAI, pricing: ImagePricing{LowQuality: 0.011, MediumQuality: 0.042, HighQuality: 0.167}}
	GPTImage1Mini = ImageModel{id: "gpt-image-1-mini", provider: ProviderOpenAI, pricing: ImagePricing{LowQuality: 0.005, MediumQuality: 0.013, HighQuality: 0.052}}

	// DefaultGPTImageModel is the recommended default OpenAI image model.
	DefaultGPTImageModel = GPTImage1
)

// Google Imagen Models
// Model pricing last verified: December 14, 2025
var (
	// Imagen 4 Series
	Imagen4      = ImageModel{id: "imagen-4.0-generate-001", provider: ProviderGoogle, pricing: ImagePricing{PerImage: 0.04}}
	Imagen4Fast  = ImageModel{id: "imagen-4.0-fast-generate-001", provider: ProviderGoogle, pricing: ImagePricing{PerImage: 0.04}}
	Imagen4Ultra = ImageModel{id: "imagen-4.0-ultra-generate-001", provider: ProviderGoogle, pricing: ImagePricing{PerImage: 0.06}}

	// DefaultImagenModel is the recommended default Google image model.
	DefaultImagenModel = Imagen4
)
