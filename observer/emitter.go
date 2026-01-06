package observer

import (
	"context"
	"time"

	"github.com/spetersoncode/gains/event"
)

// Emitter wraps an Observer and TraceContext to emit events with automatic
// trace context application. It provides a convenient way to emit events
// within a traced operation.
type Emitter struct {
	observer Observer
	tc       TraceContext
}

// NewEmitter creates an Emitter with the given observer and trace context.
// If tc is invalid, a new root trace context is created.
func NewEmitter(obs Observer, tc TraceContext) *Emitter {
	if !tc.IsValid() {
		tc = NewTraceContext()
	}
	return &Emitter{
		observer: obs,
		tc:       tc,
	}
}

// EmitterFromContext creates an Emitter from context.
// If no trace context exists in the context, a new root trace is created.
func EmitterFromContext(ctx context.Context, obs Observer) *Emitter {
	tc := TraceContextFromContext(ctx)
	return NewEmitter(obs, tc)
}

// Emit sends an event with the emitter's trace context applied.
func (e *Emitter) Emit(ctx context.Context, ev event.Event) {
	ev.Timestamp = time.Now()
	ApplyTraceContext(&ev, e.tc)
	e.observer.Observe(ctx, ev)
}

// EmitType sends an event with just a type (convenience for simple events).
func (e *Emitter) EmitType(ctx context.Context, eventType event.Type) {
	e.Emit(ctx, event.Event{Type: eventType})
}

// TraceContext returns the emitter's trace context.
func (e *Emitter) TraceContext() TraceContext {
	return e.tc
}

// Observer returns the underlying observer.
func (e *Emitter) Observer() Observer {
	return e.observer
}

// Child creates a child emitter with a new span.
// The child span's ParentSpanID is set to this emitter's SpanID.
func (e *Emitter) Child() *Emitter {
	return &Emitter{
		observer: e.observer,
		tc:       e.tc.WithSpan(NewSpanID()),
	}
}

// ChildWithName creates a child emitter and emits a StepStart event.
func (e *Emitter) ChildWithName(ctx context.Context, stepName string) *Emitter {
	child := e.Child()
	child.Emit(ctx, event.Event{
		Type:     event.StepStart,
		StepName: stepName,
	})
	return child
}

// RunStart emits a RunStart event.
func (e *Emitter) RunStart(ctx context.Context) {
	e.EmitType(ctx, event.RunStart)
}

// RunEnd emits a RunEnd event.
func (e *Emitter) RunEnd(ctx context.Context) {
	e.EmitType(ctx, event.RunEnd)
}

// RunError emits a RunError event.
func (e *Emitter) RunError(ctx context.Context, err error) {
	e.Emit(ctx, event.Event{
		Type:  event.RunError,
		Error: err,
	})
}

// StepStart emits a StepStart event.
func (e *Emitter) StepStart(ctx context.Context, name string) {
	e.Emit(ctx, event.Event{
		Type:     event.StepStart,
		StepName: name,
	})
}

// StepEnd emits a StepEnd event.
func (e *Emitter) StepEnd(ctx context.Context, name string) {
	e.Emit(ctx, event.Event{
		Type:     event.StepEnd,
		StepName: name,
	})
}

// WithSpanContext returns a new context with this emitter's trace context.
func (e *Emitter) WithSpanContext(ctx context.Context) context.Context {
	return WithTraceContext(ctx, e.tc)
}
