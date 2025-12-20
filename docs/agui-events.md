# AG-UI Event Sequence Compliance

This document describes how the gains `agui` package implements the AG-UI protocol event sequences, ensuring full compliance with the protocol specification.

## Table of Contents

- [Overview](#overview)
- [Core Pattern: Start-Content-End](#core-pattern-start-content-end)
- [Mandatory Run Lifecycle](#mandatory-run-lifecycle)
- [Event Mappings](#event-mappings)
  - [Run Lifecycle Events](#run-lifecycle-events)
  - [Message Streaming Events](#message-streaming-events)
  - [Tool Call Events](#tool-call-events)
  - [Step Events](#step-events)
  - [State Synchronization Events](#state-synchronization-events)
  - [Activity Events](#activity-events)
  - [Custom Events](#custom-events)
- [Nested Runs](#nested-runs)
- [Usage Examples](#usage-examples)
  - [Basic Agent Integration](#basic-agent-integration)
  - [Workflow Integration](#workflow-integration)
  - [Full SSE Server Example](#full-sse-server-example)
- [Compliance Verification](#compliance-verification)

---

## Overview

The AG-UI (Agent-User Interface) protocol standardizes how AI agents communicate with frontends via Server-Sent Events (SSE). The gains `agui` package provides a `Mapper` that converts gains events to AG-UI events, handling the protocol's stateful requirements automatically.

Key compliance features:
- **1:1 Event Mapping**: Each gains event maps to exactly one AG-UI event
- **Start-Content-End Pattern**: Properly structured streaming sequences
- **Nested Run Handling**: Correct lifecycle events for sub-agents and workflows
- **State Synchronization**: Full support for shared state between agent and frontend

---

## Core Pattern: Start-Content-End

AG-UI uses a three-stage streaming pattern for content delivery:

1. **Start Event** — Initiates a stream with a unique identifier
2. **Content Events** — Deliver incremental data via `delta` fields
3. **End Event** — Signals completion

This pattern applies to:
- **Text Messages**: `TEXT_MESSAGE_START` → `TEXT_MESSAGE_CONTENT`* → `TEXT_MESSAGE_END`
- **Tool Calls**: `TOOL_CALL_START` → `TOOL_CALL_ARGS`* → `TOOL_CALL_END`

The gains event system mirrors this pattern:

| Gains Event | AG-UI Event |
|-------------|-------------|
| `MessageStart` | `TEXT_MESSAGE_START` |
| `MessageDelta` | `TEXT_MESSAGE_CONTENT` |
| `MessageEnd` | `TEXT_MESSAGE_END` |
| `ToolCallStart` | `TOOL_CALL_START` |
| `ToolCallArgs` | `TOOL_CALL_ARGS` |
| `ToolCallEnd` | `TOOL_CALL_END` |
| `ToolCallResult` | `TOOL_CALL_RESULT` |

---

## Mandatory Run Lifecycle

Every agent run **must** emit:
1. `RUN_STARTED` — At the beginning
2. Either `RUN_FINISHED` (success) or `RUN_ERROR` (failure) — At the end

The gains agent package emits `RunStart` and `RunEnd`/`RunError` events automatically, which the mapper converts to the corresponding AG-UI events.

```
RUN_STARTED
  └── (agent work: messages, tool calls, steps)
  └── RUN_FINISHED or RUN_ERROR
```

---

## Event Mappings

### Run Lifecycle Events

| Gains Event | AG-UI Event | When Emitted |
|-------------|-------------|--------------|
| `RunStart` | `RUN_STARTED` | Agent/workflow execution begins (outermost only) |
| `RunEnd` | `RUN_FINISHED` | Agent/workflow completes successfully |
| `RunError` | `RUN_ERROR` | Unrecoverable error (emits at any nesting depth) |

### Message Streaming Events

| Gains Event | AG-UI Event | Fields |
|-------------|-------------|--------|
| `MessageStart` | `TEXT_MESSAGE_START` | `messageId`, `role` |
| `MessageDelta` | `TEXT_MESSAGE_CONTENT` | `messageId`, `delta` |
| `MessageEnd` | `TEXT_MESSAGE_END` | `messageId` |

**Example sequence**:
```
TEXT_MESSAGE_START (messageId="msg-123", role="assistant")
TEXT_MESSAGE_CONTENT (messageId="msg-123", delta="Hello")
TEXT_MESSAGE_CONTENT (messageId="msg-123", delta=" there!")
TEXT_MESSAGE_END (messageId="msg-123")
```

### Tool Call Events

| Gains Event | AG-UI Event | Fields |
|-------------|-------------|--------|
| `ToolCallStart` | `TOOL_CALL_START` | `toolCallId`, `toolCallName` |
| `ToolCallArgs` | `TOOL_CALL_ARGS` | `toolCallId`, `delta` (JSON fragment) |
| `ToolCallEnd` | `TOOL_CALL_END` | `toolCallId` |
| `ToolCallResult` | `TOOL_CALL_RESULT` | `messageId`, `toolCallId`, `result` |

**Example sequence**:
```
TOOL_CALL_START (toolCallId="call-456", toolCallName="get_weather")
TOOL_CALL_ARGS (toolCallId="call-456", delta="{\"location\":")
TOOL_CALL_ARGS (toolCallId="call-456", delta=" \"Paris\"}")
TOOL_CALL_END (toolCallId="call-456")
TOOL_CALL_RESULT (toolCallId="call-456", result="{\"temp\": 22}")
```

### Step Events

| Gains Event | AG-UI Event | Description |
|-------------|-------------|-------------|
| `StepStart` | `STEP_STARTED` | Workflow step or agent iteration begins |
| `StepEnd` | `STEP_FINISHED` | Step completes |
| `StepSkipped` | `STEP_FINISHED` | Step was skipped (treated as finished) |
| `ParallelStart` | `STEP_STARTED` | Parallel block begins |
| `ParallelEnd` | `STEP_FINISHED` | Parallel block completes |

### State Synchronization Events

| Gains Event | AG-UI Event | Description |
|-------------|-------------|-------------|
| `StateSnapshot` | `STATE_SNAPSHOT` | Full state object for frontend |
| `StateDelta` | `STATE_DELTA` | JSON Patch operations (RFC 6902) |
| `MessagesSnapshot` | `MESSAGES_SNAPSHOT` | Full message history |

**State delta example**:
```go
// Emit incremental state updates
mapper.StateDelta(
    event.Replace("/progress", 75),
    event.Add("/items/-", "new item"),
)
```

### Activity Events

Activity events support human-in-the-loop interactions:

| Gains Event | AG-UI Event | Description |
|-------------|-------------|-------------|
| `ActivitySnapshot` | `ACTIVITY_SNAPSHOT` | New activity (e.g., tool approval pending) |
| `ActivityDelta` | `ACTIVITY_DELTA` | Activity state update (e.g., approved) |

### Custom Events

Workflow-specific events map to AG-UI `CUSTOM` events:

| Gains Event | Custom Event Name | Value |
|-------------|-------------------|-------|
| `RouteSelected` | `gains.route_selected` | `{stepName, routeName}` |
| `LoopIteration` | `gains.loop_iteration` | `{stepName, iteration}` |

---

## Nested Runs

Workflows and agents can contain nested runs (e.g., an AgentStep within a workflow, or sub-agents). The mapper handles this via depth tracking:

- Only the **outermost** `RunStart`/`RunEnd` emit AG-UI lifecycle events
- Nested runs are filtered to prevent duplicate `RUN_STARTED`/`RUN_FINISHED`
- `RunError` always bubbles up regardless of nesting depth

```go
// Mapper tracks depth internally
mapper := agui.NewMapper(threadID, runID)

// First RunStart → emits RUN_STARTED (depth=1)
// Nested RunStart → returns nil (depth=2)
// Nested RunEnd → returns nil (depth=1)
// First RunEnd → emits RUN_FINISHED (depth=0)
```

This ensures AG-UI frontends see exactly one run lifecycle per top-level execution, even with complex nested workflows.

---

## Usage Examples

### Basic Agent Integration

```go
package main

import (
    "context"

    ai "github.com/spetersoncode/gains"
    "github.com/spetersoncode/gains/agent"
    "github.com/spetersoncode/gains/agui"
    "github.com/spetersoncode/gains/client"
)

func handleAgentRun(ctx context.Context, c *client.Client, threadID, runID string) {
    // Create mapper for this run
    mapper := agui.NewMapper(threadID, runID)

    // Create agent
    a := agent.New(c, registry)

    // Run with streaming
    events := a.RunStream(ctx, messages)

    // Map and emit events
    for ev := range events {
        if aguiEvent := mapper.MapEvent(ev); aguiEvent != nil {
            // Send to frontend via SSE
            writeSSE(aguiEvent)
        }
    }
}
```

### Workflow Integration

```go
func handleWorkflowRun(ctx context.Context, threadID, runID string) {
    // Create mapper with initial state
    initialState := map[string]any{"progress": 0}
    mapper := agui.NewMapper(threadID, runID, agui.WithInitialState(initialState))

    // Run workflow with streaming
    events := myWorkflow.RunStream(ctx, state)

    // Use MapStream for convenience (auto-filters nil events)
    aguiEvents := mapper.MapStream(events)

    for ev := range aguiEvents {
        writeSSE(ev)
    }
}
```

### Full SSE Server Example

```go
package main

import (
    "context"
    "net/http"

    "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
    "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/server/sse"

    ai "github.com/spetersoncode/gains"
    "github.com/spetersoncode/gains/agent"
    "github.com/spetersoncode/gains/agui"
    "github.com/spetersoncode/gains/client"
)

func main() {
    c := client.New(client.WithAnthropic(os.Getenv("ANTHROPIC_API_KEY")))
    registry := tool.NewRegistry()
    // ... register tools ...

    a := agent.New(c, registry)

    http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
        // Parse AG-UI request
        var input events.RunAgentInput
        json.NewDecoder(r.Body).Decode(&input)

        // Convert input messages to gains format
        messages := agui.ToGainsMessages(input.Messages)

        // Create mapper for this run
        mapper := agui.NewMapper(input.ThreadID, input.RunID)

        // Set up SSE writer
        writer := sse.NewWriter(w)
        writer.WriteEvent(mapper.RunStarted())

        // Run agent with streaming
        ctx := r.Context()
        for ev := range a.RunStream(ctx, messages) {
            if aguiEvent := mapper.MapEvent(ev); aguiEvent != nil {
                writer.WriteEvent(aguiEvent)
            }
        }

        writer.WriteEvent(mapper.RunFinished())
    })

    http.ListenAndServe(":8080", nil)
}
```

---

## Compliance Verification

The implementation passes comprehensive tests verifying AG-UI compliance:

| Requirement | Status | Test Coverage |
|-------------|--------|---------------|
| Mandatory RUN_STARTED/RUN_FINISHED | ✅ | `TestMapper_MapEvent_RunLifecycle` |
| Start-Content-End for messages | ✅ | `TestMapper_MapEvent_MessageLifecycle` |
| Start-Args-End for tool calls | ✅ | `TestMapper_MapEvent_ToolCallLifecycle` |
| Nested run filtering | ✅ | `TestMapper_NestedRuns` |
| Error bubbling | ✅ | `TestMapper_NestedRuns/RunError_emits_regardless_of_depth` |
| State sync events | ✅ | `TestMapper_MapEvent_State` |
| Activity events | ✅ | `TestMapper_ActivityEvents` |
| Custom workflow events | ✅ | `TestMapper_MapEvent_CustomWorkflowEvents` |
| Initial state emission | ✅ | `TestMapper_WithInitialState` |

Run tests with:
```bash
go test ./agui/... -v
```

---

## Event Ordering Guarantees

The gains event system preserves ordering:

1. Events are emitted in causal order (Start before Content before End)
2. Message IDs and Tool Call IDs correlate related events
3. Delta chunks should be concatenated in order received
4. The Mapper maintains internal state to track sequences correctly

For frontends implementing AG-UI:
- Process events in the order received
- Use `messageId` and `toolCallId` to correlate events into logical streams
- Apply state deltas incrementally using JSON Patch operations
