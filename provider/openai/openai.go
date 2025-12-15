package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
	"github.com/spetersoncode/gains"
)

const DefaultModel = "gpt-4o"
const DefaultImageModel = "dall-e-3"
const DefaultEmbeddingModel = "text-embedding-3-small"

// Client wraps the OpenAI SDK to implement gains.ChatProvider.
type Client struct {
	client *openai.Client
	model  string
}

// New creates a new OpenAI client with the given API key.
func New(apiKey string, opts ...ClientOption) *Client {
	client := openai.NewClient(option.WithAPIKey(apiKey))
	c := &Client{
		client: &client,
		model:  DefaultModel,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ClientOption configures the OpenAI client.
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
	if len(options.Tools) > 0 {
		params.Tools = convertTools(options.Tools)
		if options.ToolChoice != "" {
			params.ToolChoice = convertToolChoice(options.ToolChoice)
		}
	}

	// Handle JSON mode / response schema
	if options.ResponseSchema != nil {
		params.ResponseFormat = buildOpenAISchemaFormat(options.ResponseSchema)
	} else if options.ResponseFormat == gains.ResponseFormatJSON {
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{
				Type: "json_object",
			},
		}
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
		ToolCalls: extractToolCalls(resp.Choices[0].Message),
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
	if len(options.Tools) > 0 {
		params.Tools = convertTools(options.Tools)
		if options.ToolChoice != "" {
			params.ToolChoice = convertToolChoice(options.ToolChoice)
		}
	}

	// Handle JSON mode / response schema
	if options.ResponseSchema != nil {
		params.ResponseFormat = buildOpenAISchemaFormat(options.ResponseSchema)
	} else if options.ResponseFormat == gains.ResponseFormatJSON {
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{
				Type: "json_object",
			},
		}
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
				ToolCalls: extractToolCallsFromAccumulator(completion.Message.ToolCalls),
			},
		}
	}()

	return ch, nil
}

