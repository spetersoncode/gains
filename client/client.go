// Package client provides a unified client for accessing all AI provider capabilities.
package client

import (
	"context"
	"fmt"

	"github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/provider/anthropic"
	"github.com/spetersoncode/gains/provider/google"
	"github.com/spetersoncode/gains/provider/openai"
)

// ProviderName identifies supported AI providers.
type ProviderName string

const (
	ProviderAnthropic ProviderName = "anthropic"
	ProviderOpenAI    ProviderName = "openai"
	ProviderGoogle    ProviderName = "google"
)

// Feature represents a capability that a provider may support.
type Feature string

const (
	FeatureChat      Feature = "chat"
	FeatureImage     Feature = "image"
	FeatureEmbedding Feature = "embedding"
)

// providerCapabilities defines which features each provider supports.
var providerCapabilities = map[ProviderName]map[Feature]bool{
	ProviderAnthropic: {
		FeatureChat:      true,
		FeatureImage:     false,
		FeatureEmbedding: false,
	},
	ProviderOpenAI: {
		FeatureChat:      true,
		FeatureImage:     true,
		FeatureEmbedding: true,
	},
	ProviderGoogle: {
		FeatureChat:      true,
		FeatureImage:     true,
		FeatureEmbedding: true,
	},
}

// Config holds configuration for creating a unified client.
type Config struct {
	// Provider specifies which AI provider to use.
	Provider ProviderName
	// APIKey is the authentication key for the provider.
	APIKey string
	// ChatModel is the default model for chat operations.
	ChatModel string
	// ImageModel is the default model for image generation.
	ImageModel string
	// EmbeddingModel is the default model for embeddings.
	EmbeddingModel string
	// RequiredFeatures lists features that must be available.
	// Construction fails if any required feature is unsupported.
	RequiredFeatures []Feature
}

// ErrFeatureNotSupported is returned when a feature is unavailable for the provider.
type ErrFeatureNotSupported struct {
	Provider string
	Feature  string
}

func (e *ErrFeatureNotSupported) Error() string {
	return fmt.Sprintf("%s provider does not support %s", e.Provider, e.Feature)
}

// ErrInvalidProvider is returned when an unknown provider name is specified.
type ErrInvalidProvider struct {
	Provider string
}

func (e *ErrInvalidProvider) Error() string {
	return fmt.Sprintf("unknown provider: %q (valid providers: anthropic, openai, google)", e.Provider)
}

// Client is a unified interface to all AI provider capabilities.
type Client struct {
	provider       ProviderName
	chatProvider   gains.ChatProvider
	imageProvider  gains.ImageProvider
	embedProvider  gains.EmbeddingProvider
	chatModel      string
	imageModel     string
	embeddingModel string
}

// New creates a unified client with the given configuration.
// It validates that all required features are supported by the provider.
// The context is required for Google provider initialization.
func New(ctx context.Context, cfg Config) (*Client, error) {
	// Validate provider name
	caps, ok := providerCapabilities[cfg.Provider]
	if !ok {
		return nil, &ErrInvalidProvider{Provider: string(cfg.Provider)}
	}

	// Validate required features
	for _, feature := range cfg.RequiredFeatures {
		if !caps[feature] {
			return nil, &ErrFeatureNotSupported{
				Provider: string(cfg.Provider),
				Feature:  string(feature),
			}
		}
	}

	// Create provider-specific client
	var (
		chatProv  gains.ChatProvider
		imageProv gains.ImageProvider
		embedProv gains.EmbeddingProvider
	)

	switch cfg.Provider {
	case ProviderAnthropic:
		var opts []anthropic.ClientOption
		if cfg.ChatModel != "" {
			opts = append(opts, anthropic.WithModel(cfg.ChatModel))
		}
		ac := anthropic.New(cfg.APIKey, opts...)
		chatProv = ac

	case ProviderOpenAI:
		var opts []openai.ClientOption
		if cfg.ChatModel != "" {
			opts = append(opts, openai.WithModel(cfg.ChatModel))
		}
		oc := openai.New(cfg.APIKey, opts...)
		chatProv = oc
		imageProv = oc
		embedProv = oc

	case ProviderGoogle:
		var opts []google.ClientOption
		if cfg.ChatModel != "" {
			opts = append(opts, google.WithModel(cfg.ChatModel))
		}
		gc, err := google.New(ctx, cfg.APIKey, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create Google client: %w", err)
		}
		chatProv = gc
		imageProv = gc
		embedProv = gc
	}

	return &Client{
		provider:       cfg.Provider,
		chatProvider:   chatProv,
		imageProvider:  imageProv,
		embedProvider:  embedProv,
		chatModel:      cfg.ChatModel,
		imageModel:     cfg.ImageModel,
		embeddingModel: cfg.EmbeddingModel,
	}, nil
}

// Chat sends a conversation and returns a complete response.
func (c *Client) Chat(ctx context.Context, messages []gains.Message, opts ...gains.Option) (*gains.Response, error) {
	return c.chatProvider.Chat(ctx, messages, opts...)
}

// ChatStream sends a conversation and returns a channel of streaming events.
func (c *Client) ChatStream(ctx context.Context, messages []gains.Message, opts ...gains.Option) (<-chan gains.StreamEvent, error) {
	return c.chatProvider.ChatStream(ctx, messages, opts...)
}

// GenerateImage creates images from a text prompt.
// Returns ErrFeatureNotSupported if the provider doesn't support image generation.
func (c *Client) GenerateImage(ctx context.Context, prompt string, opts ...gains.ImageOption) (*gains.ImageResponse, error) {
	if c.imageProvider == nil {
		return nil, &ErrFeatureNotSupported{
			Provider: string(c.provider),
			Feature:  string(FeatureImage),
		}
	}

	// Prepend default model if set
	if c.imageModel != "" {
		opts = append([]gains.ImageOption{gains.WithImageModel(c.imageModel)}, opts...)
	}

	return c.imageProvider.GenerateImage(ctx, prompt, opts...)
}

// Embed generates embeddings for the provided texts.
// Returns ErrFeatureNotSupported if the provider doesn't support embeddings.
func (c *Client) Embed(ctx context.Context, texts []string, opts ...gains.EmbeddingOption) (*gains.EmbeddingResponse, error) {
	if c.embedProvider == nil {
		return nil, &ErrFeatureNotSupported{
			Provider: string(c.provider),
			Feature:  string(FeatureEmbedding),
		}
	}

	// Prepend default model if set
	if c.embeddingModel != "" {
		opts = append([]gains.EmbeddingOption{gains.WithEmbeddingModel(c.embeddingModel)}, opts...)
	}

	return c.embedProvider.Embed(ctx, texts, opts...)
}

// SupportsFeature returns true if the client's provider supports the given feature.
func (c *Client) SupportsFeature(f Feature) bool {
	switch f {
	case FeatureChat:
		return c.chatProvider != nil
	case FeatureImage:
		return c.imageProvider != nil
	case FeatureEmbedding:
		return c.embedProvider != nil
	default:
		return false
	}
}

// Provider returns the name of the underlying provider.
func (c *Client) Provider() ProviderName {
	return c.provider
}
