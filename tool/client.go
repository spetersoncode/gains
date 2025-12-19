package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/client"
)

// ImageClient is the interface for image generation capabilities.
type ImageClient interface {
	GenerateImage(ctx context.Context, prompt string, opts ...ai.ImageOption) (*ai.ImageResponse, error)
}

// EmbeddingClient is the interface for embedding capabilities.
type EmbeddingClient interface {
	Embed(ctx context.Context, texts []string, opts ...ai.EmbeddingOption) (*ai.EmbeddingResponse, error)
}

// Chatter is the interface for non-streaming chat capabilities.
// Use workflow.ChatClient for the full interface with streaming support.
type Chatter interface {
	Chat(ctx context.Context, messages []ai.Message, opts ...ai.Option) (*ai.Response, error)
}

// imageArgs defines the arguments for the image generation tool.
type imageArgs struct {
	Prompt  string `json:"prompt" desc:"Description of the image to generate" required:"true"`
	Size    string `json:"size" desc:"Image dimensions" enum:"1024x1024,1024x1792,1792x1024"`
	Quality string `json:"quality" desc:"Quality level (DALL-E 3 only)" enum:"standard,hd"`
	Style   string `json:"style" desc:"Visual style (DALL-E 3 only)" enum:"vivid,natural"`
}

// ImageToolOption configures the image generation tool.
type ImageToolOption func(*imageToolConfig)

type imageToolConfig struct {
	name     string
	defaults []ai.ImageOption
}

// WithImageName sets a custom name for the image tool.
// Default is "generate_image".
func WithImageName(name string) ImageToolOption {
	return func(c *imageToolConfig) {
		c.name = name
	}
}

// WithImageDefaults sets default options for image generation.
func WithImageDefaults(opts ...ai.ImageOption) ImageToolOption {
	return func(c *imageToolConfig) {
		c.defaults = opts
	}
}

// NewImageTool creates a tool for generating images.
// The tool accepts a prompt and optional size, quality, and style parameters.
func NewImageTool(c ImageClient, opts ...ImageToolOption) (ai.Tool, Handler) {
	cfg := &imageToolConfig{
		name: "generate_image",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	schema := MustSchemaFor[imageArgs]()

	t := ai.Tool{
		Name:        cfg.name,
		Description: "Generate an image from a text description",
		Parameters:  schema,
	}

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args imageArgs
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", err
		}

		imageOpts := append([]ai.ImageOption{}, cfg.defaults...)

		if args.Size != "" {
			imageOpts = append(imageOpts, ai.WithImageSize(parseImageSize(args.Size)))
		}
		if args.Quality != "" {
			imageOpts = append(imageOpts, ai.WithImageQuality(parseImageQuality(args.Quality)))
		}
		if args.Style != "" {
			imageOpts = append(imageOpts, ai.WithImageStyle(parseImageStyle(args.Style)))
		}

		resp, err := c.GenerateImage(ctx, args.Prompt, imageOpts...)
		if err != nil {
			return "", err
		}

		result, err := json.Marshal(resp)
		if err != nil {
			return "", err
		}
		return string(result), nil
	}

	return t, handler
}

func parseImageSize(s string) ai.ImageSize {
	switch s {
	case "1024x1792":
		return ai.ImageSize1024x1792
	case "1792x1024":
		return ai.ImageSize1792x1024
	default:
		return ai.ImageSize1024x1024
	}
}

func parseImageQuality(q string) ai.ImageQuality {
	if q == "hd" {
		return ai.ImageQualityHD
	}
	return ai.ImageQualityStandard
}

func parseImageStyle(s string) ai.ImageStyle {
	if s == "natural" {
		return ai.ImageStyleNatural
	}
	return ai.ImageStyleVivid
}

// embeddingArgs defines the arguments for the embedding tool.
type embeddingArgs struct {
	Texts      []string `json:"texts" desc:"Texts to generate embeddings for" required:"true" minItems:"1"`
	Dimensions int      `json:"dimensions" desc:"Output vector dimensions (optional)"`
}

// EmbeddingToolOption configures the embedding tool.
type EmbeddingToolOption func(*embeddingToolConfig)

type embeddingToolConfig struct {
	name     string
	defaults []ai.EmbeddingOption
}

// WithEmbeddingName sets a custom name for the embedding tool.
// Default is "embed_text".
func WithEmbeddingName(name string) EmbeddingToolOption {
	return func(c *embeddingToolConfig) {
		c.name = name
	}
}

// WithEmbeddingDefaults sets default options for embedding generation.
func WithEmbeddingDefaults(opts ...ai.EmbeddingOption) EmbeddingToolOption {
	return func(c *embeddingToolConfig) {
		c.defaults = opts
	}
}

