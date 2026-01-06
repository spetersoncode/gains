package observer

import (
	"sync"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/model"
)

// CostTracker accumulates token usage and calculates costs.
// It is safe for concurrent use.
type CostTracker struct {
	mu           sync.Mutex
	inputTokens  int
	outputTokens int
	pricing      model.ChatPricing
}

// NewCostTracker creates a cost tracker with the given model pricing.
func NewCostTracker(pricing model.ChatPricing) *CostTracker {
	return &CostTracker{pricing: pricing}
}

// NewCostTrackerForModel creates a cost tracker for the given chat model.
func NewCostTrackerForModel(m model.ChatModel) *CostTracker {
	return &CostTracker{pricing: m.Pricing()}
}

// Track updates the tracker with usage from an event.
// Returns true if the event contained usage information.
func (ct *CostTracker) Track(e event.Event) bool {
	// Usage is available on MessageEnd and RunEnd events that have a Response
	if e.Response == nil {
		return false
	}
	if e.Response.Usage.InputTokens == 0 && e.Response.Usage.OutputTokens == 0 {
		return false
	}

	ct.mu.Lock()
	ct.inputTokens += e.Response.Usage.InputTokens
	ct.outputTokens += e.Response.Usage.OutputTokens
	ct.mu.Unlock()
	return true
}

// Usage returns the accumulated token usage.
func (ct *CostTracker) Usage() ai.Usage {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ai.Usage{
		InputTokens:  ct.inputTokens,
		OutputTokens: ct.outputTokens,
	}
}

// Cost returns the accumulated cost in USD.
func (ct *CostTracker) Cost() float64 {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return model.CalculateCost(ai.Usage{
		InputTokens:  ct.inputTokens,
		OutputTokens: ct.outputTokens,
	}, ct.pricing)
}

// Reset clears the accumulated usage.
func (ct *CostTracker) Reset() {
	ct.mu.Lock()
	ct.inputTokens = 0
	ct.outputTokens = 0
	ct.mu.Unlock()
}

// Summary returns a human-readable summary of usage and cost.
func (ct *CostTracker) Summary() CostSummary {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	usage := ai.Usage{
		InputTokens:  ct.inputTokens,
		OutputTokens: ct.outputTokens,
	}
	return CostSummary{
		InputTokens:  ct.inputTokens,
		OutputTokens: ct.outputTokens,
		TotalTokens:  ct.inputTokens + ct.outputTokens,
		Cost:         model.CalculateCost(usage, ct.pricing),
	}
}

// CostSummary contains token usage and cost information.
type CostSummary struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	Cost         float64 `json:"cost_usd"`
}

// CalculateCost computes the cost in USD for the given usage and model.
// This is a convenience wrapper around model.CalculateCost.
func CalculateCost(usage ai.Usage, m model.ChatModel) float64 {
	return model.CalculateCost(usage, m.Pricing())
}

// CalculateCostWithPricing computes the cost in USD for the given usage and pricing.
// This is a convenience wrapper around model.CalculateCost.
func CalculateCostWithPricing(usage ai.Usage, pricing model.ChatPricing) float64 {
	return model.CalculateCost(usage, pricing)
}
