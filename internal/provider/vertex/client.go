package vertex

import (
	"context"
	"encoding/base64"
	"fmt"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/internal/provider/google"
	"google.golang.org/genai"
)

// Client wraps the Google GenAI SDK configured for Vertex AI backend.
type Client struct {
	client   *genai.Client
	project  string
	location string
	model    google.ChatModel
}

// New creates a new Vertex AI client with the given project and location.
// Uses Application Default Credentials (ADC) for authentication.
func New(ctx context.Context, project, location string, opts ...ClientOption) (*Client, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Backend:  genai.BackendVertexAI,
		Project:  project,
		Location: location,
	})
	if err != nil {
		return nil, err
	}
	c := &Client{
		client:   client,
		project:  project,
		location: location,
		model:    google.DefaultChatModel,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// ClientOption configures the Vertex AI client.
type ClientOption func(*Client)

// WithModel sets the default model for requests.
func WithModel(model google.ChatModel) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

// Chat sends a conversation and returns a complete response.
func (c *Client) Chat(ctx context.Context, messages []ai.Message, opts ...ai.Option) (*ai.Response, error) {
	options := ai.ApplyOptions(opts...)
	model := c.model
	if options.Model != nil {
		model = google.ChatModel(options.Model.String())
	}

	contents, err := google.ConvertMessages(messages)
	if err != nil {
		return nil, err
	}
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
		config.Tools = google.ConvertTools(options.Tools)
		if options.ToolChoice != "" {
			config.ToolConfig = google.ConvertToolChoice(options.ToolChoice)
		}
	}

	// Handle JSON mode / response schema
	if options.ResponseSchema != nil {
		config.ResponseMIMEType = "application/json"
		config.ResponseSchema = google.ConvertJSONSchemaToGenaiSchema(options.ResponseSchema.Schema)
	} else if options.ResponseFormat == ai.ResponseFormatJSON {
		config.ResponseMIMEType = "application/json"
	}

	// Enable image output if requested
	if options.ImageOutput {
		config.ResponseModalities = []string{"TEXT", "IMAGE"}
	}

	resp, err := c.client.Models.GenerateContent(ctx, model.String(), contents, config)
	if err != nil {
		return nil, google.WrapError(err)
	}

	content := ""
	var toolCalls []ai.ToolCall
	var parts []ai.ContentPart
	if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		for _, part := range resp.Candidates[0].Content.Parts {
			if part.Text != "" {
				content += part.Text
				if options.ImageOutput {
					parts = append(parts, ai.NewTextPart(part.Text))
				}
			}
			if part.InlineData != nil && len(part.InlineData.Data) > 0 {
				parts = append(parts, ai.ContentPart{
					Type:     ai.ContentPartTypeImage,
					Base64:   base64.StdEncoding.EncodeToString(part.InlineData.Data),
					MimeType: part.InlineData.MIMEType,
				})
			}
		}
		toolCalls = google.ExtractToolCalls(resp.Candidates[0].Content.Parts)
	}

	finishReason := ""
	if len(resp.Candidates) > 0 {
		finishReason = string(resp.Candidates[0].FinishReason)
	}

	usage := ai.Usage{}
	if resp.UsageMetadata != nil {
		usage.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
		usage.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
	}

	return &ai.Response{
		Content:      content,
		FinishReason: finishReason,
		Usage:        usage,
		ToolCalls:    toolCalls,
		Parts:        parts,
	}, nil
}

