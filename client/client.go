package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/internal/provider/anthropic"
	"github.com/spetersoncode/gains/internal/provider/google"
	"github.com/spetersoncode/gains/internal/provider/openai"
	"github.com/spetersoncode/gains/internal/retry"
)


// Feature represents a capability that a provider may support.
type Feature string

const (
	FeatureChat      Feature = "chat"
	FeatureImage     Feature = "image"
	FeatureEmbedding Feature = "embedding"
)

// providerCapabilities defines which features each provider supports.
var providerCapabilities = map[ai.Provider]map[Feature]bool{
	ai.ProviderAnthropic: {
		FeatureChat:      true,
		FeatureImage:     false,
		FeatureEmbedding: false,
	},
	ai.ProviderOpenAI: {
		FeatureChat:      true,
		FeatureImage:     true,
		FeatureEmbedding: true,
	},
	ai.ProviderGoogle: {
		FeatureChat:      true,
		FeatureImage:     true,
		FeatureEmbedding: true,
	},
}

// APIKeys holds API keys for different providers.
// Only configure keys for providers you intend to use.
type APIKeys struct {
	Anthropic string
	OpenAI    string
	Google    string
}

// Defaults holds default models for each capability.
// The model's provider determines which backend is used.
type Defaults struct {
	Chat      ai.Model
	Embedding ai.Model
	Image     ai.Model
}

// Config holds configuration for creating a unified client.
type Config struct {
	// APIKeys contains authentication keys for each provider.
	// Only configure keys for providers you intend to use.
	APIKeys APIKeys

	// Defaults contains default models for each capability.
	// The model's provider determines which backend is used.
	Defaults Defaults

	// RetryConfig configures retry behavior for transient errors.
	// If nil, uses default retry configuration (10 retries with exponential backoff).
	RetryConfig *retry.Config

	// Events is an optional channel for receiving client operation events.
	// Events are sent non-blocking; if the channel is full, events are dropped.
	Events chan<- Event
}

// ErrFeatureNotSupported is returned when a feature is unavailable for the provider.
type ErrFeatureNotSupported struct {
	Provider string
	Feature  string
}

func (e *ErrFeatureNotSupported) Error() string {
	return fmt.Sprintf("%s provider does not support %s", e.Provider, e.Feature)
}

// ErrMissingAPIKey is returned when a model is used but no API key
// is configured for that model's provider.
type ErrMissingAPIKey struct {
	Provider string
	Model    string
}

func (e *ErrMissingAPIKey) Error() string {
	if e.Model != "" {
		return fmt.Sprintf("no API key configured for %s (required by model %q)", e.Provider, e.Model)
	}
	return fmt.Sprintf("no API key configured for %s", e.Provider)
}

// ErrNoModel is returned when no model is specified and no default is configured.
type ErrNoModel struct {
	Operation string
}

// operationHints maps operation names to their config field and option function.
var operationHints = map[string]struct {
	configField string
	optionFunc  string
}{
	"chat":        {"Defaults.Chat", "gains.WithModel()"},
	"chat_stream": {"Defaults.Chat", "gains.WithModel()"},
	"image":       {"Defaults.Image", "gains.WithImageModel()"},
	"embedding":   {"Defaults.Embedding", "gains.WithEmbeddingModel()"},
}