// NewEmbeddingTool creates a tool for generating text embeddings.
// The tool accepts an array of texts and returns their embedding vectors.
func NewEmbeddingTool(c EmbeddingClient, opts ...EmbeddingToolOption) (ai.Tool, Handler) {
	cfg := &embeddingToolConfig{
		name: "embed_text",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	schema := MustSchemaFor[embeddingArgs]()

	t := ai.Tool{
		Name:        cfg.name,
		Description: "Generate embedding vectors for text",
		Parameters:  schema,
	}

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args embeddingArgs
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", err
		}

		embOpts := append([]ai.EmbeddingOption{}, cfg.defaults...)

		if args.Dimensions > 0 {
			embOpts = append(embOpts, ai.WithEmbeddingDimensions(args.Dimensions))
		}

		resp, err := c.Embed(ctx, args.Texts, embOpts...)
		if err != nil {
			return "", err
		}

		// Return a summary rather than full vectors (which can be very large)
		result := struct {
			Count      int   `json:"count"`
			Dimensions int   `json:"dimensions"`
			Usage      any   `json:"usage,omitempty"`
			Embeddings []any `json:"embeddings,omitempty"`
		}{
			Count: len(resp.Embeddings),
		}
		if len(resp.Embeddings) > 0 {
			result.Dimensions = len(resp.Embeddings[0])
		}
		if resp.Usage.InputTokens > 0 {
			result.Usage = resp.Usage
		}

		out, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(out), nil
	}

	return t, handler
}

// chatArgs defines the arguments for the chat tool.
type chatArgs struct {
	Prompt  string `json:"prompt" desc:"Question or task for the assistant" required:"true"`
	Context string `json:"context" desc:"Additional context to include"`
}

// ChatToolOption configures the chat tool.
type ChatToolOption func(*chatToolConfig)

type chatToolConfig struct {
	name         string
	systemPrompt string
	defaults     []ai.Option
}

// WithChatName sets a custom name for the chat tool.
// Default is "ask_assistant".
func WithChatName(name string) ChatToolOption {
	return func(c *chatToolConfig) {
		c.name = name
	}
}

// WithSystemPrompt sets a system prompt for the chat tool.
func WithSystemPrompt(prompt string) ChatToolOption {
	return func(c *chatToolConfig) {
		c.systemPrompt = prompt
	}
}

// WithChatDefaults sets default options for chat requests.
func WithChatDefaults(opts ...ai.Option) ChatToolOption {
	return func(c *chatToolConfig) {
		c.defaults = opts
	}
}

// NewChatTool creates a tool that makes LLM calls (sub-agent pattern).
// This allows an agent to delegate tasks to another LLM call.
func NewChatTool(c Chatter, opts ...ChatToolOption) (ai.Tool, Handler) {
	cfg := &chatToolConfig{
		name: "ask_assistant",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	schema := MustSchemaFor[chatArgs]()

	t := ai.Tool{
		Name:        cfg.name,
		Description: "Ask an AI assistant a question or delegate a task",
		Parameters:  schema,
	}

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args chatArgs
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", err
		}

		var messages []ai.Message

		if cfg.systemPrompt != "" {
			messages = append(messages, ai.Message{
				Role:    ai.RoleSystem,
				Content: cfg.systemPrompt,
			})
		}

		content := args.Prompt
		if args.Context != "" {
			content = fmt.Sprintf("%s\n\nContext:\n%s", args.Prompt, args.Context)
		}

		messages = append(messages, ai.Message{
			Role:    ai.RoleUser,
			Content: content,
		})

		resp, err := c.Chat(ctx, messages, cfg.defaults...)
		if err != nil {
			return "", err
		}

		return resp.Content, nil
	}

	return t, handler
}

// ClientTools returns tools for image, embedding, and chat capabilities.
// Only tools for supported features are returned.
func ClientTools(c *client.Client, opts ...ClientToolsOption) []ToolPair {
	cfg := &clientToolsConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var pairs []ToolPair

	// Add image tool if supported
	if c.SupportsFeature(client.FeatureImage) {
		t, h := NewImageTool(c, cfg.imageOpts...)
		pairs = append(pairs, ToolPair{Tool: t, Handler: h})
	}

	// Add embedding tool if supported
	if c.SupportsFeature(client.FeatureEmbedding) {
		t, h := NewEmbeddingTool(c, cfg.embeddingOpts...)
		pairs = append(pairs, ToolPair{Tool: t, Handler: h})
	}

	// Always add chat tool
	t, h := NewChatTool(c, cfg.chatOpts...)
	pairs = append(pairs, ToolPair{Tool: t, Handler: h})

	return pairs
}

// ClientToolsOption configures the ClientTools function.
type ClientToolsOption func(*clientToolsConfig)

type clientToolsConfig struct {
	imageOpts     []ImageToolOption
	embeddingOpts []EmbeddingToolOption
	chatOpts      []ChatToolOption
}

// WithImageToolOptions sets options for the image tool in ClientTools.
func WithImageToolOptions(opts ...ImageToolOption) ClientToolsOption {
	return func(c *clientToolsConfig) {
		c.imageOpts = opts
	}
}

// WithEmbeddingToolOptions sets options for the embedding tool in ClientTools.
func WithEmbeddingToolOptions(opts ...EmbeddingToolOption) ClientToolsOption {
	return func(c *clientToolsConfig) {
		c.embeddingOpts = opts
	}
}

// WithChatToolOptions sets options for the chat tool in ClientTools.
func WithChatToolOptions(opts ...ChatToolOption) ClientToolsOption {
	return func(c *clientToolsConfig) {
		c.chatOpts = opts
	}
}

// ToolPair holds a tool definition and its handler for easy registration.
type ToolPair struct {
	Tool    ai.Tool
	Handler Handler
}

// splitSize parses a size string like "1024x1024" into width and height.
func splitSize(s string) (int, int) {
	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return 1024, 1024
	}
	var w, h int
	fmt.Sscanf(parts[0], "%d", &w)
	fmt.Sscanf(parts[1], "%d", &h)
	if w == 0 {
		w = 1024
	}
	if h == 0 {
		h = 1024
	}
	return w, h
}
