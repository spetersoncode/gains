package openai

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spetersoncode/gains"
)

const DefaultModel = "gpt-4o"

// Client wraps the OpenAI SDK to implement gains.ChatProvider.
type Client struct {
	client *openai.Client
	model  string
}

// New creates a new OpenAI client.
// It reads the API key from the OPENAI_API_KEY environment variable.
func New(opts ...ClientOption) *Client {
	c := &Client{
		model: DefaultModel,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.client == nil {
		client := openai.NewClient()
		c.client = &client
	}
	return c
}

// ClientOption configures the OpenAI client.
type ClientOption func(*Client)

// WithAPIKey sets the API key explicitly instead of using the environment variable.
func WithAPIKey(key string) ClientOption {
	return func(c *Client) {
		client := openai.NewClient(option.WithAPIKey(key))
		c.client = &client
	}
}

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

	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: convertMessages(messages),
	}
	if options.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(options.MaxTokens))
	}
	if options.Temperature != nil {
		params.Temperature = openai.Float(*options.Temperature)
	}

	resp, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	return &gains.Response{
		Content:      resp.Choices[0].Message.Content,
		FinishReason: string(resp.Choices[0].FinishReason),
		Usage: gains.Usage{
			InputTokens:  int(resp.Usage.PromptTokens),
			OutputTokens: int(resp.Usage.CompletionTokens),
		},
	}, nil
}

// ChatStream sends a conversation and returns a channel of streaming events.
func (c *Client) ChatStream(ctx context.Context, messages []gains.Message, opts ...gains.Option) (<-chan gains.StreamEvent, error) {
	options := gains.ApplyOptions(opts...)
	model := c.model
	if options.Model != "" {
		model = options.Model
	}

	params := openai.ChatCompletionNewParams{
		Model:    model,
		Messages: convertMessages(messages),
		StreamOptions: openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		},
	}
	if options.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(options.MaxTokens))
	}
	if options.Temperature != nil {
		params.Temperature = openai.Float(*options.Temperature)
	}

	stream := c.client.Chat.Completions.NewStreaming(ctx, params)
	ch := make(chan gains.StreamEvent)

	go func() {
		defer close(ch)
		var acc openai.ChatCompletionAccumulator

		for stream.Next() {
			chunk := stream.Current()
			acc.AddChunk(chunk)

			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				ch <- gains.StreamEvent{
					Delta: chunk.Choices[0].Delta.Content,
				}
			}
		}

		if err := stream.Err(); err != nil {
			ch <- gains.StreamEvent{Err: err}
			return
		}

		// Send final event with complete response
		completion := acc.Choices[0]
		ch <- gains.StreamEvent{
			Done: true,
			Response: &gains.Response{
				Content:      completion.Message.Content,
				FinishReason: string(completion.FinishReason),
				Usage: gains.Usage{
					InputTokens:  int(acc.Usage.PromptTokens),
					OutputTokens: int(acc.Usage.CompletionTokens),
				},
			},
		}
	}()

	return ch, nil
}

func convertMessages(messages []gains.Message) []openai.ChatCompletionMessageParamUnion {
	result := make([]openai.ChatCompletionMessageParamUnion, len(messages))
	for i, msg := range messages {
		switch msg.Role {
		case gains.RoleUser:
			result[i] = openai.UserMessage(msg.Content)
		case gains.RoleAssistant:
			result[i] = openai.AssistantMessage(msg.Content)
		case gains.RoleSystem:
			result[i] = openai.SystemMessage(msg.Content)
		default:
			result[i] = openai.UserMessage(msg.Content)
		}
	}
	return result
}

var _ gains.ChatProvider = (*Client)(nil)
