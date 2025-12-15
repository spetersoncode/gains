package agent

import (
	"context"
	"sync"
	"time"

	"github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/store"
)

// Agent orchestrates autonomous tool-calling conversations.
type Agent struct {
	provider gains.ChatProvider
	registry *Registry
}

// New creates a new Agent with the given provider and tool registry.
func New(provider gains.ChatProvider, registry *Registry) *Agent {
	return &Agent{
		provider: provider,
		registry: registry,
	}
}

// Run executes the agent loop and returns the final result.
// This is a blocking call that runs until the agent completes.
func (a *Agent) Run(ctx context.Context, messages []gains.Message, opts ...Option) (*Result, error) {
	eventCh := a.RunStream(ctx, messages, opts...)

	result := &Result{
		History: store.NewMessageStoreFrom(messages, nil),
	}

	var totalUsage gains.Usage
	var lastResponse *gains.Response
	var pendingAssistantMsg *gains.Message
	var pendingToolResults []gains.ToolResult

	for event := range eventCh {
		result.Steps = event.Step

		switch event.Type {
		case EventStepStart:
			// Commit pending messages from previous step
			if pendingAssistantMsg != nil {
				result.History.Append(*pendingAssistantMsg)
				pendingAssistantMsg = nil
			}
			if len(pendingToolResults) > 0 {
				result.History.Append(gains.NewToolResultMessage(pendingToolResults...))
				pendingToolResults = nil
			}

		case EventStepComplete:
			lastResponse = event.Response
			if event.Response != nil {
				totalUsage.InputTokens += event.Response.Usage.InputTokens
				totalUsage.OutputTokens += event.Response.Usage.OutputTokens

				if len(event.Response.ToolCalls) > 0 {
					pendingAssistantMsg = &gains.Message{
						Role:      gains.RoleAssistant,
						Content:   event.Response.Content,
						ToolCalls: event.Response.ToolCalls,
					}
				}
			}

		case EventToolResult:
			if event.ToolResult != nil {
				pendingToolResults = append(pendingToolResults, *event.ToolResult)
			}

		case EventAgentComplete:
			result.Response = event.Response
			result.Termination = TerminationReason(event.Message)
			if result.Response == nil {
				result.Response = lastResponse
			}

		case EventError:
			result.Error = event.Error
			result.Termination = TerminationError
		}
	}

	// Commit any remaining messages
	if pendingAssistantMsg != nil {
		result.History.Append(*pendingAssistantMsg)
	}
	if len(pendingToolResults) > 0 {
		result.History.Append(gains.NewToolResultMessage(pendingToolResults...))
	}

	result.TotalUsage = totalUsage
	return result, result.Error
}

// RunStream executes the agent loop and returns a channel of events.
// The channel is closed when the agent completes or encounters a fatal error.
// Callers should drain the channel to ensure proper cleanup.
func (a *Agent) RunStream(ctx context.Context, messages []gains.Message, opts ...Option) <-chan Event {
	eventCh := make(chan Event, 100) // Buffered to prevent blocking

	go a.runLoop(ctx, messages, eventCh, opts...)

	return eventCh
}

func (a *Agent) runLoop(ctx context.Context, messages []gains.Message, eventCh chan<- Event, opts ...Option) {
	defer close(eventCh)

	options := ApplyOptions(opts...)

	// Apply overall timeout if specified
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// Prepare chat options with tools
	chatOpts := append([]gains.Option{gains.WithTools(a.registry.Tools())}, options.ChatOptions...)

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

		a.emit(eventCh, Event{Type: EventStepStart, Step: step})

		// Execute chat call with streaming
		response, err := a.executeStep(ctx, history.Messages(), chatOpts, step, eventCh)
		if err != nil {
			a.emit(eventCh, Event{Type: EventError, Step: step, Error: err})
			return
		}

		a.emit(eventCh, Event{Type: EventStepComplete, Step: step, Response: response})

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
		history.Append(gains.Message{
			Role:      gains.RoleAssistant,
			Content:   response.Content,
			ToolCalls: response.ToolCalls,
		})

		// Append tool results to history
		history.Append(gains.NewToolResultMessage(toolResults...))

		// If all tools were rejected, stop
		if allRejected {
			a.emitComplete(eventCh, step, response, TerminationRejected)
			return
		}
	}
}

func (a *Agent) executeStep(ctx context.Context, messages []gains.Message, chatOpts []gains.Option, step int, eventCh chan<- Event) (*gains.Response, error) {
	// Use streaming to emit deltas
	streamCh, err := a.provider.ChatStream(ctx, messages, chatOpts...)
	if err != nil {
		return nil, err
	}

	var response *gains.Response

	for event := range streamCh {
		if event.Err != nil {
			return nil, event.Err
		}

		if event.Delta != "" {
			a.emit(eventCh, Event{
				Type:  EventStreamDelta,
				Step:  step,
				Delta: event.Delta,
			})
		}

		if event.Done && event.Response != nil {
			response = event.Response
		}
	}

	if response == nil {
		return nil, context.Canceled
	}

	return response, nil
}

