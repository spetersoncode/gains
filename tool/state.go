package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spetersoncode/gains/event"
)

// SharedStateTools returns tools for reading and updating shared UI state.
// These tools enable the LLM to interact with frontend state via AG-UI protocol.
//
// Tools provided:
//   - read_state: Read the current shared state
//   - write_state: Replace the entire shared state
//   - update_state: Update specific fields
//
// The tools automatically emit STATE_SNAPSHOT/STATE_DELTA events to sync
// with the frontend. No manual event emission is needed.
//
// Usage:
//
//	// In handler, set up shared state in context
//	sharedState := event.NewSharedState(prepared.State)
//	ctx = event.WithSharedState(ctx, sharedState)
//
//	// Register tools
//	registry.Add(tool.SharedStateTools()...)
func SharedStateTools() []Registration {
	return []Registration{
		readStateTool(),
		writeStateTool(),
		updateStateTool(),
	}
}

// ReadStateArgs are the arguments for the read_state tool.
type ReadStateArgs struct {
	Field string `json:"field,omitempty" desc:"Optional field path to read (e.g. '/counter'). If empty, returns entire state."`
}

func readStateTool() Registration {
	return Func("read_state",
		"Read the current shared UI state. Returns the entire state or a specific field.",
		func(ctx context.Context, args ReadStateArgs) (string, error) {
			state := event.SharedStateFromContext(ctx)
			if state == nil {
				return `{"error": "no shared state available"}`, nil
			}

			var result any
			if args.Field != "" {
				result = state.GetField(args.Field)
			} else {
				result = state.Get()
			}

			data, err := json.Marshal(result)
			if err != nil {
				return "", fmt.Errorf("failed to marshal state: %w", err)
			}
			return string(data), nil
		},
	)
}

// WriteStateArgs are the arguments for the write_state tool.
type WriteStateArgs struct {
	State map[string]any `json:"state" desc:"The new state to set" required:"true"`
}

func writeStateTool() Registration {
	return Func("write_state",
		"Replace the entire shared UI state. This overwrites all existing state and notifies the frontend.",
		func(ctx context.Context, args WriteStateArgs) (string, error) {
			state := event.SharedStateFromContext(ctx)
			if state == nil {
				return `{"error": "no shared state available"}`, nil
			}

			state.Set(ctx, args.State)
			return `{"success": true}`, nil
		},
	)
}

// UpdateStateArgs are the arguments for the update_state tool.
type UpdateStateArgs struct {
	Path  string `json:"path" desc:"JSON Pointer path to update (e.g. '/counter')" required:"true"`
	Value any    `json:"value" desc:"The new value to set" required:"true"`
}

func updateStateTool() Registration {
	return Func("update_state",
		"Update a specific field in the shared UI state. More efficient than write_state for single field changes.",
		func(ctx context.Context, args UpdateStateArgs) (string, error) {
			state := event.SharedStateFromContext(ctx)
			if state == nil {
				return `{"error": "no shared state available"}`, nil
			}

			state.Update(ctx, event.Replace(args.Path, args.Value))
			return fmt.Sprintf(`{"success": true, "path": %q, "value": %v}`, args.Path, args.Value), nil
		},
	)
}
