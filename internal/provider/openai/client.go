package openai

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	ai "github.com/spetersoncode/gains"
)

// Client wraps the OpenAI SDK to implement ai.ChatProvider.
type Client struct {
	client *openai.Client
	model  ChatModel
}

// New creates a new OpenAI client with the given API key.
func New(apiKey string, opts ...ClientOption) *Client {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	c := &Client{
		client: &client,
		model:  DefaultChatModel,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ClientOption configures the OpenAI client.
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

	convertedMessages, err := convertMessages(messages)
	if err != nil {
		return nil, err
	}

	params := openai.ChatCompletionNewParams{
		Model:    model.String(),
		Messages: convertedMessages,
	}
	if options.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(options.MaxTokens))
	}
	if options.Temperature != nil {
		params.Temperature = openai.Float(*options.Temperature)
	}
	if len(options.Tools) > 0 {
		params.Tools = convertTools(options.Tools)
		if options.ToolChoice != "" {
			params.ToolChoice = convertToolChoice(options.ToolChoice)
		}
	}

	// Handle JSON mode / response schema
	if options.ResponseSchema != nil {
		params.ResponseFormat = buildOpenAISchemaFormat(options.ResponseSchema)
	} else if options.ResponseFormat == ai.ResponseFormatJSON {
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{
				Type: "json_object",
			},
		}
	}

	resp, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, wrapError(err)
	}

	return &ai.Response{
		Content:      resp.Choices[0].Message.Content,
		FinishReason: string(resp.Choices[0].FinishReason),
		Usage: ai.Usage{
			InputTokens:  int(resp.Usage.PromptTokens),
			OutputTokens: int(resp.Usage.CompletionTokens),
		},
		ToolCalls: extractToolCalls(resp.Choices[0].Message),
	}, nil
}

// ChatStream sends a conversation and returns a channel of streaming events.
func (c *Client) ChatStream(ctx context.Context, messages []ai.Message, opts ...ai.Option) (<-chan ai.StreamEvent, error) {
	options := ai.ApplyOptions(opts...)
	model := c.model
	if options.Model != nil {
		model = ChatModel(options.Model.String())
	}

	convertedMessages, err := convertMessages(messages)
	if err != nil {
		return nil, err
	}

	params := openai.ChatCompletionNewParams{
		Model:    model.String(),
		Messages: convertedMessages,
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
	if len(options.Tools) > 0 {
		params.Tools = convertTools(options.Tools)
		if options.ToolChoice != "" {
			params.ToolChoice = convertToolChoice(options.ToolChoice)
		}
	}

	// Handle JSON mode / response schema
	if options.ResponseSchema != nil {
		params.ResponseFormat = buildOpenAISchemaFormat(options.ResponseSchema)
	} else if options.ResponseFormat == ai.ResponseFormatJSON {
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{
				Type: "json_object",
			},
		}
	}

	stream := c.client.Chat.Completions.NewStreaming(ctx, params)
	ch := make(chan ai.StreamEvent)

	go func() {
		defer close(ch)
		var acc openai.ChatCompletionAccumulator

		for stream.Next() {
			chunk := stream.Current()
			acc.AddChunk(chunk)

			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				ch <- ai.StreamEvent{
					Delta: chunk.Choices[0].Delta.Content,
				}
			}
		}

		if err := stream.Err(); err != nil {
			ch <- ai.StreamEvent{Err: wrapError(err)}
			return
		}

		// Send final event with complete response
		completion := acc.Choices[0]
		ch <- ai.StreamEvent{
			Done: true,
			Response: &ai.Response{
				Content:      completion.Message.Content,
				FinishReason: string(completion.FinishReason),
				Usage: ai.Usage{
					InputTokens:  int(acc.Usage.PromptTokens),
					OutputTokens: int(acc.Usage.CompletionTokens),
				},
				ToolCalls: extractToolCallsFromAccumulator(completion.Message.ToolCalls),
			},
		}
	}()

	return ch, nil
}

var _ ai.ChatProvider = (*Client)(nil)
var _ ai.ImageProvider = (*Client)(nil)
var _ ai.EmbeddingProvider = (*Client)(nil)
