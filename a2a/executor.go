package a2a

import (
	"context"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/event"
)

// Executor handles A2A task execution.
// Implementations convert A2A messages to gains format, execute the
// underlying agent or workflow, and convert results back to A2A format.
type Executor interface {
	// Execute runs a task synchronously and returns the final task.
	Execute(ctx context.Context, req SendMessageRequest) (*Task, error)

	// ExecuteStream runs a task and streams status updates.
	// The channel closes when execution completes.
	ExecuteStream(ctx context.Context, req SendMessageRequest) <-chan Event
}

// SendMessageRequest represents an A2A message/send request.
type SendMessageRequest struct {
	Message       Message                  `json:"message"`
	Configuration *SendMessageConfiguration `json:"configuration,omitempty"`
	Metadata      map[string]any           `json:"metadata,omitempty"`
}

// SendMessageConfiguration contains options for the send request.
type SendMessageConfiguration struct {
	// AcceptedOutputModes specifies the output formats the client can handle.
	AcceptedOutputModes []string `json:"acceptedOutputModes,omitempty"`

	// HistoryLength controls how much conversation context to include.
	HistoryLength *int `json:"historyLength,omitempty"`

	// Blocking waits for task completion before returning.
	Blocking bool `json:"blocking,omitempty"`

	// PushNotificationConfig for async updates (not implemented yet).
	PushNotificationConfig map[string]any `json:"pushNotificationConfig,omitempty"`
}

// AgentExecutor wraps a gains Agent to implement the A2A Executor interface.
type AgentExecutor struct {
	agent   AgentRunner
	options []agent.Option
}

// AgentRunner is the interface required from gains agents.
// This allows for easier testing and decoupling.
type AgentRunner interface {
	Run(ctx context.Context, messages []ai.Message, opts ...agent.Option) (*agent.Result, error)
	RunStream(ctx context.Context, messages []ai.Message, opts ...agent.Option) <-chan event.Event
}

// NewAgentExecutor creates a new AgentExecutor wrapping the given agent.
func NewAgentExecutor(a AgentRunner, opts ...agent.Option) *AgentExecutor {
	return &AgentExecutor{
		agent:   a,
		options: opts,
	}
}

// Execute runs the agent synchronously and returns the final task.
func (e *AgentExecutor) Execute(ctx context.Context, req SendMessageRequest) (*Task, error) {
	// Convert A2A message to gains messages
	gainsMessages := []ai.Message{ToGainsMessage(req.Message)}

	// Run the agent
	result, err := e.agent.Run(ctx, gainsMessages, e.options...)
	if err != nil {
		// Create failed task
		mapper := NewMapper("", getContextID(req))
		task := mapper.CreateTask()
		task.Status = NewTaskStatusWithMessage(TaskStateFailed, &Message{
			Kind:      "message",
			MessageID: "error",
			Role:      MessageRoleAgent,
			Parts:     []Part{NewTextPart(err.Error())},
		})
		return task, nil
	}

	// Create successful task with result
	mapper := NewMapper("", getContextID(req))
	task := mapper.CreateTask()
	task.Status = NewTaskStatus(TaskStateCompleted)

	// Add result as message
	if result.Response != nil && result.Response.Content != "" {
		msg := FromGainsMessage(ai.Message{
			Role:    ai.RoleAssistant,
			Content: result.Response.Content,
		})
		task.Status.Message = &msg
	}

	// Convert history
	task.History = FromGainsMessages(result.Messages())

	return task, nil
}

// ExecuteStream runs the agent and streams status updates.
func (e *AgentExecutor) ExecuteStream(ctx context.Context, req SendMessageRequest) <-chan Event {
	output := make(chan Event, 100)

	go func() {
		defer close(output)

		// Convert A2A message to gains messages
		gainsMessages := []ai.Message{ToGainsMessage(req.Message)}

		// Create mapper for this task
		mapper := NewMapper("", getContextID(req))

		// Run the agent with streaming
		eventCh := e.agent.RunStream(ctx, gainsMessages, e.options...)

		// Map gains events to A2A events
		for evt := range eventCh {
			if a2aEvent := mapper.MapEvent(evt); a2aEvent != nil {
				output <- a2aEvent
			}
		}
	}()

	return output
}

// getContextID extracts or generates a context ID from the request.
func getContextID(req SendMessageRequest) string {
	if req.Message.ContextID != nil {
		return *req.Message.ContextID
	}
	return ""
}

// Ensure AgentExecutor implements Executor
var _ Executor = (*AgentExecutor)(nil)

// WorkflowRunner is the interface required from gains workflow runners.
type WorkflowRunner interface {
	RunStream(ctx context.Context, state any, opts ...interface{}) <-chan event.Event
}

// WorkflowExecutor wraps a gains workflow Runner to implement the A2A Executor interface.
type WorkflowExecutor struct {
	runner WorkflowRunner
}

// NewWorkflowExecutor creates a new WorkflowExecutor wrapping the given runner.
func NewWorkflowExecutor(runner WorkflowRunner) *WorkflowExecutor {
	return &WorkflowExecutor{runner: runner}
}

// Execute runs the workflow synchronously and returns the final task.
func (e *WorkflowExecutor) Execute(ctx context.Context, req SendMessageRequest) (*Task, error) {
	// Create mapper for this task
	mapper := NewMapper("", getContextID(req))

	// Convert A2A message to workflow input
	input := messageToWorkflowInput(req.Message)

	// Run the workflow with streaming to capture all events
	var lastError error
	for evt := range e.runner.RunStream(ctx, input) {
		if evt.Type == event.RunError && evt.Error != nil {
			lastError = evt.Error
		}
		mapper.MapEvent(evt)
	}

	// Create task with final state
	task := mapper.CreateTask()
	if lastError != nil {
		task.Status = NewTaskStatusWithMessage(TaskStateFailed, &Message{
			Kind:      "message",
			MessageID: "error",
			Role:      MessageRoleAgent,
			Parts:     []Part{NewTextPart(lastError.Error())},
		})
	}

	return task, nil
}

// ExecuteStream runs the workflow and streams status updates.
func (e *WorkflowExecutor) ExecuteStream(ctx context.Context, req SendMessageRequest) <-chan Event {
	output := make(chan Event, 100)

	go func() {
		defer close(output)

		// Create mapper for this task
		mapper := NewMapper("", getContextID(req))

		// Convert A2A message to workflow input
		input := messageToWorkflowInput(req.Message)

		// Run the workflow with streaming
		eventCh := e.runner.RunStream(ctx, input)

		// Map gains events to A2A events
		for evt := range eventCh {
			if a2aEvent := mapper.MapEvent(evt); a2aEvent != nil {
				output <- a2aEvent
			}
		}
	}()

	return output
}

// messageToWorkflowInput converts an A2A message to workflow input.
// It extracts data parts as input and text as a "query" field.
func messageToWorkflowInput(msg Message) map[string]any {
	input := make(map[string]any)

	// Extract text content
	if text := msg.TextContent(); text != "" {
		input["query"] = text
	}

	// Extract data parts as additional input
	for _, part := range msg.Parts {
		if dp, ok := part.(DataPart); ok {
			if data, ok := dp.Data.(map[string]any); ok {
				for k, v := range data {
					input[k] = v
				}
			}
		}
	}

	// Include metadata
	if msg.Metadata != nil {
		for k, v := range msg.Metadata {
			input[k] = v
		}
	}

	return input
}

// Ensure WorkflowExecutor implements Executor
var _ Executor = (*WorkflowExecutor)(nil)
