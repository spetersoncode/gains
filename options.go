package gains

import "encoding/json"

// Model is an interface implemented by all provider model types.
// It allows strongly-typed model selection while maintaining a unified API.
// Every model must identify its provider for automatic routing.
type Model interface {
	String() string
	Provider() Provider
}

// ResponseFormat specifies how the model should format its response.
type ResponseFormat string

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
	Model          Model
	MaxTokens      int
	Temperature    *float64
	Tools          []Tool
	ToolChoice     ToolChoice
	ResponseFormat ResponseFormat
	ResponseSchema *ResponseSchema
	RetryConfig    *RetryConfig // Per-call retry config override (nil = use client default)
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

// ApplyOptions applies functional options to an Options struct.
func ApplyOptions(opts ...Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
