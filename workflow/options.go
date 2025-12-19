package workflow

import (
	"context"
	"time"

	ai "github.com/spetersoncode/gains"
)

// ErrorHandler is called when a step encounters an error.
// Return nil to suppress the error, or return an error to propagate it.
type ErrorHandler func(ctx context.Context, stepName string, err error) error

// Options contains configuration for workflow execution.
type Options struct {
	// Timeout sets a deadline for the entire workflow.
	Timeout time.Duration

	// StepTimeout sets default timeout for individual steps.
	StepTimeout time.Duration

	// MaxConcurrency limits parallel step execution (0 = unlimited).
	MaxConcurrency int

	// ErrorHandler is called on step errors.
	ErrorHandler ErrorHandler

	// ContinueOnError allows workflow to continue after step errors.
	ContinueOnError bool

	// ChatOptions are passed to LLM calls within steps.
	ChatOptions []ai.Option
}

// Option is a functional option for workflow configuration.
type Option func(*Options)

// WithTimeout sets the overall workflow timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.Timeout = d
	}
}

// WithStepTimeout sets the default timeout for each step.
func WithStepTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.StepTimeout = d
	}
}

// WithMaxConcurrency limits parallel step execution.
// A value of 0 means unlimited concurrency.
func WithMaxConcurrency(n int) Option {
	return func(o *Options) {
		o.MaxConcurrency = n
	}
}

// WithErrorHandler sets a custom error handler.
func WithErrorHandler(fn ErrorHandler) Option {
	return func(o *Options) {
		o.ErrorHandler = fn
	}
}

// WithContinueOnError allows the workflow to continue after errors.
func WithContinueOnError(enabled bool) Option {
	return func(o *Options) {
		o.ContinueOnError = enabled
	}
}

// WithChatOptions passes options to LLM calls.
func WithChatOptions(opts ...ai.Option) Option {
	return func(o *Options) {
		o.ChatOptions = append(o.ChatOptions, opts...)
	}
}

// WithModel is a convenience option to set the model for chat calls.
func WithModel(model ai.Model) Option {
	return func(o *Options) {
		o.ChatOptions = append(o.ChatOptions, ai.WithModel(model))
	}
}

// WithMaxTokens is a convenience option to set max tokens for chat calls.
func WithMaxTokens(n int) Option {
	return func(o *Options) {
		o.ChatOptions = append(o.ChatOptions, ai.WithMaxTokens(n))
	}
}

// WithTemperature is a convenience option to set temperature for chat calls.
func WithTemperature(t float64) Option {
	return func(o *Options) {
		o.ChatOptions = append(o.ChatOptions, ai.WithTemperature(t))
	}
}

// ApplyOptions applies functional options with defaults.
func ApplyOptions(opts ...Option) *Options {
	o := &Options{
		StepTimeout:     2 * time.Minute,
		ContinueOnError: false,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