// ChatStream sends a conversation and returns a channel of streaming events.
func (c *Client) ChatStream(ctx context.Context, messages []ai.Message, opts ...ai.Option) (<-chan ai.StreamEvent, error) {
	options := ai.ApplyOptions(opts...)
	model := c.model
	if options.Model != nil {
		model = google.ChatModel(options.Model.String())
	}

	contents, err := google.ConvertMessages(messages)
	if err != nil {
		return nil, err
	}
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
		config.Tools = google.ConvertTools(options.Tools)
		if options.ToolChoice != "" {
			config.ToolConfig = google.ConvertToolChoice(options.ToolChoice)
		}
	}

	// Handle JSON mode / response schema
	if options.ResponseSchema != nil {
		config.ResponseMIMEType = "application/json"
		config.ResponseSchema = google.ConvertJSONSchemaToGenaiSchema(options.ResponseSchema.Schema)
	} else if options.ResponseFormat == ai.ResponseFormatJSON {
		config.ResponseMIMEType = "application/json"
	}

	// Enable image output if requested
	if options.ImageOutput {
		config.ResponseModalities = []string{"TEXT", "IMAGE"}
	}

	ch := make(chan ai.StreamEvent)

	go func() {
		defer close(ch)

		var fullContent string
		var finishReason string
		var usage ai.Usage
		var allParts []*genai.Part
		var contentParts []ai.ContentPart
		var iterCount int

		for resp, err := range c.client.Models.GenerateContentStream(ctx, model.String(), contents, config) {
			iterCount++
			if err != nil {
				ch <- ai.StreamEvent{Err: google.WrapError(fmt.Errorf("stream error at iteration %d: %w", iterCount, err))}
				return
			}

			// Check for content filtering/blocking
			if resp.PromptFeedback != nil && resp.PromptFeedback.BlockReason != "" {
				ch <- ai.StreamEvent{
					Err: &google.BlockedError{Reason: string(resp.PromptFeedback.BlockReason)},
				}
				return
			}

			if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				for _, part := range resp.Candidates[0].Content.Parts {
					allParts = append(allParts, part)
					if part.Text != "" {
						ch <- ai.StreamEvent{Delta: part.Text}
						fullContent += part.Text
						if options.ImageOutput {
							contentParts = append(contentParts, ai.NewTextPart(part.Text))
						}
					}
					if part.InlineData != nil && len(part.InlineData.Data) > 0 {
						contentParts = append(contentParts, ai.ContentPart{
							Type:     ai.ContentPartTypeImage,
							Base64:   base64.StdEncoding.EncodeToString(part.InlineData.Data),
							MimeType: part.InlineData.MIMEType,
						})
					}
				}
				finishReason = string(resp.Candidates[0].FinishReason)
			}

			if resp.UsageMetadata != nil {
				usage.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
				usage.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
			}
		}

		// Debug: if no iterations happened, something is wrong
		if iterCount == 0 {
			ch <- ai.StreamEvent{Err: fmt.Errorf("stream returned no data")}
			return
		}

		ch <- ai.StreamEvent{
			Done: true,
			Response: &ai.Response{
				Content:      fullContent,
				FinishReason: finishReason,
				Usage:        usage,
				ToolCalls:    google.ExtractToolCalls(allParts),
				Parts:        contentParts,
			},
		}
	}()

	return ch, nil
}

// Embed generates embeddings for the provided texts using Vertex AI's embedding API.
func (c *Client) Embed(ctx context.Context, texts []string, opts ...ai.EmbeddingOption) (*ai.EmbeddingResponse, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("%w: at least one text is required for embedding", ai.ErrEmptyInput)
	}

	options := ai.ApplyEmbeddingOptions(opts...)

	// Determine model
	model := google.DefaultEmbeddingModel
	if options.Model != nil {
		model = google.EmbeddingModel(options.Model.String())
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
	resp, err := c.client.Models.EmbedContent(ctx, model.String(), contents, config)
	if err != nil {
		return nil, google.WrapError(err)
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

	return &ai.EmbeddingResponse{
		Embeddings: embeddings,
		Usage: ai.Usage{
			// Google doesn't return token usage for embeddings
			InputTokens:  0,
			OutputTokens: 0,
		},
	}, nil
}

// GenerateImage generates images from a text prompt using Vertex AI Imagen.
func (c *Client) GenerateImage(ctx context.Context, prompt string, opts ...ai.ImageOption) (*ai.ImageResponse, error) {
	options := ai.ApplyImageOptions(opts...)

	// Determine model
	model := google.DefaultImageModel
	if options.Model != nil {
		model = google.ImageModel(options.Model.String())
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
	resp, err := c.client.Models.GenerateImages(ctx, model.String(), prompt, config)
	if err != nil {
		return nil, google.WrapError(err)
	}

	// Convert response
	images := make([]ai.GeneratedImage, len(resp.GeneratedImages))
	for i, img := range resp.GeneratedImages {
		var b64 string

		// Google returns image bytes directly
		if img.Image != nil && len(img.Image.ImageBytes) > 0 {
			b64 = encodeBase64(img.Image.ImageBytes)
		}

		images[i] = ai.GeneratedImage{
			Base64: b64,
			// Imagen doesn't provide URLs or revised prompts
		}
	}

	return &ai.ImageResponse{Images: images}, nil
}

// convertSizeToAspectRatio maps ImageSize to Imagen aspect ratio strings.
func convertSizeToAspectRatio(size ai.ImageSize) string {
	switch size {
	case ai.ImageSize1024x1024:
		return "1:1"
	case ai.ImageSize1024x1792:
		return "9:16" // Portrait
	case ai.ImageSize1792x1024:
		return "16:9" // Landscape
	default:
		return "1:1"
	}
}

// encodeBase64 encodes bytes to base64 string.
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

var _ ai.ChatProvider = (*Client)(nil)
var _ ai.ImageProvider = (*Client)(nil)
var _ ai.EmbeddingProvider = (*Client)(nil)
