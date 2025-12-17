package workflow

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: State tests are in the internal/store package.

// --- Mock Provider ---

type mockProvider struct {
	responses []mockResponse
	callCount int
	mu        sync.Mutex
}

type mockResponse struct {
	content   string
	toolCalls []ai.ToolCall
	err       error
}

func (m *mockProvider) Chat(ctx context.Context, messages []ai.Message, opts ...ai.Option) (*ai.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

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

func (m *mockProvider) ChatStream(ctx context.Context, messages []ai.Message, opts ...ai.Option) (<-chan event.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan event.Event)

	if m.callCount >= len(m.responses) {
		go func() {
			defer close(ch)
			msgID := "msg-default"
			ch <- event.Event{Type: event.MessageStart, MessageID: msgID}
			ch <- event.Event{Type: event.MessageDelta, MessageID: msgID, Delta: "No more responses"}
			ch <- event.Event{
				Type:      event.MessageEnd,
				MessageID: msgID,
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
			ch <- event.Event{Type: event.RunError, Error: resp.err}
		}()
		return ch, nil
	}

	go func() {
		defer close(ch)
		msgID := "msg-test"
		ch <- event.Event{Type: event.MessageStart, MessageID: msgID}
		for _, c := range resp.content {
			select {
			case <-ctx.Done():
				ch <- event.Event{Type: event.RunError, Error: ctx.Err()}
				return
			case ch <- event.Event{Type: event.MessageDelta, MessageID: msgID, Delta: string(c)}:
			}
		}
		ch <- event.Event{
			Type:      event.MessageEnd,
			MessageID: msgID,
			Response: &ai.Response{
				Content:   resp.content,
				ToolCalls: resp.toolCalls,
				Usage:     ai.Usage{InputTokens: 10, OutputTokens: 20},
			},
		}
	}()

	return ch, nil
}

// --- FuncStep Tests ---

func TestFuncStep_Run(t *testing.T) {
	executed := false
	step := NewFuncStep("test", func(ctx context.Context, state *State) error {
		executed = true
		state.Set("result", "done")
		return nil
	})

	state := NewState(nil)
	result, err := step.Run(context.Background(), state)

	require.NoError(t, err)
	assert.True(t, executed)
	assert.Equal(t, "test", result.StepName)
	assert.Equal(t, "done", state.GetString("result"))
}

func TestFuncStep_RunError(t *testing.T) {
	expectedErr := errors.New("test error")
	step := NewFuncStep("test", func(ctx context.Context, state *State) error {
		return expectedErr
	})

	_, err := step.Run(context.Background(), NewState(nil))
	assert.ErrorIs(t, err, expectedErr)
}

func TestFuncStep_RunStream(t *testing.T) {
	step := NewFuncStep("test", func(ctx context.Context, state *State) error {
		state.Set("result", "done")
		return nil
	})

	state := NewState(nil)
	events := step.RunStream(context.Background(), state)

	var eventTypes []event.Type
	for ev := range events {
		eventTypes = append(eventTypes, ev.Type)
	}

	assert.Equal(t, []event.Type{event.StepStart, event.StepEnd}, eventTypes)
	assert.Equal(t, "done", state.GetString("result"))
}

// --- PromptStep Tests ---

func TestPromptStep_Run(t *testing.T) {
	provider := &mockProvider{
		responses: []mockResponse{{content: "Hello, World!"}},
	}

	step := NewPromptStep("prompt", provider,
		func(s *State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: s.GetString("input")},
			}
		},
		"output",
	)

	state := NewStateFrom(map[string]any{"input": "Hi"})
	result, err := step.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result.Output)
	assert.Equal(t, "Hello, World!", state.GetString("output"))
	assert.Equal(t, 10, result.Usage.InputTokens)
	assert.Equal(t, 20, result.Usage.OutputTokens)
}

func TestPromptStep_RunStream(t *testing.T) {
	provider := &mockProvider{
		responses: []mockResponse{{content: "Hello"}},
	}

	step := NewPromptStep("prompt", provider,
		func(s *State) []ai.Message {
			return []ai.Message{{Role: ai.RoleUser, Content: "Hi"}}
		},
		"output",
	)

	state := NewState(nil)
	events := step.RunStream(context.Background(), state)

	var deltas string
	var completed bool
	for ev := range events {
		if ev.Type == event.MessageDelta {
			deltas += ev.Delta
		}
		if ev.Type == event.StepEnd {
			completed = true
		}
	}

	assert.Equal(t, "Hello", deltas)
	assert.True(t, completed)
	assert.Equal(t, "Hello", state.GetString("output"))
}

