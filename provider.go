package gains

// Provider identifies an AI provider.
type Provider string

// String returns the provider identifier.
func (p Provider) String() string { return string(p) }

// Supported providers.
const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
	ProviderGoogle    Provider = "google"
	ProviderVertex    Provider = "vertex"
)
