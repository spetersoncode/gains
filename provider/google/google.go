package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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

// GenerateImage generates images from a text prompt using Imagen.
func (c *Client) GenerateImage(ctx context.Context, prompt string, opts ...gains.ImageOption) (*gains.ImageResponse, error) {
	options := gains.ApplyImageOptions(opts...)

	// Determine model
	model := DefaultImageModel
	if options.Model != "" {
		model = options.Model
	}

	// Build image generation config
	config := &genai.GenerateImagesConfig{}

	// Apply count (Imagen supports 1-4)
	n := options.Count
	if n <= 0 {
		n = 1
	}
	config.NumberOfImages = int32(n)

	// Apply aspect ratio based on size
	if options.Size != "" {
		config.AspectRatio = convertSizeToAspectRatio(options.Size)
	}

	// Make API call
	resp, err := c.client.Models.GenerateImages(ctx, model, prompt, config)
	if err != nil {
		return nil, err
	}

	// Convert response
	images := make([]gains.GeneratedImage, len(resp.GeneratedImages))
	for i, img := range resp.GeneratedImages {
		var b64 string

		// Google returns image bytes directly
		if img.Image != nil && len(img.Image.ImageBytes) > 0 {
			b64 = base64.StdEncoding.EncodeToString(img.Image.ImageBytes)
		}

		images[i] = gains.GeneratedImage{
			Base64: b64,
			// Imagen doesn't provide URLs or revised prompts
		}
	}

	return &gains.ImageResponse{Images: images}, nil
}

// Embed generates embeddings for the provided texts using Google's embedding API.
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

	// Build embed content config
	config := &genai.EmbedContentConfig{}

	// Apply dimensions
	if options.Dimensions > 0 {
		dims := int32(options.Dimensions)
		config.OutputDimensionality = &dims
	}

	// Apply task type
	if options.TaskType != "" {
		config.TaskType = string(options.TaskType)
	}

	// Build contents from texts
	contents := make([]*genai.Content, len(texts))
	for i, text := range texts {
		contents[i] = &genai.Content{
			Parts: []*genai.Part{{Text: text}},
		}
	}

	// Make API call
	resp, err := c.client.Models.EmbedContent(ctx, model, contents, config)
	if err != nil {
		return nil, err
	}

	// Convert response
	embeddings := make([][]float64, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		// Convert []float32 to []float64
		embeddings[i] = make([]float64, len(emb.Values))
		for j, v := range emb.Values {
			embeddings[i][j] = float64(v)
		}
	}

	return &gains.EmbeddingResponse{
		Embeddings: embeddings,
		Usage: gains.Usage{
			// Google doesn't return token usage for embeddings
			InputTokens:  0,
			OutputTokens: 0,
		},
	}, nil
}

// convertSizeToAspectRatio maps ImageSize to Imagen aspect ratio strings.
func convertSizeToAspectRatio(size gains.ImageSize) string {
	switch size {
	case gains.ImageSize1024x1024:
		return "1:1"
	case gains.ImageSize1024x1792:
		return "9:16" // Portrait
	case gains.ImageSize1792x1024:
		return "16:9" // Landscape
	default:
		return "1:1"
	}
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
		case gains.RoleTool:
			// Tool results are sent as user messages with FunctionResponse parts
			role = "user"
		}

		var parts []*genai.Part

		// Handle multimodal content
		if msg.HasParts() {
			parts = convertPartsToGoogleParts(msg.Parts)
		} else if msg.Content != "" {
			parts = append(parts, &genai.Part{Text: msg.Content})
		}

		// Handle tool calls (assistant messages)
		for _, tc := range msg.ToolCalls {
			var args map[string]any
			json.Unmarshal([]byte(tc.Arguments), &args)
			parts = append(parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					Name: tc.Name,
					Args: args,
				},
			})
		}

		// Handle tool results
		for _, tr := range msg.ToolResults {
			// Parse the content as JSON if possible, otherwise wrap as text
			var result map[string]any
			if err := json.Unmarshal([]byte(tr.Content), &result); err != nil {
				result = map[string]any{"result": tr.Content}
			}
			parts = append(parts, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     tr.ToolCallID, // Google uses the function name, but we store ID; handle in extractToolCalls
					Response: result,
				},
			})
		}

		if len(parts) > 0 {
			contents = append(contents, &genai.Content{
				Role:  role,
				Parts: parts,
			})
		}
	}

	return contents
}