func (e *ErrNoModel) Error() string {
	if hint, ok := operationHints[e.Operation]; ok {
		return fmt.Sprintf("no model specified for %s: set client.Config %s or use %s",
			e.Operation, hint.configField, hint.optionFunc)
	}
	return fmt.Sprintf("no model specified for %s and no default configured", e.Operation)
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithDefaultTemperature sets the default temperature for chat requests.
// Per-request options override this default.
func WithDefaultTemperature(t float64) ClientOption {
	return func(c *Client) {
		c.defaultChatOpts = append(c.defaultChatOpts, ai.WithTemperature(t))
	}
}

// WithDefaultMaxTokens sets the default max tokens for chat requests.
// Per-request options override this default.
func WithDefaultMaxTokens(n int) ClientOption {
	return func(c *Client) {
		c.defaultChatOpts = append(c.defaultChatOpts, ai.WithMaxTokens(n))
	}
}

// WithDefaultChatOptions sets default options for all chat requests.
// Per-request options override these defaults.
func WithDefaultChatOptions(opts ...ai.Option) ClientOption {
	return func(c *Client) {
		c.defaultChatOpts = append(c.defaultChatOpts, opts...)
	}
}

// Client is a unified interface to all AI provider capabilities.
// Provider clients are lazily initialized when first needed.
type Client struct {
	apiKeys         APIKeys
	defaults        Defaults
	retryConfig     retry.Config
	events          chan<- Event
	defaultChatOpts []ai.Option

	// Lazy-initialized providers (protected by mutex)
	mu              sync.RWMutex
	anthropicClient *anthropic.Client
	openaiClient    *openai.Client
	googleClient    *google.Client
	googleInitErr   error
}

// New creates a unified client with the given configuration.
// Provider clients are lazily initialized when first needed based on the model used.
// Optional ClientOption arguments configure default behaviors like temperature.
func New(cfg Config, opts ...ClientOption) *Client {
	retryConfig := retry.DefaultConfig()
	if cfg.RetryConfig != nil {
		retryConfig = *cfg.RetryConfig
	}

	c := &Client{
		apiKeys:     cfg.APIKeys,
		defaults:    cfg.Defaults,
		retryConfig: retryConfig,
		events:      cfg.Events,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// getAnthropicClient returns the Anthropic client, initializing it if needed.
func (c *Client) getAnthropicClient() (*anthropic.Client, error) {
	c.mu.RLock()
	if c.anthropicClient != nil {
		defer c.mu.RUnlock()
		return c.anthropicClient, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.anthropicClient != nil {
		return c.anthropicClient, nil
	}

	if c.apiKeys.Anthropic == "" {
		return nil, &ErrMissingAPIKey{Provider: "anthropic"}
	}

	c.anthropicClient = anthropic.New(c.apiKeys.Anthropic)
	return c.anthropicClient, nil
}

// getOpenAIClient returns the OpenAI client, initializing it if needed.
func (c *Client) getOpenAIClient() (*openai.Client, error) {
	c.mu.RLock()
	if c.openaiClient != nil {
		defer c.mu.RUnlock()
		return c.openaiClient, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.openaiClient != nil {
		return c.openaiClient, nil
	}

	if c.apiKeys.OpenAI == "" {
		return nil, &ErrMissingAPIKey{Provider: "openai"}
	}

	c.openaiClient = openai.New(c.apiKeys.OpenAI)
	return c.openaiClient, nil
}

// getGoogleClient returns the Google client, initializing it if needed.
func (c *Client) getGoogleClient(ctx context.Context) (*google.Client, error) {
	c.mu.RLock()
	if c.googleClient != nil {
		defer c.mu.RUnlock()
		return c.googleClient, nil
	}
	if c.googleInitErr != nil {
		defer c.mu.RUnlock()
		return nil, c.googleInitErr
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.googleClient != nil {
		return c.googleClient, nil
	}
	if c.googleInitErr != nil {
		return nil, c.googleInitErr
	}

	if c.apiKeys.Google == "" {
		return nil, &ErrMissingAPIKey{Provider: "google"}
	}

	client, err := google.New(ctx, c.apiKeys.Google)
	if err != nil {
		c.googleInitErr = fmt.Errorf("failed to initialize Google client: %w", err)
		return nil, c.googleInitErr
	}

	c.googleClient = client
	return c.googleClient, nil
}

// resolveProvider determines which provider to use for a given model.
func (c *Client) resolveProvider(model ai.Model) ai.Provider {
	return model.Provider()
}

// getChatProvider returns the chat provider for the given model.
func (c *Client) getChatProvider(ctx context.Context, model ai.Model) (ai.ChatProvider, ai.Provider, error) {
	provider := c.resolveProvider(model)

	switch provider {
	case ai.ProviderAnthropic:
		client, err := c.getAnthropicClient()
		if err != nil {
			return nil, "", err
		}
		return client, provider, nil
	case ai.ProviderOpenAI:
		client, err := c.getOpenAIClient()
		if err != nil {
			return nil, "", err
		}
		return client, provider, nil
	case ai.ProviderGoogle:
		client, err := c.getGoogleClient(ctx)
		if err != nil {
			return nil, "", err
		}
		return client, provider, nil
	default:
		return nil, "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

// Chat sends a conversation and returns a complete response.
// The model can be specified via WithModel option, or the default chat model is used.
// Automatically retries on transient errors according to the client's retry configuration.
func (c *Client) Chat(ctx context.Context, messages []ai.Message, opts ...ai.Option) (*ai.Response, error) {
	// Prepend default options so per-request options override them
	opts = append(c.defaultChatOpts, opts...)
	options := ai.ApplyOptions(opts...)

	// Determine which model to use
	model := options.Model
	if model == nil {
		model = c.defaults.Chat
	}
	if model == nil {
		return nil, &ErrNoModel{Operation: "chat"}
	}

	// Get the appropriate provider
	chatProvider, provider, err := c.getChatProvider(ctx, model)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	emit(c.events, Event{
		Type:      EventRequestStart,
		Operation: "chat",
		Provider:  provider,
	})

	// Ensure model is passed to the underlying provider
	if options.Model == nil {
		opts = append([]ai.Option{ai.WithModel(model)}, opts...)
	}

	// Create retry events channel if client events are enabled
	var retryEvents chan retry.Event
	if c.events != nil {
		retryEvents = make(chan retry.Event, 10)
		go c.forwardRetryEvents(retryEvents, "chat", provider)
	}

	resp, err := retry.DoWithEvents(ctx, c.retryConfig, retryEvents, func() (*ai.Response, error) {
		return chatProvider.Chat(ctx, messages, opts...)
	})

	if retryEvents != nil {
		close(retryEvents)
	}

	if err != nil {
		emit(c.events, Event{
			Type:      EventRequestError,
			Operation: "chat",
			Provider:  provider,
			Duration:  time.Since(start),
			Error:     err,
		})
		return nil, err
	}

	var usage *ai.Usage
	if resp != nil {
		usage = &resp.Usage
	}
	emit(c.events, Event{
		Type:      EventRequestComplete,
		Operation: "chat",
		Provider:  provider,
		Duration:  time.Since(start),
		Usage:     usage,
	})
	return resp, nil
}

// ChatStream sends a conversation and returns a channel of streaming events.
// The model can be specified via WithModel option, or the default chat model is used.
// Automatically retries on transient errors when establishing the stream connection.
func (c *Client) ChatStream(ctx context.Context, messages []ai.Message, opts ...ai.Option) (<-chan ai.StreamEvent, error) {
	// Prepend default options so per-request options override them
	opts = append(c.defaultChatOpts, opts...)
	options := ai.ApplyOptions(opts...)

	// Determine which model to use
	model := options.Model
	if model == nil {
		model = c.defaults.Chat
	}
	if model == nil {
		return nil, &ErrNoModel{Operation: "chat_stream"}
	}

	// Get the appropriate provider
	chatProvider, provider, err := c.getChatProvider(ctx, model)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	emit(c.events, Event{
		Type:      EventRequestStart,
		Operation: "chat_stream",
		Provider:  provider,
	})

	// Ensure model is passed to the underlying provider
	if options.Model == nil {
		opts = append([]ai.Option{ai.WithModel(model)}, opts...)
	}

	// Create retry events channel if client events are enabled
	var retryEvents chan retry.Event
	if c.events != nil {
		retryEvents = make(chan retry.Event, 10)
		go c.forwardRetryEvents(retryEvents, "chat_stream", provider)
	}

	ch, err := retry.DoStreamWithEvents(ctx, c.retryConfig, retryEvents, func() (<-chan ai.StreamEvent, error) {
		return chatProvider.ChatStream(ctx, messages, opts...)
	})

	if retryEvents != nil {
		close(retryEvents)
	}

	if err != nil {
		emit(c.events, Event{
			Type:      EventRequestError,
			Operation: "chat_stream",
			Provider:  provider,
			Duration:  time.Since(start),
			Error:     err,
		})
		return nil, err
	}

	emit(c.events, Event{
		Type:      EventRequestComplete,
		Operation: "chat_stream",
		Provider:  provider,
		Duration:  time.Since(start),
	})
	return ch, nil
}

// GenerateImage creates images from a text prompt.
// The model can be specified via WithImageModel option, or the default image model is used.
// Returns ErrFeatureNotSupported if the provider doesn't support image generation.
// Automatically retries on transient errors according to the client's retry configuration.
func (c *Client) GenerateImage(ctx context.Context, prompt string, opts ...ai.ImageOption) (*ai.ImageResponse, error) {
	options := ai.ApplyImageOptions(opts...)

	// Determine which model to use
	model := options.Model
	if model == nil {
		model = c.defaults.Image
	}
	if model == nil {
		return nil, &ErrNoModel{Operation: "image"}
	}

	// Resolve provider and check capability
	provider := c.resolveProvider(model)

	if !providerCapabilities[provider][FeatureImage] {
		return nil, &ErrFeatureNotSupported{Provider: provider.String(), Feature: "image"}
	}

	// Get the image provider
	var imageProvider ai.ImageProvider
	switch provider {
	case ai.ProviderOpenAI:
		client, err := c.getOpenAIClient()
		if err != nil {
			return nil, err
		}
		imageProvider = client
	case ai.ProviderGoogle:
		client, err := c.getGoogleClient(ctx)
		if err != nil {
			return nil, err
		}
		imageProvider = client
	default:
		return nil, &ErrFeatureNotSupported{Provider: provider.String(), Feature: "image"}
	}

	start := time.Now()
	emit(c.events, Event{
		Type:      EventRequestStart,
		Operation: "image",
		Provider:  provider,
	})

	// Ensure model is passed to the underlying provider
	if options.Model == nil {
		opts = append([]ai.ImageOption{ai.WithImageModel(model)}, opts...)
	}

	// Create retry events channel if client events are enabled
	var retryEvents chan retry.Event
	if c.events != nil {
		retryEvents = make(chan retry.Event, 10)
		go c.forwardRetryEvents(retryEvents, "image", provider)
	}

	resp, err := retry.DoWithEvents(ctx, c.retryConfig, retryEvents, func() (*ai.ImageResponse, error) {
		return imageProvider.GenerateImage(ctx, prompt, opts...)
	})

	if retryEvents != nil {
		close(retryEvents)
	}

	if err != nil {
		emit(c.events, Event{
			Type:      EventRequestError,
			Operation: "image",
			Provider:  provider,
			Duration:  time.Since(start),
			Error:     err,
		})
		return nil, err
	}

	emit(c.events, Event{
		Type:      EventRequestComplete,
		Operation: "image",
		Provider:  provider,
		Duration:  time.Since(start),
	})
	return resp, nil
}

// Embed generates embeddings for the provided texts.
// The model can be specified via WithEmbeddingModel option, or the default embedding model is used.
// Returns ErrFeatureNotSupported if the provider doesn't support embeddings.
// Automatically retries on transient errors according to the client's retry configuration.
func (c *Client) Embed(ctx context.Context, texts []string, opts ...ai.EmbeddingOption) (*ai.EmbeddingResponse, error) {
	options := ai.ApplyEmbeddingOptions(opts...)

	// Determine which model to use
	model := options.Model
	if model == nil {
		model = c.defaults.Embedding
	}
	if model == nil {
		return nil, &ErrNoModel{Operation: "embedding"}
	}

	// Resolve provider and check capability
	provider := c.resolveProvider(model)

	if !providerCapabilities[provider][FeatureEmbedding] {
		return nil, &ErrFeatureNotSupported{Provider: provider.String(), Feature: "embedding"}
	}

	// Get the embedding provider
	var embedProvider ai.EmbeddingProvider
	switch provider {
	case ai.ProviderOpenAI:
		client, err := c.getOpenAIClient()
		if err != nil {
			return nil, err
		}
		embedProvider = client
	case ai.ProviderGoogle:
		client, err := c.getGoogleClient(ctx)
		if err != nil {
			return nil, err
		}
		embedProvider = client
	default:
		return nil, &ErrFeatureNotSupported{Provider: provider.String(), Feature: "embedding"}
	}

	start := time.Now()
	emit(c.events, Event{
		Type:      EventRequestStart,
		Operation: "embed",
		Provider:  provider,
	})

	// Ensure model is passed to the underlying provider
	if options.Model == nil {
		opts = append([]ai.EmbeddingOption{ai.WithEmbeddingModel(model)}, opts...)
	}

	// Create retry events channel if client events are enabled
	var retryEvents chan retry.Event
	if c.events != nil {
		retryEvents = make(chan retry.Event, 10)
		go c.forwardRetryEvents(retryEvents, "embed", provider)
	}

	resp, err := retry.DoWithEvents(ctx, c.retryConfig, retryEvents, func() (*ai.EmbeddingResponse, error) {
		return embedProvider.Embed(ctx, texts, opts...)
	})

	if retryEvents != nil {
		close(retryEvents)
	}

	if err != nil {
		emit(c.events, Event{
			Type:      EventRequestError,
			Operation: "embed",
			Provider:  provider,
			Duration:  time.Since(start),
			Error:     err,
		})
		return nil, err
	}

	emit(c.events, Event{
		Type:      EventRequestComplete,
		Operation: "embed",
		Provider:  provider,
		Duration:  time.Since(start),
	})
	return resp, nil
}

// SupportsFeature returns true if the given feature is supported by any configured provider.
func (c *Client) SupportsFeature(f Feature) bool {
	switch f {
	case FeatureChat:
		return c.apiKeys.Anthropic != "" || c.apiKeys.OpenAI != "" || c.apiKeys.Google != ""
	case FeatureImage:
		return c.apiKeys.OpenAI != "" || c.apiKeys.Google != ""
	case FeatureEmbedding:
		return c.apiKeys.OpenAI != "" || c.apiKeys.Google != ""
	default:
		return false
	}
}

// forwardRetryEvents reads from a retry events channel and forwards events
// to the client's event channel as EventRetry events.
func (c *Client) forwardRetryEvents(retryEvents <-chan retry.Event, operation string, provider ai.Provider) {
	for re := range retryEvents {
		reCopy := re // Copy to avoid pointer issues
		emit(c.events, Event{
			Type:       EventRetry,
			Operation:  operation,
			Provider:   provider,
			RetryEvent: &reCopy,
		})
	}
}
