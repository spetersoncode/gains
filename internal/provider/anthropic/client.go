package anthropic

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	ai "github.com/spetersoncode/gains"
)

// Client wraps the Anthropic SDK to implement ai.ChatProvider.
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
func (c *Client) Chat(ctx context.Context, messages []ai.Message, opts ...ai.Option) (*ai.Response, error) {
	options := ai.ApplyOptions(opts...)
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
	useJSONTool := options.ResponseFormat == ai.ResponseFormatJSON || options.ResponseSchema != nil

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
		if options.ToolChoice != "" && options.ToolChoice != ai.ToolChoiceNone {
			params.ToolChoice = convertToolChoice(options.ToolChoice)
		}
	}

	resp, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, wrapError(err)
	}

	content := ""
	var toolCalls []ai.ToolCall
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		} else if block.Type == "tool_use" {
			if useJSONTool && block.Name == jsonResponseToolName {
				// Extract tool input as the JSON response
				content = string(block.Input)
			} else {
				// Regular tool call
				toolCalls = append(toolCalls, ai.ToolCall{
					ID:        block.ID,
					Name:      block.Name,
					Arguments: string(block.Input),
				})
			}
		}
	}

	return &ai.Response{
		Content:      content,
		FinishReason: string(resp.StopReason),
		Usage: ai.Usage{
			InputTokens:  int(resp.Usage.InputTokens),
			OutputTokens: int(resp.Usage.OutputTokens),
		},
		ToolCalls: toolCalls,
	}, nil
}

// ChatStream sends a conversation and returns a channel of streaming events.
func (c *Client) ChatStream(ctx context.Context, messages []ai.Message, opts ...ai.Option) (<-chan ai.StreamEvent, error) {
	options := ai.ApplyOptions(opts...)
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
	useJSONTool := options.ResponseFormat == ai.ResponseFormatJSON || options.ResponseSchema != nil

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
		if options.ToolChoice != "" && options.ToolChoice != ai.ToolChoiceNone {
			params.ToolChoice = convertToolChoice(options.ToolChoice)
		}
	}

	stream := c.client.Messages.NewStreaming(ctx, params)
	ch := make(chan ai.StreamEvent)

	go func() {
		defer close(ch)
		var acc anthropic.Message

		for stream.Next() {
			event := stream.Current()
			acc.Accumulate(event)

			if event.Type == "content_block_delta" {
				delta := event.AsContentBlockDelta()
				if textDelta := delta.Delta.AsTextDelta(); textDelta.Type == "text_delta" {
					ch <- ai.StreamEvent{
						Delta: textDelta.Text,
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			ch <- ai.StreamEvent{Err: wrapError(err)}
			return
		}

		// Send final event with complete response
		content := ""
		var toolCalls []ai.ToolCall
		for _, block := range acc.Content {
			if block.Type == "text" {
				content += block.Text
			} else if block.Type == "tool_use" {
				if useJSONTool && block.Name == jsonResponseToolName {
					// Extract tool input as the JSON response
					content = string(block.Input)
				} else {
					// Regular tool call
					toolCalls = append(toolCalls, ai.ToolCall{
						ID:        block.ID,
						Name:      block.Name,
						Arguments: string(block.Input),
					})
				}
			}
		}

		ch <- ai.StreamEvent{
			Done: true,
			Response: &ai.Response{
				Content:      content,
				FinishReason: string(acc.StopReason),
				Usage: ai.Usage{
					InputTokens:  int(acc.Usage.InputTokens),
					OutputTokens: int(acc.Usage.OutputTokens),
				},
				ToolCalls: toolCalls,
			},
		}
	}()

	return ch, nil
}

var _ ai.ChatProvider = (*Client)(nil)