// --- Chain Tests ---

func TestChain_Run(t *testing.T) {
	var order []string

	step1 := NewFuncStep("step1", func(ctx context.Context, state *State) error {
		order = append(order, "step1")
		state.Set("step1_done", true)
		return nil
	})

	step2 := NewFuncStep("step2", func(ctx context.Context, state *State) error {
		order = append(order, "step2")
		// Verify step1 ran first
		assert.True(t, state.GetBool("step1_done"))
		return nil
	})

	chain := NewChain("test-chain", step1, step2)
	state := NewState(nil)

	result, err := chain.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, []string{"step1", "step2"}, order)
	assert.Equal(t, "test-chain", result.StepName)
}

func TestChain_RunWithError(t *testing.T) {
	expectedErr := errors.New("step2 error")

	step1 := NewFuncStep("step1", func(ctx context.Context, state *State) error {
		return nil
	})

	step2 := NewFuncStep("step2", func(ctx context.Context, state *State) error {
		return expectedErr
	})

	step3 := NewFuncStep("step3", func(ctx context.Context, state *State) error {
		t.Fatal("step3 should not be reached")
		return nil
	})

	chain := NewChain("test-chain", step1, step2, step3)

	_, err := chain.Run(context.Background(), NewState(nil))

	var stepErr *StepError
	require.ErrorAs(t, err, &stepErr)
	assert.Equal(t, "step2", stepErr.StepName)
	assert.ErrorIs(t, stepErr.Err, expectedErr)
}

func TestChain_RunStream(t *testing.T) {
	chain := NewChain("test-chain",
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			return nil
		}),
		NewFuncStep("step2", func(ctx context.Context, state *State) error {
			return nil
		}),
	)

	events := chain.RunStream(context.Background(), NewState(nil))

	var eventTypes []event.Type
	for ev := range events {
		eventTypes = append(eventTypes, ev.Type)
	}

	expected := []event.Type{
		event.RunStart,
		event.StepStart, event.StepEnd,
		event.StepStart, event.StepEnd,
		event.RunEnd,
	}
	assert.Equal(t, expected, eventTypes)
}

func TestChain_Timeout(t *testing.T) {
	slowStep := NewFuncStep("slow", func(ctx context.Context, state *State) error {
		select {
		case <-time.After(1 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	chain := NewChain("test-chain", slowStep)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := chain.Run(ctx, NewState(nil))
	assert.Error(t, err)
}

// --- Parallel Tests ---

func TestParallel_Run(t *testing.T) {
	var count atomic.Int32

	steps := []Step{
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			count.Add(1)
			state.Set("step1", "done")
			return nil
		}),
		NewFuncStep("step2", func(ctx context.Context, state *State) error {
			count.Add(1)
			state.Set("step2", "done")
			return nil
		}),
		NewFuncStep("step3", func(ctx context.Context, state *State) error {
			count.Add(1)
			state.Set("step3", "done")
			return nil
		}),
	}

	parallel := NewParallel("test-parallel", steps, nil)
	state := NewState(nil)

	result, err := parallel.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, int32(3), count.Load())
	assert.Equal(t, "test-parallel", result.StepName)

	// With default aggregation, all branch states should be merged
	assert.Equal(t, "done", state.GetString("step1"))
	assert.Equal(t, "done", state.GetString("step2"))
	assert.Equal(t, "done", state.GetString("step3"))
}

func TestParallel_RunWithAggregator(t *testing.T) {
	steps := []Step{
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			state.Set("value", 1)
			return nil
		}),
		NewFuncStep("step2", func(ctx context.Context, state *State) error {
			state.Set("value", 2)
			return nil
		}),
	}

	aggregator := func(state *State, results map[string]*StepResult) error {
		state.Set("aggregated", len(results))
		return nil
	}

	parallel := NewParallel("test-parallel", steps, aggregator)
	state := NewState(nil)

	_, err := parallel.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, 2, state.GetInt("aggregated"))
}

