package gains

import "encoding/json"

// Model is an interface implemented by all provider model types.
// It allows strongly-typed model selection while maintaining a unified API.
// Every model must identify its provider for automatic routing.
type Model interface {
	String() string
	Provider() Provider
}

// ImageOutputCapable is an optional interface that models can implement
// to indicate they support image generation in chat responses.
type ImageOutputCapable interface {
	SupportsImageOutput() bool
}

// ModelSupportsImageOutput checks if a model supports image output.
// Returns true if the model implements ImageOutputCapable and returns true.
func ModelSupportsImageOutput(m Model) bool {
	if ioc, ok := m.(ImageOutputCapable); ok {
		return ioc.SupportsImageOutput()
	}
	return false
}

// ResponseFormat specifies how the model should format its response.
type ResponseFormat string

// ImageAspectRatio specifies the aspect ratio for generated images in chat responses.
// Only supported by Google/Vertex AI models with image output capability.
type ImageAspectRatio string

const (
	ImageAspectRatio1x1   ImageAspectRatio = "1:1"
	ImageAspectRatio2x3   ImageAspectRatio = "2:3"
	ImageAspectRatio3x2   ImageAspectRatio = "3:2"
	ImageAspectRatio3x4   ImageAspectRatio = "3:4"
	ImageAspectRatio4x3   ImageAspectRatio = "4:3"
	ImageAspectRatio9x16  ImageAspectRatio = "9:16"
	ImageAspectRatio16x9  ImageAspectRatio = "16:9"
	ImageAspectRatio21x9  ImageAspectRatio = "21:9"
)

// ImageOutputSize specifies the resolution for generated images in chat responses.
// Only supported by Google/Vertex AI models with image output capability.
type ImageOutputSize string

const (
	ImageOutputSize1K ImageOutputSize = "1K" // Default
	ImageOutputSize2K ImageOutputSize = "2K"
	ImageOutputSize4K ImageOutputSize = "4K"
)

const (
	// ResponseFormatText is the default text response format.
	ResponseFormatText ResponseFormat = "text"
	// ResponseFormatJSON forces the model to output valid JSON.
	ResponseFormatJSON ResponseFormat = "json"
)

// ResponseSchema defines a JSON schema for structured output.
type ResponseSchema struct {
	// Name is a descriptive name for the schema (required by some providers).
	Name string
	// Description explains what this schema represents (optional).
	Description string
	// Schema is the JSON Schema definition.
	Schema json.RawMessage
	// Strict enables strict schema enforcement (OpenAI only, defaults to true).
	Strict bool
}

// Options contains configuration for a chat request.
type Options struct {
	Model            Model
	MaxTokens        int
	Temperature      *float64
	Tools            []Tool
	ToolChoice       ToolChoice
	ResponseFormat   ResponseFormat
	ResponseSchema   *ResponseSchema
	RetryConfig      *RetryConfig     // Per-call retry config override (nil = use client default)
	ImageOutput      bool             // Enable image output for models that support it
	ImageAspectRatio ImageAspectRatio // Aspect ratio for generated images (Google/Vertex only)
	ImageOutputSize  ImageOutputSize  // Resolution for generated images (Google/Vertex only)
}

// Option is a functional option for configuring chat requests.
type Option func(*Options)

// WithModel sets the model to use for the request.
func WithModel(model Model) Option {
	return func(o *Options) {
		o.Model = model
	}
}

// WithMaxTokens sets the maximum number of tokens to generate.
func WithMaxTokens(n int) Option {
	return func(o *Options) {
		o.MaxTokens = n
	}
}

// WithTemperature sets the sampling temperature (0.0 to 2.0).
func WithTemperature(t float64) Option {
	return func(o *Options) {
		o.Temperature = &t
	}
}

// WithTools sets the tools available to the model.
// This is used internally by the agent package. For tool-calling use cases,
// prefer [github.com/spetersoncode/gains/agent] which handles the tool loop.
func WithTools(tools []Tool) Option {
	return func(o *Options) {
		o.Tools = tools
	}
}

// WithToolChoice controls how the model uses tools.
func WithToolChoice(choice ToolChoice) Option {
	return func(o *Options) {
		o.ToolChoice = choice
	}
}

// WithJSONMode forces the model to output valid JSON.
// Note: For Anthropic, this uses a tool-based approach since native JSON mode is not available.
func WithJSONMode() Option {
	return func(o *Options) {
		o.ResponseFormat = ResponseFormatJSON
	}
}

// WithResponseSchema sets a JSON schema for structured output.
// This automatically enables JSON mode.
func WithResponseSchema(schema ResponseSchema) Option {
	return func(o *Options) {
		o.ResponseFormat = ResponseFormatJSON
		o.ResponseSchema = &schema
	}
}

// WithRetry overrides the client's default retry configuration for this request.
// Use DefaultRetryConfig(), DisabledRetryConfig(), or NewRetryConfig() to create configs.
func WithRetry(cfg RetryConfig) Option {
	return func(o *Options) {
		o.RetryConfig = &cfg
	}
}

// WithRetryDisabled disables retry for this request (single attempt).
func WithRetryDisabled() Option {
	return func(o *Options) {
		disabled := DisabledRetryConfig()
		o.RetryConfig = &disabled
	}
}

// WithImageOutput enables image generation in chat responses.
// When enabled, models that support image output (e.g., Gemini image models)
// will include generated images in Response.Parts.
// Note: Only supported by Google and Vertex AI with specific models.
func WithImageOutput() Option {
	return func(o *Options) {
		o.ImageOutput = true
	}
}

// WithImageAspectRatio sets the aspect ratio for generated images in chat responses.
// Supported values: "1:1", "2:3", "3:2", "3:4", "4:3", "9:16", "16:9", "21:9".
// Note: Only supported by Google and Vertex AI with image output enabled.
func WithImageAspectRatio(ratio ImageAspectRatio) Option {
	return func(o *Options) {
		o.ImageAspectRatio = ratio
	}
}

// WithImageOutputSize sets the resolution for generated images in chat responses.
// Supported values: "1K" (default), "2K", "4K".
// Note: Only supported by Google and Vertex AI with image output enabled.
func WithImageOutputSize(size ImageOutputSize) Option {
	return func(o *Options) {
		o.ImageOutputSize = size
	}
}

// ApplyOptions applies functional options to an Options struct.
func ApplyOptions(opts ...Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
