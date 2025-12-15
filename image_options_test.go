package gains

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyImageOptions(t *testing.T) {
	t.Run("returns empty options when no options provided", func(t *testing.T) {
		opts := ApplyImageOptions()
		assert.NotNil(t, opts)
		assert.Empty(t, opts.Model)
		assert.Empty(t, opts.Size)
		assert.Zero(t, opts.Count)
		assert.Empty(t, opts.Quality)
		assert.Empty(t, opts.Style)
		assert.Empty(t, opts.Format)
	})

	t.Run("applies multiple options", func(t *testing.T) {
		opts := ApplyImageOptions(
			WithImageModel("dall-e-3"),
			WithImageSize(ImageSize1024x1024),
			WithImageCount(1),
			WithImageQuality(ImageQualityHD),
			WithImageStyle(ImageStyleVivid),
			WithImageFormat(ImageFormatURL),
		)

		assert.Equal(t, "dall-e-3", opts.Model)
		assert.Equal(t, ImageSize1024x1024, opts.Size)
		assert.Equal(t, 1, opts.Count)
		assert.Equal(t, ImageQualityHD, opts.Quality)
		assert.Equal(t, ImageStyleVivid, opts.Style)
		assert.Equal(t, ImageFormatURL, opts.Format)
	})
}

func TestWithImageModel(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected string
	}{
		{"sets dall-e-3", "dall-e-3", "dall-e-3"},
		{"sets dall-e-2", "dall-e-2", "dall-e-2"},
		{"sets Google Imagen", "imagen-3.0-generate-002", "imagen-3.0-generate-002"},
		{"handles empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyImageOptions(WithImageModel(tt.model))
			assert.Equal(t, tt.expected, opts.Model)
		})
	}
}

func TestWithImageSize(t *testing.T) {
	tests := []struct {
		name     string
		size     ImageSize
		expected ImageSize
	}{
		{"sets square 1024x1024", ImageSize1024x1024, ImageSize1024x1024},
		{"sets portrait 1024x1792", ImageSize1024x1792, ImageSize1024x1792},
		{"sets landscape 1792x1024", ImageSize1792x1024, ImageSize1792x1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyImageOptions(WithImageSize(tt.size))
			assert.Equal(t, tt.expected, opts.Size)
		})
	}
}

func TestWithImageCount(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected int
	}{
		{"sets 1 image", 1, 1},
		{"sets 4 images", 4, 4},
		{"handles zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyImageOptions(WithImageCount(tt.count))
			assert.Equal(t, tt.expected, opts.Count)
		})
	}
}

func TestWithImageQuality(t *testing.T) {
	tests := []struct {
		name     string
		quality  ImageQuality
		expected ImageQuality
	}{
		{"sets standard quality", ImageQualityStandard, ImageQualityStandard},
		{"sets HD quality", ImageQualityHD, ImageQualityHD},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyImageOptions(WithImageQuality(tt.quality))
			assert.Equal(t, tt.expected, opts.Quality)
		})
	}
}

func TestWithImageStyle(t *testing.T) {
	tests := []struct {
		name     string
		style    ImageStyle
		expected ImageStyle
	}{
		{"sets vivid style", ImageStyleVivid, ImageStyleVivid},
		{"sets natural style", ImageStyleNatural, ImageStyleNatural},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyImageOptions(WithImageStyle(tt.style))
			assert.Equal(t, tt.expected, opts.Style)
		})
	}
}

func TestWithImageFormat(t *testing.T) {
	tests := []struct {
		name     string
		format   ImageFormat
		expected ImageFormat
	}{
		{"sets URL format", ImageFormatURL, ImageFormatURL},
		{"sets Base64 format", ImageFormatBase64, ImageFormatBase64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyImageOptions(WithImageFormat(tt.format))
			assert.Equal(t, tt.expected, opts.Format)
		})
	}
}

func TestImageSizeConstants(t *testing.T) {
	assert.Equal(t, ImageSize("1024x1024"), ImageSize1024x1024)
	assert.Equal(t, ImageSize("1024x1792"), ImageSize1024x1792)
	assert.Equal(t, ImageSize("1792x1024"), ImageSize1792x1024)
}

func TestImageQualityConstants(t *testing.T) {
	assert.Equal(t, ImageQuality("standard"), ImageQualityStandard)
	assert.Equal(t, ImageQuality("hd"), ImageQualityHD)
}

func TestImageStyleConstants(t *testing.T) {
	assert.Equal(t, ImageStyle("vivid"), ImageStyleVivid)
	assert.Equal(t, ImageStyle("natural"), ImageStyleNatural)
}

func TestImageFormatConstants(t *testing.T) {
	assert.Equal(t, ImageFormat("url"), ImageFormatURL)
	assert.Equal(t, ImageFormat("b64_json"), ImageFormatBase64)
}

func TestImageOptionsOverride(t *testing.T) {
	t.Run("later option overrides earlier", func(t *testing.T) {
		opts := ApplyImageOptions(
			WithImageModel("dall-e-2"),
			WithImageSize(ImageSize1024x1024),
			WithImageModel("dall-e-3"),
			WithImageSize(ImageSize1792x1024),
		)
		assert.Equal(t, "dall-e-3", opts.Model)
		assert.Equal(t, ImageSize1792x1024, opts.Size)
	})
}

func TestImageResponseStruct(t *testing.T) {
	t.Run("creates response with URL images", func(t *testing.T) {
		resp := ImageResponse{
			Images: []GeneratedImage{
				{
					URL:           "https://example.com/image1.png",
					RevisedPrompt: "A detailed prompt",
				},
			},
		}

		assert.Len(t, resp.Images, 1)
		assert.Equal(t, "https://example.com/image1.png", resp.Images[0].URL)
		assert.Equal(t, "A detailed prompt", resp.Images[0].RevisedPrompt)
		assert.Empty(t, resp.Images[0].Base64)
	})

	t.Run("creates response with Base64 images", func(t *testing.T) {
		resp := ImageResponse{
			Images: []GeneratedImage{
				{
					Base64: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAAB",
				},
			},
		}

		assert.Len(t, resp.Images, 1)
		assert.NotEmpty(t, resp.Images[0].Base64)
		assert.Empty(t, resp.Images[0].URL)
	})

	t.Run("creates response with multiple images", func(t *testing.T) {
		resp := ImageResponse{
			Images: []GeneratedImage{
				{URL: "https://example.com/image1.png"},
				{URL: "https://example.com/image2.png"},
				{URL: "https://example.com/image3.png"},
			},
		}

		assert.Len(t, resp.Images, 3)
	})
}

func TestGeneratedImageStruct(t *testing.T) {
	t.Run("creates image with all fields", func(t *testing.T) {
		img := GeneratedImage{
			URL:           "https://example.com/image.png",
			Base64:        "",
			RevisedPrompt: "An enhanced version of the original prompt",
		}

		assert.Equal(t, "https://example.com/image.png", img.URL)
		assert.Empty(t, img.Base64)
		assert.Equal(t, "An enhanced version of the original prompt", img.RevisedPrompt)
	})
}
