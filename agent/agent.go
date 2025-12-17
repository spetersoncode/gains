package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/internal/store"
	"github.com/spetersoncode/gains/tool"
)

// ChatClient is the interface for chat capabilities needed by the agent.
type ChatClient interface {
	ChatStream(ctx context.Context, messages []ai.Message, opts ...ai.Option) (<-chan event.Event, error)
}

// Agent orchestrates autonomous tool-calling conversations.
type Agent struct {
	chatClient ChatClient
	registry   *tool.Registry
}

// New creates a new Agent with the given chat client and tool registry.
func New(c ChatClient, registry *tool.Registry) *Agent {
	return &Agent{
		chatClient: c,
		registry:   registry,
	}
}

// Run executes the agent loop and returns the final result.
// This is a blocking call that runs until the agent completes.
func (a *Agent) Run(ctx context.Context, messages []ai.Message, opts ...Option) (*Result, error) {
	eventCh := a.RunStream(ctx, messages, opts...)

	result := &Result{
		history: store.NewMessageStoreFrom(messages, nil),
	}

	var totalUsage ai.Usage
	var lastResponse *ai.Response
	var pendingAssistantMsg *ai.Message
	var pendingToolResults []ai.ToolResult

	for ev := range eventCh {
		result.Steps = ev.Step

		switch ev.Type {
		case event.StepStart:
			// Commit pending messages from previous step
			if pendingAssistantMsg != nil {
				result.history.Append(*pendingAssistantMsg)
				pendingAssistantMsg = nil
			}
			if len(pendingToolResults) > 0 {
				result.history.Append(ai.NewToolResultMessage(pendingToolResults...))
				pendingToolResults = nil
			}

		case event.StepEnd:
			lastResponse = ev.Response
			if ev.Response != nil {
				totalUsage.InputTokens += ev.Response.Usage.InputTokens
				totalUsage.OutputTokens += ev.Response.Usage.OutputTokens

				if len(ev.Response.ToolCalls) > 0 {
					pendingAssistantMsg = &ai.Message{
						Role:      ai.RoleAssistant,
						Content:   ev.Response.Content,
						ToolCalls: ev.Response.ToolCalls,
					}
				}
			}

		case event.ToolCallResult:
			if ev.ToolResult != nil {
				pendingToolResults = append(pendingToolResults, *ev.ToolResult)
			}

		case event.RunEnd:
			result.Response = ev.Response
			result.Termination = TerminationReason(ev.Message)
			if result.Response == nil {
				result.Response = lastResponse
			}

		case event.RunError:
			result.Error = ev.Error
			result.Termination = TerminationError
		}
	}

	// Commit any remaining messages
	if pendingAssistantMsg != nil {
		result.history.Append(*pendingAssistantMsg)
	}
	if len(pendingToolResults) > 0 {
		result.history.Append(ai.NewToolResultMessage(pendingToolResults...))
	}

	result.TotalUsage = totalUsage
	return result, result.Error
}

// RunStream executes the agent loop and returns a channel of events.
// The channel is closed when the agent completes or encounters a fatal error.
// Callers should drain the channel to ensure proper cleanup.
func (a *Agent) RunStream(ctx context.Context, messages []ai.Message, opts ...Option) <-chan Event {
	eventCh := event.NewChannel()

	go a.runLoop(ctx, messages, eventCh, opts...)

	return eventCh
}