// GenerateImage generates images from a text prompt using DALL-E.
func (c *Client) GenerateImage(ctx context.Context, prompt string, opts ...gains.ImageOption) (*gains.ImageResponse, error) {
	options := gains.ApplyImageOptions(opts...)

	// Determine model
	model := DefaultImageModel
	if options.Model != "" {
		model = options.Model
	}

	// Build request params
	params := openai.ImageGenerateParams{
		Model:  openai.ImageModel(model),
		Prompt: prompt,
	}

	// Apply size (default: 1024x1024)
	size := options.Size
	if size == "" {
		size = gains.ImageSize1024x1024
	}
	params.Size = openai.ImageGenerateParamsSize(size)

	// Apply count (DALL-E 3 only supports n=1)
	n := options.Count
	if n <= 0 {
		n = 1
	}
	params.N = openai.Int(int64(n))

	// Apply quality
	if options.Quality != "" {
		params.Quality = openai.ImageGenerateParamsQuality(options.Quality)
	}

	// Apply style
	if options.Style != "" {
		params.Style = openai.ImageGenerateParamsStyle(options.Style)
	}

	// Apply format
	format := options.Format
	if format == "" {
		format = gains.ImageFormatURL
	}
	params.ResponseFormat = openai.ImageGenerateParamsResponseFormat(format)

	// Make API call
	resp, err := c.client.Images.Generate(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert response
	images := make([]gains.GeneratedImage, len(resp.Data))
	for i, img := range resp.Data {
		images[i] = gains.GeneratedImage{
			URL:           img.URL,
			Base64:        img.B64JSON,
			RevisedPrompt: img.RevisedPrompt,
		}
	}

	return &gains.ImageResponse{Images: images}, nil
}

// Embed generates embeddings for the provided texts using OpenAI's embedding API.
func (c *Client) Embed(ctx context.Context, texts []string, opts ...gains.EmbeddingOption) (*gains.EmbeddingResponse, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("at least one text is required for embedding")
	}

	options := gains.ApplyEmbeddingOptions(opts...)

	// Determine model
	model := DefaultEmbeddingModel
	if options.Model != "" {
		model = options.Model
	}

	// Build request params
	params := openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(model),
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
	}

	// Apply dimensions (only for text-embedding-3-* models)
	if options.Dimensions > 0 {
		params.Dimensions = openai.Int(int64(options.Dimensions))
	}

	// Make API call
	resp, err := c.client.Embeddings.New(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert response - embeddings are returned in order
	embeddings := make([][]float64, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return &gains.EmbeddingResponse{
		Embeddings: embeddings,
		Usage: gains.Usage{
			InputTokens:  int(resp.Usage.PromptTokens),
			OutputTokens: 0, // Embeddings don't have output tokens
		},
	}, nil
}

func convertMessages(messages []gains.Message) []openai.ChatCompletionMessageParamUnion {
	var result []openai.ChatCompletionMessageParamUnion
	for _, msg := range messages {
		switch msg.Role {
		case gains.RoleUser:
			if msg.HasParts() {
				contentParts := convertPartsToOpenAIParts(msg.Parts)
				result = append(result, openai.ChatCompletionMessageParamUnion{
					OfUser: &openai.ChatCompletionUserMessageParam{
						Content: openai.ChatCompletionUserMessageParamContentUnion{
							OfArrayOfContentParts: contentParts,
						},
					},
				})
			} else {
				result = append(result, openai.UserMessage(msg.Content))
			}
		case gains.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				// Assistant message with tool calls
				toolCalls := make([]openai.ChatCompletionMessageToolCallParam, len(msg.ToolCalls))
				for i, tc := range msg.ToolCalls {
					toolCalls[i] = openai.ChatCompletionMessageToolCallParam{
						ID: tc.ID,
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      tc.Name,
							Arguments: tc.Arguments,
						},
					}
				}
				assistantMsg := openai.ChatCompletionAssistantMessageParam{
					ToolCalls: toolCalls,
				}
				result = append(result, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &assistantMsg,
				})
			} else {
				result = append(result, openai.AssistantMessage(msg.Content))
			}
		case gains.RoleSystem:
			result = append(result, openai.SystemMessage(msg.Content))
		case gains.RoleTool:
			// Tool result messages - one message per tool result
			for _, tr := range msg.ToolResults {
				result = append(result, openai.ToolMessage(tr.Content, tr.ToolCallID))
			}
		default:
			result = append(result, openai.UserMessage(msg.Content))
		}
	}
	return result
}

func convertPartsToOpenAIParts(parts []gains.ContentPart) []openai.ChatCompletionContentPartUnionParam {
	var result []openai.ChatCompletionContentPartUnionParam
	for _, part := range parts {
		switch part.Type {
		case gains.ContentPartTypeText:
			result = append(result, openai.TextContentPart(part.Text))
		case gains.ContentPartTypeImage:
			var imageURL string
			if part.Base64 != "" {
				// Convert to data URI format
				mimeType := part.MimeType
				if mimeType == "" {
					mimeType = "image/jpeg" // Default
				}
				imageURL = fmt.Sprintf("data:%s;base64,%s", mimeType, part.Base64)
			} else if part.ImageURL != "" {
				// Fetch HTTP/HTTPS URLs client-side and convert to base64
				// This avoids issues where OpenAI's servers can't fetch certain URLs
				if strings.HasPrefix(part.ImageURL, "http://") || strings.HasPrefix(part.ImageURL, "https://") {
					data, mimeType, err := fetchImageFromURL(part.ImageURL)
					if err == nil {
						if part.MimeType != "" {
							mimeType = part.MimeType
						}
						b64 := base64.StdEncoding.EncodeToString(data)
						imageURL = fmt.Sprintf("data:%s;base64,%s", mimeType, b64)
					} else {
						// Fall back to passing the URL directly if fetching fails
						imageURL = part.ImageURL
					}
				} else {
					// Non-HTTP URLs (data URIs, etc.) pass through directly
					imageURL = part.ImageURL
				}
			}
			if imageURL != "" {
				result = append(result, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
					URL: imageURL,
				}))
			}
		}
	}
	return result
}

