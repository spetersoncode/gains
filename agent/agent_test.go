package agent

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider implements ai.ChatProvider for testing.
type mockProvider struct {
	responses []mockResponse
	callCount int
}

type mockResponse struct {
	content   string
	toolCalls []ai.ToolCall
	err       error
}

func (m *mockProvider) Chat(ctx context.Context, messages []ai.Message, opts ...ai.Option) (*ai.Response, error) {
	if m.callCount >= len(m.responses) {
		return &ai.Response{Content: "No more responses"}, nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	if resp.err != nil {
		return nil, resp.err
	}
	return &ai.Response{
		Content:   resp.content,
		ToolCalls: resp.toolCalls,
		Usage:     ai.Usage{InputTokens: 10, OutputTokens: 20},
	}, nil
}

func (m *mockProvider) ChatStream(ctx context.Context, messages []ai.Message, opts ...ai.Option) (<-chan ai.StreamEvent, error) {
	ch := make(chan ai.StreamEvent)

	if m.callCount >= len(m.responses) {
		go func() {
			defer close(ch)
			ch <- ai.StreamEvent{
				Delta: "No more responses",
				Done:  true,
				Response: &ai.Response{
					Content: "No more responses",
				},
			}
		}()
		return ch, nil
	}

	resp := m.responses[m.callCount]
	m.callCount++

	if resp.err != nil {
		go func() {
			defer close(ch)
			ch <- ai.StreamEvent{Err: resp.err}
		}()
		return ch, nil
	}

	go func() {
		defer close(ch)
		// Simulate streaming by sending content character by character
		for _, c := range resp.content {
			select {
			case <-ctx.Done():
				ch <- ai.StreamEvent{Err: ctx.Err()}
				return
			case ch <- ai.StreamEvent{Delta: string(c)}:
			}
		}
		ch <- ai.StreamEvent{
			Done: true,
			Response: &ai.Response{
				Content:   resp.content,
				ToolCalls: resp.toolCalls,
				Usage:     ai.Usage{InputTokens: 10, OutputTokens: 20},
			},
		}
	}()

	return ch, nil
}

// --- Registry Tests ---

func TestRegistry_Register(t *testing.T) {
	t.Run("registers tool successfully", func(t *testing.T) {
		r := tool.NewRegistry()
		testTool := ai.Tool{Name: "test_tool", Description: "A test tool"}
		handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
			return "result", nil
		}

		err := r.Register(testTool, handler)

		assert.NoError(t, err)
		assert.Equal(t, 1, r.Len())
	})

	t.Run("returns error for duplicate registration", func(t *testing.T) {
		r := tool.NewRegistry()
		testTool := ai.Tool{Name: "test_tool", Description: "A test tool"}
		handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
			return "result", nil
		}

		err := r.Register(testTool, handler)
		require.NoError(t, err)

		err = r.Register(testTool, handler)
		assert.Error(t, err)
		var errAlreadyRegistered *tool.ErrToolAlreadyRegistered
		assert.ErrorAs(t, err, &errAlreadyRegistered)
	})
}

func TestRegistry_MustRegister(t *testing.T) {
	t.Run("panics on duplicate registration", func(t *testing.T) {
		r := tool.NewRegistry()
		testTool := ai.Tool{Name: "test_tool", Description: "A test tool"}
		handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
			return "result", nil
		}

		r.MustRegister(testTool, handler)

		assert.Panics(t, func() {
			r.MustRegister(testTool, handler)
		})
	})
}

