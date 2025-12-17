package workflow

import (
	"context"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/tool"
)

// AgentResult contains the structured output from an AgentStep execution.
type AgentResult struct {
	// Response is the final model response.
	Response *ai.Response
	// Messages is the complete conversation history.
	Messages []ai.Message
	// Steps is the number of agent iterations.
	Steps int
	// Termination indicates why the agent stopped.
	Termination agent.TerminationReason
}

// AgentStep wraps the agent package for autonomous tool-calling within a workflow.
// It runs an agent loop to completion and stores the final result in state.
type AgentStep struct {
	name       string
	chatClient ChatClient
	registry   *tool.Registry
	prompt     PromptFunc
	outputKey  string
	agentOpts  []agent.Option
	chatOpts   []ai.Option
}

// NewAgentStep creates a step that runs an autonomous tool-calling agent.
//
// Parameters:
//   - name: Unique identifier for the step
//   - chatClient: Client supporting ChatStream for the agent
//   - registry: Tool registry with registered handlers
//   - prompt: Function that builds initial messages from state
//   - outputKey: State key for storing agent result (empty to skip storage)
//   - agentOpts: Options passed to agent.Run/RunStream
//   - chatOpts: Options passed to each chat call
//
// Example:
//
//	registry := tool.NewRegistry()
//	tool.RegisterFunc(registry, "search", "Search the web", searchHandler)
//
//	step := workflow.NewAgentStep(
//	    "research",
//	    client,
//	    registry,
//	    func(s *workflow.State) []ai.Message {
//	        topic := s.GetString("topic")
//	        return []ai.Message{{
//	            Role: ai.RoleUser,
//	            Content: fmt.Sprintf("Research %s and provide a summary", topic),
//	        }}
//	    },
//	    "research_result",
//	    []agent.Option{agent.WithMaxSteps(5)},
//	    ai.WithModel(model.Claude35Sonnet),
//	)
func NewAgentStep(
	name string,
	chatClient ChatClient,
	registry *tool.Registry,
	prompt PromptFunc,
	outputKey string,
	agentOpts []agent.Option,
	chatOpts ...ai.Option,
) *AgentStep {
	return &AgentStep{
		name:       name,
		chatClient: chatClient,
		registry:   registry,
		prompt:     prompt,
		outputKey:  outputKey,
		agentOpts:  agentOpts,
		chatOpts:   chatOpts,
	}
}

// Name returns the step name.
func (a *AgentStep) Name() string { return a.name }

// Run executes the agent to completion.
func (a *AgentStep) Run(ctx context.Context, state *State, opts ...Option) (*StepResult, error) {
	options := ApplyOptions(opts...)

	// Apply workflow timeout if set
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// Build messages from state
	msgs := a.prompt(state)

	// Merge chat options: step options first, then workflow options
	chatOpts := make([]ai.Option, 0, len(a.chatOpts)+len(options.ChatOptions))
	chatOpts = append(chatOpts, a.chatOpts...)
	chatOpts = append(chatOpts, options.ChatOptions...)

	// Build agent options with chat options
	agentOpts := make([]agent.Option, 0, len(a.agentOpts)+1)
	agentOpts = append(agentOpts, a.agentOpts...)
	if len(chatOpts) > 0 {
		agentOpts = append(agentOpts, agent.WithChatOptions(chatOpts...))
	}

	// Create and run agent
	ag := agent.New(a.chatClient, a.registry)
	result, err := ag.Run(ctx, msgs, agentOpts...)
	if err != nil {
		return nil, &StepError{StepName: a.name, Err: err}
	}

	// Store result in state
	if a.outputKey != "" {
		agentResult := &AgentResult{
			Response:    result.Response,
			Messages:    result.Messages(),
			Steps:       result.Steps,
			Termination: result.Termination,
		}
		state.Set(a.outputKey, agentResult)
	}

	return &StepResult{
		StepName: a.name,
		Output:   result.Response.Content,
		Response: result.Response,
		Usage:    result.TotalUsage,
		Metadata: map[string]any{
			"steps":       result.Steps,
			"termination": string(result.Termination),
		},
	}, nil
}

