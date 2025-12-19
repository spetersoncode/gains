// Package main provides a reference AG-UI HTTP server that exposes a gains
// agent via the AG-UI protocol over Server-Sent Events (SSE).
//
// This server demonstrates how to integrate gains with AG-UI compatible
// frontends like CopilotKit. It uses only the Go standard library for HTTP.
//
// Configuration is via environment variables:
//
//	AGUI_PORT         - Server port (default: 8000)
//	AGUI_LOG_LEVEL    - Log level: debug, info, warn, error (default: info)
//	GAINS_PROVIDER    - Provider: anthropic, openai, or google (required)
//	GAINS_MODEL       - Model override (optional, uses provider default)
//	GAINS_MAX_STEPS   - Max agent iterations (default: 10)
//	GAINS_TIMEOUT     - Agent timeout (default: 2m)
//	GAINS_DEMO_TOOLS  - Enable demo tools (default: true)
//	ANTHROPIC_API_KEY - Anthropic API key
//	OPENAI_API_KEY    - OpenAI API key
//	GOOGLE_API_KEY    - Google API key
//
// Usage:
//
//	GAINS_PROVIDER=anthropic go run ./cmd/aguiserver
//
// Debug logging (shows all events):
//
//	AGUI_LOG_LEVEL=debug GAINS_PROVIDER=anthropic go run ./cmd/aguiserver
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/client"
	"github.com/spetersoncode/gains/model"
	"github.com/spetersoncode/gains/tool"
)

func main() {
	// Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	// Setup structured logger
	logger := setupLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	// Create gains client
	gainsClient, err := createClient(cfg)
	if err != nil {
		slog.Error("failed to create client", "error", err)
		os.Exit(1)
	}

	// Create tool registry
	registry := tool.NewRegistry()
	if cfg.EnableDemoTools {
		SetupDemoTools(registry)
		slog.Info("registered demo tools", "count", registry.Len())
	}

	// Create agent
	a := agent.New(gainsClient, registry)

	// Create workflow registry with demo workflows
	workflowRegistry := SetupDemoWorkflows(gainsClient)
	slog.Info("registered demo workflows", "count", workflowRegistry.Len(), "names", workflowRegistry.Names())

	// Create HTTP handlers
	handler := NewAgentHandler(a, registry, cfg)
	workflowHandler := NewWorkflowHandler(workflowRegistry, cfg)

	// Setup routes
	mux := http.NewServeMux()
	mux.Handle("/api/agent", corsMiddleware(handler))
	mux.Handle("/api/workflow", corsMiddleware(workflowHandler))
	mux.HandleFunc("/health", healthHandler)

	// Create server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // SSE needs no write timeout
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		slog.Info("shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("shutdown error", "error", err)
		}
	}()

	// Start server
	slog.Info("server starting",
		"port", cfg.Port,
		"provider", cfg.Provider,
		"log_level", cfg.LogLevel,
		"agent_endpoint", fmt.Sprintf("POST http://localhost:%s/api/agent", cfg.Port),
		"workflow_endpoint", fmt.Sprintf("POST http://localhost:%s/api/workflow", cfg.Port),
		"health", fmt.Sprintf("GET http://localhost:%s/health", cfg.Port),
	)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

func createClient(cfg *Config) (*client.Client, error) {
	// Determine default model based on provider
	var defaultChat gains.Model
	switch cfg.Provider {
	case "anthropic":
		defaultChat = model.ClaudeSonnet45
	case "openai":
		defaultChat = model.GPT52
	case "google":
		defaultChat = model.Gemini25Flash
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}

	// Note: GAINS_MODEL override is not supported in this reference implementation.
	// The default model for each provider is used. For custom models, modify
	// this function or use the gains library directly.
	_ = cfg.Model // Acknowledge but don't use

	return client.New(client.Config{
		APIKeys: client.APIKeys{
			Anthropic: cfg.AnthropicKey,
			OpenAI:    cfg.OpenAIKey,
			Google:    cfg.GoogleKey,
		},
		Defaults: client.Defaults{
			Chat: defaultChat,
		},
	}), nil
}

// setupLogger creates a structured logger with the specified level.
func setupLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