func TestRegistry_Get(t *testing.T) {
	r := tool.NewRegistry()
	testTool := ai.Tool{Name: "test_tool", Description: "A test tool"}
	handler := func(ctx context.Context, call ai.ToolCall) (string, error) {
		return "result", nil
	}
	r.MustRegister(testTool, handler)

	t.Run("returns handler for registered tool", func(t *testing.T) {
		h, ok := r.Get("test_tool")
		assert.True(t, ok)
		assert.NotNil(t, h)
	})

	t.Run("returns false for unregistered tool", func(t *testing.T) {
		h, ok := r.Get("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, h)
	})
}

func TestRegistry_Tools(t *testing.T) {
	r := tool.NewRegistry()
	r.MustRegister(ai.Tool{Name: "tool1"}, func(ctx context.Context, call ai.ToolCall) (string, error) { return "", nil })
	r.MustRegister(ai.Tool{Name: "tool2"}, func(ctx context.Context, call ai.ToolCall) (string, error) { return "", nil })

	tools := r.Tools()
	assert.Len(t, tools, 2)

	names := make(map[string]bool)
	for _, t := range tools {
		names[t.Name] = true
	}
	assert.True(t, names["tool1"])
	assert.True(t, names["tool2"])
}

func TestRegistry_Execute(t *testing.T) {
	t.Run("executes handler successfully", func(t *testing.T) {
		r := tool.NewRegistry()
		r.MustRegister(
			ai.Tool{Name: "test_tool"},
			func(ctx context.Context, call ai.ToolCall) (string, error) {
				return "success: " + call.Arguments, nil
			},
		)

		result, err := r.Execute(context.Background(), ai.ToolCall{
			ID:        "call_1",
			Name:      "test_tool",
			Arguments: `{"key":"value"}`,
		})

		assert.NoError(t, err)
		assert.Equal(t, "call_1", result.ToolCallID)
		assert.Equal(t, `success: {"key":"value"}`, result.Content)
		assert.False(t, result.IsError)
	})

	t.Run("returns error result when handler fails", func(t *testing.T) {
		r := tool.NewRegistry()
		r.MustRegister(
			ai.Tool{Name: "failing_tool"},
			func(ctx context.Context, call ai.ToolCall) (string, error) {
				return "", errors.New("handler error")
			},
		)

		result, err := r.Execute(context.Background(), ai.ToolCall{
			ID:   "call_1",
			Name: "failing_tool",
		})

		// Error is captured in result, not returned as error
		assert.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Equal(t, "handler error", result.Content)
	})

	t.Run("returns error for unknown tool", func(t *testing.T) {
		r := tool.NewRegistry()

		_, err := r.Execute(context.Background(), ai.ToolCall{
			ID:   "call_1",
			Name: "unknown_tool",
		})

		assert.Error(t, err)
		var errNotFound *tool.ErrToolNotFound
		assert.ErrorAs(t, err, &errNotFound)
	})
}

// --- Options Tests ---

func TestApplyOptions(t *testing.T) {
	t.Run("applies defaults", func(t *testing.T) {
		opts := ApplyOptions()

		assert.Equal(t, 10, opts.MaxSteps)
		assert.Equal(t, 30*time.Second, opts.HandlerTimeout)
		assert.True(t, opts.ParallelToolCalls)
	})

	t.Run("applies custom options", func(t *testing.T) {
		opts := ApplyOptions(
			WithMaxSteps(5),
			WithTimeout(time.Minute),
			WithHandlerTimeout(10*time.Second),
			WithParallelToolCalls(false),
		)

		assert.Equal(t, 5, opts.MaxSteps)
		assert.Equal(t, time.Minute, opts.Timeout)
		assert.Equal(t, 10*time.Second, opts.HandlerTimeout)
		assert.False(t, opts.ParallelToolCalls)
	})
}

// --- Agent Tests ---

func TestAgent_Run_SimpleConversation(t *testing.T) {
	provider := &mockProvider{
		responses: []mockResponse{
			{content: "Hello! How can I help you?"},
		},
	}

	registry := tool.NewRegistry()
	agent := New(provider, registry)

	result, err := agent.Run(context.Background(), []ai.Message{
		{Role: ai.RoleUser, Content: "Hi"},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Steps)
	assert.Equal(t, TerminationComplete, result.Termination)
	assert.Equal(t, "Hello! How can I help you?", result.Response.Content)
}

func TestAgent_Run_WithToolCalls(t *testing.T) {
	provider := &mockProvider{
		responses: []mockResponse{
			{
				content: "Let me check the weather.",
				toolCalls: []ai.ToolCall{
					{ID: "call_1", Name: "get_weather", Arguments: `{"location":"Tokyo"}`},
				},
			},
			{content: "The weather in Tokyo is 72°F and sunny."},
		},
	}

	registry := tool.NewRegistry()
	registry.MustRegister(
		ai.Tool{Name: "get_weather", Description: "Get weather", Parameters: json.RawMessage(`{"type":"object"}`)},
		func(ctx context.Context, call ai.ToolCall) (string, error) {
			return `{"temp": 72, "conditions": "sunny"}`, nil
		},
	)

	agent := New(provider, registry)

	result, err := agent.Run(context.Background(), []ai.Message{
		{Role: ai.RoleUser, Content: "What's the weather in Tokyo?"},
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result.Steps)
	assert.Equal(t, TerminationComplete, result.Termination)
	assert.Contains(t, result.Response.Content, "72°F")

	// Verify conversation history includes tool results
	assert.True(t, len(result.Messages()) > 1)
}

func TestAgent_Run_MaxSteps(t *testing.T) {
	// Provider always returns tool calls, causing infinite loop
	provider := &mockProvider{
		responses: []mockResponse{
			{content: "Step 1", toolCalls: []ai.ToolCall{{ID: "c1", Name: "tool1", Arguments: "{}"}}},
			{content: "Step 2", toolCalls: []ai.ToolCall{{ID: "c2", Name: "tool1", Arguments: "{}"}}},
			{content: "Step 3", toolCalls: []ai.ToolCall{{ID: "c3", Name: "tool1", Arguments: "{}"}}},
			{content: "Step 4"},
		},
	}

	registry := tool.NewRegistry()
	registry.MustRegister(
		ai.Tool{Name: "tool1"},
		func(ctx context.Context, call ai.ToolCall) (string, error) { return "ok", nil },
	)

	agent := New(provider, registry)

	result, err := agent.Run(context.Background(), []ai.Message{
		{Role: ai.RoleUser, Content: "Go"},
	}, WithMaxSteps(2))

	require.NoError(t, err)
	assert.Equal(t, TerminationMaxSteps, result.Termination)
	// MaxSteps=2 means we can complete steps 1 and 2
	// Step 3 starts, checks termination, and stops immediately
	// The step counter reflects when termination was detected
	assert.Equal(t, 3, result.Steps)
}

func TestAgent_Run_Timeout(t *testing.T) {
	provider := &mockProvider{
		responses: []mockResponse{
			{content: "Processing...", toolCalls: []ai.ToolCall{{ID: "c1", Name: "slow_tool", Arguments: "{}"}}},
		},
	}

	registry := tool.NewRegistry()
	registry.MustRegister(
		ai.Tool{Name: "slow_tool"},
		func(ctx context.Context, call ai.ToolCall) (string, error) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(5 * time.Second):
				return "done", nil
			}
		},
	)

	agent := New(provider, registry)

	result, _ := agent.Run(context.Background(), []ai.Message{
		{Role: ai.RoleUser, Content: "Go"},
	}, WithTimeout(50*time.Millisecond))

	assert.Equal(t, TerminationTimeout, result.Termination)
}

func TestAgent_Run_CustomStopPredicate(t *testing.T) {
	provider := &mockProvider{
		responses: []mockResponse{
			{content: "Step 1"},
		},
	}

	registry := tool.NewRegistry()
	agent := New(provider, registry)

	result, err := agent.Run(context.Background(), []ai.Message{
		{Role: ai.RoleUser, Content: "Go"},
	}, WithStopPredicate(func(step int, response *ai.Response) bool {
		return response.Content == "Step 1"
	}))

	require.NoError(t, err)
	assert.Equal(t, TerminationCustom, result.Termination)
}

func TestAgent_Run_Approval(t *testing.T) {
	t.Run("approved tool executes", func(t *testing.T) {
		provider := &mockProvider{
			responses: []mockResponse{
				{content: "Calling tool", toolCalls: []ai.ToolCall{{ID: "c1", Name: "tool1", Arguments: "{}"}}},
				{content: "Done"},
			},
		}

		registry := tool.NewRegistry()
		registry.MustRegister(
			ai.Tool{Name: "tool1"},
			func(ctx context.Context, call ai.ToolCall) (string, error) { return "result", nil },
		)

		agent := New(provider, registry)

		result, err := agent.Run(context.Background(), []ai.Message{
			{Role: ai.RoleUser, Content: "Go"},
		}, WithApprover(func(ctx context.Context, call ai.ToolCall) (bool, string) {
			return true, ""
		}))

		require.NoError(t, err)
		assert.Equal(t, TerminationComplete, result.Termination)
		assert.Equal(t, 2, result.Steps)
	})

	t.Run("rejected tool stops agent", func(t *testing.T) {
		provider := &mockProvider{
			responses: []mockResponse{
				{content: "Calling tool", toolCalls: []ai.ToolCall{{ID: "c1", Name: "tool1", Arguments: "{}"}}},
			},
		}

		registry := tool.NewRegistry()
		registry.MustRegister(
			ai.Tool{Name: "tool1"},
			func(ctx context.Context, call ai.ToolCall) (string, error) { return "result", nil },
		)

		agent := New(provider, registry)

		result, err := agent.Run(context.Background(), []ai.Message{
			{Role: ai.RoleUser, Content: "Go"},
		}, WithApprover(func(ctx context.Context, call ai.ToolCall) (bool, string) {
			return false, "Dangerous operation"
		}))

		require.NoError(t, err)
		assert.Equal(t, TerminationRejected, result.Termination)
	})

	t.Run("approval required only for specific tools", func(t *testing.T) {
		var approverCalled int32
		provider := &mockProvider{
			responses: []mockResponse{
				{content: "Calling safe", toolCalls: []ai.ToolCall{{ID: "c1", Name: "safe_tool", Arguments: "{}"}}},
				{content: "Calling dangerous", toolCalls: []ai.ToolCall{{ID: "c2", Name: "dangerous_tool", Arguments: "{}"}}},
				{content: "Done"},
			},
		}

		registry := tool.NewRegistry()
		registry.MustRegister(ai.Tool{Name: "safe_tool"}, func(ctx context.Context, call ai.ToolCall) (string, error) { return "ok", nil })
		registry.MustRegister(ai.Tool{Name: "dangerous_tool"}, func(ctx context.Context, call ai.ToolCall) (string, error) { return "ok", nil })

		agent := New(provider, registry)

		result, err := agent.Run(context.Background(), []ai.Message{
			{Role: ai.RoleUser, Content: "Go"},
		},
			WithApprover(func(ctx context.Context, call ai.ToolCall) (bool, string) {
				atomic.AddInt32(&approverCalled, 1)
				return true, ""
			}),
			WithApprovalRequired("dangerous_tool"),
		)

		require.NoError(t, err)
		assert.Equal(t, TerminationComplete, result.Termination)
		// Approver should only be called for dangerous_tool
		assert.Equal(t, int32(1), atomic.LoadInt32(&approverCalled))
	})
}

func TestAgent_RunStream_Events(t *testing.T) {
	provider := &mockProvider{
		responses: []mockResponse{
			{content: "Calling tool", toolCalls: []ai.ToolCall{{ID: "c1", Name: "tool1", Arguments: "{}"}}},
			{content: "Done"},
		},
	}

	registry := tool.NewRegistry()
	registry.MustRegister(
		ai.Tool{Name: "tool1"},
		func(ctx context.Context, call ai.ToolCall) (string, error) { return "result", nil },
	)

	agent := New(provider, registry)

	events := agent.RunStream(context.Background(), []ai.Message{
		{Role: ai.RoleUser, Content: "Go"},
	})

	var eventTypes []EventType
	for event := range events {
		eventTypes = append(eventTypes, event.Type)
	}

	// Verify we get expected event sequence
	assert.Contains(t, eventTypes, EventStepStart)
	assert.Contains(t, eventTypes, EventStreamDelta)
	assert.Contains(t, eventTypes, EventStepComplete)
	assert.Contains(t, eventTypes, EventToolCallRequested)
	assert.Contains(t, eventTypes, EventToolCallApproved)
	assert.Contains(t, eventTypes, EventToolCallStarted)
	assert.Contains(t, eventTypes, EventToolResult)
	assert.Contains(t, eventTypes, EventAgentComplete)
}

func TestAgent_ParallelToolCalls(t *testing.T) {
	var executionOrder []string
	var mu sync.Mutex

	provider := &mockProvider{
		responses: []mockResponse{
			{
				content: "Calling tools",
				toolCalls: []ai.ToolCall{
					{ID: "c1", Name: "tool1", Arguments: "{}"},
					{ID: "c2", Name: "tool2", Arguments: "{}"},
					{ID: "c3", Name: "tool3", Arguments: "{}"},
				},
			},
			{content: "Done"},
		},
	}

	registry := tool.NewRegistry()
	for _, name := range []string{"tool1", "tool2", "tool3"} {
		toolName := name
		registry.MustRegister(
			ai.Tool{Name: toolName},
			func(ctx context.Context, call ai.ToolCall) (string, error) {
				// Small delay to allow interleaving
				time.Sleep(10 * time.Millisecond)
				mu.Lock()
				executionOrder = append(executionOrder, toolName)
				mu.Unlock()
				return "ok", nil
			},
		)
	}

	agent := New(provider, registry)

	_, err := agent.Run(context.Background(), []ai.Message{
		{Role: ai.RoleUser, Content: "Go"},
	}, WithParallelToolCalls(true))

	require.NoError(t, err)
	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, executionOrder, 3)
}

