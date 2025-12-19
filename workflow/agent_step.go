package workflow

import (
	"context"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/agent"
	"github.com/spetersoncode/gains/chat"
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
// It runs an agent loop to completion and stores the final result in state via setter.
type AgentStep[S any] struct {
	name       string
	chatClient chat.Client
	registry   *tool.Registry
	prompt     PromptFunc[S]
	setter     func(*S, *AgentResult)
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
//   - setter: Function that stores the result in state (nil to skip storage)
//   - agentOpts: Options passed to agent.Run/RunStream
//   - chatOpts: Options passed to each chat call
//
// Example:
//
//	registry := tool.NewRegistry()
//	tool.RegisterFunc(registry, "search", "Search the web", searchHandler)
//
//	step := workflow.NewAgentStep[MyState](
//	    "research",
//	    client,
//	    registry,
//	    func(s *MyState) []ai.Message {
//	        return []ai.Message{{
//	            Role: ai.RoleUser,
//	            Content: fmt.Sprintf("Research %s and provide a summary", s.Topic),
//	        }}
//	    },
//	    func(s *MyState, r *AgentResult) { s.ResearchResult = r },
//	    []agent.Option{agent.WithMaxSteps(5)},
//	    ai.WithModel(model.Claude35Sonnet),
//	)
func NewAgentStep[S any](
	name string,
	chatClient chat.Client,
	registry *tool.Registry,
	prompt PromptFunc[S],
	setter func(*S, *AgentResult),
	agentOpts []agent.Option,
	chatOpts ...ai.Option,
) *AgentStep[S] {
	return &AgentStep[S]{
		name:       name,
		chatClient: chatClient,
		registry:   registry,
		prompt:     prompt,
		setter:     setter,
		agentOpts:  agentOpts,
		chatOpts:   chatOpts,
	}
}

// Name returns the step name.
func (a *AgentStep[S]) Name() string { return a.name }

// Run executes the agent to completion.
func (a *AgentStep[S]) Run(ctx context.Context, state *S, opts ...Option) error {
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
		return &StepError{StepName: a.name, Err: err}
	}

	// Store result in state via setter
	if a.setter != nil {
		agentResult := &AgentResult{
			Response:    result.Response,
			Messages:    result.Messages(),
			Steps:       result.Steps,
			Termination: result.Termination,
		}
		a.setter(state, agentResult)
	}

	return nil
}

// RunStream executes the agent and emits mapped workflow events.
func (a *AgentStep[S]) RunStream(ctx context.Context, state *S, opts ...Option) <-chan Event {
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

		// Store result in state via setter
		if a.setter != nil && lastResponse != nil {
			agentResult := &AgentResult{
				Response:    lastResponse,
				Messages:    messages,
				Steps:       steps,
				Termination: termination,
			}
			a.setter(state, agentResult)
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
