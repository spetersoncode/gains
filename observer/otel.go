package observer

import (
	"context"

	"github.com/spetersoncode/gains/event"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// OTel integrates with OpenTelemetry tracing.
// It creates spans for runs, steps, and tool calls, and records attributes
// from events. Requires OpenTelemetry SDK to be configured in the application.
type OTel struct {
	tracer     trace.Tracer
	spans      map[string]trace.Span // Active spans by trace+span ID
	attributes []attribute.KeyValue  // Default attributes for all spans
}

// OTelOption configures an OTel observer.
type OTelOption func(*OTel)

// WithTracer sets a custom tracer. If not set, uses the global tracer.
func WithTracer(tracer trace.Tracer) OTelOption {
	return func(o *OTel) {
		o.tracer = tracer
	}
}

// WithAttributes adds default attributes to all spans.
func WithAttributes(attrs ...attribute.KeyValue) OTelOption {
	return func(o *OTel) {
		o.attributes = append(o.attributes, attrs...)
	}
}

// NewOTel creates an OpenTelemetry observer.
// By default, uses the global tracer provider with the name "gains".
func NewOTel(opts ...OTelOption) *OTel {
	o := &OTel{
		spans: make(map[string]trace.Span),
	}
	for _, opt := range opts {
		opt(o)
	}
	if o.tracer == nil {
		o.tracer = otel.Tracer("gains")
	}
	return o
}

// Observe processes events and manages OpenTelemetry spans.
func (o *OTel) Observe(ctx context.Context, e event.Event) {
	switch e.Type {
	case event.RunStart:
		o.startSpan(ctx, e, "run")

	case event.RunEnd:
		if span := o.getSpan(e); span != nil {
			if e.Response != nil {
				span.SetAttributes(
					attribute.Int("gains.input_tokens", e.Response.Usage.InputTokens),
					attribute.Int("gains.output_tokens", e.Response.Usage.OutputTokens),
				)
			}
			span.End()
			o.removeSpan(e)
		}

	case event.RunError:
		if span := o.getSpan(e); span != nil {
			span.SetStatus(codes.Error, e.Error.Error())
			span.RecordError(e.Error)
			span.End()
			o.removeSpan(e)
		}

	case event.StepStart:
		o.startSpan(ctx, e, "step:"+e.StepName)

	case event.StepEnd:
		if span := o.getSpan(e); span != nil {
			span.End()
			o.removeSpan(e)
		}

	case event.ToolCallStart:
		if e.ToolCall != nil {
			o.startSpan(ctx, e, "tool:"+e.ToolCall.Name)
		}

	case event.ToolCallEnd:
		// Tool call span ends on result

	case event.ToolCallResult:
		if span := o.getSpan(e); span != nil {
			if e.ToolResult != nil {
				span.SetAttributes(
					attribute.Bool("gains.tool.is_error", e.ToolResult.IsError),
				)
				if e.ToolResult.IsError {
					span.SetStatus(codes.Error, "tool execution failed")
				}
			}
			span.End()
			o.removeSpan(e)
		}

	case event.RouteSelected:
		if span := o.getSpan(e); span != nil {
			span.SetAttributes(attribute.String("gains.route", e.RouteName))
		}

	case event.LoopIteration:
		if span := o.getSpan(e); span != nil {
			span.AddEvent("loop_iteration", trace.WithAttributes(
				attribute.Int("gains.iteration", e.Iteration),
			))
		}

	case event.RetryAttempt:
		if span := o.getSpan(e); span != nil {
			span.AddEvent("retry_attempt", trace.WithAttributes(
				attribute.Int("gains.attempt", e.Attempt),
			))
		}

	case event.RetryFailed:
		if span := o.getSpan(e); span != nil {
			span.AddEvent("retry_failed", trace.WithAttributes(
				attribute.Int("gains.attempt", e.Attempt),
				attribute.String("gains.error", e.Error.Error()),
			))
		}
	}
}

// Close is a no-op; spans are ended individually.
func (o *OTel) Close() error { return nil }

func (o *OTel) startSpan(ctx context.Context, e event.Event, name string) {
	_, span := o.tracer.Start(ctx, name,
		trace.WithAttributes(o.attributes...),
		trace.WithAttributes(
			attribute.String("gains.trace_id", e.TraceID),
			attribute.String("gains.span_id", e.SpanID),
			attribute.String("gains.parent_span_id", e.ParentSpanID),
		),
	)
	o.spans[o.spanKey(e)] = span
}

func (o *OTel) getSpan(e event.Event) trace.Span {
	return o.spans[o.spanKey(e)]
}

func (o *OTel) removeSpan(e event.Event) {
	delete(o.spans, o.spanKey(e))
}

func (o *OTel) spanKey(e event.Event) string {
	return e.TraceID + ":" + e.SpanID
}

// Verify OTel implements Observer.
var _ Observer = (*OTel)(nil)