// --- Error Tests ---

func TestErrors(t *testing.T) {
	t.Run("ErrToolNotFound", func(t *testing.T) {
		err := &tool.ErrToolNotFound{Name: "missing_tool"}
		assert.Contains(t, err.Error(), "missing_tool")
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("ErrToolExecution", func(t *testing.T) {
		inner := errors.New("connection failed")
		err := &tool.ErrToolExecution{Name: "api_tool", Err: inner}
		assert.Contains(t, err.Error(), "api_tool")
		assert.Contains(t, err.Error(), "connection failed")
		assert.ErrorIs(t, err, inner)
	})

	t.Run("ErrToolAlreadyRegistered", func(t *testing.T) {
		err := &tool.ErrToolAlreadyRegistered{Name: "duplicate"}
		assert.Contains(t, err.Error(), "duplicate")
		assert.Contains(t, err.Error(), "already registered")
	})
}

// --- Event Tests ---

func TestEventType_Constants(t *testing.T) {
	// Verify event type constants are defined
	assert.Equal(t, EventType("step_start"), EventStepStart)
	assert.Equal(t, EventType("stream_delta"), EventStreamDelta)
	assert.Equal(t, EventType("tool_call_requested"), EventToolCallRequested)
	assert.Equal(t, EventType("tool_call_approved"), EventToolCallApproved)
	assert.Equal(t, EventType("tool_call_rejected"), EventToolCallRejected)
	assert.Equal(t, EventType("tool_call_started"), EventToolCallStarted)
	assert.Equal(t, EventType("tool_result"), EventToolResult)
	assert.Equal(t, EventType("step_complete"), EventStepComplete)
	assert.Equal(t, EventType("agent_complete"), EventAgentComplete)
	assert.Equal(t, EventType("error"), EventError)
}

func TestTerminationReason_Constants(t *testing.T) {
	assert.Equal(t, TerminationReason("complete"), TerminationComplete)
	assert.Equal(t, TerminationReason("max_steps"), TerminationMaxSteps)
	assert.Equal(t, TerminationReason("timeout"), TerminationTimeout)
	assert.Equal(t, TerminationReason("custom"), TerminationCustom)
	assert.Equal(t, TerminationReason("rejected"), TerminationRejected)
	assert.Equal(t, TerminationReason("error"), TerminationError)
	assert.Equal(t, TerminationReason("cancelled"), TerminationCancelled)
}
