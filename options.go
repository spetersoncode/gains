package gains

// Options contains configuration for a chat request.
type Options struct {
	Model       string
	MaxTokens   int
	Temperature *float64
}

// Option is a functional option for configuring chat requests.
type Option func(*Options)

// WithModel sets the model to use for the request.
func WithModel(model string) Option {
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

// ApplyOptions applies functional options to an Options struct.
func ApplyOptions(opts ...Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
