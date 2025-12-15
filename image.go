package gains

import "context"

// ImageProvider defines the interface for AI image generation providers.
type ImageProvider interface {
	// GenerateImage creates images from a text prompt.
	GenerateImage(ctx context.Context, prompt string, opts ...ImageOption) (*ImageResponse, error)
}

// ImageResponse represents a complete response from an image generation provider.
type ImageResponse struct {
	Images []GeneratedImage
}

// GeneratedImage represents a single generated image.
type GeneratedImage struct {
	// URL contains the URL to the generated image (if URL format was requested).
	URL string
	// Base64 contains the base64-encoded image data (if Base64 format was requested).
	Base64 string
	// RevisedPrompt contains the prompt that was actually used.
	// DALL-E 3 rewrites prompts for better results.
	RevisedPrompt string
}

// ImageFormat specifies the output format for generated images.
type ImageFormat string

const (
	// ImageFormatURL returns images as URLs.
	ImageFormatURL ImageFormat = "url"
	// ImageFormatBase64 returns images as base64-encoded data.
	ImageFormatBase64 ImageFormat = "b64_json"
)

// ImageSize represents predefined image dimensions.
type ImageSize string

const (
	ImageSize1024x1024 ImageSize = "1024x1024"
	ImageSize1024x1792 ImageSize = "1024x1792" // Portrait
	ImageSize1792x1024 ImageSize = "1792x1024" // Landscape
)

// ImageQuality specifies the quality level for generated images.
// Note: Only supported by DALL-E 3.
type ImageQuality string

const (
	ImageQualityStandard ImageQuality = "standard"
	ImageQualityHD       ImageQuality = "hd"
)

// ImageStyle specifies the visual style for generated images.
// Note: Only supported by DALL-E 3.
type ImageStyle string

const (
	ImageStyleVivid   ImageStyle = "vivid"
	ImageStyleNatural ImageStyle = "natural"
)
