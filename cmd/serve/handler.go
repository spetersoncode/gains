package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"

	"github.com/spetersoncode/gains/a2a"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/agui"
	"github.com/spetersoncode/gains/tool"
	"github.com/spetersoncode/gains/workflow"
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

// WorkflowHandler handles AG-UI workflow requests over SSE.
type WorkflowHandler struct {
	registry *workflow.Registry
	config   *Config
}

// NewWorkflowHandler creates a new handler for the given workflow registry.
func NewWorkflowHandler(r *workflow.Registry, cfg *Config) *WorkflowHandler {
	return &WorkflowHandler{registry: r, config: cfg}
}

// ServeHTTP handles POST requests to run a workflow and stream events via SSE.
func (h *WorkflowHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Only accept POST
	if r.Method != http.MethodPost {
		slog.Warn("method not allowed", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var input agui.RunWorkflowInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		slog.Warn("invalid request body", "error", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Create request-scoped logger
	log := slog.With(
		"run_id", input.RunID,
		"thread_id", input.ThreadID,
		"workflow", input.WorkflowName,
	)

	// Validate input
	prepared, err := input.Prepare()
	if err != nil {
		log.Warn("invalid input", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check workflow exists
	if !h.registry.Has(prepared.WorkflowName) {
		log.Warn("workflow not found", "workflow", prepared.WorkflowName)
		http.Error(w, fmt.Sprintf("workflow not found: %s", prepared.WorkflowName), http.StatusNotFound)
		return
	}

	log.Info("workflow request started")

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
	mapper := agui.NewMapper(prepared.ThreadID, prepared.RunID,
		agui.WithInitialState(prepared.State),
	)

	// Run workflow with streaming
	ctx := r.Context()
	gainsEvents := h.registry.RunStream(ctx, prepared.WorkflowName, prepared.State)

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
		log.Error("workflow request failed",
			"duration_ms", duration.Milliseconds(),
			"events_sent", eventCount,
			"error", lastError,
		)
	} else {
		log.Info("workflow request completed",
			"duration_ms", duration.Milliseconds(),
			"events_sent", eventCount,
		)
	}
}

// A2AHandler handles A2A protocol requests over SSE.
// Supports the tasks/send and tasks/sendSubscribe methods.
type A2AHandler struct {
	executor a2a.Executor
	config   *Config
}

// NewA2AHandler creates a new handler for A2A requests.
func NewA2AHandler(executor a2a.Executor, cfg *Config) *A2AHandler {
	return &A2AHandler{executor: executor, config: cfg}
}

// jsonRPCRequest represents a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// jsonRPCResponse represents a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id,omitempty"`
	Result  any           `json:"result,omitempty"`
	Error   *jsonRPCError `json:"error,omitempty"`
}

// jsonRPCError represents a JSON-RPC 2.0 error.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// JSON-RPC error codes
const (
	jsonRPCParseError     = -32700
	jsonRPCInvalidRequest = -32600
	jsonRPCMethodNotFound = -32601
	jsonRPCInvalidParams  = -32602
	jsonRPCInternalError  = -32603
)

// ServeHTTP handles A2A protocol requests.
func (h *A2AHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Only accept POST
	if r.Method != http.MethodPost {
		slog.Warn("method not allowed", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON-RPC request
	var req jsonRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("invalid JSON-RPC request", "error", err)
		h.writeError(w, nil, jsonRPCParseError, "Parse error: "+err.Error())
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		h.writeError(w, req.ID, jsonRPCInvalidRequest, "Invalid JSON-RPC version")
		return
	}

	log := slog.With("method", req.Method, "id", req.ID)
	log.Info("A2A request received")

	// Dispatch based on method
	switch req.Method {
	case "tasks/send":
		h.handleSend(w, r, req, log)
	case "tasks/sendSubscribe":
		h.handleSendSubscribe(w, r, req, log, start)
	default:
		log.Warn("unknown method")
		h.writeError(w, req.ID, jsonRPCMethodNotFound, "Method not found: "+req.Method)
	}
}

// handleSend handles synchronous task execution.
func (h *A2AHandler) handleSend(w http.ResponseWriter, r *http.Request, req jsonRPCRequest, log *slog.Logger) {
	// Parse params
	var params a2a.SendMessageRequest
	if err := json.Unmarshal(req.Params, &params); err != nil {
		log.Warn("invalid params", "error", err)
		h.writeError(w, req.ID, jsonRPCInvalidParams, "Invalid params: "+err.Error())
		return
	}

	// Execute task
	task, err := h.executor.Execute(r.Context(), params)
	if err != nil {
		log.Error("execution error", "error", err)
		h.writeError(w, req.ID, jsonRPCInternalError, "Execution error: "+err.Error())
		return
	}

	// Return result
	h.writeResult(w, req.ID, task)
	log.Info("A2A request completed", "task_id", task.ID, "status", task.Status.State)
}

// handleSendSubscribe handles streaming task execution via SSE.
func (h *A2AHandler) handleSendSubscribe(w http.ResponseWriter, r *http.Request, req jsonRPCRequest, log *slog.Logger, start time.Time) {
	// Parse params
	var params a2a.SendMessageRequest
	if err := json.Unmarshal(req.Params, &params); err != nil {
		log.Warn("invalid params", "error", err)
		h.writeError(w, req.ID, jsonRPCInvalidParams, "Invalid params: "+err.Error())
		return
	}

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

	// Execute with streaming
	events := h.executor.ExecuteStream(r.Context(), params)

	// Stream events as SSE
	var eventCount int
	for evt := range events {
		eventCount++

		data, err := json.Marshal(evt)
		if err != nil {
			log.Error("failed to marshal event", "error", err)
			continue
		}

		// Determine event type
		var eventType string
		switch evt.(type) {
		case a2a.TaskStatusUpdateEvent:
			eventType = "status-update"
		case a2a.TaskArtifactUpdateEvent:
			eventType = "artifact-update"
		default:
			eventType = "message"
		}

		// Write SSE format
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(data)); err != nil {
			log.Error("failed to write SSE event", "error", err)
			return
		}

		flusher.Flush()
		log.Debug("sent A2A event", "event_type", eventType, "event_num", eventCount)
	}

	duration := time.Since(start)
	log.Info("A2A streaming request completed",
		"duration_ms", duration.Milliseconds(),
		"events_sent", eventCount,
	)
}

// writeResult writes a successful JSON-RPC response.
func (h *A2AHandler) writeResult(w http.ResponseWriter, id any, result any) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// writeError writes a JSON-RPC error response.
func (h *A2AHandler) writeError(w http.ResponseWriter, id any, code int, message string) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonRPCError{
			Code:    code,
			Message: message,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
