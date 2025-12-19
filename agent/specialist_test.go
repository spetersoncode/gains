package agent

import (
	"testing"

	"github.com/spetersoncode/gains/tool"
)

func TestSpecialistRegistry_RegisterAndGet(t *testing.T) {
	registry := NewSpecialistRegistry()

	// Create a mock agent (nil client/registry is fine for testing metadata)
	mockAgent := &Agent{}

	registry.Register("research", "Research and gather information", mockAgent)

	t.Run("Get returns registered specialist", func(t *testing.T) {
		s := registry.Get("research")
		if s == nil {
			t.Fatal("expected specialist, got nil")
		}
		if s.Name != "research" {
			t.Errorf("expected name 'research', got %q", s.Name)
		}
		if s.Description != "Research and gather information" {
			t.Errorf("unexpected description: %q", s.Description)
		}
		if s.Agent != mockAgent {
			t.Error("expected same agent instance")
		}
	})

	t.Run("GetAgent returns agent directly", func(t *testing.T) {
		a := registry.GetAgent("research")
		if a != mockAgent {
			t.Error("expected same agent instance")
		}
	})

	t.Run("Get returns nil for nonexistent", func(t *testing.T) {
		if registry.Get("nonexistent") != nil {
			t.Error("expected nil for nonexistent specialist")
		}
	})

	t.Run("GetAgent returns nil for nonexistent", func(t *testing.T) {
		if registry.GetAgent("nonexistent") != nil {
			t.Error("expected nil for nonexistent specialist")
		}
	})

	t.Run("Has returns true for existing", func(t *testing.T) {
		if !registry.Has("research") {
			t.Error("expected Has('research') to be true")
		}
	})

	t.Run("Has returns false for nonexistent", func(t *testing.T) {
		if registry.Has("nonexistent") {
			t.Error("expected Has('nonexistent') to be false")
		}
	})
}

func TestSpecialistRegistry_Unregister(t *testing.T) {
	registry := NewSpecialistRegistry()
	registry.Register("temp", "Temporary specialist", &Agent{})

	registry.Unregister("temp")

	if registry.Has("temp") {
		t.Error("expected specialist to be unregistered")
	}
}

func TestSpecialistRegistry_NamesAndLen(t *testing.T) {
	registry := NewSpecialistRegistry()
	registry.Register("a", "Agent A", &Agent{})
	registry.Register("b", "Agent B", &Agent{})
	registry.Register("c", "Agent C", &Agent{})

	t.Run("Len returns correct count", func(t *testing.T) {
		if registry.Len() != 3 {
			t.Errorf("expected Len() = 3, got %d", registry.Len())
		}
	})

	t.Run("Names returns all names", func(t *testing.T) {
		names := registry.Names()
		if len(names) != 3 {
			t.Errorf("expected 3 names, got %d", len(names))
		}

		// Check all names are present (order may vary)
		nameSet := make(map[string]bool)
		for _, n := range names {
			nameSet[n] = true
		}
		for _, expected := range []string{"a", "b", "c"} {
			if !nameSet[expected] {
				t.Errorf("expected name %q to be present", expected)
			}
		}
	})

	t.Run("All returns all specialists", func(t *testing.T) {
		specs := registry.All()
		if len(specs) != 3 {
			t.Errorf("expected 3 specialists, got %d", len(specs))
		}
	})
}

func TestSpecialistRegistry_Capabilities(t *testing.T) {
	registry := NewSpecialistRegistry()
	registry.Register("research", "Research", &Agent{},
		WithCapabilities("search", "summarize"))
	registry.Register("code", "Code", &Agent{},
		WithCapabilities("write", "analyze"))
	registry.Register("general", "General", &Agent{},
		WithCapabilities("search", "write"))

	t.Run("ByCapability returns matching specialists", func(t *testing.T) {
		matches := registry.ByCapability("search")
		if len(matches) != 2 {
			t.Errorf("expected 2 matches for 'search', got %d", len(matches))
		}

		names := make(map[string]bool)
		for _, s := range matches {
			names[s.Name] = true
		}
		if !names["research"] || !names["general"] {
			t.Error("expected 'research' and 'general' to match 'search'")
		}
	})

	t.Run("ByCapability returns empty for no matches", func(t *testing.T) {
		matches := registry.ByCapability("nonexistent")
		if len(matches) != 0 {
			t.Errorf("expected 0 matches, got %d", len(matches))
		}
	})
}

func TestSpecialistRegistry_Replace(t *testing.T) {
	registry := NewSpecialistRegistry()
	agent1 := &Agent{}
	agent2 := &Agent{}

	registry.Register("test", "Original", agent1)
	registry.Register("test", "Replaced", agent2)

	s := registry.Get("test")
	if s.Description != "Replaced" {
		t.Error("expected specialist to be replaced")
	}
	if s.Agent != agent2 {
		t.Error("expected agent to be replaced")
	}
}

func TestSpecialistRegistry_AsTools(t *testing.T) {
	registry := NewSpecialistRegistry()
	registry.Register("research", "Research agent", &Agent{})
	registry.Register("code", "Code agent", &Agent{})

	tools := registry.AsTools()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}

	// Check tools have correct names
	names := make(map[string]bool)
	for _, tt := range tools {
		names[tt.Tool.Name] = true
	}
	if !names["research"] || !names["code"] {
		t.Error("expected both 'research' and 'code' tools")
	}
}

func TestSpecialistRegistry_RegisterTo(t *testing.T) {
	specialists := NewSpecialistRegistry()
	specialists.Register("helper", "Helper agent", &Agent{})

	toolRegistry := tool.NewRegistry()
	specialists.RegisterTo(toolRegistry)

	if toolRegistry.Len() != 1 {
		t.Errorf("expected 1 tool, got %d", toolRegistry.Len())
	}

	if _, ok := toolRegistry.GetTool("helper"); !ok {
		t.Error("expected 'helper' tool to be registered")
	}
}

func TestSpecialistRegistry_Chaining(t *testing.T) {
	registry := NewSpecialistRegistry().
		Register("a", "A", &Agent{}).
		Register("b", "B", &Agent{}).
		Register("c", "C", &Agent{})

	if registry.Len() != 3 {
		t.Errorf("expected 3 specialists after chaining, got %d", registry.Len())
	}
}