func TestParallel_RunWithError(t *testing.T) {
	steps := []Step{
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			return nil
		}),
		NewFuncStep("step2", func(ctx context.Context, state *State) error {
			return errors.New("step2 error")
		}),
	}

	parallel := NewParallel("test-parallel", steps, nil)

	_, err := parallel.Run(context.Background(), NewState(nil))

	var parallelErr *ParallelError
	require.ErrorAs(t, err, &parallelErr)
	assert.Contains(t, parallelErr.Errors, "step2")
}

func TestParallel_MaxConcurrency(t *testing.T) {
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	steps := make([]Step, 5)
	for i := 0; i < 5; i++ {
		steps[i] = NewFuncStep("step", func(ctx context.Context, state *State) error {
			current := concurrent.Add(1)
			for {
				max := maxConcurrent.Load()
				if current > max {
					if maxConcurrent.CompareAndSwap(max, current) {
						break
					}
				} else {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			concurrent.Add(-1)
			return nil
		})
	}

	parallel := NewParallel("test-parallel", steps, nil)

	_, err := parallel.Run(context.Background(), NewState(nil), WithMaxConcurrency(2))

	require.NoError(t, err)
	assert.LessOrEqual(t, maxConcurrent.Load(), int32(2))
}

func TestParallel_RunStream(t *testing.T) {
	steps := []Step{
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			return nil
		}),
		NewFuncStep("step2", func(ctx context.Context, state *State) error {
			return nil
		}),
	}

	parallel := NewParallel("test-parallel", steps, nil)
	events := parallel.RunStream(context.Background(), NewState(nil))

	var hasStart, hasComplete bool
	var stepCompletes int
	for ev := range events {
		if ev.Type == event.ParallelStart {
			hasStart = true
		}
		if ev.Type == event.ParallelEnd {
			hasComplete = true
		}
		if ev.Type == event.StepEnd {
			stepCompletes++
		}
	}

	assert.True(t, hasStart)
	assert.True(t, hasComplete)
	assert.Equal(t, 2, stepCompletes)
}

// --- Router Tests ---

func TestRouter_Run(t *testing.T) {
	step1 := NewFuncStep("high", func(ctx context.Context, state *State) error {
		state.Set("route_taken", "high")
		return nil
	})

	step2 := NewFuncStep("low", func(ctx context.Context, state *State) error {
		state.Set("route_taken", "low")
		return nil
	})

	router := NewRouter("test-router",
		[]Route{
			{
				Name: "high-priority",
				Condition: func(ctx context.Context, s *State) bool {
					return s.GetString("priority") == "high"
				},
				Step: step1,
			},
			{
				Name: "low-priority",
				Condition: func(ctx context.Context, s *State) bool {
					return s.GetString("priority") == "low"
				},
				Step: step2,
			},
		},
		nil,
	)

	t.Run("takes high priority route", func(t *testing.T) {
		state := NewStateFrom(map[string]any{"priority": "high"})
		_, err := router.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, "high", state.GetString("route_taken"))
		assert.Equal(t, "high-priority", state.GetString("test-router_route"))
	})

	t.Run("takes low priority route", func(t *testing.T) {
		state := NewStateFrom(map[string]any{"priority": "low"})
		_, err := router.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, "low", state.GetString("route_taken"))
		assert.Equal(t, "low-priority", state.GetString("test-router_route"))
	})
}

func TestRouter_DefaultRoute(t *testing.T) {
	defaultStep := NewFuncStep("default", func(ctx context.Context, state *State) error {
		state.Set("route_taken", "default")
		return nil
	})

	router := NewRouter("test-router",
		[]Route{
			{
				Name: "specific",
				Condition: func(ctx context.Context, s *State) bool {
					return s.GetString("match") == "yes"
				},
				Step: NewFuncStep("specific", func(ctx context.Context, state *State) error {
					return nil
				}),
			},
		},
		defaultStep,
	)

	state := NewStateFrom(map[string]any{"match": "no"})
	_, err := router.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, "default", state.GetString("route_taken"))
}

func TestRouter_NoMatch(t *testing.T) {
	router := NewRouter("test-router",
		[]Route{
			{
				Name: "never-match",
				Condition: func(ctx context.Context, s *State) bool {
					return false
				},
				Step: NewFuncStep("step", func(ctx context.Context, state *State) error {
					return nil
				}),
			},
		},
		nil,
	)

	_, err := router.Run(context.Background(), NewState(nil))
	assert.ErrorIs(t, err, ErrNoRouteMatched)
}

