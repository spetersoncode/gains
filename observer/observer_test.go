package observer

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceIDGeneration(t *testing.T) {
	id := NewTraceID()
	assert.Len(t, id, 32, "trace ID should be 32 hex chars")

	// Ensure uniqueness
	id2 := NewTraceID()
	assert.NotEqual(t, id, id2, "trace IDs should be unique")
}

func TestSpanIDGeneration(t *testing.T) {
	id := NewSpanID()
	assert.Len(t, id, 16, "span ID should be 16 hex chars")

	// Ensure uniqueness
	id2 := NewSpanID()
	assert.NotEqual(t, id, id2, "span IDs should be unique")
}

func TestTraceContext(t *testing.T) {
	tc := NewTraceContext()
	assert.True(t, tc.IsValid(), "new trace context should be valid")
	assert.NotEmpty(t, tc.TraceID)
	assert.NotEmpty(t, tc.SpanID)
	assert.Empty(t, tc.ParentSpanID)

	// Test WithSpan
	child := tc.WithSpan(NewSpanID())
	assert.Equal(t, tc.TraceID, child.TraceID, "trace ID should be preserved")
	assert.NotEqual(t, tc.SpanID, child.SpanID, "span ID should be new")
	assert.Equal(t, tc.SpanID, child.ParentSpanID, "parent span ID should be set")
}

func TestTraceContextFromContext(t *testing.T) {
	ctx := context.Background()

	// Empty context
	tc := TraceContextFromContext(ctx)
	assert.False(t, tc.IsValid(), "empty context should return invalid trace context")

	// With trace context
	expected := NewTraceContext()
	ctx = WithTraceContext(ctx, expected)
	tc = TraceContextFromContext(ctx)
	assert.Equal(t, expected, tc)
}

func TestChildSpan(t *testing.T) {
	ctx := context.Background()

	// From empty context - creates new root
	ctx, tc := ChildSpan(ctx)
	assert.True(t, tc.IsValid())
	assert.Empty(t, tc.ParentSpanID)

	// From existing context - creates child
	ctx, child := ChildSpan(ctx)
	assert.Equal(t, tc.TraceID, child.TraceID)
	assert.Equal(t, tc.SpanID, child.ParentSpanID)
	assert.NotEqual(t, tc.SpanID, child.SpanID)

	// Verify context has child trace
	fromCtx := TraceContextFromContext(ctx)
	assert.Equal(t, child, fromCtx)
}

func TestApplyExtractTraceContext(t *testing.T) {
	tc := NewTraceContext()
	e := event.Event{Type: event.RunStart}

	ApplyTraceContext(&e, tc)
	assert.Equal(t, tc.TraceID, e.TraceID)
	assert.Equal(t, tc.SpanID, e.SpanID)
	assert.Equal(t, tc.ParentSpanID, e.ParentSpanID)

	extracted := ExtractTraceContext(e)
	assert.Equal(t, tc, extracted)
}

func TestNoopObserver(t *testing.T) {
	obs := NewNoop()
	defer obs.Close()

	ctx := context.Background()
	e := event.Event{Type: event.RunStart, Timestamp: time.Now()}

	// Should not panic
	obs.Observe(ctx, e)
	assert.NoError(t, obs.Close())
}

func TestFileObserver(t *testing.T) {
	var buf bytes.Buffer
	obs := NewFileWriter(&buf)

	ctx := context.Background()
	tc := NewTraceContext()

	// Emit events
	e := event.Event{
		Type:      event.RunStart,
		TraceID:   tc.TraceID,
		SpanID:    tc.SpanID,
		Timestamp: time.Now(),
	}
	obs.Observe(ctx, e)

	e2 := event.Event{
		Type:      event.ToolCallStart,
		TraceID:   tc.TraceID,
		SpanID:    tc.SpanID,
		ToolCall:  &ai.ToolCall{Name: "test_tool", ID: "call_123"},
		Timestamp: time.Now(),
	}
	obs.Observe(ctx, e2)

	require.NoError(t, obs.Close())

	// Parse output
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)

	var fe1 map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &fe1))
	assert.Equal(t, "run_start", fe1["type"])
	assert.Equal(t, tc.TraceID, fe1["trace_id"])

	var fe2 map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &fe2))
	assert.Equal(t, "tool_call_start", fe2["type"])
	assert.Equal(t, "test_tool", fe2["tool_name"])
}

func TestMultiObserver(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	obs1 := NewFileWriter(&buf1)
	obs2 := NewFileWriter(&buf2)

	multi := NewMulti(obs1, obs2)

	ctx := context.Background()
	e := event.Event{
		Type:      event.RunStart,
		Timestamp: time.Now(),
	}
	multi.Observe(ctx, e)
	require.NoError(t, multi.Close())

	// Both should have received the event
	assert.NotEmpty(t, buf1.String())
	assert.NotEmpty(t, buf2.String())
	assert.Equal(t, buf1.String(), buf2.String())
}

func TestMultiObserverAdd(t *testing.T) {
	var buf bytes.Buffer
	multi := NewMulti()

	// Empty multi should work
	ctx := context.Background()
	multi.Observe(ctx, event.Event{Type: event.RunStart, Timestamp: time.Now()})

	// Add observer
	multi.Add(NewFileWriter(&buf))
	multi.Observe(ctx, event.Event{Type: event.RunEnd, Timestamp: time.Now()})

	assert.Contains(t, buf.String(), "run_end")
}

