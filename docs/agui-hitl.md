# AG-UI Human-in-the-Loop Patterns

This document describes patterns for implementing human-in-the-loop (HITL) interactions with AG-UI frontends, including tool approval workflows and user input dialogs.

## Table of Contents

- [Overview](#overview)
- [Tool Approval](#tool-approval)
  - [ApprovalBroker](#approvalbroker)
  - [Selective Approval](#selective-approval)
  - [Activity Events](#activity-events)
  - [Frontend Integration](#frontend-integration)
- [User Input](#user-input)
  - [UserInputBroker](#userinputbroker)
  - [Input Types](#input-types)
  - [Activity Events for Input](#activity-events-for-input)
- [Complete Server Example](#complete-server-example)
- [Best Practices](#best-practices)

---

## Overview

Human-in-the-loop patterns allow agents to pause and wait for user decisions:

1. **Tool Approval**: User approves/rejects tool calls before execution
2. **User Input**: Agent requests text input, confirmations, or choices from user

Both patterns use AG-UI's Activity events for frontend communication:
- `ACTIVITY_SNAPSHOT`: Initial state (pending approval, awaiting input)
- `ACTIVITY_DELTA`: State updates (approved, rejected, responded)

---

## Tool Approval

### ApprovalBroker

The `ApprovalBroker` manages async approval workflows:

```go
import (
    "github.com/spetersoncode/gains/agent"
    "github.com/spetersoncode/gains/agui"
)

// Create the broker
broker := agent.NewApprovalBroker()

// Or with options
broker := agent.NewApprovalBrokerWith(
    agent.WithApprovalTimeout(5*time.Minute),
    agent.WithOnSubmit(func(call ai.ToolCall) {
        log.Printf("Approval requested for %s", call.Name)
    }),
)

// Use with agent
a := agent.New(client, registry)
result, err := a.Run(ctx, messages,
    agent.WithApprover(broker.Approver()),
)
```

**Sending decisions from frontend:**

```go
// Handle approval endpoint
http.HandleFunc("/api/approve", func(w http.ResponseWriter, r *http.Request) {
    // Parse AG-UI approval input
    input, err := agui.ParseApprovalInput(body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Send decision to broker
    if err := agui.HandleApproval(broker, input); err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    w.WriteHeader(http.StatusOK)
})

// Or parse JSON directly
err := agui.HandleApprovalJSON(broker, requestBody)
```

**Convenience methods:**

```go
// Approve a specific tool call
broker.Approve("call-123")

// Reject with reason
broker.Reject("call-123", "dangerous operation")

// Check pending approvals
if broker.HasPending() {
    count := broker.PendingCount()
}
```

### Selective Approval

Require approval only for specific tools:

```go
a.Run(ctx, messages,
    agent.WithApprover(broker.Approver()),
    agent.WithApprovalRequired("write_file", "delete_file", "execute_command"),
)
```

Or implement custom logic:

```go
approver := func(ctx context.Context, call ai.ToolCall) (bool, string) {
    // Auto-approve read operations
    if strings.HasPrefix(call.Name, "read_") {
        return true, ""
    }

    // Require approval for writes
    return broker.waitForDecision(ctx, call)
}

a.Run(ctx, messages,
    agent.WithApprover(approver),
)
```

### Activity Events

Approval requests emit Activity events for the frontend:

```go
// Agent emits when approval is needed:
event.NewToolApprovalPending(toolCallID, toolName, arguments)
// Maps to: ACTIVITY_SNAPSHOT (activityType: "tool_approval", status: "pending")

// Agent emits when approved:
event.NewToolApprovalApproved(toolCallID)
// Maps to: ACTIVITY_DELTA (patch: /status -> "approved")

// Agent emits when rejected:
event.NewToolApprovalRejected(toolCallID, reason)
// Maps to: ACTIVITY_DELTA (patches: /status -> "rejected", /reason -> reason)
```

**Activity content structure:**

```json
{
  "toolCallId": "call-123",
  "toolName": "write_file",
  "arguments": "{\"path\": \"/tmp/file.txt\", \"content\": \"...\"}",
  "status": "pending",
  "reason": ""
}
```

### Frontend Integration

Example approval UI flow:

1. Frontend receives `ACTIVITY_SNAPSHOT` with `tool_approval` type
2. UI displays approval dialog with tool name and arguments
3. User clicks Approve or Reject
4. Frontend sends POST to `/api/approve`:

```json
{
  "toolCallId": "call-123",
  "approved": true
}
```

Or for rejection:

```json
{
  "toolCallId": "call-123",
  "approved": false,
  "reason": "User rejected: seems risky"
}
```

5. Agent receives decision and continues (or handles rejection)
6. Frontend receives `ACTIVITY_DELTA` confirming the decision

---

## User Input

### UserInputBroker

Request arbitrary input from users during agent execution:

```go
// Create broker (similar pattern to ApprovalBroker)
inputBroker := agent.NewUserInputBroker()

// Use with agent
a := agent.New(client, registry,
    agent.WithUserInputBroker(inputBroker),
)
```

**Requesting input from tools:**

```go
tool.MustRegisterFunc(registry, "ask_user", "Ask user a question",
    func(ctx context.Context, args struct {
        Question string `json:"question" required:"true"`
    }) (string, error) {
        broker := agent.UserInputBrokerFromContext(ctx)
        if broker == nil {
            return "", errors.New("no user input broker available")
        }

        response, err := broker.RequestText(ctx, args.Question,
            agent.WithInputTitle("Agent Question"),
            agent.WithInputPlaceholder("Type your answer..."),
        )
        if err != nil {
            return "", err
        }

        return response.Value, nil
    },
)
```

**Handling responses from frontend:**

```go
http.HandleFunc("/api/input", func(w http.ResponseWriter, r *http.Request) {
    err := agui.HandleUserInputJSON(inputBroker, body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    w.WriteHeader(http.StatusOK)
})
```

### Input Types

**Text Input:**

```go
response, err := broker.RequestText(ctx, "What is your name?",
    agent.WithInputTitle("Name Required"),
    agent.WithInputDefault("Anonymous"),
    agent.WithInputPlaceholder("Enter your name..."),
)
// response.Value contains the user's input
```

**Confirmation Dialog:**

```go
response, err := broker.RequestConfirm(ctx, "Are you sure you want to proceed?",
    agent.WithInputTitle("Confirm Action"),
)
// response.Confirmed is true or false
```

**Choice Selection:**

```go
response, err := broker.RequestChoice(ctx, "Select your preferred format:",
    []string{"JSON", "YAML", "XML"},
    agent.WithInputTitle("Format Selection"),
    agent.WithInputDefault("JSON"),
)
// response.Value contains the selected choice
```

### Activity Events for Input

Input requests emit Activity events:

```go
// Pending input request
event.NewUserInputPending(requestID, inputType, title, message, choices, defaultVal, placeholder)
// Maps to: ACTIVITY_SNAPSHOT (activityType: "user_input")

// User responded
event.NewUserInputResponded(requestID, value, confirmed)
// Maps to: ACTIVITY_DELTA (status: "responded")

// User cancelled
event.NewUserInputCancelled(requestID)
// Maps to: ACTIVITY_DELTA (status: "cancelled")

// Request timed out
event.NewUserInputTimeout(requestID)
// Maps to: ACTIVITY_DELTA (status: "timeout")
```

**Activity content structure:**

```json
{
  "requestId": "req-456",
  "type": "text",
  "title": "Agent Question",
  "message": "What is your name?",
  "choices": null,
  "default": "",
  "placeholder": "Type your answer...",
  "status": "pending",
  "value": "",
  "confirmed": false
}
```

---

## Complete Server Example

Full AG-UI server with HITL capabilities:

```go
package main

import (
    "context"
    "encoding/json"
    "io"
    "net/http"
    "os"
    "time"

    "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/server/sse"

    ai "github.com/spetersoncode/gains"
    "github.com/spetersoncode/gains/agent"
    "github.com/spetersoncode/gains/agui"
    "github.com/spetersoncode/gains/client"
    "github.com/spetersoncode/gains/event"
    "github.com/spetersoncode/gains/tool"
)

var (
    // Global brokers for this example (use per-session in production)
    approvalBroker = agent.NewApprovalBroker()
)

func main() {
    c := client.New(client.WithAnthropic(os.Getenv("ANTHROPIC_API_KEY")))
    registry := tool.NewRegistry()

    // Register tools
    tool.MustRegisterFunc(registry, "write_file", "Write content to a file",
        func(ctx context.Context, args struct {
            Path    string `json:"path" required:"true"`
            Content string `json:"content" required:"true"`
        }) (string, error) {
            // In production: actually write the file
            return "File written successfully", nil
        },
    )

    tool.MustRegisterFunc(registry, "read_file", "Read a file",
        func(ctx context.Context, args struct {
            Path string `json:"path" required:"true"`
        }) (string, error) {
            content, err := os.ReadFile(args.Path)
            if err != nil {
                return "", err
            }
            return string(content), nil
        },
    )

    a := agent.New(c, registry)

    // Main run endpoint
    http.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
        var input agui.RunAgentInput
        json.NewDecoder(r.Body).Decode(&input)

        prepared, _ := input.Prepare()
        mapper := agui.NewMapper(input.ThreadID, input.RunID)

        writer := sse.NewWriter(w)
        writer.WriteEvent(mapper.RunStarted())

        // Enable event forwarding for activity events
        eventCh := make(chan event.Event, 100)
        ctx := event.WithForwardChannel(r.Context(), eventCh)

        // Run agent with approval
        events := a.RunStream(ctx, prepared.Messages,
            agent.WithApprover(approvalBroker.Approver()),
            agent.WithApprovalRequired("write_file", "delete_file"),
        )

        // Merge agent events with forwarded events
        go func() {
            for ev := range events {
                eventCh <- ev
            }
            close(eventCh)
        }()

        // Process all events
        for ev := range eventCh {
            if aguiEvent := mapper.MapEvent(ev); aguiEvent != nil {
                writer.WriteEvent(aguiEvent)
            }
        }

        writer.WriteEvent(mapper.RunFinished())
    })

    // Approval endpoint
    http.HandleFunc("/api/approve", func(w http.ResponseWriter, r *http.Request) {
        body, _ := io.ReadAll(r.Body)
        if err := agui.HandleApprovalJSON(approvalBroker, body); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        w.WriteHeader(http.StatusOK)
    })

    http.ListenAndServe(":8080", nil)
}
```

---

## Best Practices

### 1. Use Per-Session Brokers

Create brokers per session, not globally:

```go
// Good: Per-session broker
sessions := make(map[string]*SessionState)

type SessionState struct {
    ApprovalBroker *agent.ApprovalBroker
    InputBroker    *agent.UserInputBroker
}

func getSession(sessionID string) *SessionState {
    if s, ok := sessions[sessionID]; ok {
        return s
    }
    s := &SessionState{
        ApprovalBroker: agent.NewApprovalBroker(),
        InputBroker:    agent.NewUserInputBroker(),
    }
    sessions[sessionID] = s
    return s
}
```

### 2. Set Reasonable Timeouts

Don't wait forever for user input:

```go
broker := agent.NewApprovalBrokerWith(
    agent.WithApprovalTimeout(2*time.Minute), // Don't block too long
)
```

### 3. Provide Context in Approval Requests

Give users enough information to make decisions:

```go
// The agent will include tool name and arguments in the activity event
// Frontend should display this prominently
```

### 4. Handle Timeouts Gracefully

Timeouts are normal, not errors:

```go
response, err := broker.RequestConfirm(ctx, "Continue?")
if err == context.DeadlineExceeded {
    // User didn't respond - use default behavior
    return handleDefaultCase()
}
```

### 5. Track Activity State

Use activity IDs for correlation:

```go
// The tool call ID serves as the activity ID for approvals
// This allows the frontend to match responses to requests
```

### 6. Consider UX in Tool Design

Design tools with approval UX in mind:

```go
// Good: Separate read and write tools
tool.RegisterFunc(registry, "read_config", ...)  // No approval needed
tool.RegisterFunc(registry, "write_config", ...) // Requires approval

// Avoid: Single tool that sometimes writes
tool.RegisterFunc(registry, "config", ...) // Hard to know when to approve
```

---

## Related Documentation

- [AG-UI Event Sequences](agui-events.md) - Event mapping and compliance
- [AG-UI Shared State](agui-shared-state.md) - State synchronization patterns
