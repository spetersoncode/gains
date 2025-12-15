package gains

// ImageOptions contains configuration for an image generation request.
type ImageOptions struct {
	Model   Model
	Size    ImageSize
	Count   int
	Quality ImageQuality
	Style   ImageStyle
	Format  ImageFormat
}

// ImageOption is a functional option for configuring image generation requests.
type ImageOption func(*ImageOptions)

// WithImageModel sets the model to use for image generation.
func WithImageModel(model Model) ImageOption {
	return func(o *ImageOptions) {
		o.Model = model
	}
}

// WithImageSize sets the dimensions for generated images.
func WithImageSize(size ImageSize) ImageOption {
	return func(o *ImageOptions) {
		o.Size = size
	}
}

// WithImageCount sets the number of images to generate.
// Note: DALL-E 3 only supports n=1; Google Imagen supports up to 4.
func WithImageCount(n int) ImageOption {
	return func(o *ImageOptions) {
		o.Count = n
	}
}

// WithImageQuality sets the quality level for generated images.
// Supported values: "standard", "hd"
// Note: Only supported by DALL-E 3.
func WithImageQuality(q ImageQuality) ImageOption {
	return func(o *ImageOptions) {
		o.Quality = q
	}
}

// WithImageStyle sets the visual style for generated images.
// Supported values: "vivid", "natural"
// Note: Only supported by DALL-E 3.
func WithImageStyle(s ImageStyle) ImageOption {
	return func(o *ImageOptions) {
		o.Style = s
	}
}

// WithImageFormat sets the output format for generated images.
// Supported values: "url", "b64_json"
func WithImageFormat(f ImageFormat) ImageOption {
	return func(o *ImageOptions) {
		o.Format = f
	}
}

// ApplyImageOptions applies functional options to an ImageOptions struct.
func ApplyImageOptions(opts ...ImageOption) *ImageOptions {
	o := &ImageOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