func (a *Agent) processToolCalls(ctx context.Context, toolCalls []gains.ToolCall, options *Options, step int, eventCh chan<- Event) ([]gains.ToolResult, bool) {
	// First, emit requested events and handle approval
	type approvalResult struct {
		call     gains.ToolCall
		approved bool
		reason   string
	}

	approvals := make([]approvalResult, len(toolCalls))

	for i, tc := range toolCalls {
		a.emit(eventCh, Event{Type: EventToolCallRequested, Step: step, ToolCall: &tc})

		if a.requiresApproval(tc.Name, options) {
			approved, reason := options.Approver(ctx, tc)
			approvals[i] = approvalResult{call: tc, approved: approved, reason: reason}

			if approved {
				a.emit(eventCh, Event{Type: EventToolCallApproved, Step: step, ToolCall: &tc})
			} else {
				a.emit(eventCh, Event{Type: EventToolCallRejected, Step: step, ToolCall: &tc, Message: reason})
			}
		} else {
			// Auto-approved
			approvals[i] = approvalResult{call: tc, approved: true}
			a.emit(eventCh, Event{Type: EventToolCallApproved, Step: step, ToolCall: &tc})
		}
	}

	// Collect approved and rejected
	var approvedCalls []gains.ToolCall
	var rejectedResults []gains.ToolResult

	for _, ar := range approvals {
		if ar.approved {
			approvedCalls = append(approvedCalls, ar.call)
		} else {
			reason := ar.reason
			if reason == "" {
				reason = "Tool call rejected"
			}
			rejectedResults = append(rejectedResults, gains.ToolResult{
				ToolCallID: ar.call.ID,
				Content:    reason,
				IsError:    true,
			})
		}
	}

	// If all rejected, return early
	if len(approvedCalls) == 0 {
		for i := range rejectedResults {
			a.emit(eventCh, Event{Type: EventToolResult, Step: step, ToolCall: &toolCalls[i], ToolResult: &rejectedResults[i]})
		}
		return rejectedResults, true
	}

	// Execute approved tool calls
	var executedResults []gains.ToolResult

	if options.ParallelToolCalls && len(approvedCalls) > 1 {
		executedResults = a.executeToolCallsParallel(ctx, approvedCalls, options, step, eventCh)
	} else {
		executedResults = a.executeToolCallsSequential(ctx, approvedCalls, options, step, eventCh)
	}

	// Combine results in original order
	results := make([]gains.ToolResult, 0, len(toolCalls))
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

func (a *Agent) executeToolCallsSequential(ctx context.Context, toolCalls []gains.ToolCall, options *Options, step int, eventCh chan<- Event) []gains.ToolResult {
	results := make([]gains.ToolResult, len(toolCalls))

	for i, tc := range toolCalls {
		results[i] = a.executeToolCall(ctx, tc, options, step, eventCh)
	}

	return results
}

func (a *Agent) executeToolCallsParallel(ctx context.Context, toolCalls []gains.ToolCall, options *Options, step int, eventCh chan<- Event) []gains.ToolResult {
	results := make([]gains.ToolResult, len(toolCalls))
	var wg sync.WaitGroup

	for i, tc := range toolCalls {
		wg.Add(1)
		go func(idx int, call gains.ToolCall) {
			defer wg.Done()
			results[idx] = a.executeToolCall(ctx, call, options, step, eventCh)
		}(i, tc)
	}

	wg.Wait()
	return results
}

func (a *Agent) executeToolCall(ctx context.Context, tc gains.ToolCall, options *Options, step int, eventCh chan<- Event) gains.ToolResult {
	a.emit(eventCh, Event{Type: EventToolCallStarted, Step: step, ToolCall: &tc})

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
		result = gains.ToolResult{
			ToolCallID: tc.ID,
			Content:    err.Error(),
			IsError:    true,
		}
	}

	a.emit(eventCh, Event{Type: EventToolResult, Step: step, ToolCall: &tc, ToolResult: &result})
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

func (a *Agent) checkTermination(ctx context.Context, step int, response *gains.Response, options *Options) TerminationReason {
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

func (a *Agent) emit(ch chan<- Event, event Event) {
	event.Timestamp = time.Now()
	select {
	case ch <- event:
	default:
		// Channel full - should not happen with buffered channel
		// but we don't want to block the agent loop
	}
}

func (a *Agent) emitComplete(ch chan<- Event, step int, response *gains.Response, reason TerminationReason) {
	a.emit(ch, Event{
		Type:     EventAgentComplete,
		Step:     step,
		Response: response,
		Message:  string(reason),
	})
}