func fetchImageFromURL(url string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	// Set User-Agent header - many servers (including Wikipedia) require this
	req.Header.Set("User-Agent", "gains/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to fetch image: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Get MIME type from Content-Type header or infer from URL
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = inferMimeTypeFromURL(url)
	}

	return data, mimeType, nil
}

func inferMimeTypeFromURL(url string) string {
	lower := strings.ToLower(url)
	switch {
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	default:
		return "image/jpeg" // Default fallback
	}
}

func convertTools(tools []gains.Tool) []openai.ChatCompletionToolParam {
	if len(tools) == 0 {
		return nil
	}
	result := make([]openai.ChatCompletionToolParam, len(tools))
	for i, t := range tools {
		// Parse JSON schema to map for FunctionParameters
		var params shared.FunctionParameters
		if len(t.Parameters) > 0 {
			json.Unmarshal(t.Parameters, &params)
		}
		result[i] = openai.ChatCompletionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name:        t.Name,
				Description: openai.String(t.Description),
				Parameters:  params,
			},
		}
	}
	return result
}

func convertToolChoice(choice gains.ToolChoice) openai.ChatCompletionToolChoiceOptionUnionParam {
	switch choice {
	case gains.ToolChoiceNone:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("none"),
		}
	case gains.ToolChoiceRequired:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("required"),
		}
	default:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("auto"),
		}
	}
}

func extractToolCalls(msg openai.ChatCompletionMessage) []gains.ToolCall {
	if len(msg.ToolCalls) == 0 {
		return nil
	}
	result := make([]gains.ToolCall, len(msg.ToolCalls))
	for i, tc := range msg.ToolCalls {
		result[i] = gains.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		}
	}
	return result
}

func extractToolCallsFromAccumulator(toolCalls []openai.ChatCompletionMessageToolCall) []gains.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}
	result := make([]gains.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = gains.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		}
	}
	return result
}

func buildOpenAISchemaFormat(schema *gains.ResponseSchema) openai.ChatCompletionNewParamsResponseFormatUnion {
	var schemaMap map[string]any
	json.Unmarshal(schema.Schema, &schemaMap)

	name := schema.Name
	if name == "" {
		name = "response_schema"
	}

	strict := true // Default to strict

	// OpenAI strict mode requires additionalProperties: false on all objects
	if strict {
		addAdditionalPropertiesFalse(schemaMap)
	}

	return openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
			Type: "json_schema",
			JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:        name,
				Description: openai.String(schema.Description),
				Schema:      schemaMap,
				Strict:      openai.Bool(strict),
			},
		},
	}
}

// addAdditionalPropertiesFalse recursively adds additionalProperties: false to all object schemas.
// This is required by OpenAI's strict mode.
func addAdditionalPropertiesFalse(schema map[string]any) {
	if schema == nil {
		return
	}

	// If this is an object type, add additionalProperties: false
	if schemaType, ok := schema["type"].(string); ok && schemaType == "object" {
		schema["additionalProperties"] = false
	}

	// Recurse into properties
	if props, ok := schema["properties"].(map[string]any); ok {
		for _, propSchema := range props {
			if propMap, ok := propSchema.(map[string]any); ok {
				addAdditionalPropertiesFalse(propMap)
			}
		}
	}

	// Recurse into array items
	if items, ok := schema["items"].(map[string]any); ok {
		addAdditionalPropertiesFalse(items)
	}
}

var _ gains.ChatProvider = (*Client)(nil)
var _ gains.ImageProvider = (*Client)(nil)
var _ gains.EmbeddingProvider = (*Client)(nil)
