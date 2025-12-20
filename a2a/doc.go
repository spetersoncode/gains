// Package a2a provides utilities for integrating gains with the A2A (Agent-to-Agent) protocol.
//
// A2A is an open protocol enabling communication and interoperability between AI agent
// systems. It uses JSON-RPC 2.0 over HTTP(S) with support for streaming via Server-Sent
// Events (SSE). This package provides type definitions and message conversion utilities
// for building A2A-compatible agents with gains.
//
// # Overview
//
// This package provides:
//   - Core A2A types: [Message], [Task], [TaskState], [Artifact], and Part types
//   - Message conversion: [ToGainsMessages], [FromGainsMessages] for bidirectional conversion
//   - Event mapping: [Mapper] for converting gains events to A2A task updates
//
// The package does NOT provide HTTP handlers or transport implementations. Users are
// responsible for implementing their own JSON-RPC server using their preferred framework.
//
// # Message Conversion
//
// Use [ToGainsMessages] to convert A2A messages to gains messages for processing:
//
//	gainsMessages := a2a.ToGainsMessages(a2aMessages)
//	result := agent.Run(ctx, gainsMessages)
//
// Use [FromGainsMessages] to convert gains messages back to A2A format:
//
//	history := a2a.FromGainsMessages(gainsMessages)
//	task.History = history
//
// # Task Lifecycle
//
// A2A tasks progress through defined states:
//
//   - TaskStateSubmitted: Task received, not yet started
//   - TaskStateWorking: Task is being processed
//   - TaskStateInputRequired: Agent needs additional input
//   - TaskStateCompleted: Task finished successfully
//   - TaskStateFailed: Task failed with an error
//   - TaskStateCanceled: Task was canceled
//   - TaskStateRejected: Task was rejected by the agent
//
// # Event Mapping
//
// Use [Mapper] to convert gains events to A2A task status updates during streaming:
//
//	mapper := a2a.NewMapper(taskID, contextID)
//
//	for event := range agent.RunStream(ctx, messages) {
//	    if update := mapper.MapEvent(event); update != nil {
//	        // Send task status update to client
//	        sendSSE(update)
//	    }
//	}
//
//	// Finalize with completed or failed status
//	finalTask := mapper.Complete(artifacts)
//
// # Protocol Compliance
//
// This package implements types compatible with A2A Protocol version 0.3. For full
// protocol details, see: https://a2a-protocol.org
//
// # Thread Safety
//
// The Mapper is NOT safe for concurrent use. Each goroutine should have its own
// Mapper instance. Message conversion functions are stateless and safe for
// concurrent use.
package a2a
