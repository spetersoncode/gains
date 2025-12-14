package anthropic

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/spetersoncode/gains"
)

const DefaultModel = "claude-sonnet-4-20250514"

// Client wraps the Anthropic SDK to implement gains.ChatProvider.
type Client struct {
	client *anthropic.Client
	model  string
}

// New creates a new Anthropic client.
// It reads the API key from the ANTHROPIC_API_KEY environment variable.
func New(opts ...ClientOption) *Client {
	c := &Client{
		model: DefaultModel,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.client == nil {
		client := anthropic.NewClient()
		c.client = &client
	}
	return c
}

// ClientOption configures the Anthropic client.
type ClientOption func(*Client)

// WithAPIKey sets the API key explicitly instead of using the environment variable.
func WithAPIKey(key string) ClientOption {
	return func(c *Client) {
		client := anthropic.NewClient(option.WithAPIKey(key))
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

	maxTokens := int64(4096)
	if options.MaxTokens > 0 {
		maxTokens = int64(options.MaxTokens)
	}

	msgs, system := convertMessages(messages)
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		Messages:  msgs,
	}
	if len(system) > 0 {
		params.System = system
	}
	if options.Temperature != nil {
		params.Temperature = anthropic.Float(*options.Temperature)
	}

	resp, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, err
	}

	content := ""
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &gains.Response{
		Content:      content,
		FinishReason: string(resp.StopReason),
		Usage: gains.Usage{
			InputTokens:  int(resp.Usage.InputTokens),
			OutputTokens: int(resp.Usage.OutputTokens),
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

	maxTokens := int64(4096)
	if options.MaxTokens > 0 {
		maxTokens = int64(options.MaxTokens)
	}

	msgs, system := convertMessages(messages)
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		Messages:  msgs,
	}
	if len(system) > 0 {
		params.System = system
	}
	if options.Temperature != nil {
		params.Temperature = anthropic.Float(*options.Temperature)
	}

	stream := c.client.Messages.NewStreaming(ctx, params)
	ch := make(chan gains.StreamEvent)

	go func() {
		defer close(ch)
		var acc anthropic.Message

		for stream.Next() {
			event := stream.Current()
			acc.Accumulate(event)

			if event.Type == "content_block_delta" {
				delta := event.AsContentBlockDelta()
				if textDelta := delta.Delta.AsTextDelta(); textDelta.Type == "text_delta" {
					ch <- gains.StreamEvent{
						Delta: textDelta.Text,
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			ch <- gains.StreamEvent{Err: err}
			return
		}

		// Send final event with complete response
		content := ""
		for _, block := range acc.Content {
			if block.Type == "text" {
				content += block.Text
			}
		}

		ch <- gains.StreamEvent{
			Done: true,
			Response: &gains.Response{
				Content:      content,
				FinishReason: string(acc.StopReason),
				Usage: gains.Usage{
					InputTokens:  int(acc.Usage.InputTokens),
					OutputTokens: int(acc.Usage.OutputTokens),
				},
			},
		}
	}()

	return ch, nil
}

func convertMessages(messages []gains.Message) ([]anthropic.MessageParam, []anthropic.TextBlockParam) {
	var result []anthropic.MessageParam
	var system []anthropic.TextBlockParam

	for _, msg := range messages {
		switch msg.Role {
		case gains.RoleSystem:
			system = append(system, anthropic.TextBlockParam{Text: msg.Content})
		case gains.RoleUser:
			result = append(result, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		case gains.RoleAssistant:
			result = append(result, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
		default:
			result = append(result, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		}
	}

	return result, system
}

var _ gains.ChatProvider = (*Client)(nil)
