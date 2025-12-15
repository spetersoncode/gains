package anthropic

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/spetersoncode/gains"
)

// Client wraps the Anthropic SDK to implement gains.ChatProvider.
type Client struct {
	client *anthropic.Client
	model  ChatModel
}

// New creates a new Anthropic client with the given API key.
func New(apiKey string, opts ...ClientOption) *Client {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	c := &Client{
		client: &client,
		model:  DefaultChatModel,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ClientOption configures the Anthropic client.
type ClientOption func(*Client)

// WithModel sets the default model for requests.
func WithModel(model ChatModel) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

// Chat sends a conversation and returns a complete response.
func (c *Client) Chat(ctx context.Context, messages []gains.Message, opts ...gains.Option) (*gains.Response, error) {
	options := gains.ApplyOptions(opts...)
	model := c.model
	if options.Model != nil {
		model = ChatModel(options.Model.String())
	}

	maxTokens := int64(4096)
	if options.MaxTokens > 0 {
		maxTokens = int64(options.MaxTokens)
	}

	msgs, system := convertMessages(messages)
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model.String()),
		MaxTokens: maxTokens,
		Messages:  msgs,
	}
	if len(system) > 0 {
		params.System = system
	}
	if options.Temperature != nil {
		params.Temperature = anthropic.Float(*options.Temperature)
	}

	// Check if JSON mode is requested
	useJSONTool := options.ResponseFormat == gains.ResponseFormatJSON || options.ResponseSchema != nil

	if useJSONTool {
		jsonTool, jsonToolChoice := buildAnthropicJSONTool(options)
		// Merge with user tools if present
		if len(options.Tools) > 0 {
			params.Tools = append(convertTools(options.Tools), jsonTool)
		} else {
			params.Tools = []anthropic.ToolUnionParam{jsonTool}
		}
		params.ToolChoice = jsonToolChoice
	} else if len(options.Tools) > 0 {
		params.Tools = convertTools(options.Tools)
		if options.ToolChoice != "" && options.ToolChoice != gains.ToolChoiceNone {
			params.ToolChoice = convertToolChoice(options.ToolChoice)
		}
	}

	resp, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, err
	}

	content := ""
	var toolCalls []gains.ToolCall
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		} else if block.Type == "tool_use" {
			if useJSONTool && block.Name == jsonResponseToolName {
				// Extract tool input as the JSON response
				content = string(block.Input)
			} else {
				// Regular tool call
				toolCalls = append(toolCalls, gains.ToolCall{
					ID:        block.ID,
					Name:      block.Name,
					Arguments: string(block.Input),
				})
			}
		}
	}

	return &gains.Response{
		Content:      content,
		FinishReason: string(resp.StopReason),
		Usage: gains.Usage{
			InputTokens:  int(resp.Usage.InputTokens),
			OutputTokens: int(resp.Usage.OutputTokens),
		},
		ToolCalls: toolCalls,
	}, nil
}

// ChatStream sends a conversation and returns a channel of streaming events.
func (c *Client) ChatStream(ctx context.Context, messages []gains.Message, opts ...gains.Option) (<-chan gains.StreamEvent, error) {
	options := gains.ApplyOptions(opts...)
	model := c.model
	if options.Model != nil {
		model = ChatModel(options.Model.String())
	}

	maxTokens := int64(4096)
	if options.MaxTokens > 0 {
		maxTokens = int64(options.MaxTokens)
	}

	msgs, system := convertMessages(messages)
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model.String()),
		MaxTokens: maxTokens,
		Messages:  msgs,
	}
	if len(system) > 0 {
		params.System = system
	}
	if options.Temperature != nil {
		params.Temperature = anthropic.Float(*options.Temperature)
	}

	// Check if JSON mode is requested
	useJSONTool := options.ResponseFormat == gains.ResponseFormatJSON || options.ResponseSchema != nil

	if useJSONTool {
		jsonTool, jsonToolChoice := buildAnthropicJSONTool(options)
		// Merge with user tools if present
		if len(options.Tools) > 0 {
			params.Tools = append(convertTools(options.Tools), jsonTool)
		} else {
			params.Tools = []anthropic.ToolUnionParam{jsonTool}
		}
		params.ToolChoice = jsonToolChoice
	} else if len(options.Tools) > 0 {
		params.Tools = convertTools(options.Tools)
		if options.ToolChoice != "" && options.ToolChoice != gains.ToolChoiceNone {
			params.ToolChoice = convertToolChoice(options.ToolChoice)
		}
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
		var toolCalls []gains.ToolCall
		for _, block := range acc.Content {
			if block.Type == "text" {
				content += block.Text
			} else if block.Type == "tool_use" {
				if useJSONTool && block.Name == jsonResponseToolName {
					// Extract tool input as the JSON response
					content = string(block.Input)
				} else {
					// Regular tool call
					toolCalls = append(toolCalls, gains.ToolCall{
						ID:        block.ID,
						Name:      block.Name,
						Arguments: string(block.Input),
					})
				}
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
				ToolCalls: toolCalls,
			},
		}
	}()

	return ch, nil
}

var _ gains.ChatProvider = (*Client)(nil)
