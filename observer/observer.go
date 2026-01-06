// Package observer provides an observability interface for tracing and monitoring
// AI operations. Observers receive events from client, agent, and workflow packages
// to enable logging, tracing, metrics collection, and debugging.
//
// The package provides several built-in observer implementations:
//   - Console: Human-readable output to stdout/stderr
//   - File: JSON-formatted output to a file
//   - Noop: Discards all events (default when no observer is configured)
//   - Multi: Fans out events to multiple observers
//   - OpenTelemetry: Integration with OpenTelemetry tracing
//
// # Basic Usage
//
// Create an observer and pass it to client, agent, or workflow:
//
//	obs := observer.NewConsole(os.Stdout)
//	defer obs.Close()
//
//	agent := agent.New(client, tools, agent.WithObserver(obs))
//
// # Trace Context
//
// Events include trace context fields (TraceID, SpanID, ParentSpanID) that enable
// distributed tracing. Use the context functions to propagate trace context:
//
//	ctx = observer.WithTraceContext(ctx, observer.TraceContext{
//	    TraceID: observer.NewTraceID(),
//	    SpanID:  observer.NewSpanID(),
//	})
//
//	// Later, retrieve the context
//	tc := observer.TraceContextFromContext(ctx)
package observer

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/spetersoncode/gains/event"
)

// Observer defines the interface for receiving and processing events.
// Implementations must be safe for concurrent use.
type Observer interface {
	// Observe processes an event. Implementations should not block for
	// extended periods. For slow operations (network calls, file I/O),
	// use buffering or async processing.
	//
	// The context may contain trace context and cancellation signals.
	// Implementations should respect context cancellation.
	Observe(ctx context.Context, e event.Event)

	// Close releases resources held by the observer.
	// After Close is called, Observe should be a no-op.
	Close() error
}

// TraceContext holds distributed tracing identifiers.
// These fields enable correlation of events across services and reconstruction
// of execution traces.
type TraceContext struct {
	// TraceID is a unique identifier for the entire trace, spanning all spans
	// from the initial request through all downstream operations.
	// Format: 32 hex characters (128-bit).
	TraceID string

	// SpanID is a unique identifier for the current operation/span.
	// Format: 16 hex characters (64-bit).
	SpanID string

	// ParentSpanID links this span to its parent, enabling trace hierarchy.
	// Empty for root spans.
	// Format: 16 hex characters (64-bit).
	ParentSpanID string
}

// IsValid returns true if the trace context has at least a TraceID and SpanID.
func (tc TraceContext) IsValid() bool {
	return tc.TraceID != "" && tc.SpanID != ""
}

// WithSpan creates a new TraceContext with a new SpanID, setting the current
// SpanID as the ParentSpanID. The TraceID is preserved.
func (tc TraceContext) WithSpan(spanID string) TraceContext {
	return TraceContext{
		TraceID:      tc.TraceID,
		SpanID:       spanID,
		ParentSpanID: tc.SpanID,
	}
}

// traceContextKey is the context key for trace context.
type traceContextKey struct{}

// WithTraceContext returns a new context with the given trace context.
func WithTraceContext(ctx context.Context, tc TraceContext) context.Context {
	return context.WithValue(ctx, traceContextKey{}, tc)
}

// TraceContextFromContext retrieves the trace context from the context.
// Returns an empty TraceContext if none is set.
func TraceContextFromContext(ctx context.Context) TraceContext {
	tc, _ := ctx.Value(traceContextKey{}).(TraceContext)
	return tc
}

// NewTraceID generates a new random trace ID (128-bit, 32 hex chars).
// This follows the W3C Trace Context format for trace IDs.
func NewTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// NewSpanID generates a new random span ID (64-bit, 16 hex chars).
// This follows the W3C Trace Context format for span IDs.
func NewSpanID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// NewTraceContext creates a new root trace context with fresh TraceID and SpanID.
func NewTraceContext() TraceContext {
	return TraceContext{
		TraceID: NewTraceID(),
		SpanID:  NewSpanID(),
	}
}

// ChildSpan creates a child span from the given trace context.
// If the parent context has no trace context, a new root trace is created.
func ChildSpan(ctx context.Context) (context.Context, TraceContext) {
	parent := TraceContextFromContext(ctx)
	var tc TraceContext
	if parent.TraceID == "" {
		// No parent, create a new root trace
		tc = NewTraceContext()
	} else {
		// Create child span
		tc = parent.WithSpan(NewSpanID())
	}
	return WithTraceContext(ctx, tc), tc
}

// ApplyTraceContext copies trace context fields from a TraceContext to an Event.
func ApplyTraceContext(e *event.Event, tc TraceContext) {
	e.TraceID = tc.TraceID
	e.SpanID = tc.SpanID
	e.ParentSpanID = tc.ParentSpanID
}

// ExtractTraceContext extracts trace context fields from an Event.
func ExtractTraceContext(e event.Event) TraceContext {
	return TraceContext{
		TraceID:      e.TraceID,
		SpanID:       e.SpanID,
		ParentSpanID: e.ParentSpanID,
	}
}
