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
	"github.com/spetersoncode/gains/tool"
)

// AgentHandler handles AG-UI agent requests over SSE.
type AgentHandler struct {
	agent    *agent.Agent
	registry *tool.Registry
	config   *Config
}

// NewAgentHandler creates a new handler for the given agent and registry.
func NewAgentHandler(a *agent.Agent, r *tool.Registry, cfg *Config) *AgentHandler {
	return &AgentHandler{agent: a, registry: r, config: cfg}
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
	var input agui.RunAgentInput
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

	// Validate and convert input
	prepared, err := input.Prepare()
	if err != nil {
		log.Warn("invalid input", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Register frontend tools if provided
	if len(prepared.Tools) > 0 {
		gainsTools := prepared.GainsTools()
		if err := h.registry.RegisterClientTools(gainsTools); err != nil {
			log.Warn("failed to register frontend tools", "error", err)
		} else {
			log.Info("registered frontend tools", "count", len(prepared.ToolNames), "names", prepared.ToolNames)
		}
	}

	// Ensure cleanup of client tools after request
	defer func() {
		for _, name := range prepared.ToolNames {
			h.registry.Unregister(name)
		}
		if len(prepared.ToolNames) > 0 {
			log.Debug("unregistered frontend tools", "count", len(prepared.ToolNames))
		}
	}()

	log.Info("request started", "message_count", len(prepared.Messages))

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
	mapper := agui.NewMapper(prepared.ThreadID, prepared.RunID)

	// Run agent with streaming
	ctx := r.Context()
	gainsEvents := h.agent.RunStream(ctx, prepared.Messages,
		agent.WithMaxSteps(h.config.MaxSteps),
		agent.WithTimeout(h.config.Timeout),
	)

	// Stream events as SSE using the mapper's filtered stream
	var eventCount int
	var lastError error
	for aguiEvent := range mapper.MapStream(gainsEvents) {
		eventCount++
		log.Debug("sending SSE event",
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
