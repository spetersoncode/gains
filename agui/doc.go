// Package agui provides utilities for integrating gains with the AG-UI protocol.
//
// AG-UI (Agent-User Interface) is an open, lightweight, event-based protocol that
// standardizes how AI agents connect to user-facing applications. This package
// provides mapping utilities to convert gains events to AG-UI events, enabling
// easy integration with AG-UI-compatible frontends.
//
// # Overview
//
// This package provides:
//   - [Mapper]: Stateful event converter that handles AG-UI's Start-Content-End pattern
//   - Message conversion utilities: [ToGainsMessages], [FromGainsMessages]
//
// The package does NOT provide HTTP handlers or transport implementations. Users
// are responsible for implementing their own server using the AG-UI SDK's SSE
// writer or their preferred transport mechanism.
//
// # Usage
//
// Create a Mapper for each run and use it to convert gains events:
//
//	// Create mapper for this run
//	mapper := agui.NewMapper(threadID, runID)
//
//	// Emit run started
//	writeEvent(mapper.RunStarted())
//
//	// Run agent and map events
//	for event := range myAgent.RunStream(ctx, messages) {
//	    for _, aguiEvent := range mapper.MapAgentEvent(event) {
//	        writeEvent(aguiEvent)
//	    }
//	}
//
//	// Emit run finished
//	writeEvent(mapper.RunFinished())
//
// # Event Mapping
//
// The Mapper tracks state to properly emit AG-UI's Start-Content-End sequences:
//
//   - gains EventStreamDelta → TEXT_MESSAGE_START (on first delta), TEXT_MESSAGE_CONTENT
//   - gains EventStepComplete → TEXT_MESSAGE_END (if message active), STEP_FINISHED
//   - gains EventToolCallRequested → TOOL_CALL_START, TOOL_CALL_ARGS
//   - gains EventToolResult → TOOL_CALL_END, TOOL_CALL_RESULT
//
// # Message Conversion
//
// Use [ToGainsMessages] to convert AG-UI messages to gains messages for input:
//
//	messages := agui.ToGainsMessages(aguiMessages)
//	result := agent.Run(ctx, messages)
//
// Use [FromGainsMessages] to convert gains messages to AG-UI format for snapshots:
//
//	snapshot := events.NewMessagesSnapshotEvent(agui.FromGainsMessages(history))
//
// # Thread Safety
//
// The Mapper is NOT safe for concurrent use. Each goroutine should have its own
// Mapper instance. Message conversion functions are stateless and safe for
// concurrent use.
package agui
