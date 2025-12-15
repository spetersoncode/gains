package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/spetersoncode/gains"
	"google.golang.org/genai"
)

const DefaultModel = "gemini-2.0-flash"
const DefaultImageModel = "imagen-3.0-generate-002"

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

		// Handle text content
		if msg.Content != "" {
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