func TestRouter_RunStream(t *testing.T) {
	step := NewFuncStep("target", func(ctx context.Context, state *State) error {
		return nil
	})

	router := NewRouter("test-router",
		[]Route{
			{
				Name: "always",
				Condition: func(ctx context.Context, s *State) bool {
					return true
				},
				Step: step,
			},
		},
		nil,
	)

	events := router.RunStream(context.Background(), NewState(nil))

	var hasRouteSelected bool
	var selectedRoute string
	for ev := range events {
		if ev.Type == event.RouteSelected {
			hasRouteSelected = true
			selectedRoute = ev.RouteName
		}
	}

	assert.True(t, hasRouteSelected)
	assert.Equal(t, "always", selectedRoute)
}

func TestClassifierRouter_Run(t *testing.T) {
	provider := &mockProvider{
		responses: []mockResponse{{content: "billing"}},
	}

	billingStep := NewFuncStep("billing", func(ctx context.Context, state *State) error {
		state.Set("handled_by", "billing")
		return nil
	})

	technicalStep := NewFuncStep("technical", func(ctx context.Context, state *State) error {
		state.Set("handled_by", "technical")
		return nil
	})

	router := NewClassifierRouter("classifier", provider,
		func(s *State) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleSystem, Content: "Classify as: billing, technical"},
				{Role: ai.RoleUser, Content: s.GetString("ticket")},
			}
		},
		map[string]Step{
			"billing":   billingStep,
			"technical": technicalStep,
		},
	)

	state := NewStateFrom(map[string]any{"ticket": "I have a billing question"})
	_, err := router.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, "billing", state.GetString("handled_by"))
	assert.Equal(t, "billing", state.GetString("classifier_classification"))
}

func TestClassifierRouter_UnknownClassification(t *testing.T) {
	provider := &mockProvider{
		responses: []mockResponse{{content: "unknown"}},
	}

	router := NewClassifierRouter("classifier", provider,
		func(s *State) []ai.Message {
			return []ai.Message{{Role: ai.RoleUser, Content: "test"}}
		},
		map[string]Step{
			"known": NewFuncStep("known", func(ctx context.Context, state *State) error {
				return nil
			}),
		},
	)

	_, err := router.Run(context.Background(), NewState(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown classification")
}

// --- Workflow Tests ---

func TestWorkflow_Run(t *testing.T) {
	chain := NewChain("inner",
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			state.Set("step1", "done")
			return nil
		}),
		NewFuncStep("step2", func(ctx context.Context, state *State) error {
			state.Set("step2", "done")
			return nil
		}),
	)

	wf := New("test-workflow", chain)
	result, err := wf.Run(context.Background(), NewState(nil))

	require.NoError(t, err)
	assert.Equal(t, "test-workflow", result.WorkflowName)
	assert.Equal(t, TerminationComplete, result.Termination)
	assert.Equal(t, "done", result.State.GetString("step1"))
	assert.Equal(t, "done", result.State.GetString("step2"))
}

func TestWorkflow_RunWithNilState(t *testing.T) {
	step := NewFuncStep("step", func(ctx context.Context, state *State) error {
		state.Set("key", "value")
		return nil
	})

	wf := New("test-workflow", step)
	result, err := wf.Run(context.Background(), nil)

	require.NoError(t, err)
	assert.NotNil(t, result.State)
	assert.Equal(t, "value", result.State.GetString("key"))
}

func TestWorkflow_RunWithError(t *testing.T) {
	step := NewFuncStep("failing", func(ctx context.Context, state *State) error {
		return errors.New("intentional error")
	})

	wf := New("test-workflow", step)
	result, err := wf.Run(context.Background(), NewState(nil))

	assert.Error(t, err)
	assert.Equal(t, TerminationError, result.Termination)
	assert.NotNil(t, result.Error)
}

func TestWorkflow_RunStream(t *testing.T) {
	chain := NewChain("inner",
		NewFuncStep("step1", func(ctx context.Context, state *State) error {
			return nil
		}),
	)

	wf := New("test-workflow", chain)
	events := wf.RunStream(context.Background(), NewState(nil))

	var eventTypes []event.Type
	for ev := range events {
		eventTypes = append(eventTypes, ev.Type)
	}

	assert.Contains(t, eventTypes, event.RunStart)
	assert.Contains(t, eventTypes, event.RunEnd)
}

