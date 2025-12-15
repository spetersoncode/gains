package anthropic

// Model pricing last verified: December 14, 2025
// Source: https://platform.claude.com/docs/en/about-claude/models/overview

// ChatModel represents an Anthropic chat model.
type ChatModel string

const (
	// Claude 4.5 Family (Current)
	ClaudeOpus45   ChatModel = "claude-opus-4-5"   // Alias - auto-updates
	ClaudeSonnet45 ChatModel = "claude-sonnet-4-5" // Alias - auto-updates
	ClaudeHaiku45  ChatModel = "claude-haiku-4-5"  // Alias - auto-updates

	// Pinned versions (use for production stability)
	ClaudeOpus45_20251101   ChatModel = "claude-opus-4-5-20251101"
	ClaudeSonnet45_20250929 ChatModel = "claude-sonnet-4-5-20250929"
	ClaudeHaiku45_20251001  ChatModel = "claude-haiku-4-5-20251001"

	// DefaultChatModel is the recommended default model.
	DefaultChatModel ChatModel = ClaudeSonnet45
)

// ChatModelPricing contains pricing per million tokens (USD).
type ChatModelPricing struct {
	InputPerMillion  float64
	OutputPerMillion float64
}

// Pricing returns the pricing for this model.
func (m ChatModel) Pricing() ChatModelPricing {
	switch m {
	case ClaudeOpus45, ClaudeOpus45_20251101:
		return ChatModelPricing{InputPerMillion: 5.00, OutputPerMillion: 25.00}
	case ClaudeSonnet45, ClaudeSonnet45_20250929:
		return ChatModelPricing{InputPerMillion: 3.00, OutputPerMillion: 15.00}
	case ClaudeHaiku45, ClaudeHaiku45_20251001:
		return ChatModelPricing{InputPerMillion: 1.00, OutputPerMillion: 5.00}
	default:
		return ChatModelPricing{} // Unknown model
	}
}

// String returns the model identifier string.
func (m ChatModel) String() string { return string(m) }
