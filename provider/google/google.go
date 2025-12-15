package google

import (
	"context"

	"google.golang.org/genai"
	"github.com/spetersoncode/gains"
)

const DefaultModel = "gemini-2.0-flash"

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

	resp, err := c.client.Models.GenerateContent(ctx, model, contents, config)
	if err != nil {
		return nil, err
	}

	content := ""
	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.Text != "" {
				content += part.Text
			}
		}
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

	ch := make(chan gains.StreamEvent)

	go func() {
		defer close(ch)

		var fullContent string
		var finishReason string
		var usage gains.Usage

		for resp := range c.client.Models.GenerateContentStream(ctx, model, contents, config) {
			if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				for _, part := range resp.Candidates[0].Content.Parts {
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
			},
		}
	}()

	return ch, nil
}

func convertMessages(messages []gains.Message) []*genai.Content {
	var contents []*genai.Content

	for _, msg := range messages {
		role := "user"
		switch msg.Role {
		case gains.RoleUser:
			role = "user"
		case gains.RoleAssistant:
			role = "model"
		case gains.RoleSystem:
			// Gemini handles system prompts differently - prepend to first user message
			// For simplicity, treat as user message with context
			role = "user"
		}

		contents = append(contents, &genai.Content{
			Role: role,
			Parts: []*genai.Part{
				{Text: msg.Content},
			},
		})
	}

	return contents
}

var _ gains.ChatProvider = (*Client)(nil)
