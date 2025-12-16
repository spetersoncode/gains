package model

import ai "github.com/spetersoncode/gains"

// ChatModel represents a chat/completion model from any provider.
type ChatModel struct {
	id       string
	provider ai.Provider
	pricing  ChatPricing
}

// String returns the API identifier for this model.
func (m ChatModel) String() string { return m.id }

// Provider returns which provider this model belongs to.
func (m ChatModel) Provider() ai.Provider { return m.provider }

// Pricing returns the pricing for this model.
func (m ChatModel) Pricing() ChatPricing { return m.pricing }

// Anthropic Claude Models
// Model pricing last verified: December 14, 2025
var (
	// Claude 4.5 Family (Current) - auto-updating aliases
	ClaudeOpus45   = ChatModel{id: "claude-opus-4-5", provider: ai.ProviderAnthropic, pricing: ChatPricing{InputPerMillion: 5.00, OutputPerMillion: 25.00}}
	ClaudeSonnet45 = ChatModel{id: "claude-sonnet-4-5", provider: ai.ProviderAnthropic, pricing: ChatPricing{InputPerMillion: 3.00, OutputPerMillion: 15.00}}
	ClaudeHaiku45  = ChatModel{id: "claude-haiku-4-5", provider: ai.ProviderAnthropic, pricing: ChatPricing{InputPerMillion: 1.00, OutputPerMillion: 5.00}}

	// Pinned versions (use for production stability)
	ClaudeOpus45_20251101   = ChatModel{id: "claude-opus-4-5-20251101", provider: ai.ProviderAnthropic, pricing: ChatPricing{InputPerMillion: 5.00, OutputPerMillion: 25.00}}
	ClaudeSonnet45_20250929 = ChatModel{id: "claude-sonnet-4-5-20250929", provider: ai.ProviderAnthropic, pricing: ChatPricing{InputPerMillion: 3.00, OutputPerMillion: 15.00}}
	ClaudeHaiku45_20251001  = ChatModel{id: "claude-haiku-4-5-20251001", provider: ai.ProviderAnthropic, pricing: ChatPricing{InputPerMillion: 1.00, OutputPerMillion: 5.00}}

	// DefaultClaudeModel is the recommended default Anthropic model.
	DefaultClaudeModel = ClaudeSonnet45
)

// OpenAI GPT and O-Series Models
// Model pricing last verified: December 14, 2025
var (
	// GPT-5.2 Series (Latest - December 2025)
	GPT52    = ChatModel{id: "gpt-5.2", provider: ai.ProviderOpenAI, pricing: ChatPricing{InputPerMillion: 1.75, OutputPerMillion: 14.00, CachedInputPerMillion: 0.175}}
	GPT52Pro = ChatModel{id: "gpt-5.2-pro", provider: ai.ProviderOpenAI, pricing: ChatPricing{InputPerMillion: 3.50, OutputPerMillion: 28.00, CachedInputPerMillion: 0.35}}

	// GPT-5.1 Series
	GPT51      = ChatModel{id: "gpt-5.1", provider: ai.ProviderOpenAI, pricing: ChatPricing{InputPerMillion: 1.25, OutputPerMillion: 10.00, CachedInputPerMillion: 0.125}}
	GPT51Mini  = ChatModel{id: "gpt-5.1-mini", provider: ai.ProviderOpenAI, pricing: ChatPricing{InputPerMillion: 0.30, OutputPerMillion: 1.25, CachedInputPerMillion: 0.03}}
	GPT51Codex = ChatModel{id: "gpt-5.1-codex", provider: ai.ProviderOpenAI, pricing: ChatPricing{InputPerMillion: 1.25, OutputPerMillion: 10.00, CachedInputPerMillion: 0.125}}

	// GPT-5 Series
	GPT5     = ChatModel{id: "gpt-5", provider: ai.ProviderOpenAI, pricing: ChatPricing{InputPerMillion: 1.25, OutputPerMillion: 10.00, CachedInputPerMillion: 0.125}}
	GPT5Mini = ChatModel{id: "gpt-5-mini", provider: ai.ProviderOpenAI, pricing: ChatPricing{InputPerMillion: 0.25, OutputPerMillion: 1.00, CachedInputPerMillion: 0.025}}
	GPT5Nano = ChatModel{id: "gpt-5-nano", provider: ai.ProviderOpenAI, pricing: ChatPricing{InputPerMillion: 0.10, OutputPerMillion: 0.40, CachedInputPerMillion: 0.01}}
	GPT5Pro  = ChatModel{id: "gpt-5-pro", provider: ai.ProviderOpenAI, pricing: ChatPricing{InputPerMillion: 2.50, OutputPerMillion: 20.00, CachedInputPerMillion: 0.25}}

	// O-Series Reasoning Models
	O3     = ChatModel{id: "o3", provider: ai.ProviderOpenAI, pricing: ChatPricing{InputPerMillion: 2.00, OutputPerMillion: 16.00, CachedInputPerMillion: 0.20}}
	O3Mini = ChatModel{id: "o3-mini", provider: ai.ProviderOpenAI, pricing: ChatPricing{InputPerMillion: 0.50, OutputPerMillion: 2.00, CachedInputPerMillion: 0.05}}
	O4Mini = ChatModel{id: "o4-mini", provider: ai.ProviderOpenAI, pricing: ChatPricing{InputPerMillion: 0.50, OutputPerMillion: 2.00, CachedInputPerMillion: 0.05}}

	// DefaultGPTModel is the recommended default OpenAI model.
	DefaultGPTModel = GPT52
)

// Google Gemini Models
// Model pricing last verified: December 14, 2025
var (
	// Gemini 3.0 (Latest - November 2025)
	Gemini3Pro       = ChatModel{id: "gemini-3.0-pro", provider: ai.ProviderGoogle, pricing: ChatPricing{InputPerMillion: 2.00, OutputPerMillion: 12.00, InputPerMillionLong: 4.00, OutputPerMillionLong: 18.00}}
	Gemini3DeepThink = ChatModel{id: "gemini-3.0-deep-think", provider: ai.ProviderGoogle, pricing: ChatPricing{InputPerMillion: 4.00, OutputPerMillion: 24.00, InputPerMillionLong: 8.00, OutputPerMillionLong: 36.00}}

	// Gemini 2.5 Series
	Gemini25Pro       = ChatModel{id: "gemini-2.5-pro", provider: ai.ProviderGoogle, pricing: ChatPricing{InputPerMillion: 1.25, OutputPerMillion: 10.00, InputPerMillionLong: 2.50, OutputPerMillionLong: 15.00}}
	Gemini25Flash     = ChatModel{id: "gemini-2.5-flash", provider: ai.ProviderGoogle, pricing: ChatPricing{InputPerMillion: 0.15, OutputPerMillion: 0.60, InputPerMillionLong: 0.15, OutputPerMillionLong: 0.60}}
	Gemini25FlashLite = ChatModel{id: "gemini-2.5-flash-lite", provider: ai.ProviderGoogle, pricing: ChatPricing{InputPerMillion: 0.075, OutputPerMillion: 0.30, InputPerMillionLong: 0.075, OutputPerMillionLong: 0.30}}

	// DefaultGeminiModel is the recommended default Google model.
	DefaultGeminiModel = Gemini25Flash
)
