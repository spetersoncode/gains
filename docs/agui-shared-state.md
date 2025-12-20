# AG-UI Shared State Patterns

This document describes patterns and best practices for bidirectional state synchronization between gains agents/workflows and AG-UI frontends.

## Table of Contents

- [Overview](#overview)
- [State Flow](#state-flow)
- [Reading Frontend State](#reading-frontend-state)
  - [Agent Input State](#agent-input-state)
  - [Workflow Input State](#workflow-input-state)
  - [Type-Safe State Decoding](#type-safe-state-decoding)
- [Emitting State to Frontend](#emitting-state-to-frontend)
  - [State Snapshots](#state-snapshots)
  - [State Deltas](#state-deltas)
  - [From Tool Handlers](#from-tool-handlers)
- [Best Practices](#best-practices)
- [Complete Example](#complete-example)

---

## Overview

AG-UI supports bidirectional state synchronization:

1. **Frontend → Agent**: State sent with each `RunAgentInput` or `RunWorkflowInput`
2. **Agent → Frontend**: State updates via `STATE_SNAPSHOT` and `STATE_DELTA` events

This enables rich UI patterns:
- Progress tracking
- Form state persistence
- Multi-step wizard flows
- Real-time data visualization
- Collaborative editing

---

## State Flow

```
┌─────────────┐                         ┌─────────────┐
│   Frontend  │  ──── RunAgentInput ──► │    Agent    │
│             │       (with state)       │             │
│             │                          │             │
│             │  ◄── STATE_SNAPSHOT ──── │             │
│             │  ◄── STATE_DELTA ─────── │             │
└─────────────┘                         └─────────────┘
```

---

## Reading Frontend State

### Agent Input State

Use `DecodeState` to extract typed state from `RunAgentInput`:

```go
// Define your state struct
type ChatState struct {
    UserID      string   `json:"userId"`
    Preferences struct {
        Language string `json:"language"`
        Theme    string `json:"theme"`
    } `json:"preferences"`
    History []string `json:"history,omitempty"`
}

func handleRun(w http.ResponseWriter, r *http.Request) {
    var input agui.RunAgentInput
    json.NewDecoder(r.Body).Decode(&input)

    prepared, err := input.Prepare()
    if err != nil {
        // Handle error
    }

    // Decode state into typed struct
    state, err := agui.DecodeState[ChatState](prepared)
    if err != nil {
        // Handle decode error
    }

    // Or use MustDecodeState (panics on error)
    state := agui.MustDecodeState[ChatState](prepared)

    // Use state
    fmt.Printf("User %s prefers %s\n", state.UserID, state.Preferences.Language)
}
```

### Workflow Input State

For workflow-specific inputs:

```go
type WorkflowState struct {
    DocumentID string `json:"documentId"`
    Stage      string `json:"stage"`
    Data       any    `json:"data"`
}

func handleWorkflow(w http.ResponseWriter, r *http.Request) {
    var input agui.RunWorkflowInput
    json.NewDecoder(r.Body).Decode(&input)

    prepared, err := input.Prepare()
    if err != nil {
        // Handle error
    }

    state, err := agui.DecodeWorkflowState[WorkflowState](prepared)
    // Or: state := agui.MustDecodeWorkflowState[WorkflowState](prepared)
}
```

### Type-Safe State Decoding

For initializing workflow state structs:

```go
// Create a new state struct initialized from frontend
state, err := agui.InitializeState[MyState](prepared)
// state is *MyState, ready for workflow execution

// Or merge into existing state with defaults
state := &MyState{
    DefaultTimeout: 30,
    Retries:        3,
}
agui.MustMergeState(state, prepared) // Overwrites with frontend values
```

---

## Emitting State to Frontend

### State Snapshots

Send the complete state when you want to replace the frontend's state entirely:

```go
// Via Mapper helper
mapper := agui.NewMapper(threadID, runID)
writeEvent(mapper.StateSnapshot(map[string]any{
    "progress": 50,
    "items":    []string{"step1", "step2"},
    "status":   "processing",
}))

// Via gains events (for integration with RunStream)
events <- event.NewStateSnapshot(map[string]any{
    "progress": 100,
    "status":   "complete",
})
```

### State Deltas

Use JSON Patch (RFC 6902) for incremental updates:

```go
// Via Mapper helper
writeEvent(mapper.StateDelta(
    event.Replace("/progress", 75),
    event.Add("/items/-", "step3"),       // Append to array
    event.Remove("/tempData"),             // Remove a field
))

// Via gains events
events <- event.NewStateDelta(
    event.Replace("/progress", 100),
    event.Add("/results/0", result),
)
```

**Available patch operations:**

| Operation | Description | Example |
|-----------|-------------|---------|
| `Replace` | Replace a value | `event.Replace("/status", "done")` |
| `Add` | Add a new value | `event.Add("/items/-", item)` |
| `Remove` | Remove a value | `event.Remove("/tempField")` |
| `Move` | Move a value | `event.Move("/old", "/new")` |
| `Copy` | Copy a value | `event.Copy("/src", "/dest")` |
| `Test` | Test a value exists | `event.Test("/field", expected)` |

### From Tool Handlers

Emit state updates from within tool handlers:

```go
tool.MustRegisterFunc(registry, "process_item", "Process an item",
    func(ctx context.Context, args ProcessArgs) (string, error) {
        // Get the event channel from context
        eventCh := event.ForwardChannelFromContext(ctx)
        if eventCh != nil {
            // Update progress
            event.EmitField(eventCh, "/progress", 25)

            // Do work...

            // Update again
            event.EmitDelta(eventCh,
                event.Replace("/progress", 50),
                event.Add("/completedItems/-", args.ItemID),
            )
        }

        // Complete the work
        return "processed", nil
    },
)
```

**Convenience functions:**

```go
// Send a full state snapshot
event.EmitSnapshot(eventCh, state)

// Send multiple patch operations
event.EmitDelta(eventCh,
    event.Replace("/field1", value1),
    event.Add("/field2", value2),
)

// Update a single field
event.EmitField(eventCh, "/progress", 75)
```

---

## Best Practices

### 1. Use Snapshots Sparingly

Snapshots replace the entire frontend state. Use them for:
- Initial state after `RUN_STARTED`
- Major state transitions
- Error recovery

Prefer deltas for incremental updates.

### 2. Initial State Pattern

Emit initial state immediately after run starts:

```go
// Using WithInitialState option (recommended)
mapper := agui.NewMapper(threadID, runID,
    agui.WithInitialState(map[string]any{
        "progress": 0,
        "status":   "starting",
        "items":    []string{},
    }),
)

// MapStream will automatically emit STATE_SNAPSHOT after RUN_STARTED
```

### 3. Define State Schema

Document your state structure for frontend developers:

```go
// State schema for the document processor
type ProcessorState struct {
    // Progress percentage (0-100)
    Progress int `json:"progress"`

    // Current processing stage
    Stage string `json:"stage"` // "parsing" | "analyzing" | "generating"

    // Errors encountered (if any)
    Errors []string `json:"errors,omitempty"`

    // Results when complete
    Results *ProcessingResult `json:"results,omitempty"`
}
```

### 4. Batch Related Updates

Combine related updates into a single delta:

```go
// Good: Single event with multiple patches
event.EmitDelta(eventCh,
    event.Replace("/progress", 100),
    event.Replace("/status", "complete"),
    event.Add("/completedAt", time.Now().Format(time.RFC3339)),
)

// Avoid: Multiple separate events
event.EmitField(eventCh, "/progress", 100)
event.EmitField(eventCh, "/status", "complete")  // Unnecessary extra event
```

### 5. Use Typed State

Define typed state structs rather than raw maps:

```go
// Good: Type-safe state
type MyState struct {
    Progress int      `json:"progress"`
    Items    []string `json:"items"`
}

state := agui.MustDecodeState[MyState](prepared)
// state.Progress is int, state.Items is []string

// Avoid: Untyped map access
rawState := prepared.State.(map[string]any)
progress := rawState["progress"].(float64) // Type assertion risks
```

---

## Complete Example

Full AG-UI server with shared state:

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    "time"

    "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/server/sse"

    ai "github.com/spetersoncode/gains"
    "github.com/spetersoncode/gains/agent"
    "github.com/spetersoncode/gains/agui"
    "github.com/spetersoncode/gains/client"
    "github.com/spetersoncode/gains/event"
    "github.com/spetersoncode/gains/tool"
)

// State shared between frontend and agent
type AppState struct {
    Progress    int      `json:"progress"`
    Stage       string   `json:"stage"`
    Items       []string `json:"items"`
    CompletedAt string   `json:"completedAt,omitempty"`
}

func main() {
    c := client.New(client.WithAnthropic(os.Getenv("ANTHROPIC_API_KEY")))
    registry := tool.NewRegistry()

    // Register tools that emit state updates
    tool.MustRegisterFunc(registry, "process", "Process items",
        func(ctx context.Context, args struct {
            Items []string `json:"items" required:"true"`
        }) (string, error) {
            eventCh := event.ForwardChannelFromContext(ctx)

            for i, item := range args.Items {
                // Update progress
                progress := (i + 1) * 100 / len(args.Items)
                if eventCh != nil {
                    event.EmitDelta(eventCh,
                        event.Replace("/progress", progress),
                        event.Replace("/stage", "processing"),
                        event.Add("/items/-", item),
                    )
                }
                time.Sleep(100 * time.Millisecond) // Simulate work
            }

            // Mark complete
            if eventCh != nil {
                event.EmitDelta(eventCh,
                    event.Replace("/stage", "complete"),
                    event.Replace("/completedAt", time.Now().Format(time.RFC3339)),
                )
            }

            return "processed all items", nil
        },
    )

    a := agent.New(c, registry)

    http.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
        var input agui.RunAgentInput
        json.NewDecoder(r.Body).Decode(&input)

        prepared, _ := input.Prepare()

        // Decode frontend state
        frontendState := agui.MustDecodeState[AppState](prepared)

        // Initialize with defaults + frontend overrides
        initialState := AppState{
            Progress: frontendState.Progress,
            Stage:    "starting",
            Items:    frontendState.Items,
        }

        // Create mapper with initial state
        mapper := agui.NewMapper(input.ThreadID, input.RunID,
            agui.WithInitialState(initialState),
        )

        // SSE setup
        writer := sse.NewWriter(w)

        // Run agent
        ctx := event.WithForwardChannel(r.Context(), make(chan event.Event, 100))
        events := a.RunStream(ctx, prepared.Messages)

        // Map and emit events
        for ev := range mapper.MapStream(events) {
            writer.WriteEvent(ev)
        }
    })

    http.ListenAndServe(":8080", nil)
}
```

---

## Related Documentation

- [AG-UI Event Sequences](agui-events.md) - Event mapping and compliance
- [AG-UI HITL Patterns](agui-hitl.md) - Human-in-the-loop interactions
