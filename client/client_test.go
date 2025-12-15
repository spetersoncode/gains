package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderNameConstants(t *testing.T) {
	assert.Equal(t, ProviderName("anthropic"), ProviderAnthropic)
	assert.Equal(t, ProviderName("openai"), ProviderOpenAI)
	assert.Equal(t, ProviderName("google"), ProviderGoogle)
}

func TestFeatureConstants(t *testing.T) {
	assert.Equal(t, Feature("chat"), FeatureChat)
	assert.Equal(t, Feature("image"), FeatureImage)
	assert.Equal(t, Feature("embedding"), FeatureEmbedding)
}

func TestErrInvalidProvider(t *testing.T) {
	t.Run("Error returns formatted message", func(t *testing.T) {
		err := &ErrInvalidProvider{Provider: "unknown"}
		expected := `unknown provider: "unknown" (valid providers: anthropic, openai, google)`
		assert.Equal(t, expected, err.Error())
	})

	t.Run("Error with empty provider", func(t *testing.T) {
		err := &ErrInvalidProvider{Provider: ""}
		expected := `unknown provider: "" (valid providers: anthropic, openai, google)`
		assert.Equal(t, expected, err.Error())
	})
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

func TestNewWithInvalidProvider(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		provider ProviderName
	}{
		{"unknown provider", ProviderName("unknown")},
		{"empty provider", ProviderName("")},
		{"typo in provider", ProviderName("opnai")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Provider: tt.provider,
				APIKey:   "test-key",
			}

			client, err := New(ctx, cfg)
			assert.Nil(t, client)
			require.Error(t, err)

			var invalidErr *ErrInvalidProvider
			assert.ErrorAs(t, err, &invalidErr)
			assert.Equal(t, string(tt.provider), invalidErr.Provider)
		})
	}
}

func TestNewWithUnsupportedRequiredFeature(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name             string
		provider         ProviderName
		requiredFeatures []Feature
		expectedFeature  string
	}{
		{
			name:             "anthropic image not supported",
			provider:         ProviderAnthropic,
			requiredFeatures: []Feature{FeatureImage},
			expectedFeature:  "image",
		},
		{
			name:             "anthropic embedding not supported",
			provider:         ProviderAnthropic,
			requiredFeatures: []Feature{FeatureEmbedding},
			expectedFeature:  "embedding",
		},
		{
			name:             "anthropic multiple unsupported",
			provider:         ProviderAnthropic,
			requiredFeatures: []Feature{FeatureChat, FeatureImage},
			expectedFeature:  "image",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Provider:         tt.provider,
				APIKey:           "test-key",
				RequiredFeatures: tt.requiredFeatures,
			}

			client, err := New(ctx, cfg)
			assert.Nil(t, client)
			require.Error(t, err)

			var featureErr *ErrFeatureNotSupported
			assert.ErrorAs(t, err, &featureErr)
			assert.Equal(t, tt.expectedFeature, featureErr.Feature)
		})
	}
}

func TestNewWithValidProviders(t *testing.T) {
	ctx := context.Background()

	t.Run("creates Anthropic client", func(t *testing.T) {
		cfg := Config{
			Provider: ProviderAnthropic,
			APIKey:   "test-anthropic-key",
		}

		client, err := New(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, client)

		assert.Equal(t, ProviderAnthropic, client.Provider())
		assert.True(t, client.SupportsFeature(FeatureChat))
		assert.False(t, client.SupportsFeature(FeatureImage))
		assert.False(t, client.SupportsFeature(FeatureEmbedding))
	})

	t.Run("creates OpenAI client", func(t *testing.T) {
		cfg := Config{
			Provider: ProviderOpenAI,
			APIKey:   "test-openai-key",
		}

		client, err := New(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, client)

		assert.Equal(t, ProviderOpenAI, client.Provider())
		assert.True(t, client.SupportsFeature(FeatureChat))
		assert.True(t, client.SupportsFeature(FeatureImage))
		assert.True(t, client.SupportsFeature(FeatureEmbedding))
	})

	t.Run("creates Google client", func(t *testing.T) {
		cfg := Config{
			Provider: ProviderGoogle,
			APIKey:   "test-google-key",
		}

		client, err := New(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, client)

		assert.Equal(t, ProviderGoogle, client.Provider())
		assert.True(t, client.SupportsFeature(FeatureChat))
		assert.True(t, client.SupportsFeature(FeatureImage))
		assert.True(t, client.SupportsFeature(FeatureEmbedding))
	})
}