func TestConsoleObserver(t *testing.T) {
	var buf bytes.Buffer
	obs := NewConsole(&buf)

	ctx := context.Background()
	tc := NewTraceContext()

	// Run start
	obs.Observe(ctx, event.Event{
		Type:      event.RunStart,
		TraceID:   tc.TraceID,
		Timestamp: time.Now(),
	})
	assert.Contains(t, buf.String(), "▶ Run started")

	// Message delta
	buf.Reset()
	obs.Observe(ctx, event.Event{
		Type:      event.MessageDelta,
		Delta:     "Hello",
		Timestamp: time.Now(),
	})
	assert.Equal(t, "Hello", buf.String())

	// Tool call
	buf.Reset()
	obs.Observe(ctx, event.Event{
		Type:      event.ToolCallStart,
		ToolCall:  &ai.ToolCall{Name: "search", ID: "call_1"},
		Timestamp: time.Now(),
	})
	assert.Contains(t, buf.String(), "⚙ Tool: search")

	// Run end with usage
	buf.Reset()
	startTime := time.Now().Add(-100 * time.Millisecond)
	obs.Observe(ctx, event.Event{
		Type:      event.RunStart,
		Timestamp: startTime,
	})
	obs.Observe(ctx, event.Event{
		Type: event.RunEnd,
		Response: &ai.Response{
			Usage: ai.Usage{InputTokens: 100, OutputTokens: 50},
		},
		Timestamp: time.Now(),
	})
	assert.Contains(t, buf.String(), "■ Run completed")
	assert.Contains(t, buf.String(), "100 in / 50 out")
}

func TestConsoleObserverWithCostTracking(t *testing.T) {
	var buf bytes.Buffer
	obs := NewConsole(&buf, WithCostTracking(model.ClaudeSonnet45))

	ctx := context.Background()

	obs.Observe(ctx, event.Event{
		Type:      event.RunStart,
		Timestamp: time.Now(),
	})
	obs.Observe(ctx, event.Event{
		Type: event.RunEnd,
		Response: &ai.Response{
			Usage: ai.Usage{InputTokens: 1000, OutputTokens: 500},
		},
		Timestamp: time.Now(),
	})

	assert.Contains(t, buf.String(), "$")
	summary := obs.CostSummary()
	require.NotNil(t, summary)
	assert.Equal(t, 1000, summary.InputTokens)
	assert.Equal(t, 500, summary.OutputTokens)
	assert.Greater(t, summary.Cost, 0.0)
}

func TestCostTracker(t *testing.T) {
	ct := NewCostTracker(model.ChatPricing{
		InputPerMillion:  1.0,  // $1 per million input
		OutputPerMillion: 2.0,  // $2 per million output
	})

	// Track usage from event
	e := event.Event{
		Type: event.RunEnd,
		Response: &ai.Response{
			Usage: ai.Usage{InputTokens: 1000000, OutputTokens: 500000},
		},
	}
	tracked := ct.Track(e)
	assert.True(t, tracked)

	usage := ct.Usage()
	assert.Equal(t, 1000000, usage.InputTokens)
	assert.Equal(t, 500000, usage.OutputTokens)

	cost := ct.Cost()
	// 1M input @ $1/M = $1.00, 500K output @ $2/M = $1.00, total = $2.00
	assert.InDelta(t, 2.0, cost, 0.001)

	summary := ct.Summary()
	assert.Equal(t, 1500000, summary.TotalTokens)
	assert.InDelta(t, 2.0, summary.Cost, 0.001)

	// Reset
	ct.Reset()
	assert.Equal(t, 0, ct.Usage().InputTokens)
}

func TestCostTrackerNoUsage(t *testing.T) {
	ct := NewCostTracker(model.ChatPricing{})

	// Event without response
	tracked := ct.Track(event.Event{Type: event.RunStart})
	assert.False(t, tracked)

	// Event with zero usage
	tracked = ct.Track(event.Event{
		Type:     event.RunEnd,
		Response: &ai.Response{},
	})
	assert.False(t, tracked)
}

func TestEmitter(t *testing.T) {
	var buf bytes.Buffer
	obs := NewFileWriter(&buf)

	tc := NewTraceContext()
	emitter := NewEmitter(obs, tc)

	ctx := context.Background()
	emitter.RunStart(ctx)
	emitter.RunEnd(ctx)

	require.NoError(t, obs.Close())

	// Verify events have trace context
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)

	var e map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &e))
	assert.Equal(t, tc.TraceID, e["trace_id"])
	assert.Equal(t, tc.SpanID, e["span_id"])
}

func TestEmitterChild(t *testing.T) {
	var buf bytes.Buffer
	obs := NewFileWriter(&buf)

	parent := NewEmitter(obs, NewTraceContext())
	child := parent.Child()

	assert.Equal(t, parent.TraceContext().TraceID, child.TraceContext().TraceID)
	assert.Equal(t, parent.TraceContext().SpanID, child.TraceContext().ParentSpanID)
	assert.NotEqual(t, parent.TraceContext().SpanID, child.TraceContext().SpanID)
}

func TestEmitterFromContext(t *testing.T) {
	var buf bytes.Buffer
	obs := NewFileWriter(&buf)

	// From empty context - creates new trace
	ctx := context.Background()
	emitter := EmitterFromContext(ctx, obs)
	assert.True(t, emitter.TraceContext().IsValid())

	// From context with trace
	tc := NewTraceContext()
	ctx = WithTraceContext(ctx, tc)
	emitter = EmitterFromContext(ctx, obs)
	assert.Equal(t, tc, emitter.TraceContext())
}
