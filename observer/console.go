package observer

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/model"
)

// Console outputs human-readable event information to a writer.
// Useful for debugging and development.
type Console struct {
	w           io.Writer
	mu          sync.Mutex
	startTime   time.Time
	costTracker *CostTracker
	verbose     bool
}

// ConsoleOption configures a Console observer.
type ConsoleOption func(*Console)

// WithCostTracking enables cost tracking with the given model pricing.
func WithCostTracking(m model.ChatModel) ConsoleOption {
	return func(c *Console) {
		c.costTracker = NewCostTrackerForModel(m)
	}
}

// WithCostTrackingPricing enables cost tracking with explicit pricing.
func WithCostTrackingPricing(pricing model.ChatPricing) ConsoleOption {
	return func(c *Console) {
		c.costTracker = NewCostTracker(pricing)
	}
}

// WithVerbose enables verbose output including all event types.
func WithVerbose() ConsoleOption {
	return func(c *Console) {
		c.verbose = true
	}
}

// NewConsole creates a new console observer that writes to the given writer.
func NewConsole(w io.Writer, opts ...ConsoleOption) *Console {
	c := &Console{w: w}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewStdoutConsole creates a console observer that writes to stdout.
func NewStdoutConsole(opts ...ConsoleOption) *Console {
	return NewConsole(os.Stdout, opts...)
}

// NewStderrConsole creates a console observer that writes to stderr.
func NewStderrConsole(opts ...ConsoleOption) *Console {
	return NewConsole(os.Stderr, opts...)
}

// Observe outputs the event to the console in human-readable format.
func (c *Console) Observe(ctx context.Context, e event.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Track costs if enabled
	if c.costTracker != nil {
		c.costTracker.Track(e)
	}

	switch e.Type {
	case event.RunStart:
		c.startTime = e.Timestamp
		c.printf("▶ Run started")
		if e.TraceID != "" {
			c.printf(" [trace:%s]", e.TraceID[:8])
		}
		c.printf("\n")

	case event.RunEnd:
		elapsed := e.Timestamp.Sub(c.startTime)
		c.printf("■ Run completed in %s", elapsed.Round(time.Millisecond))
		if e.Response != nil && e.Response.Usage.InputTokens > 0 {
			c.printf(" [%d in / %d out tokens]",
				e.Response.Usage.InputTokens, e.Response.Usage.OutputTokens)
		}
		if c.costTracker != nil {
			summary := c.costTracker.Summary()
			c.printf(" [$%.4f]", summary.Cost)
		}
		c.printf("\n")

	case event.RunError:
		c.printf("✖ Run error: %v\n", e.Error)

	case event.MessageStart:
		if c.verbose {
			c.printf("◆ Message started [%s]\n", e.MessageID)
		}

	case event.MessageDelta:
		// Stream content directly without prefix
		c.printf("%s", e.Delta)

	case event.MessageEnd:
		// Add newline after streaming content
		c.printf("\n")

	case event.ToolCallStart:
		if e.ToolCall != nil {
			c.printf("⚙ Tool: %s", e.ToolCall.Name)
			if c.verbose && e.ToolCall.ID != "" {
				c.printf(" [%s]", e.ToolCall.ID)
			}
			c.printf("\n")
		}

	case event.ToolCallArgs:
		if c.verbose && e.ToolCall != nil {
			c.printf("  Args: %s\n", e.ToolCall.Arguments)
		}

	case event.ToolCallEnd:
		// No output in normal mode

	case event.ToolCallExecuting:
		if c.verbose {
			c.printf("  Executing...\n")
		}

	case event.ToolCallResult:
		if e.ToolResult != nil {
			if e.ToolResult.IsError {
				c.printf("  ✖ Error: %s\n", truncate(e.ToolResult.Content, 200))
			} else if c.verbose {
				c.printf("  Result: %s\n", truncate(e.ToolResult.Content, 200))
			}
		}

	case event.ToolCallApproved:
		c.printf("  ✓ Approved\n")

	case event.ToolCallRejected:
		c.printf("  ✗ Rejected: %s\n", e.Message)

	case event.StepStart:
		c.printf("→ Step: %s\n", e.StepName)

	case event.StepEnd:
		if c.verbose {
			c.printf("← Step completed: %s\n", e.StepName)
		}

	case event.StepSkipped:
		if c.verbose {
			c.printf("⊘ Step skipped: %s\n", e.StepName)
		}

	case event.RouteSelected:
		c.printf("↳ Route: %s\n", e.RouteName)

	case event.LoopIteration:
		c.printf("↺ Loop iteration %d\n", e.Iteration)

	case event.ParallelStart:
		if c.verbose {
			c.printf("⊕ Parallel execution started\n")
		}

	case event.ParallelEnd:
		if c.verbose {
			c.printf("⊕ Parallel execution completed\n")
		}

	case event.RetryAttempt:
		c.printf("↻ Retry attempt %d\n", e.Attempt)

	case event.RetryFailed:
		c.printf("↻ Attempt %d failed: %v\n", e.Attempt, e.Error)

	case event.RetryScheduled:
		if c.verbose {
			c.printf("↻ Retry scheduled\n")
		}

	case event.RetrySuccess:
		if c.verbose {
			c.printf("↻ Retry succeeded on attempt %d\n", e.Attempt)
		}

	case event.RetryExhausted:
		c.printf("↻ All retry attempts exhausted\n")

	default:
		if c.verbose {
			c.printf("? Unknown event: %s\n", e.Type)
		}
	}
}

// Close is a no-op for the console observer.
func (c *Console) Close() error { return nil }

// CostSummary returns the accumulated cost summary, or nil if cost tracking is disabled.
func (c *Console) CostSummary() *CostSummary {
	if c.costTracker == nil {
		return nil
	}
	summary := c.costTracker.Summary()
	return &summary
}

func (c *Console) printf(format string, args ...any) {
	fmt.Fprintf(c.w, format, args...)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// Verify Console implements Observer.
var _ Observer = (*Console)(nil)