// --- Nested Workflow Tests ---

func TestNestedWorkflows(t *testing.T) {
	// Create an inner chain
	innerChain := NewChain("inner-chain",
		NewFuncStep("inner-step1", func(ctx context.Context, state *State) error {
			state.Set("inner1", "done")
			return nil
		}),
		NewFuncStep("inner-step2", func(ctx context.Context, state *State) error {
			state.Set("inner2", "done")
			return nil
		}),
	)

	// Create outer chain that contains inner chain
	outerChain := NewChain("outer-chain",
		NewFuncStep("outer-step1", func(ctx context.Context, state *State) error {
			state.Set("outer1", "done")
			return nil
		}),
		innerChain, // Chain implements Step, so it can be nested
		NewFuncStep("outer-step2", func(ctx context.Context, state *State) error {
			// Verify inner steps ran
			assert.Equal(t, "done", state.GetString("inner1"))
			assert.Equal(t, "done", state.GetString("inner2"))
			state.Set("outer2", "done")
			return nil
		}),
	)

	wf := New("nested-workflow", outerChain)
	result, err := wf.Run(context.Background(), NewState(nil))

	require.NoError(t, err)
	assert.Equal(t, "done", result.State.GetString("outer1"))
	assert.Equal(t, "done", result.State.GetString("inner1"))
	assert.Equal(t, "done", result.State.GetString("inner2"))
	assert.Equal(t, "done", result.State.GetString("outer2"))
}

// --- Options Tests ---

func TestApplyOptions_Defaults(t *testing.T) {
	opts := ApplyOptions()

	assert.Equal(t, 30*time.Second, opts.StepTimeout)
	assert.False(t, opts.ContinueOnError)
	assert.Zero(t, opts.Timeout)
	assert.Zero(t, opts.MaxConcurrency)
}

func TestApplyOptions_Custom(t *testing.T) {
	opts := ApplyOptions(
		WithTimeout(5*time.Minute),
		WithStepTimeout(1*time.Minute),
		WithMaxConcurrency(4),
		WithContinueOnError(true),
	)

	assert.Equal(t, 5*time.Minute, opts.Timeout)
	assert.Equal(t, 1*time.Minute, opts.StepTimeout)
	assert.Equal(t, 4, opts.MaxConcurrency)
	assert.True(t, opts.ContinueOnError)
}

func TestContinueOnError(t *testing.T) {
	step1 := NewFuncStep("step1", func(ctx context.Context, state *State) error {
		state.Set("step1", "done")
		return nil
	})

	step2 := NewFuncStep("step2", func(ctx context.Context, state *State) error {
		return errors.New("step2 error")
	})

	step3 := NewFuncStep("step3", func(ctx context.Context, state *State) error {
		state.Set("step3", "done")
		return nil
	})

	chain := NewChain("test-chain", step1, step2, step3)

	t.Run("without continue on error", func(t *testing.T) {
		state := NewState(nil)
		_, err := chain.Run(context.Background(), state)
		assert.Error(t, err)
		assert.False(t, state.Has("step3"))
	})

	t.Run("with continue on error", func(t *testing.T) {
		state := NewState(nil)
		_, err := chain.Run(context.Background(), state,
			WithContinueOnError(true),
			WithErrorHandler(func(ctx context.Context, stepName string, err error) error {
				return nil // suppress error
			}),
		)
		require.NoError(t, err)
		assert.Equal(t, "done", state.GetString("step3"))
	})
}

// --- Error Tests ---

func TestStepError(t *testing.T) {
	inner := errors.New("inner error")
	err := &StepError{StepName: "test-step", Err: inner}

	assert.Contains(t, err.Error(), "test-step")
	assert.Contains(t, err.Error(), "inner error")
	assert.ErrorIs(t, err, inner)
}

func TestParallelError(t *testing.T) {
	err := &ParallelError{
		Errors: map[string]error{
			"step1": errors.New("error1"),
			"step2": errors.New("error2"),
		},
	}

	assert.Contains(t, err.Error(), "2 errors")
}
