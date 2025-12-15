package agent

import (
	"context"
	"encoding/json"
	"sync"

	ai "github.com/spetersoncode/gains"
)

// Handler is a function that executes a tool call and returns a result.
// The context supports cancellation and timeout.
// The call contains the tool name, ID, and arguments as a JSON string.
// Returns the result content string, or an error if execution failed.
type Handler func(ctx context.Context, call ai.ToolCall) (string, error)

// registeredTool combines a tool definition with its handler.
type registeredTool struct {
	tool    ai.Tool
	handler Handler
}

// Registry manages registered tools and their handlers.
// It is safe for concurrent use.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]registeredTool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]registeredTool),
	}
}

// Register adds a tool with its handler to the registry.
// Returns an error if a tool with the same name is already registered.
func (r *Registry) Register(tool ai.Tool, handler Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return &ErrToolAlreadyRegistered{Name: tool.Name}
	}

	r.tools[tool.Name] = registeredTool{
		tool:    tool,
		handler: handler,
	}
	return nil
}

// MustRegister is like Register but panics on error.
func (r *Registry) MustRegister(tool ai.Tool, handler Handler) {
	if err := r.Register(tool, handler); err != nil {
		panic(err)
	}
}

// Unregister removes a tool from the registry.
// It is a no-op if the tool is not registered.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// Get retrieves a handler by tool name.
// Returns the handler and true if found, or nil and false if not found.
func (r *Registry) Get(name string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rt, ok := r.tools[name]
	if !ok {
		return nil, false
	}
	return rt.handler, true
}

// GetTool retrieves a tool definition by name.
// Returns the tool and true if found, or empty tool and false if not found.
func (r *Registry) GetTool(name string) (ai.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rt, ok := r.tools[name]
	if !ok {
		return ai.Tool{}, false
	}
	return rt.tool, true
}

// Tools returns all registered tool definitions.
// This is used to pass the tools to the ChatProvider.
func (r *Registry) Tools() []ai.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]ai.Tool, 0, len(r.tools))
	for _, rt := range r.tools {
		tools = append(tools, rt.tool)
	}
	return tools
}

// Names returns the names of all registered tools.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Len returns the number of registered tools.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// TypedHandler is a function that executes a tool call with typed arguments.
type TypedHandler[T any] func(ctx context.Context, args T) (string, error)

// RegisterFunc registers a tool with a typed handler that automatically
// unmarshals the arguments JSON into the specified type T.
// This provides a cleaner API compared to manually unmarshaling in each handler.
func RegisterFunc[T any](r *Registry, name, description string, params json.RawMessage, fn TypedHandler[T]) error {
	tool := ai.Tool{
		Name:        name,
		Description: description,
		Parameters:  params,
	}

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args T
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", err
		}
		return fn(ctx, args)
	}

	return r.Register(tool, handler)
}

// MustRegisterFunc is like RegisterFunc but panics on error.
func MustRegisterFunc[T any](r *Registry, name, description string, params json.RawMessage, fn TypedHandler[T]) {
	if err := RegisterFunc(r, name, description, params, fn); err != nil {
		panic(err)
	}
}

// Execute runs the handler for a tool call and returns a ToolResult.
// If the tool is not found, returns ErrToolNotFound.
// If the handler returns an error, the error is captured in ToolResult.IsError
// and the error message is returned as the content (allowing the model to recover).
func (r *Registry) Execute(ctx context.Context, call ai.ToolCall) (ai.ToolResult, error) {
	handler, ok := r.Get(call.Name)
	if !ok {
		return ai.ToolResult{}, &ErrToolNotFound{Name: call.Name}
	}

	content, err := handler(ctx, call)
	if err != nil {
		// Return error as tool result so model can potentially recover
		return ai.ToolResult{
			ToolCallID: call.ID,
			Content:    err.Error(),
			IsError:    true,
		}, nil
	}

	return ai.ToolResult{
		ToolCallID: call.ID,
		Content:    content,
		IsError:    false,
	}, nil
}
