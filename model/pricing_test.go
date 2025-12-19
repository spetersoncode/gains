package model

import (
	"testing"

	ai "github.com/spetersoncode/gains"
	"github.com/stretchr/testify/assert"
)

func TestCalculateCost(t *testing.T) {
	pricing := ChatPricing{
		InputPerMillion:  1.00,
		OutputPerMillion: 2.00,
	}

	t.Run("calculates cost for standard usage", func(t *testing.T) {
		usage := ai.Usage{InputTokens: 1000, OutputTokens: 500}
		cost := CalculateCost(usage, pricing)
		// 1000/1M * $1 + 500/1M * $2 = $0.001 + $0.001 = $0.002
		assert.InDelta(t, 0.002, cost, 0.0001)
	})

	t.Run("calculates cost for million tokens", func(t *testing.T) {
		usage := ai.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000}
		cost := CalculateCost(usage, pricing)
		// 1M/1M * $1 + 1M/1M * $2 = $1 + $2 = $3
		assert.InDelta(t, 3.0, cost, 0.0001)
	})

	t.Run("returns zero for zero usage", func(t *testing.T) {
		usage := ai.Usage{InputTokens: 0, OutputTokens: 0}
		cost := CalculateCost(usage, pricing)
		assert.Equal(t, 0.0, cost)
	})
}

func TestChatModel_Cost(t *testing.T) {
	t.Run("calculates cost using model pricing", func(t *testing.T) {
		// Claude Sonnet 4.5: $3/M input, $15/M output
		usage := ai.Usage{InputTokens: 10000, OutputTokens: 5000}
		cost := ClaudeSonnet45.Cost(usage)
		// 10000/1M * $3 + 5000/1M * $15 = $0.03 + $0.075 = $0.105
		assert.InDelta(t, 0.105, cost, 0.0001)
	})

	t.Run("works with different models", func(t *testing.T) {
		usage := ai.Usage{InputTokens: 100000, OutputTokens: 50000}

		// Compare costs between models
		sonnetCost := ClaudeSonnet45.Cost(usage)
		haikuCost := ClaudeHaiku45.Cost(usage)

		// Haiku should be cheaper than Sonnet
		assert.Greater(t, sonnetCost, haikuCost)
	})
}

func TestChatPricing_HasCachedPricing(t *testing.T) {
	t.Run("returns true when cached pricing set", func(t *testing.T) {
		pricing := GPT52.Pricing()
		assert.True(t, pricing.HasCachedPricing())
	})

	t.Run("returns false when no cached pricing", func(t *testing.T) {
		pricing := ClaudeSonnet45.Pricing()
		assert.False(t, pricing.HasCachedPricing())
	})
}

func TestChatPricing_HasLongContextPricing(t *testing.T) {
	t.Run("returns true for Google models", func(t *testing.T) {
		pricing := Gemini25Pro.Pricing()
		assert.True(t, pricing.HasLongContextPricing())
	})

	t.Run("returns false for non-Google models", func(t *testing.T) {
		pricing := ClaudeSonnet45.Pricing()
		assert.False(t, pricing.HasLongContextPricing())
	})
}
