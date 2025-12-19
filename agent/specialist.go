package agent

import (
	"sync"

	"github.com/spetersoncode/gains/tool"
)

// Specialist represents a specialized agent with metadata.
type Specialist struct {
	Name        string   // Unique identifier
	Description string   // Description for tool registration
	Agent       *Agent   // The underlying agent
	Capabilities []string // Optional: capabilities this specialist provides
}

// SpecialistOption configures a Specialist.
type SpecialistOption func(*Specialist)

// WithCapabilities sets capabilities for the specialist.
// Capabilities are used for capability-based routing.
func WithCapabilities(caps ...string) SpecialistOption {
	return func(s *Specialist) {
		s.Capabilities = caps
	}
}

// SpecialistRegistry manages a collection of specialized agents.
// It provides named lookup and easy tool registration for sub-agent patterns.
//
// Example:
//
//	registry := agent.NewSpecialistRegistry()
//	registry.Register("research", "Research and gather information", researchAgent)
//	registry.Register("code", "Write and analyze code", codeAgent)
//
//	// Add all specialists as tools to a tool registry
//	toolRegistry := tool.NewRegistry()
//	for _, t := range registry.AsTools() {
//	    toolRegistry.Add(t)
//	}
type SpecialistRegistry struct {
	mu          sync.RWMutex
	specialists map[string]*Specialist
}

// NewSpecialistRegistry creates an empty specialist registry.
func NewSpecialistRegistry() *SpecialistRegistry {
	return &SpecialistRegistry{
		specialists: make(map[string]*Specialist),
	}
}

// Register adds a specialist agent to the registry.
// If a specialist with the same name already exists, it is replaced.
func (r *SpecialistRegistry) Register(name, description string, agent *Agent, opts ...SpecialistOption) *SpecialistRegistry {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := &Specialist{
		Name:        name,
		Description: description,
		Agent:       agent,
	}

	for _, opt := range opts {
		opt(s)
	}

	r.specialists[name] = s
	return r
}

// Unregister removes a specialist from the registry.
// It is a no-op if the specialist does not exist.
func (r *SpecialistRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.specialists, name)
}

// Get retrieves a specialist by name.
// Returns nil if not found.
func (r *SpecialistRegistry) Get(name string) *Specialist {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.specialists[name]
}

// GetAgent retrieves an agent by specialist name.
// Returns nil if not found.
func (r *SpecialistRegistry) GetAgent(name string) *Agent {
	s := r.Get(name)
	if s == nil {
		return nil
	}
	return s.Agent
}

// Has returns true if a specialist with the given name exists.
func (r *SpecialistRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.specialists[name]
	return ok
}

// Names returns all registered specialist names.
func (r *SpecialistRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.specialists))
	for name := range r.specialists {
		names = append(names, name)
	}
	return names
}

// All returns all registered specialists.
func (r *SpecialistRegistry) All() []*Specialist {
	r.mu.RLock()
	defer r.mu.RUnlock()

	specs := make([]*Specialist, 0, len(r.specialists))
	for _, s := range r.specialists {
		specs = append(specs, s)
	}
	return specs
}

// Len returns the number of registered specialists.
func (r *SpecialistRegistry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.specialists)
}

// ByCapability returns all specialists that have the given capability.
func (r *SpecialistRegistry) ByCapability(capability string) []*Specialist {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*Specialist
	for _, s := range r.specialists {
		for _, cap := range s.Capabilities {
			if cap == capability {
				matches = append(matches, s)
				break
			}
		}
	}
	return matches
}

// AsTools converts all specialists to tool registrations.
// Use this to add all specialists as callable tools to a tool.Registry.
//
// Example:
//
//	toolRegistry := tool.NewRegistry()
//	for _, t := range specialists.AsTools() {
//	    toolRegistry.Add(t)
//	}
func (r *SpecialistRegistry) AsTools(opts ...ToolOption) []tool.Registration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]tool.Registration, 0, len(r.specialists))
	for _, s := range r.specialists {
		toolOpts := append([]ToolOption{WithToolDescription(s.Description)}, opts...)
		tools = append(tools, NewTool(s.Name, s.Agent, toolOpts...))
	}
	return tools
}

// AsToolsWith converts specialists to tools with per-specialist options.
// The optsFunc receives the specialist and returns tool options for it.
func (r *SpecialistRegistry) AsToolsWith(optsFunc func(*Specialist) []ToolOption) []tool.Registration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]tool.Registration, 0, len(r.specialists))
	for _, s := range r.specialists {
		opts := optsFunc(s)
		opts = append([]ToolOption{WithToolDescription(s.Description)}, opts...)
		tools = append(tools, NewTool(s.Name, s.Agent, opts...))
	}
	return tools
}

// RegisterTo adds all specialists as tools to the given tool registry.
// This is a convenience method combining AsTools and registry.Add.
func (r *SpecialistRegistry) RegisterTo(toolRegistry *tool.Registry, opts ...ToolOption) {
	for _, t := range r.AsTools(opts...) {
		toolRegistry.Add(t)
	}
}