func TestNewWithCustomModels(t *testing.T) {
	ctx := context.Background()

	t.Run("sets custom chat model for Anthropic", func(t *testing.T) {
		cfg := Config{
			Provider:  ProviderAnthropic,
			APIKey:    "test-key",
			ChatModel: "claude-3-opus",
		}

		client, err := New(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("sets custom models for OpenAI", func(t *testing.T) {
		cfg := Config{
			Provider:       ProviderOpenAI,
			APIKey:         "test-key",
			ChatModel:      "gpt-4-turbo",
			ImageModel:     "dall-e-2",
			EmbeddingModel: "text-embedding-3-large",
		}

		client, err := New(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, client)
	})
}

func TestNewWithRequiredFeatures(t *testing.T) {
	ctx := context.Background()

	t.Run("succeeds when required features are supported", func(t *testing.T) {
		cfg := Config{
			Provider:         ProviderOpenAI,
			APIKey:           "test-key",
			RequiredFeatures: []Feature{FeatureChat, FeatureImage, FeatureEmbedding},
		}

		client, err := New(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("succeeds with chat only for Anthropic", func(t *testing.T) {
		cfg := Config{
			Provider:         ProviderAnthropic,
			APIKey:           "test-key",
			RequiredFeatures: []Feature{FeatureChat},
		}

		client, err := New(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("succeeds with empty required features", func(t *testing.T) {
		cfg := Config{
			Provider:         ProviderAnthropic,
			APIKey:           "test-key",
			RequiredFeatures: []Feature{},
		}

		client, err := New(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, client)
	})
}

func TestClientSupportsFeature(t *testing.T) {
	ctx := context.Background()

	t.Run("Anthropic capabilities", func(t *testing.T) {
		cfg := Config{
			Provider: ProviderAnthropic,
			APIKey:   "test-key",
		}

		client, err := New(ctx, cfg)
		require.NoError(t, err)

		assert.True(t, client.SupportsFeature(FeatureChat))
		assert.False(t, client.SupportsFeature(FeatureImage))
		assert.False(t, client.SupportsFeature(FeatureEmbedding))
		assert.False(t, client.SupportsFeature(Feature("unknown")))
	})

	t.Run("OpenAI capabilities", func(t *testing.T) {
		cfg := Config{
			Provider: ProviderOpenAI,
			APIKey:   "test-key",
		}

		client, err := New(ctx, cfg)
		require.NoError(t, err)

		assert.True(t, client.SupportsFeature(FeatureChat))
		assert.True(t, client.SupportsFeature(FeatureImage))
		assert.True(t, client.SupportsFeature(FeatureEmbedding))
	})

	t.Run("Google capabilities", func(t *testing.T) {
		cfg := Config{
			Provider: ProviderGoogle,
			APIKey:   "test-key",
		}

		client, err := New(ctx, cfg)
		require.NoError(t, err)

		assert.True(t, client.SupportsFeature(FeatureChat))
		assert.True(t, client.SupportsFeature(FeatureImage))
		assert.True(t, client.SupportsFeature(FeatureEmbedding))
	})
}

func TestClientProvider(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		provider ProviderName
	}{
		{"Anthropic", ProviderAnthropic},
		{"OpenAI", ProviderOpenAI},
		{"Google", ProviderGoogle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Provider: tt.provider,
				APIKey:   "test-key",
			}

			client, err := New(ctx, cfg)
			require.NoError(t, err)

			assert.Equal(t, tt.provider, client.Provider())
		})
	}
}

func TestClientGenerateImageFeatureCheck(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error for Anthropic", func(t *testing.T) {
		cfg := Config{
			Provider: ProviderAnthropic,
			APIKey:   "test-key",
		}

		client, err := New(ctx, cfg)
		require.NoError(t, err)

		_, err = client.GenerateImage(ctx, "test prompt")
		require.Error(t, err)

		var featureErr *ErrFeatureNotSupported
		assert.ErrorAs(t, err, &featureErr)
		assert.Equal(t, "image", featureErr.Feature)
	})
}

func TestClientEmbedFeatureCheck(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error for Anthropic", func(t *testing.T) {
		cfg := Config{
			Provider: ProviderAnthropic,
			APIKey:   "test-key",
		}

		client, err := New(ctx, cfg)
		require.NoError(t, err)

		_, err = client.Embed(ctx, []string{"test text"})
		require.Error(t, err)

		var featureErr *ErrFeatureNotSupported
		assert.ErrorAs(t, err, &featureErr)
		assert.Equal(t, "embedding", featureErr.Feature)
	})
}

func TestConfigStruct(t *testing.T) {
	t.Run("creates config with all fields", func(t *testing.T) {
		cfg := Config{
			Provider:         ProviderOpenAI,
			APIKey:           "sk-test-key",
			ChatModel:        "gpt-4",
			ImageModel:       "dall-e-3",
			EmbeddingModel:   "text-embedding-3-small",
			RequiredFeatures: []Feature{FeatureChat, FeatureImage},
		}

		assert.Equal(t, ProviderOpenAI, cfg.Provider)
		assert.Equal(t, "sk-test-key", cfg.APIKey)
		assert.Equal(t, "gpt-4", cfg.ChatModel)
		assert.Equal(t, "dall-e-3", cfg.ImageModel)
		assert.Equal(t, "text-embedding-3-small", cfg.EmbeddingModel)
		assert.Len(t, cfg.RequiredFeatures, 2)
	})
}

func TestProviderCapabilities(t *testing.T) {
	// Test that the providerCapabilities map is correctly defined
	t.Run("Anthropic has correct capabilities", func(t *testing.T) {
		caps := providerCapabilities[ProviderAnthropic]
		assert.True(t, caps[FeatureChat])
		assert.False(t, caps[FeatureImage])
		assert.False(t, caps[FeatureEmbedding])
	})

	t.Run("OpenAI has correct capabilities", func(t *testing.T) {
		caps := providerCapabilities[ProviderOpenAI]
		assert.True(t, caps[FeatureChat])
		assert.True(t, caps[FeatureImage])
		assert.True(t, caps[FeatureEmbedding])
	})

	t.Run("Google has correct capabilities", func(t *testing.T) {
		caps := providerCapabilities[ProviderGoogle]
		assert.True(t, caps[FeatureChat])
		assert.True(t, caps[FeatureImage])
		assert.True(t, caps[FeatureEmbedding])
	})
}
