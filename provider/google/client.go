package google

import (
	"context"

	"github.com/spetersoncode/gains"
	"google.golang.org/genai"
)

const DefaultModel = "gemini-2.0-flash"
const DefaultImageModel = "imagen-3.0-generate-002"
const DefaultEmbeddingModel = "text-embedding-004"

// Client wraps the Google GenAI SDK to implement gains.ChatProvider.
type Client struct {
	client *genai.Client
	model  string
}

// New creates a new Google GenAI client with the given API key.
func New(ctx context.Context, apiKey string, opts ...ClientOption) (*Client, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}
	c := &Client{
		client: client,
		model:  DefaultModel,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// ClientOption configures the Google client.
type ClientOption func(*Client)

// WithModel sets the default model for requests.
func WithModel(model string) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

// Chat sends a conversation and returns a complete response.
func (c *Client) Chat(ctx context.Context, messages []gains.Message, opts ...gains.Option) (*gains.Response, error) {
	options := gains.ApplyOptions(opts...)
	model := c.model
	if options.Model != "" {
		model = options.Model
	}

	contents := convertMessages(messages)
	config := &genai.GenerateContentConfig{}
	if options.MaxTokens > 0 {
		maxTokens := int32(options.MaxTokens)
		config.MaxOutputTokens = maxTokens
	}
	if options.Temperature != nil {
		temp := float32(*options.Temperature)
		config.Temperature = &temp
	}
	if len(options.Tools) > 0 {
		config.Tools = convertTools(options.Tools)
		if options.ToolChoice != "" {
			config.ToolConfig = convertToolChoice(options.ToolChoice)
		}
	}

	// Handle JSON mode / response schema
	if options.ResponseSchema != nil {
		config.ResponseMIMEType = "application/json"
		config.ResponseSchema = convertJSONSchemaToGenaiSchema(options.ResponseSchema.Schema)
	} else if options.ResponseFormat == gains.ResponseFormatJSON {
		config.ResponseMIMEType = "application/json"
	}

	resp, err := c.client.Models.GenerateContent(ctx, model, contents, config)
	if err != nil {
		return nil, err
	}

	content := ""
	var toolCalls []gains.ToolCall
	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.Text != "" {
				content += part.Text
			}
		}
		toolCalls = extractToolCalls(resp.Candidates[0].Content.Parts)
	}

	finishReason := ""
	if len(resp.Candidates) > 0 {
		finishReason = string(resp.Candidates[0].FinishReason)
	}

	usage := gains.Usage{}
	if resp.UsageMetadata != nil {
		usage.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
		usage.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
	}

	return &gains.Response{
		Content:      content,
		FinishReason: finishReason,
		Usage:        usage,
		ToolCalls:    toolCalls,
	}, nil
}

// ChatStream sends a conversation and returns a channel of streaming events.
func (c *Client) ChatStream(ctx context.Context, messages []gains.Message, opts ...gains.Option) (<-chan gains.StreamEvent, error) {
	options := gains.ApplyOptions(opts...)
	model := c.model
	if options.Model != "" {
		model = options.Model
	}

	contents := convertMessages(messages)
	config := &genai.GenerateContentConfig{}
	if options.MaxTokens > 0 {
		maxTokens := int32(options.MaxTokens)
		config.MaxOutputTokens = maxTokens
	}
	if options.Temperature != nil {
		temp := float32(*options.Temperature)
		config.Temperature = &temp
	}
	if len(options.Tools) > 0 {
		config.Tools = convertTools(options.Tools)
		if options.ToolChoice != "" {
			config.ToolConfig = convertToolChoice(options.ToolChoice)
		}
	}

	// Handle JSON mode / response schema
	if options.ResponseSchema != nil {
		config.ResponseMIMEType = "application/json"
		config.ResponseSchema = convertJSONSchemaToGenaiSchema(options.ResponseSchema.Schema)
	} else if options.ResponseFormat == gains.ResponseFormatJSON {
		config.ResponseMIMEType = "application/json"
	}

	ch := make(chan gains.StreamEvent)

	go func() {
		defer close(ch)

		var fullContent string
		var finishReason string
		var usage gains.Usage
		var allParts []*genai.Part

		for resp := range c.client.Models.GenerateContentStream(ctx, model, contents, config) {
			if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				for _, part := range resp.Candidates[0].Content.Parts {
					allParts = append(allParts, part)
					if part.Text != "" {
						ch <- gains.StreamEvent{Delta: part.Text}
						fullContent += part.Text
					}
				}
				finishReason = string(resp.Candidates[0].FinishReason)
			}

			if resp.UsageMetadata != nil {
				usage.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
				usage.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
			}
		}

		ch <- gains.StreamEvent{
			Done: true,
			Response: &gains.Response{
				Content:      fullContent,
				FinishReason: finishReason,
				Usage:        usage,
				ToolCalls:    extractToolCalls(allParts),
			},
		}
	}()

	return ch, nil
}

var _ gains.ChatProvider = (*Client)(nil)
var _ gains.ImageProvider = (*Client)(nil)
var _ gains.EmbeddingProvider = (*Client)(nil)
