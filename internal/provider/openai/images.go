package openai

import (
	"context"

	"github.com/openai/openai-go"
	ai "github.com/spetersoncode/gains"
)

// GenerateImage generates images from a text prompt using DALL-E.
func (c *Client) GenerateImage(ctx context.Context, prompt string, opts ...ai.ImageOption) (*ai.ImageResponse, error) {
	options := ai.ApplyImageOptions(opts...)

	// Determine model
	model := DefaultImageModel
	if options.Model != nil {
		model = ImageModel(options.Model.String())
	}

	// Build request params
	params := openai.ImageGenerateParams{
		Model:  openai.ImageModel(model.String()),
		Prompt: prompt,
	}

	// Apply size (default: 1024x1024)
	size := options.Size
	if size == "" {
		size = ai.ImageSize1024x1024
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
		format = ai.ImageFormatURL
	}
	params.ResponseFormat = openai.ImageGenerateParamsResponseFormat(format)

	// Make API call
	resp, err := c.client.Images.Generate(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert response
	images := make([]ai.GeneratedImage, len(resp.Data))
	for i, img := range resp.Data {
		images[i] = ai.GeneratedImage{
			URL:           img.URL,
			Base64:        img.B64JSON,
			RevisedPrompt: img.RevisedPrompt,
		}
	}

	return &ai.ImageResponse{Images: images}, nil
}
