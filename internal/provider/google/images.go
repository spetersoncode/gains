package google

import (
	"context"
	"encoding/base64"

	ai "github.com/spetersoncode/gains"
	"google.golang.org/genai"
)

// GenerateImage generates images from a text prompt using Imagen.
func (c *Client) GenerateImage(ctx context.Context, prompt string, opts ...ai.ImageOption) (*ai.ImageResponse, error) {
	options := ai.ApplyImageOptions(opts...)

	// Determine model
	model := DefaultImageModel
	if options.Model != nil {
		model = ImageModel(options.Model.String())
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
		return nil, wrapError(err)
	}

	// Convert response
	images := make([]ai.GeneratedImage, len(resp.GeneratedImages))
	for i, img := range resp.GeneratedImages {
		var b64 string

		// Google returns image bytes directly
		if img.Image != nil && len(img.Image.ImageBytes) > 0 {
			b64 = base64.StdEncoding.EncodeToString(img.Image.ImageBytes)
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
