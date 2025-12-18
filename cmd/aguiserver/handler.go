package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"

	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/agui"
	"github.com/spetersoncode/gains/event"
)

// RunAgentInput represents the AG-UI request body for running an agent.
// This mirrors the AG-UI protocol specification.
type RunAgentInput struct {
	ThreadID       string                `json:"thread_id"`
	RunID          string                `json:"run_id"`
	Messages       []aguievents.Message  `json:"messages"`
	Tools          []any                 `json:"tools,omitempty"`    // Frontend-provided tools (not used by this server)
	Context        []any                 `json:"context,omitempty"`  // Context items (not used by this server)
	State          any                   `json:"state,omitempty"`    // State (not used by this server)
	ForwardedProps any                   `json:"forwarded_props,omitempty"`
}

// AgentHandler handles AG-UI agent requests over SSE.
type AgentHandler struct {
	agent  *agent.Agent
	config *Config
}

// NewAgentHandler creates a new handler for the given agent.
func NewAgentHandler(a *agent.Agent, cfg *Config) *AgentHandler {
	return &AgentHandler{agent: a, config: cfg}
}

// ServeHTTP handles POST requests to run the agent and stream events via SSE.
func (h *AgentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Only accept POST
	if r.Method != http.MethodPost {
		slog.Warn("method not allowed", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var input RunAgentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		slog.Warn("invalid request body", "error", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Create request-scoped logger
	log := slog.With(
		"run_id", input.RunID,
		"thread_id", input.ThreadID,
	)

	// Convert AG-UI messages to gains messages
	messages := agui.ToGainsMessages(input.Messages)
	if len(messages) == 0 {
		log.Warn("no messages provided")
		http.Error(w, "No messages provided", http.StatusBadRequest)
		return
	}

	log.Info("request started", "message_count", len(messages))

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Error("streaming not supported")
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Create mapper for this run
	mapper := agui.NewMapper(input.ThreadID, input.RunID)

	// Run agent with streaming
	ctx := r.Context()
	eventCh := h.agent.RunStream(ctx, messages,
		agent.WithMaxSteps(h.config.MaxSteps),
		agent.WithTimeout(h.config.Timeout),
	)

	// Stream events as SSE
	var eventCount int
	var lastError error
	for ev := range eventCh {
		// Log the gains event at debug level
		logGainsEvent(log, ev)

		aguiEvent := mapper.MapEvent(ev)
		if aguiEvent == nil {
			continue // Skip events with no AG-UI equivalent
		}

		eventCount++
		slog.Debug("sending SSE event",
			"run_id", input.RunID,
			"event_type", aguiEvent.Type(),
			"event_num", eventCount,
		)

		if err := writeSSE(w, flusher, aguiEvent); err != nil {
			log.Error("failed to write SSE event", "error", err, "event_type", aguiEvent.Type())
			lastError = err
			return
		}
	}

	duration := time.Since(start)
	if lastError != nil {
		log.Error("request failed",
			"duration_ms", duration.Milliseconds(),
			"events_sent", eventCount,
			"error", lastError,
		)
	} else {
		log.Info("request completed",
			"duration_ms", duration.Milliseconds(),
			"events_sent", eventCount,
		)
	}
}

// writeSSE writes an AG-UI event in SSE format.
func writeSSE(w http.ResponseWriter, flusher http.Flusher, ev aguievents.Event) error {
	data, err := ev.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// Write SSE format: event: TYPE\ndata: {json}\n\n
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type(), string(data)); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	flusher.Flush()
	return nil
}

// logGainsEvent logs a gains event at debug level with relevant details.
func logGainsEvent(log *slog.Logger, ev event.Event) {
	attrs := []any{
		"type", string(ev.Type),
	}

	switch ev.Type {
	case event.RunStart:
		// Run started
	case event.RunEnd:
		if ev.Error != nil {
			attrs = append(attrs, "error", ev.Error.Error())
		}
	case event.MessageStart:
		attrs = append(attrs, "message_id", ev.MessageID)
	case event.MessageDelta:
		if len(ev.Delta) > 0 {
			preview := ev.Delta
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			attrs = append(attrs, "content_preview", preview)
		}
	case event.MessageEnd:
		attrs = append(attrs, "message_id", ev.MessageID)
	case event.ToolCallStart, event.ToolCallEnd, event.ToolCallArgs:
		if ev.ToolCall != nil {
			attrs = append(attrs, "tool", ev.ToolCall.Name, "call_id", ev.ToolCall.ID)
		}
	case event.ToolCallResult:
		if ev.ToolResult != nil {
			attrs = append(attrs, "call_id", ev.ToolResult.ToolCallID)
			if ev.ToolResult.IsError {
				attrs = append(attrs, "is_error", true)
			}
		}
	case event.StepStart, event.StepEnd:
		attrs = append(attrs, "step", ev.StepName)
	}

	log.Debug("gains event", attrs...)
}

// corsMiddleware adds CORS headers for cross-origin frontend requests.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// healthHandler returns a simple health check response.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