func (a *Agent) runLoop(ctx context.Context, messages []ai.Message, eventCh chan<- Event, opts ...Option) {
	defer close(eventCh)

	options := ApplyOptions(opts...)

	// Apply overall timeout if specified
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// Emit run start
	event.Emit(eventCh, Event{Type: event.RunStart})

	// Prepare chat options with tools
	chatOpts := append([]ai.Option{ai.WithTools(a.registry.Tools())}, options.ChatOptions...)

	// Copy messages to avoid mutating the original
	history := store.NewMessageStoreFrom(messages, nil)

	step := 0

	for {
		step++

		// Check termination conditions before step
		if reason := a.checkTermination(ctx, step, nil, options); reason != "" {
			a.emitComplete(eventCh, step, nil, reason)
			return
		}

		event.Emit(eventCh, Event{Type: event.StepStart, Step: step})

		// Execute chat call with streaming
		response, err := a.executeStep(ctx, history.Messages(), chatOpts, step, eventCh)
		if err != nil {
			event.Emit(eventCh, Event{Type: event.RunError, Step: step, Error: err})
			return
		}

		event.Emit(eventCh, Event{Type: event.StepEnd, Step: step, Response: response})

		// Check custom stop predicate
		if options.StopPredicate != nil && options.StopPredicate(step, response) {
			a.emitComplete(eventCh, step, response, TerminationCustom)
			return
		}

		// No tool calls = natural completion
		if len(response.ToolCalls) == 0 {
			a.emitComplete(eventCh, step, response, TerminationComplete)
			return
		}

		// Process tool calls
		toolResults, allRejected := a.processToolCalls(ctx, response.ToolCalls, options, step, eventCh)

		// Append assistant message with tool calls to history
		history.Append(ai.Message{
			Role:      ai.RoleAssistant,
			Content:   response.Content,
			ToolCalls: response.ToolCalls,
		})

		// Append tool results to history
		history.Append(ai.NewToolResultMessage(toolResults...))

		// If all tools were rejected, stop
		if allRejected {
			a.emitComplete(eventCh, step, response, TerminationRejected)
			return
		}
	}
}

func (a *Agent) executeStep(ctx context.Context, messages []ai.Message, chatOpts []ai.Option, step int, eventCh chan<- Event) (*ai.Response, error) {
	// Use streaming to emit deltas
	streamCh, err := a.chatClient.ChatStream(ctx, messages, chatOpts...)
	if err != nil {
		return nil, err
	}

	var response *ai.Response
	messageID := fmt.Sprintf("msg_%d_%d", step, time.Now().UnixNano())
	messageStarted := false

	for ev := range streamCh {
		switch ev.Type {
		case event.RunError:
			return nil, ev.Error

		case event.MessageStart:
			// Forward message start with our step-scoped message ID
			event.Emit(eventCh, Event{
				Type:      event.MessageStart,
				Step:      step,
				MessageID: messageID,
			})
			messageStarted = true

		case event.MessageDelta:
			if !messageStarted {
				// Emit start if we haven't yet (defensive)
				event.Emit(eventCh, Event{
					Type:      event.MessageStart,
					Step:      step,
					MessageID: messageID,
				})
				messageStarted = true
			}
			event.Emit(eventCh, Event{
				Type:      event.MessageDelta,
				Step:      step,
				MessageID: messageID,
				Delta:     ev.Delta,
			})

		case event.MessageEnd:
			if !messageStarted {
				event.Emit(eventCh, Event{
					Type:      event.MessageStart,
					Step:      step,
					MessageID: messageID,
				})
			}
			event.Emit(eventCh, Event{
				Type:      event.MessageEnd,
				Step:      step,
				MessageID: messageID,
				Response:  ev.Response,
			})
			response = ev.Response
		}
	}

	if response == nil {
		return nil, context.Canceled
	}

	return response, nil
}

func (a *Agent) processToolCalls(ctx context.Context, toolCalls []ai.ToolCall, options *Options, step int, eventCh chan<- Event) ([]ai.ToolResult, bool) {
	// First, emit requested events and handle approval
	type approvalResult struct {
		call     ai.ToolCall
		approved bool
		reason   string
	}

	approvals := make([]approvalResult, len(toolCalls))

	for i, tc := range toolCalls {
		// Emit tool call start (name only) and args (arguments)
		event.Emit(eventCh, Event{Type: event.ToolCallStart, Step: step, ToolCall: &tc})
		event.Emit(eventCh, Event{Type: event.ToolCallArgs, Step: step, ToolCall: &tc})

		if a.requiresApproval(tc.Name, options) {
			approved, reason := options.Approver(ctx, tc)
			approvals[i] = approvalResult{call: tc, approved: approved, reason: reason}

			if approved {
				event.Emit(eventCh, Event{Type: event.ToolCallApproved, Step: step, ToolCall: &tc})
			} else {
				event.Emit(eventCh, Event{Type: event.ToolCallRejected, Step: step, ToolCall: &tc, Message: reason})
			}
		} else {
			// Auto-approved
			approvals[i] = approvalResult{call: tc, approved: true}
			event.Emit(eventCh, Event{Type: event.ToolCallApproved, Step: step, ToolCall: &tc})
		}
	}

	// Collect approved and rejected
	var approvedCalls []ai.ToolCall
	var rejectedResults []ai.ToolResult

	for _, ar := range approvals {
		if ar.approved {
			approvedCalls = append(approvedCalls, ar.call)
		} else {
			reason := ar.reason
			if reason == "" {
				reason = "Tool call rejected"
			}
			rejectedResults = append(rejectedResults, ai.ToolResult{
				ToolCallID: ar.call.ID,
				Content:    reason,
				IsError:    true,
			})
		}
	}

	// If all rejected, return early
	if len(approvedCalls) == 0 {
		for i := range rejectedResults {
			tc := toolCalls[i]
			event.Emit(eventCh, Event{Type: event.ToolCallEnd, Step: step, ToolCall: &tc})
			event.Emit(eventCh, Event{Type: event.ToolCallResult, Step: step, ToolCall: &tc, ToolResult: &rejectedResults[i]})
		}
		return rejectedResults, true
	}

	// Execute approved tool calls
	var executedResults []ai.ToolResult

	if options.ParallelToolCalls && len(approvedCalls) > 1 {
		executedResults = a.executeToolCallsParallel(ctx, approvedCalls, options, step, eventCh)
	} else {
		executedResults = a.executeToolCallsSequential(ctx, approvedCalls, options, step, eventCh)
	}

	// Combine results in original order
	results := make([]ai.ToolResult, 0, len(toolCalls))
	approvedIdx := 0
	rejectedIdx := 0

	for _, ar := range approvals {
		if ar.approved {
			results = append(results, executedResults[approvedIdx])
			approvedIdx++
		} else {
			results = append(results, rejectedResults[rejectedIdx])
			rejectedIdx++
		}
	}

	return results, false
}

