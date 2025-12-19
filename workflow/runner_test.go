package workflow

import (
	"context"
	"testing"

	"github.com/spetersoncode/gains/event"
)

type testRunnerState struct {
	Query  string   `json:"query"`
	Result string   `json:"result"`
	Items  []string `json:"items"`
}

func TestNewRunnerJSON(t *testing.T) {
	// Create a simple workflow step
	step := NewFuncStep("process", func(ctx context.Context, state *testRunnerState) error {
		state.Result = "processed: " + state.Query
		return nil
	})

	runner := NewRunnerJSON[testRunnerState]("test-workflow", step)

	t.Run("name returns workflow name", func(t *testing.T) {
		if runner.Name() != "test-workflow" {
			t.Errorf("expected 'test-workflow', got %q", runner.Name())
		}
	})

	t.Run("RunStream with map input", func(t *testing.T) {
		input := map[string]any{
			"query": "hello world",
		}

		events := collectEvents(runner.RunStream(context.Background(), input))

		// Check we got run start and end
		hasStart := false
		hasEnd := false
		for _, ev := range events {
			if ev.Type == event.RunStart {
				hasStart = true
			}
			if ev.Type == event.RunEnd {
				hasEnd = true
			}
		}

		if !hasStart {
			t.Error("expected RunStart event")
		}
		if !hasEnd {
			t.Error("expected RunEnd event")
		}
	})

	t.Run("RunStream with nil input", func(t *testing.T) {
		events := collectEvents(runner.RunStream(context.Background(), nil))

		// Should complete without error
		for _, ev := range events {
			if ev.Type == event.RunError {
				t.Errorf("unexpected error: %v", ev.Error)
			}
		}
	})

	t.Run("RunStream with typed input", func(t *testing.T) {
		input := &testRunnerState{
			Query: "direct input",
		}

		events := collectEvents(runner.RunStream(context.Background(), input))

		for _, ev := range events {
			if ev.Type == event.RunError {
				t.Errorf("unexpected error: %v", ev.Error)
			}
		}
	})
}

func TestNewRunner_CustomFactory(t *testing.T) {
	step := NewFuncStep("process", func(ctx context.Context, state *testRunnerState) error {
		state.Result = "custom: " + state.Query
		return nil
	})

	runner := NewRunner("custom", step, func(input any) (*testRunnerState, error) {
		state := &testRunnerState{
			Query: "default query",
		}
		if m, ok := input.(map[string]any); ok {
			if q, ok := m["q"].(string); ok {
				state.Query = q // Use "q" instead of "query"
			}
		}
		return state, nil
	})

	input := map[string]any{"q": "short key"}
	events := collectEvents(runner.RunStream(context.Background(), input))

	for _, ev := range events {
		if ev.Type == event.RunError {
			t.Errorf("unexpected error: %v", ev.Error)
		}
	}
}

func TestRegistry(t *testing.T) {
	t.Run("Register and Get", func(t *testing.T) {
		reg := NewRegistry()
		step := NewFuncStep("noop", func(ctx context.Context, state *testRunnerState) error {
			return nil
		})
		runner := NewRunnerJSON[testRunnerState]("workflow-a", step)

		reg.Register(runner)

		if !reg.Has("workflow-a") {
			t.Error("expected Has('workflow-a') to be true")
		}
		if reg.Get("workflow-a") != runner {
			t.Error("expected Get to return the registered runner")
		}
	})

	t.Run("Get nonexistent", func(t *testing.T) {
		reg := NewRegistry()

		if reg.Get("nonexistent") != nil {
			t.Error("expected nil for nonexistent workflow")
		}
		if reg.Has("nonexistent") {
			t.Error("expected Has('nonexistent') to be false")
		}
	})

	t.Run("Unregister", func(t *testing.T) {
		reg := NewRegistry()
		step := NewFuncStep("noop", func(ctx context.Context, state *testRunnerState) error {
			return nil
		})
		runner := NewRunnerJSON[testRunnerState]("workflow-b", step)

		reg.Register(runner)
		reg.Unregister("workflow-b")

		if reg.Has("workflow-b") {
			t.Error("expected Has('workflow-b') to be false after unregister")
		}
	})

	t.Run("Names and Len", func(t *testing.T) {
		reg := NewRegistry()

		step1 := NewFuncStep("noop", func(ctx context.Context, state *testRunnerState) error {
			return nil
		})
		step2 := NewFuncStep("noop", func(ctx context.Context, state *testRunnerState) error {
			return nil
		})

		reg.Register(NewRunnerJSON[testRunnerState]("wf-1", step1))
		reg.Register(NewRunnerJSON[testRunnerState]("wf-2", step2))

		if reg.Len() != 2 {
			t.Errorf("expected Len() = 2, got %d", reg.Len())
		}

		names := reg.Names()
		if len(names) != 2 {
			t.Errorf("expected 2 names, got %d", len(names))
		}
	})

	t.Run("RunStream with registered workflow", func(t *testing.T) {
		reg := NewRegistry()
		step := NewFuncStep("process", func(ctx context.Context, state *testRunnerState) error {
			state.Result = "registry run"
			return nil
		})
		reg.Register(NewRunnerJSON[testRunnerState]("my-workflow", step))

		events := collectEvents(reg.RunStream(context.Background(), "my-workflow", nil))

		hasEnd := false
		for _, ev := range events {
			if ev.Type == event.RunEnd {
				hasEnd = true
			}
			if ev.Type == event.RunError {
				t.Errorf("unexpected error: %v", ev.Error)
			}
		}

		if !hasEnd {
			t.Error("expected RunEnd event")
		}
	})

	t.Run("RunStream with nonexistent workflow", func(t *testing.T) {
		reg := NewRegistry()

		events := collectEvents(reg.RunStream(context.Background(), "nonexistent", nil))

		hasError := false
		for _, ev := range events {
			if ev.Type == event.RunError {
				hasError = true
			}
		}

		if !hasError {
			t.Error("expected RunError event for nonexistent workflow")
		}
	})
}

func collectEvents(ch <-chan event.Event) []event.Event {
	var events []event.Event
	for ev := range ch {
		events = append(events, ev)
	}
	return events
}
