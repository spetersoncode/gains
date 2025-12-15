package model

// ChatPricing contains pricing per million tokens (USD) for chat models.
// Fields are zero if not applicable to a specific provider's model.
type ChatPricing struct {
	// InputPerMillion is the standard input token pricing (all providers).
	InputPerMillion float64
	// OutputPerMillion is the standard output token pricing (all providers).
	OutputPerMillion float64
	// CachedInputPerMillion is for cached/prompt-cached input tokens (OpenAI only).
	// Check HasCachedPricing() before using.
	CachedInputPerMillion float64
	// InputPerMillionLong is for long context >200K tokens (Google only).
	// Check HasLongContextPricing() before using.
	InputPerMillionLong float64
	// OutputPerMillionLong is for long context >200K tokens (Google only).
	// Check HasLongContextPricing() before using.
	OutputPerMillionLong float64
}

// HasCachedPricing returns true if the model supports cached input pricing.
func (p ChatPricing) HasCachedPricing() bool {
	return p.CachedInputPerMillion > 0
}

// HasLongContextPricing returns true if the model has tiered pricing for long context.
func (p ChatPricing) HasLongContextPricing() bool {
	return p.InputPerMillionLong > 0 || p.OutputPerMillionLong > 0
}

// ImagePricing contains image generation pricing (USD).
// Different providers use different pricing models.
type ImagePricing struct {
	// PerImage is a flat per-image price (Google Imagen).
	PerImage float64
	// LowQuality is the price for low quality images (OpenAI).
	LowQuality float64
	// MediumQuality is the price for medium quality images (OpenAI).
	MediumQuality float64
	// HighQuality is the price for high quality images (OpenAI).
	HighQuality float64
}

// HasQualityTiers returns true if the model has quality-based pricing tiers.
func (p ImagePricing) HasQualityTiers() bool {
	return p.LowQuality > 0 || p.MediumQuality > 0 || p.HighQuality > 0
}

// HasFlatPricing returns true if the model uses flat per-image pricing.
func (p ImagePricing) HasFlatPricing() bool {
	return p.PerImage > 0
}

// EmbeddingPricing contains embedding pricing per million tokens (USD).
type EmbeddingPricing struct {
	// PerMillion is the price per million tokens.
	PerMillion float64
}
