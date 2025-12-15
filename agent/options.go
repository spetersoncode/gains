package agent

import (
	"context"
	"time"

	"github.com/spetersoncode/gains"
)

// ApproverFunc is called when a tool call requires approval.
// It returns true to approve the call, or false with a reason to reject it.
// The rejection reason is sent back to the model as an error result.
type ApproverFunc func(ctx context.Context, call gains.ToolCall) (approved bool, reason string)

// StopFunc is a custom predicate to determine if the agent should stop.
// It receives the current step number and the latest response.
// Return true to stop the agent.
type StopFunc func(step int, response *gains.Response) bool

// Options contains configuration for agent execution.
type Options struct {
	// MaxSteps limits the number of agent iterations.
	// Set to 0 for unlimited (not recommended). Default is 10.
	MaxSteps int

	// Timeout sets a deadline for the entire agent execution.
	// A value of 0 means no timeout (context deadline applies).
	Timeout time.Duration

	// HandlerTimeout sets the timeout for each individual tool handler.
	// A value of 0 means no per-handler timeout. Default is 30 seconds.
	HandlerTimeout time.Duration

	// ParallelToolCalls enables concurrent execution of multiple tool calls.
	// Default is true.
	ParallelToolCalls bool

	// Approver enables human-in-the-loop approval for tool calls.
	// If nil, all tool calls are automatically approved.
	Approver ApproverFunc

	// ApprovalRequired specifies which tool names require approval.
	// If empty and Approver is set, all tools require approval.
	// If non-empty, only the listed tools require approval.
	ApprovalRequired []string

	// StopPredicate is a custom termination condition.
	// Called after each step; return true to stop the agent.
	StopPredicate StopFunc

	// ChatOptions are passed through to the underlying ChatProvider.
	ChatOptions []gains.Option
}

// Option is a functional option for configuring agent execution.
type Option func(*Options)

// WithMaxSteps sets the maximum number of agent iterations.
// Default is 10. Set to 0 for unlimited (not recommended).
func WithMaxSteps(n int) Option {
	return func(o *Options) {
		o.MaxSteps = n
	}
}

// WithTimeout sets a deadline for the entire agent execution.
func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.Timeout = d
	}
}

// WithHandlerTimeout sets the timeout for each individual tool handler.
// Default is 30 seconds. Set to 0 for no per-handler timeout.
func WithHandlerTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.HandlerTimeout = d
	}
}

// WithParallelToolCalls enables or disables concurrent tool execution.
// Default is true.
func WithParallelToolCalls(enabled bool) Option {
	return func(o *Options) {
		o.ParallelToolCalls = enabled
	}
}

// WithApprover sets the human-in-the-loop approval function.
// The function is called before each tool execution and must return
// an approval decision.
func WithApprover(fn ApproverFunc) Option {
	return func(o *Options) {
		o.Approver = fn
	}
}

// WithApprovalRequired specifies which tools require approval.
// If not called but WithApprover is used, all tools require approval.
// Call with specific tool names to only require approval for those tools.
func WithApprovalRequired(tools ...string) Option {
	return func(o *Options) {
		o.ApprovalRequired = tools
	}
}

// WithStopPredicate sets a custom termination condition.
// The predicate is called after each step with the step number and response.
// Return true to stop the agent.
func WithStopPredicate(fn StopFunc) Option {
	return func(o *Options) {
		o.StopPredicate = fn
	}
}

// WithChatOptions passes options through to the ChatProvider.
// These options are applied to every chat call made by the agent.
func WithChatOptions(opts ...gains.Option) Option {
	return func(o *Options) {
		o.ChatOptions = append(o.ChatOptions, opts...)
	}
}

// WithModel is a convenience option to set the model for chat calls.
func WithModel(model gains.Model) Option {
	return func(o *Options) {
		o.ChatOptions = append(o.ChatOptions, gains.WithModel(model))
	}
}

// WithMaxTokens is a convenience option to set max tokens for chat calls.
func WithMaxTokens(n int) Option {
	return func(o *Options) {
		o.ChatOptions = append(o.ChatOptions, gains.WithMaxTokens(n))
	}
}

// WithTemperature is a convenience option to set temperature for chat calls.
func WithTemperature(t float64) Option {
	return func(o *Options) {
		o.ChatOptions = append(o.ChatOptions, gains.WithTemperature(t))
	}
}

// ApplyOptions applies functional options to an Options struct with defaults.
func ApplyOptions(opts ...Option) *Options {
	o := &Options{
		MaxSteps:          10,
		HandlerTimeout:    30 * time.Second,
		ParallelToolCalls: true,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
