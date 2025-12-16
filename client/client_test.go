package client

import (
	"testing"

	ai "github.com/spetersoncode/gains"
	"github.com/stretchr/testify/assert"
)

// testModel implements gains.Model for testing.
type testModel struct {
	id       string
	provider ai.Provider
}

func (m testModel) String() string      { return m.id }
func (m testModel) Provider() ai.Provider { return m.provider }

func TestFeatureConstants(t *testing.T) {
	assert.Equal(t, Feature("chat"), FeatureChat)
	assert.Equal(t, Feature("image"), FeatureImage)
	assert.Equal(t, Feature("embedding"), FeatureEmbedding)
}

func TestErrFeatureNotSupported(t *testing.T) {
	t.Run("Error returns formatted message", func(t *testing.T) {
		err := &ErrFeatureNotSupported{
			Provider: "anthropic",
			Feature:  "image",
		}
		expected := "anthropic provider does not support image"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Error for embedding", func(t *testing.T) {
		err := &ErrFeatureNotSupported{
			Provider: "anthropic",
			Feature:  "embedding",
		}
		expected := "anthropic provider does not support embedding"
		assert.Equal(t, expected, err.Error())
	})
}

func TestErrMissingAPIKey(t *testing.T) {
	t.Run("Error with model", func(t *testing.T) {
		err := &ErrMissingAPIKey{Provider: "anthropic", Model: "claude-sonnet"}
		expected := `no API key configured for anthropic (required by model "claude-sonnet")`
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Error without model", func(t *testing.T) {
		err := &ErrMissingAPIKey{Provider: "openai"}
		expected := "no API key configured for openai"
		assert.Equal(t, expected, err.Error())
	})
}

func TestErrNoModel(t *testing.T) {
	t.Run("Error returns formatted message", func(t *testing.T) {
		err := &ErrNoModel{Operation: "chat"}
		expected := "no model specified for chat and no default configured"
		assert.Equal(t, expected, err.Error())
	})
}

func TestNew(t *testing.T) {
	t.Run("creates client with API keys", func(t *testing.T) {
		cfg := Config{
			APIKeys: APIKeys{
				Anthropic: "test-anthropic-key",
				OpenAI:    "test-openai-key",
			},
		}

		c := New(cfg)
		assert.NotNil(t, c)
	})

	t.Run("creates client with defaults", func(t *testing.T) {
		chatModel := testModel{id: "claude-sonnet", provider: ai.ProviderAnthropic}
		cfg := Config{
			APIKeys: APIKeys{
				Anthropic: "test-key",
			},
			Defaults: Defaults{
				Chat: chatModel,
			},
		}

		c := New(cfg)
		assert.NotNil(t, c)
	})
}

func TestSupportsFeature(t *testing.T) {
	t.Run("chat supported with any API key", func(t *testing.T) {
		c := New(Config{
			APIKeys: APIKeys{Anthropic: "key"},
		})
		assert.True(t, c.SupportsFeature(FeatureChat))
	})

	t.Run("image supported with OpenAI or Google", func(t *testing.T) {
		c1 := New(Config{
			APIKeys: APIKeys{OpenAI: "key"},
		})
		assert.True(t, c1.SupportsFeature(FeatureImage))

		c2 := New(Config{
			APIKeys: APIKeys{Google: "key"},
		})
		assert.True(t, c2.SupportsFeature(FeatureImage))

		c3 := New(Config{
			APIKeys: APIKeys{Anthropic: "key"},
		})
		assert.False(t, c3.SupportsFeature(FeatureImage))
	})

	t.Run("embedding supported with OpenAI or Google", func(t *testing.T) {
		c1 := New(Config{
			APIKeys: APIKeys{OpenAI: "key"},
		})
		assert.True(t, c1.SupportsFeature(FeatureEmbedding))

		c2 := New(Config{
			APIKeys: APIKeys{Google: "key"},
		})
		assert.True(t, c2.SupportsFeature(FeatureEmbedding))

		c3 := New(Config{
			APIKeys: APIKeys{Anthropic: "key"},
		})
		assert.False(t, c3.SupportsFeature(FeatureEmbedding))
	})

	t.Run("unknown feature not supported", func(t *testing.T) {
		c := New(Config{
			APIKeys: APIKeys{OpenAI: "key", Anthropic: "key", Google: "key"},
		})
		assert.False(t, c.SupportsFeature(Feature("unknown")))
	})
}

func TestProviderCapabilities(t *testing.T) {
	t.Run("Anthropic has correct capabilities", func(t *testing.T) {
		caps := providerCapabilities[ai.ProviderAnthropic]
		assert.True(t, caps[FeatureChat])
		assert.False(t, caps[FeatureImage])
		assert.False(t, caps[FeatureEmbedding])
	})

	t.Run("OpenAI has correct capabilities", func(t *testing.T) {
		caps := providerCapabilities[ai.ProviderOpenAI]
		assert.True(t, caps[FeatureChat])
		assert.True(t, caps[FeatureImage])
		assert.True(t, caps[FeatureEmbedding])
	})

	t.Run("Google has correct capabilities", func(t *testing.T) {
		caps := providerCapabilities[ai.ProviderGoogle]
		assert.True(t, caps[FeatureChat])
		assert.True(t, caps[FeatureImage])
		assert.True(t, caps[FeatureEmbedding])
	})
}

func TestConfigStruct(t *testing.T) {
	t.Run("creates config with all fields", func(t *testing.T) {
		chatModel := testModel{id: "gpt-4", provider: ai.ProviderOpenAI}
		imageModel := testModel{id: "dall-e-3", provider: ai.ProviderOpenAI}
		embedModel := testModel{id: "text-embedding-3-small", provider: ai.ProviderOpenAI}

		cfg := Config{
			APIKeys: APIKeys{
				Anthropic: "anthropic-key",
				OpenAI:    "openai-key",
				Google:    "google-key",
			},
			Defaults: Defaults{
				Chat:      chatModel,
				Image:     imageModel,
				Embedding: embedModel,
			},
		}

		assert.Equal(t, "anthropic-key", cfg.APIKeys.Anthropic)
		assert.Equal(t, "openai-key", cfg.APIKeys.OpenAI)
		assert.Equal(t, "google-key", cfg.APIKeys.Google)
		assert.Equal(t, "gpt-4", cfg.Defaults.Chat.String())
		assert.Equal(t, "dall-e-3", cfg.Defaults.Image.String())
		assert.Equal(t, "text-embedding-3-small", cfg.Defaults.Embedding.String())
	})
}