func (a *Agent) executeToolCallsSequential(ctx context.Context, toolCalls []ai.ToolCall, options *Options, step int, eventCh chan<- Event) []ai.ToolResult {
	results := make([]ai.ToolResult, len(toolCalls))

	for i, tc := range toolCalls {
		results[i] = a.executeToolCall(ctx, tc, options, step, eventCh)
	}

	return results
}

func (a *Agent) executeToolCallsParallel(ctx context.Context, toolCalls []ai.ToolCall, options *Options, step int, eventCh chan<- Event) []ai.ToolResult {
	results := make([]ai.ToolResult, len(toolCalls))
	var wg sync.WaitGroup

	for i, tc := range toolCalls {
		wg.Add(1)
		go func(idx int, call ai.ToolCall) {
			defer wg.Done()
			results[idx] = a.executeToolCall(ctx, call, options, step, eventCh)
		}(i, tc)
	}

	wg.Wait()
	return results
}

func (a *Agent) executeToolCall(ctx context.Context, tc ai.ToolCall, options *Options, step int, eventCh chan<- Event) ai.ToolResult {
	event.Emit(eventCh, Event{Type: event.ToolCallExecuting, Step: step, ToolCall: &tc})

	// Apply handler timeout
	execCtx := ctx
	if options.HandlerTimeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, options.HandlerTimeout)
		defer cancel()
	}

	result, err := a.registry.Execute(execCtx, tc)
	if err != nil {
		// Tool not found or other registry error
		result = ai.ToolResult{
			ToolCallID: tc.ID,
			Content:    err.Error(),
			IsError:    true,
		}
	}

	event.Emit(eventCh, Event{Type: event.ToolCallEnd, Step: step, ToolCall: &tc})
	event.Emit(eventCh, Event{Type: event.ToolCallResult, Step: step, ToolCall: &tc, ToolResult: &result})
	return result
}

func (a *Agent) requiresApproval(toolName string, options *Options) bool {
	if options.Approver == nil {
		return false
	}
	if len(options.ApprovalRequired) == 0 {
		return true // All tools require approval
	}
	for _, name := range options.ApprovalRequired {
		if name == toolName {
			return true
		}
	}
	return false
}

func (a *Agent) checkTermination(ctx context.Context, step int, response *ai.Response, options *Options) TerminationReason {
	// Check context cancellation/timeout
	if ctx.Err() != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return TerminationTimeout
		}
		return TerminationCancelled
	}

	// Check max steps (step is 1-indexed, check before executing)
	if options.MaxSteps > 0 && step > options.MaxSteps {
		return TerminationMaxSteps
	}

	return ""
}

func (a *Agent) emitComplete(ch chan<- Event, step int, response *ai.Response, reason TerminationReason) {
	event.Emit(ch, Event{
		Type:     event.RunEnd,
		Step:     step,
		Response: response,
		Message:  string(reason),
	})
}