// RunStream executes the agent and emits mapped workflow events.
func (a *AgentStep) RunStream(ctx context.Context, state *State, opts ...Option) <-chan Event {
	ch := make(chan Event, 100)

	go func() {
		defer close(ch)

		options := ApplyOptions(opts...)

		if options.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, options.Timeout)
			defer cancel()
		}

		event.Emit(ch, Event{Type: event.StepStart, StepName: a.name})

		msgs := a.prompt(state)

		// Merge chat options
		chatOpts := make([]ai.Option, 0, len(a.chatOpts)+len(options.ChatOptions))
		chatOpts = append(chatOpts, a.chatOpts...)
		chatOpts = append(chatOpts, options.ChatOptions...)

		// Build agent options
		agentOpts := make([]agent.Option, 0, len(a.agentOpts)+1)
		agentOpts = append(agentOpts, a.agentOpts...)
		if len(chatOpts) > 0 {
			agentOpts = append(agentOpts, agent.WithChatOptions(chatOpts...))
		}

		ag := agent.New(a.chatClient, a.registry)
		agentCh := ag.RunStream(ctx, msgs, agentOpts...)

		var totalUsage ai.Usage
		var lastResponse *ai.Response
		var steps int
		var termination agent.TerminationReason
		var messages []ai.Message

		// Track message history from events
		pendingAssistantMsg := (*ai.Message)(nil)
		var pendingToolResults []ai.ToolResult
		messageHistory := append([]ai.Message{}, msgs...)

		// Map agent events to workflow events
		for agentEvent := range agentCh {
			steps = agentEvent.Step

			switch agentEvent.Type {
			case event.StepStart:
				// Commit pending messages from previous step
				if pendingAssistantMsg != nil {
					messageHistory = append(messageHistory, *pendingAssistantMsg)
					pendingAssistantMsg = nil
				}
				if len(pendingToolResults) > 0 {
					messageHistory = append(messageHistory, ai.NewToolResultMessage(pendingToolResults...))
					pendingToolResults = nil
				}

				event.Emit(ch, Event{
					Type:     event.StepStart,
					StepName: a.name,
					Step:     agentEvent.Step,
					Message:  "agent_iteration",
				})

			case event.MessageStart:
				event.Emit(ch, Event{
					Type:      event.MessageStart,
					StepName:  a.name,
					MessageID: agentEvent.MessageID,
				})

			case event.MessageDelta:
				event.Emit(ch, Event{
					Type:      event.MessageDelta,
					StepName:  a.name,
					MessageID: agentEvent.MessageID,
					Delta:     agentEvent.Delta,
				})

			case event.MessageEnd:
				event.Emit(ch, Event{
					Type:      event.MessageEnd,
					StepName:  a.name,
					MessageID: agentEvent.MessageID,
					Response:  agentEvent.Response,
				})

			case event.ToolCallStart:
				event.Emit(ch, Event{
					Type:     event.ToolCallStart,
					StepName: a.name,
					ToolCall: agentEvent.ToolCall,
				})

			case event.ToolCallArgs:
				event.Emit(ch, Event{
					Type:     event.ToolCallArgs,
					StepName: a.name,
					ToolCall: agentEvent.ToolCall,
				})

			case event.ToolCallApproved:
				event.Emit(ch, Event{
					Type:     event.ToolCallApproved,
					StepName: a.name,
					ToolCall: agentEvent.ToolCall,
				})

			case event.ToolCallRejected:
				event.Emit(ch, Event{
					Type:     event.ToolCallRejected,
					StepName: a.name,
					ToolCall: agentEvent.ToolCall,
					Message:  agentEvent.Message,
				})

			case event.ToolCallExecuting:
				event.Emit(ch, Event{
					Type:     event.ToolCallExecuting,
					StepName: a.name,
					ToolCall: agentEvent.ToolCall,
				})

			case event.ToolCallEnd:
				event.Emit(ch, Event{
					Type:     event.ToolCallEnd,
					StepName: a.name,
					ToolCall: agentEvent.ToolCall,
				})

			case event.ToolCallResult:
				if agentEvent.ToolResult != nil {
					pendingToolResults = append(pendingToolResults, *agentEvent.ToolResult)
				}
				event.Emit(ch, Event{
					Type:       event.ToolCallResult,
					StepName:   a.name,
					ToolCall:   agentEvent.ToolCall,
					ToolResult: agentEvent.ToolResult,
				})

			case event.StepEnd:
				if agentEvent.Response != nil {
					totalUsage.InputTokens += agentEvent.Response.Usage.InputTokens
					totalUsage.OutputTokens += agentEvent.Response.Usage.OutputTokens
					lastResponse = agentEvent.Response

					if len(agentEvent.Response.ToolCalls) > 0 {
						pendingAssistantMsg = &ai.Message{
							Role:      ai.RoleAssistant,
							Content:   agentEvent.Response.Content,
							ToolCalls: agentEvent.Response.ToolCalls,
						}
					}
				}

			case event.RunEnd:
				termination = agent.TerminationReason(agentEvent.Message)
				if agentEvent.Response != nil {
					lastResponse = agentEvent.Response
				}

			case event.RunError:
				event.Emit(ch, Event{
					Type:     event.RunError,
					StepName: a.name,
					Error:    agentEvent.Error,
				})
				return
			}
		}

		// Commit remaining pending messages
		if pendingAssistantMsg != nil {
			messageHistory = append(messageHistory, *pendingAssistantMsg)
		}
		if len(pendingToolResults) > 0 {
			messageHistory = append(messageHistory, ai.NewToolResultMessage(pendingToolResults...))
		}
		messages = messageHistory

		// Store result in state
		if a.outputKey != "" && lastResponse != nil {
			agentResult := &AgentResult{
				Response:    lastResponse,
				Messages:    messages,
				Steps:       steps,
				Termination: termination,
			}
			state.Set(a.outputKey, agentResult)
		}

		var output string
		if lastResponse != nil {
			output = lastResponse.Content
		}

		event.Emit(ch, Event{
			Type:     event.StepEnd,
			StepName: a.name,
			Response: lastResponse,
			Message:  output,
		})
	}()

	return ch
}

// NewAgentStepWithKey creates an AgentStep that stores output using a typed key.
func NewAgentStepWithKey(
	name string,
	chatClient ChatClient,
	registry *tool.Registry,
	prompt PromptFunc,
	outputKey Key[*AgentResult],
	agentOpts []agent.Option,
	chatOpts ...ai.Option,
) *AgentStep {
	return &AgentStep{
		name:       name,
		chatClient: chatClient,
		registry:   registry,
		prompt:     prompt,
		outputKey:  outputKey.Name(),
		agentOpts:  agentOpts,
		chatOpts:   chatOpts,
	}
}

// OutputKey returns a typed key for accessing the AgentResult in state.
// The key name is the step's outputKey.
func (a *AgentStep) OutputKey() Key[*AgentResult] {
	return NewKey[*AgentResult](a.outputKey)
}