func convertPartsToGoogleParts(parts []gains.ContentPart) []*genai.Part {
	var result []*genai.Part
	for _, part := range parts {
		switch part.Type {
		case gains.ContentPartTypeText:
			result = append(result, &genai.Part{Text: part.Text})
		case gains.ContentPartTypeImage:
			if part.Base64 != "" {
				// Decode base64 to bytes
				data, err := base64.StdEncoding.DecodeString(part.Base64)
				if err == nil {
					mimeType := part.MimeType
					if mimeType == "" {
						mimeType = "image/jpeg" // Default
					}
					result = append(result, &genai.Part{
						InlineData: &genai.Blob{
							Data:     data,
							MIMEType: mimeType,
						},
					})
				}
			} else if part.ImageURL != "" {
				// Google supports GCS URIs directly
				if strings.HasPrefix(part.ImageURL, "gs://") {
					mimeType := part.MimeType
					if mimeType == "" {
						mimeType = inferMimeTypeFromURL(part.ImageURL)
					}
					result = append(result, &genai.Part{
						FileData: &genai.FileData{
							FileURI:  part.ImageURL,
							MIMEType: mimeType,
						},
					})
				} else {
					// HTTP/HTTPS URLs need to be fetched and converted to inline data
					data, mimeType, err := fetchImageFromURL(part.ImageURL)
					if err == nil {
						if part.MimeType != "" {
							mimeType = part.MimeType
						}
						result = append(result, &genai.Part{
							InlineData: &genai.Blob{
								Data:     data,
								MIMEType: mimeType,
							},
						})
					}
				}
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

func convertTools(tools []gains.Tool) []*genai.Tool {
	if len(tools) == 0 {
		return nil
	}

	funcs := make([]*genai.FunctionDeclaration, len(tools))
	for i, t := range tools {
		funcs[i] = &genai.FunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  convertJSONSchemaToGenaiSchema(t.Parameters),
		}
	}

	return []*genai.Tool{{FunctionDeclarations: funcs}}
}

func convertJSONSchemaToGenaiSchema(schemaJSON json.RawMessage) *genai.Schema {
	if len(schemaJSON) == 0 {
		return nil
	}

	var schema map[string]any
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return nil
	}

	return convertSchemaObject(schema)
}

func convertSchemaObject(schema map[string]any) *genai.Schema {
	if schema == nil {
		return nil
	}

	result := &genai.Schema{}

	// Handle type
	if typeVal, ok := schema["type"].(string); ok {
		switch typeVal {
		case "string":
			result.Type = genai.TypeString
		case "number":
			result.Type = genai.TypeNumber
		case "integer":
			result.Type = genai.TypeInteger
		case "boolean":
			result.Type = genai.TypeBoolean
		case "array":
			result.Type = genai.TypeArray
		case "object":
			result.Type = genai.TypeObject
		}
	}

	// Handle description
	if desc, ok := schema["description"].(string); ok {
		result.Description = desc
	}

	// Handle enum
	if enumVal, ok := schema["enum"].([]any); ok {
		for _, e := range enumVal {
			if s, ok := e.(string); ok {
				result.Enum = append(result.Enum, s)
			}
		}
	}

	// Handle properties (for objects)
	if props, ok := schema["properties"].(map[string]any); ok {
		result.Properties = make(map[string]*genai.Schema)
		for name, propSchema := range props {
			if propMap, ok := propSchema.(map[string]any); ok {
				result.Properties[name] = convertSchemaObject(propMap)
			}
		}
	}

	// Handle required fields
	if required, ok := schema["required"].([]any); ok {
		for _, r := range required {
			if s, ok := r.(string); ok {
				result.Required = append(result.Required, s)
			}
		}
	}

	// Handle array items
	if items, ok := schema["items"].(map[string]any); ok {
		result.Items = convertSchemaObject(items)
	}

	return result
}

func convertToolChoice(choice gains.ToolChoice) *genai.ToolConfig {
	switch choice {
	case gains.ToolChoiceNone:
		return &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeNone,
			},
		}
	case gains.ToolChoiceRequired:
		return &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAny,
			},
		}
	default:
		return &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAuto,
			},
		}
	}
}

func extractToolCalls(parts []*genai.Part) []gains.ToolCall {
	var calls []gains.ToolCall
	for i, part := range parts {
		if part.FunctionCall != nil {
			args, _ := json.Marshal(part.FunctionCall.Args)
			calls = append(calls, gains.ToolCall{
				ID:        fmt.Sprintf("call_%d_%s", i, part.FunctionCall.Name),
				Name:      part.FunctionCall.Name,
				Arguments: string(args),
			})
		}
	}
	return calls
}

var _ gains.ChatProvider = (*Client)(nil)
var _ gains.ImageProvider = (*Client)(nil)
var _ gains.EmbeddingProvider = (*Client)(nil)
