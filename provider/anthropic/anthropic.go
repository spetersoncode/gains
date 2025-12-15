package anthropic

import (
	"context"
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/spetersoncode/gains"
)

const DefaultModel = "claude-sonnet-4-20250514"

// jsonResponseToolName is the name of the synthetic tool used for JSON mode.
const jsonResponseToolName = "__gains_json_response__"

// Client wraps the Anthropic SDK to implement gains.ChatProvider.
type Client struct {
	client *anthropic.Client
	model  string
}

// New creates a new Anthropic client with the given API key.
func New(apiKey string, opts ...ClientOption) *Client {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	c := &Client{
		client: &client,
		model:  DefaultModel,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ClientOption configures the Anthropic client.
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
			if len(msg.ToolCalls) > 0 {
				// Assistant message with tool calls
				var blocks []anthropic.ContentBlockParamUnion
				if msg.Content != "" {
					blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
				}
				for _, tc := range msg.ToolCalls {
					var input any
					json.Unmarshal([]byte(tc.Arguments), &input)
					blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Name))
				}
				result = append(result, anthropic.MessageParam{
					Role:    anthropic.MessageParamRoleAssistant,
					Content: blocks,
				})
			} else {
				result = append(result, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
			}
		case gains.RoleTool:
			// Tool results are sent as user messages with tool_result blocks
			var blocks []anthropic.ContentBlockParamUnion
			for _, tr := range msg.ToolResults {
				blocks = append(blocks, anthropic.NewToolResultBlock(tr.ToolCallID, tr.Content, tr.IsError))
			}
			result = append(result, anthropic.MessageParam{
				Role:    anthropic.MessageParamRoleUser,
				Content: blocks,
			})
		default:
			result = append(result, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		}
	}

	return result, system
}

func convertTools(tools []gains.Tool) []anthropic.ToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	result := make([]anthropic.ToolUnionParam, len(tools))
	for i, t := range tools {
		// Parse the JSON Schema to get the input schema
		var schema map[string]interface{}
		if len(t.Parameters) > 0 {
			json.Unmarshal(t.Parameters, &schema)
		}

		// Extract required as []string
		var required []string
		if reqVal, ok := schema["required"].([]interface{}); ok {
			for _, r := range reqVal {
				if s, ok := r.(string); ok {
					required = append(required, s)
				}
			}
		}

		inputSchema := anthropic.ToolInputSchemaParam{
			Properties: schema["properties"],
			Required:   required,
		}

		toolParam := anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: inputSchema,
		}

		result[i] = anthropic.ToolUnionParam{
			OfTool: &toolParam,
		}
	}
	return result
}

func convertToolChoice(choice gains.ToolChoice) anthropic.ToolChoiceUnionParam {
	switch choice {
	case gains.ToolChoiceNone:
		return anthropic.ToolChoiceUnionParam{
			OfNone: &anthropic.ToolChoiceNoneParam{},
		}
	case gains.ToolChoiceRequired:
		return anthropic.ToolChoiceUnionParam{
			OfAny: &anthropic.ToolChoiceAnyParam{},
		}
	default:
		return anthropic.ToolChoiceUnionParam{
			OfAuto: &anthropic.ToolChoiceAutoParam{},
		}
	}
}

func extractToolCalls(content []anthropic.ContentBlockUnion) []gains.ToolCall {
	var calls []gains.ToolCall
	for _, block := range content {
		if block.Type == "tool_use" {
			calls = append(calls, gains.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: string(block.Input),
			})
		}
	}
	return calls
}

func buildAnthropicJSONTool(options *gains.Options) (anthropic.ToolUnionParam, anthropic.ToolChoiceUnionParam) {
	var schema map[string]any
	if options.ResponseSchema != nil && len(options.ResponseSchema.Schema) > 0 {
		json.Unmarshal(options.ResponseSchema.Schema, &schema)
	} else {
		// Generic object schema for basic JSON mode
		schema = map[string]any{
			"type":                 "object",
			"additionalProperties": true,
		}
	}

	description := "Output the response as structured JSON"
	if options.ResponseSchema != nil && options.ResponseSchema.Description != "" {
		description = options.ResponseSchema.Description
	}

	// Extract required fields
	var required []string
	if reqVal, ok := schema["required"].([]any); ok {
		for _, r := range reqVal {
			if s, ok := r.(string); ok {
				required = append(required, s)
			}
		}
	}

	inputSchema := anthropic.ToolInputSchemaParam{
		Properties: schema["properties"],
		Required:   required,
	}

	tool := anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        jsonResponseToolName,
			Description: anthropic.String(description),
			InputSchema: inputSchema,
		},
	}

	toolChoice := anthropic.ToolChoiceUnionParam{
		OfTool: &anthropic.ToolChoiceToolParam{
			Name: jsonResponseToolName,
		},
	}

	return tool, toolChoice
}

var _ gains.ChatProvider = (*Client)(nil)
