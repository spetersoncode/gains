package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"

	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/agui"
)

// RunAgentInput represents the AG-UI request body for running an agent.
// This mirrors the AG-UI protocol specification.
type RunAgentInput struct {
	ThreadID       string           `json:"thread_id"`
	RunID          string           `json:"run_id"`
	Messages       []events.Message `json:"messages"`
	Tools          []any            `json:"tools,omitempty"`    // Frontend-provided tools (not used by this server)
	Context        []any            `json:"context,omitempty"`  // Context items (not used by this server)
	State          any              `json:"state,omitempty"`    // State (not used by this server)
	ForwardedProps any              `json:"forwarded_props,omitempty"`
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
	// Only accept POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var input RunAgentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Convert AG-UI messages to gains messages
	messages := agui.ToGainsMessages(input.Messages)
	if len(messages) == 0 {
		http.Error(w, "No messages provided", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
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
	for ev := range eventCh {
		aguiEvent := mapper.MapEvent(ev)
		if aguiEvent == nil {
			continue // Skip events with no AG-UI equivalent
		}

		if err := writeSSE(w, flusher, aguiEvent); err != nil {
			log.Printf("Error writing SSE event: %v", err)
			return
		}
	}
}

// writeSSE writes an AG-UI event in SSE format.
func writeSSE(w http.ResponseWriter, flusher http.Flusher, event events.Event) error {
	data, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// Write SSE format: event: TYPE\ndata: {json}\n\n
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type(), string(data)); err != nil {
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
