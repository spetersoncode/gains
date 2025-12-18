package tool

import (
	"context"
	"encoding/json"
	"sync"

	ai "github.com/spetersoncode/gains"
)

// registeredTool combines a tool definition with its handler.
type registeredTool struct {
	tool     ai.Tool
	handler  Handler
	isClient bool // true for client-side tools that have no local handler
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

// RegisterClientTool registers a tool definition without a handler.
// Client tools are executed by the frontend/client, not the backend.
// When the agent encounters a call to a client tool, it emits events
// but does not execute locally.
func (r *Registry) RegisterClientTool(tool ai.Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return &ErrToolAlreadyRegistered{Name: tool.Name}
	}

	r.tools[tool.Name] = registeredTool{
		tool:     tool,
		handler:  nil,
		isClient: true,
	}
	return nil
}

// RegisterClientTools registers multiple client tool definitions.
func (r *Registry) RegisterClientTools(tools []ai.Tool) error {
	for _, t := range tools {
		if err := r.RegisterClientTool(t); err != nil {
			return err
		}
	}
	return nil
}

// IsClientTool returns true if the named tool is a client-side tool.
func (r *Registry) IsClientTool(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rt, ok := r.tools[name]
	return ok && rt.isClient
}

// ClientToolNames returns the names of all registered client tools.
func (r *Registry) ClientToolNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name, rt := range r.tools {
		if rt.isClient {
			names = append(names, name)
		}
	}
	return names
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

// RegisterFunc registers a tool with a typed handler that automatically
// unmarshals the arguments JSON into the specified type T.
//
// Example:
//
//	type SearchArgs struct {
//	    Query string `json:"query" desc:"Search query" required:"true"`
//	}
//
//	tool.RegisterFunc(registry, "search", "Search the web",
//	    func(ctx context.Context, args SearchArgs) (string, error) {
//	        return doSearch(args.Query), nil
//	    },
//	)
func RegisterFunc[T any](r *Registry, name, description string, fn TypedHandler[T]) error {
	schema, err := SchemaFor[T]()
	if err != nil {
		return err
	}

	t := ai.Tool{
		Name:        name,
		Description: description,
		Parameters:  schema,
	}

	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args T
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", err
		}
		return fn(ctx, args)
	}

	return r.Register(t, handler)
}

// MustRegisterFunc is like RegisterFunc but panics on error.
func MustRegisterFunc[T any](r *Registry, name, description string, fn TypedHandler[T]) {
	if err := RegisterFunc(r, name, description, fn); err != nil {
		panic(err)
	}
}

// Execute runs the handler for a tool call and returns a ToolResult.
// If the tool is not found, returns ErrToolNotFound.
// If the tool is a client-side tool, returns ErrClientTool.
// If the handler returns an error, the error is captured in ToolResult.IsError
// and the error message is returned as the content (allowing the model to recover).
func (r *Registry) Execute(ctx context.Context, call ai.ToolCall) (ai.ToolResult, error) {
	r.mu.RLock()
	rt, ok := r.tools[call.Name]
	r.mu.RUnlock()

	if !ok {
		return ai.ToolResult{}, &ErrToolNotFound{Name: call.Name}
	}

	if rt.isClient {
		return ai.ToolResult{}, &ErrClientTool{Name: call.Name}
	}

	content, err := rt.handler(ctx, call)
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

// Registration holds a tool and its handler for fluent registration.
type Registration struct {
	Tool    ai.Tool
	Handler Handler
}

// Func creates a Registration with automatic schema generation from the typed handler.
// Panics if schema generation fails.
//
// Example:
//
//	registry := tool.NewRegistry().Add(
//	    tool.Func("weather", "Get weather", func(ctx context.Context, args WeatherArgs) (string, error) {
//	        return getWeather(args.Location), nil
//	    }),
//	    tool.Func("search", "Search web", searchHandler),
//	)
func Func[T any](name, description string, fn TypedHandler[T]) Registration {
	schema := MustSchemaFor[T]()
	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		var args T
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return "", err
		}
		return fn(ctx, args)
	}
	return Registration{
		Tool: ai.Tool{
			Name:        name,
			Description: description,
			Parameters:  schema,
		},
		Handler: handler,
	}
}

// WithHandler creates a Registration from a Handler and schema.
// Use this when you have a pre-built Handler implementation.
func WithHandler(name, description string, schema json.RawMessage, h Handler) Registration {
	return Registration{
		Tool: ai.Tool{
			Name:        name,
			Description: description,
			Parameters:  schema,
		},
		Handler: h,
	}
}

// WithTool creates a Registration from an existing Tool and Handler.
// Use this when you have pre-built tool definitions.
func WithTool(t ai.Tool, h Handler) Registration {
	return Registration{
		Tool:    t,
		Handler: h,
	}
}

// Add registers one or more tools to the registry.
// Panics if any tool is already registered.
// Returns the registry for fluent chaining.
//
// Example:
//
//	registry := tool.NewRegistry().Add(
//	    tool.Func("weather", "Get weather", weatherFn),
//	    tool.Func("search", "Search web", searchFn),
//	    tool.Func("calc", "Calculate", calcFn),
//	)
func (r *Registry) Add(regs ...Registration) *Registry {
	for _, reg := range regs {
		r.MustRegister(reg.Tool, reg.Handler)
	}
	return r
}
