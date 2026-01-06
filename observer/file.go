package observer

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/spetersoncode/gains/event"
)

// File writes events as JSON lines to a file or writer.
// Each event is written as a single JSON object on its own line (JSONL format).
type File struct {
	w      io.Writer
	closer io.Closer // Only set if we opened the file
	mu     sync.Mutex
	enc    *json.Encoder
	closed bool
}

// fileEvent is the JSON structure for file output.
type fileEvent struct {
	Type         string `json:"type"`
	TraceID      string `json:"trace_id,omitempty"`
	SpanID       string `json:"span_id,omitempty"`
	ParentSpanID string `json:"parent_span_id,omitempty"`
	Timestamp    string `json:"timestamp"`

	// Event-specific fields
	MessageID    string `json:"message_id,omitempty"`
	Delta        string `json:"delta,omitempty"`
	Step         int    `json:"step,omitempty"`
	StepName     string `json:"step_name,omitempty"`
	RouteName    string `json:"route_name,omitempty"`
	Iteration    int    `json:"iteration,omitempty"`
	Attempt      int    `json:"attempt,omitempty"`
	Error        string `json:"error,omitempty"`
	Message      string `json:"message,omitempty"`
	ToolName     string `json:"tool_name,omitempty"`
	ToolCallID   string `json:"tool_call_id,omitempty"`
	ActivityID   string `json:"activity_id,omitempty"`
	ActivityType string `json:"activity_type,omitempty"`
}

// NewFile creates a new file observer that writes to the specified path.
// The file is created if it doesn't exist, or appended to if it does.
func NewFile(path string) (*File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &File{
		w:      f,
		closer: f,
		enc:    json.NewEncoder(f),
	}, nil
}

// NewFileWriter creates a new file observer that writes to the given writer.
// The caller is responsible for closing the writer.
func NewFileWriter(w io.Writer) *File {
	return &File{
		w:   w,
		enc: json.NewEncoder(w),
	}
}

// Observe writes the event as JSON to the file.
func (f *File) Observe(ctx context.Context, e event.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return
	}

	fe := fileEvent{
		Type:         string(e.Type),
		TraceID:      e.TraceID,
		SpanID:       e.SpanID,
		ParentSpanID: e.ParentSpanID,
		Timestamp:    e.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
		MessageID:    e.MessageID,
		Delta:        e.Delta,
		Step:         e.Step,
		StepName:     e.StepName,
		RouteName:    e.RouteName,
		Iteration:    e.Iteration,
		Attempt:      e.Attempt,
		Message:      e.Message,
		ActivityID:   e.ActivityID,
		ActivityType: string(e.Activity),
	}

	if e.Error != nil {
		fe.Error = e.Error.Error()
	}

	if e.ToolCall != nil {
		fe.ToolName = e.ToolCall.Name
		fe.ToolCallID = e.ToolCall.ID
	}

	_ = f.enc.Encode(fe)
}

// Close closes the file if it was opened by NewFile.
func (f *File) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.closed = true
	if f.closer != nil {
		return f.closer.Close()
	}
	return nil
}

// Verify File implements Observer.
var _ Observer = (*File)(nil)
