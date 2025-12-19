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

// testState is the common state struct for most workflow tests
type testState struct {
	Input        string
	Output       string
	Result       string
	Step1Done    bool
	Step1        string
	Step2        string
	Step3        string
	Outer1       string
	Outer2       string
	Inner1       string
	Inner2       string
	RouteTaken   string
	HandledBy    string
	Priority     string
	Match        string
	Value        int
	Aggregated   int
	Ticket       string
}

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
	step := NewFuncStep[testState]("test", func(ctx context.Context, state *testState) error {
		executed = true
		state.Result = "done"
		return nil
	})

	state := &testState{}
	err := step.Run(context.Background(), state)

	require.NoError(t, err)
	assert.True(t, executed)
	assert.Equal(t, "done", state.Result)
}

func TestFuncStep_RunError(t *testing.T) {
	expectedErr := errors.New("test error")
	step := NewFuncStep[testState]("test", func(ctx context.Context, state *testState) error {
		return expectedErr
	})

	err := step.Run(context.Background(), &testState{})
	assert.ErrorIs(t, err, expectedErr)
}

func TestFuncStep_RunStream(t *testing.T) {
	step := NewFuncStep[testState]("test", func(ctx context.Context, state *testState) error {
		state.Result = "done"
		return nil
	})

	state := &testState{}
	events := step.RunStream(context.Background(), state)

	var eventTypes []event.Type
	for ev := range events {
		eventTypes = append(eventTypes, ev.Type)
	}

	assert.Equal(t, []event.Type{event.StepStart, event.StepEnd}, eventTypes)
	assert.Equal(t, "done", state.Result)
}

// --- PromptStep Tests ---

func TestPromptStep_Run(t *testing.T) {
	provider := &mockProvider{
		responses: []mockResponse{{content: "Hello, World!"}},
	}

	step := NewPromptStep("prompt", provider,
		func(s *testState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleUser, Content: s.Input},
			}
		},
		nil,
		func(s *testState) *string { return &s.Output },
	)

	state := &testState{Input: "Hi"}
	err := step.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", state.Output)
}

func TestPromptStep_RunStream(t *testing.T) {
	provider := &mockProvider{
		responses: []mockResponse{{content: "Hello"}},
	}

	step := NewPromptStep("prompt", provider,
		func(s *testState) []ai.Message {
			return []ai.Message{{Role: ai.RoleUser, Content: "Hi"}}
		},
		nil,
		func(s *testState) *string { return &s.Output },
	)

	state := &testState{}
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
	assert.Equal(t, "Hello", state.Output)
}

// --- Chain Tests ---

func TestChain_Run(t *testing.T) {
	var order []string

	step1 := NewFuncStep[testState]("step1", func(ctx context.Context, state *testState) error {
		order = append(order, "step1")
		state.Step1Done = true
		return nil
	})

	step2 := NewFuncStep[testState]("step2", func(ctx context.Context, state *testState) error {
		order = append(order, "step2")
		assert.True(t, state.Step1Done)
		return nil
	})

	chain := NewChain("test-chain", step1, step2)
	state := &testState{}

	err := chain.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, []string{"step1", "step2"}, order)
}

func TestChain_RunWithError(t *testing.T) {
	expectedErr := errors.New("step2 error")

	step1 := NewFuncStep[testState]("step1", func(ctx context.Context, state *testState) error {
		return nil
	})

	step2 := NewFuncStep[testState]("step2", func(ctx context.Context, state *testState) error {
		return expectedErr
	})

	step3 := NewFuncStep[testState]("step3", func(ctx context.Context, state *testState) error {
		t.Fatal("step3 should not be reached")
		return nil
	})

	chain := NewChain("test-chain", step1, step2, step3)

	err := chain.Run(context.Background(), &testState{})

	var stepErr *StepError
	require.ErrorAs(t, err, &stepErr)
	assert.Equal(t, "step2", stepErr.StepName)
	assert.ErrorIs(t, stepErr.Err, expectedErr)
}

func TestChain_RunStream(t *testing.T) {
	chain := NewChain("test-chain",
		NewFuncStep[testState]("step1", func(ctx context.Context, state *testState) error {
			return nil
		}),
		NewFuncStep[testState]("step2", func(ctx context.Context, state *testState) error {
			return nil
		}),
	)

	events := chain.RunStream(context.Background(), &testState{})

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
	slowStep := NewFuncStep[testState]("slow", func(ctx context.Context, state *testState) error {
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

	err := chain.Run(ctx, &testState{})
	assert.Error(t, err)
}

// --- Parallel Tests ---

func TestParallel_Run(t *testing.T) {
	var count atomic.Int32

	steps := []Step[testState]{
		NewFuncStep[testState]("step1", func(ctx context.Context, state *testState) error {
			count.Add(1)
			state.Step1 = "done"
			return nil
		}),
		NewFuncStep[testState]("step2", func(ctx context.Context, state *testState) error {
			count.Add(1)
			state.Step2 = "done"
			return nil
		}),
		NewFuncStep[testState]("step3", func(ctx context.Context, state *testState) error {
			count.Add(1)
			state.Step3 = "done"
			return nil
		}),
	}

	// Aggregator that merges step results
	aggregator := func(state *testState, branches map[string]*testState, errs map[string]error) error {
		for name, br := range branches {
			switch name {
			case "step1":
				state.Step1 = br.Step1
			case "step2":
				state.Step2 = br.Step2
			case "step3":
				state.Step3 = br.Step3
			}
		}
		return nil
	}

	parallel := NewParallel("test-parallel", steps, aggregator)
	state := &testState{}

	err := parallel.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, int32(3), count.Load())

	assert.Equal(t, "done", state.Step1)
	assert.Equal(t, "done", state.Step2)
	assert.Equal(t, "done", state.Step3)
}

func TestParallel_RunWithAggregator(t *testing.T) {
	steps := []Step[testState]{
		NewFuncStep[testState]("step1", func(ctx context.Context, state *testState) error {
			state.Value = 1
			return nil
		}),
		NewFuncStep[testState]("step2", func(ctx context.Context, state *testState) error {
			state.Value = 2
			return nil
		}),
	}

	aggregator := func(state *testState, branches map[string]*testState, errs map[string]error) error {
		state.Aggregated = len(branches)
		return nil
	}

	parallel := NewParallel("test-parallel", steps, aggregator)
	state := &testState{}

	err := parallel.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, 2, state.Aggregated)
}

func TestParallel_RunWithError(t *testing.T) {
	steps := []Step[testState]{
		NewFuncStep[testState]("step1", func(ctx context.Context, state *testState) error {
			return nil
		}),
		NewFuncStep[testState]("step2", func(ctx context.Context, state *testState) error {
			return errors.New("step2 error")
		}),
	}

	parallel := NewParallel[testState]("test-parallel", steps, nil)

	err := parallel.Run(context.Background(), &testState{})

	var parallelErr *ParallelError
	require.ErrorAs(t, err, &parallelErr)
	assert.Contains(t, parallelErr.Errors, "step2")
}

func TestParallel_ContinueOnError(t *testing.T) {
	steps := []Step[testState]{
		NewFuncStep[testState]("step1", func(ctx context.Context, state *testState) error {
			state.Step1 = "done"
			return nil
		}),
		NewFuncStep[testState]("step2", func(ctx context.Context, state *testState) error {
			return errors.New("step2 error")
		}),
	}

	t.Run("aggregator receives errors map", func(t *testing.T) {
		var receivedErrors map[string]error
		aggregator := func(state *testState, branches map[string]*testState, errs map[string]error) error {
			receivedErrors = errs
			return nil
		}

		parallel := NewParallel("test-parallel", steps, aggregator)
		state := &testState{}

		err := parallel.Run(context.Background(), state, WithContinueOnError(true))

		require.NoError(t, err)
		assert.Len(t, receivedErrors, 1)
		assert.Contains(t, receivedErrors, "step2")
	})

	t.Run("streaming emits StepSkipped for failures", func(t *testing.T) {
		parallel := NewParallel[testState]("test-parallel", steps, nil)
		state := &testState{}

		events := parallel.RunStream(context.Background(), state, WithContinueOnError(true))

		var skippedSteps []string
		for ev := range events {
			if ev.Type == event.StepSkipped {
				skippedSteps = append(skippedSteps, ev.StepName)
				assert.NotNil(t, ev.Error)
				assert.Equal(t, "step failed, continuing", ev.Message)
			}
		}

		assert.Contains(t, skippedSteps, "step2")
	})
}

func TestParallel_MaxConcurrency(t *testing.T) {
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	steps := make([]Step[testState], 5)
	for i := 0; i < 5; i++ {
		steps[i] = NewFuncStep[testState]("step", func(ctx context.Context, state *testState) error {
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

	parallel := NewParallel[testState]("test-parallel", steps, nil)

	err := parallel.Run(context.Background(), &testState{}, WithMaxConcurrency(2))

	require.NoError(t, err)
	assert.LessOrEqual(t, maxConcurrent.Load(), int32(2))
}

func TestParallel_RunStream(t *testing.T) {
	steps := []Step[testState]{
		NewFuncStep[testState]("step1", func(ctx context.Context, state *testState) error {
			return nil
		}),
		NewFuncStep[testState]("step2", func(ctx context.Context, state *testState) error {
			return nil
		}),
	}

	parallel := NewParallel[testState]("test-parallel", steps, nil)
	events := parallel.RunStream(context.Background(), &testState{})

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
	step1 := NewFuncStep[testState]("high", func(ctx context.Context, state *testState) error {
		state.RouteTaken = "high"
		return nil
	})

	step2 := NewFuncStep[testState]("low", func(ctx context.Context, state *testState) error {
		state.RouteTaken = "low"
		return nil
	})

	router := NewRouter("test-router",
		[]Route[testState]{
			{
				Name: "high-priority",
				Condition: func(ctx context.Context, s *testState) bool {
					return s.Priority == "high"
				},
				Step: step1,
			},
			{
				Name: "low-priority",
				Condition: func(ctx context.Context, s *testState) bool {
					return s.Priority == "low"
				},
				Step: step2,
			},
		},
		nil,
	)

	t.Run("takes high priority route", func(t *testing.T) {
		state := &testState{Priority: "high"}
		err := router.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, "high", state.RouteTaken)
	})

	t.Run("takes low priority route", func(t *testing.T) {
		state := &testState{Priority: "low"}
		err := router.Run(context.Background(), state)
		require.NoError(t, err)
		assert.Equal(t, "low", state.RouteTaken)
	})
}

func TestRouter_DefaultRoute(t *testing.T) {
	defaultStep := NewFuncStep[testState]("default", func(ctx context.Context, state *testState) error {
		state.RouteTaken = "default"
		return nil
	})

	router := NewRouter("test-router",
		[]Route[testState]{
			{
				Name: "specific",
				Condition: func(ctx context.Context, s *testState) bool {
					return s.Match == "yes"
				},
				Step: NewFuncStep[testState]("specific", func(ctx context.Context, state *testState) error {
					return nil
				}),
			},
		},
		defaultStep,
	)

	state := &testState{Match: "no"}
	err := router.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, "default", state.RouteTaken)
}

func TestRouter_NoMatch(t *testing.T) {
	router := NewRouter("test-router",
		[]Route[testState]{
			{
				Name: "never-match",
				Condition: func(ctx context.Context, s *testState) bool {
					return false
				},
				Step: NewFuncStep[testState]("step", func(ctx context.Context, state *testState) error {
					return nil
				}),
			},
		},
		nil,
	)

	err := router.Run(context.Background(), &testState{})
	assert.ErrorIs(t, err, ErrNoRouteMatched)
}

func TestRouter_RunStream(t *testing.T) {
	step := NewFuncStep[testState]("target", func(ctx context.Context, state *testState) error {
		return nil
	})

	router := NewRouter("test-router",
		[]Route[testState]{
			{
				Name: "always",
				Condition: func(ctx context.Context, s *testState) bool {
					return true
				},
				Step: step,
			},
		},
		nil,
	)

	events := router.RunStream(context.Background(), &testState{})

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

	billingStep := NewFuncStep[testState]("billing", func(ctx context.Context, state *testState) error {
		state.HandledBy = "billing"
		return nil
	})

	technicalStep := NewFuncStep[testState]("technical", func(ctx context.Context, state *testState) error {
		state.HandledBy = "technical"
		return nil
	})

	router := NewClassifierRouter("classifier", provider,
		func(s *testState) []ai.Message {
			return []ai.Message{
				{Role: ai.RoleSystem, Content: "Classify as: billing, technical"},
				{Role: ai.RoleUser, Content: s.Ticket},
			}
		},
		map[string]Step[testState]{
			"billing":   billingStep,
			"technical": technicalStep,
		},
	)

	state := &testState{Ticket: "I have a billing question"}
	err := router.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, "billing", state.HandledBy)
}

func TestClassifierRouter_UnknownClassification(t *testing.T) {
	provider := &mockProvider{
		responses: []mockResponse{{content: "unknown"}},
	}

	router := NewClassifierRouter("classifier", provider,
		func(s *testState) []ai.Message {
			return []ai.Message{{Role: ai.RoleUser, Content: "test"}}
		},
		map[string]Step[testState]{
			"known": NewFuncStep[testState]("known", func(ctx context.Context, state *testState) error {
				return nil
			}),
		},
	)

	err := router.Run(context.Background(), &testState{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown classification")
}

// --- Workflow Tests ---

func TestWorkflow_Run(t *testing.T) {
	chain := NewChain("inner",
		NewFuncStep[testState]("step1", func(ctx context.Context, state *testState) error {
			state.Step1 = "done"
			return nil
		}),
		NewFuncStep[testState]("step2", func(ctx context.Context, state *testState) error {
			state.Step2 = "done"
			return nil
		}),
	)

	wf := New("test-workflow", chain)
	state := &testState{}
	result, err := wf.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, "test-workflow", result.WorkflowName)
	assert.Equal(t, TerminationComplete, result.Termination)
	assert.Equal(t, "done", result.State.Step1)
	assert.Equal(t, "done", result.State.Step2)
}

func TestWorkflow_RunWithError(t *testing.T) {
	step := NewFuncStep[testState]("failing", func(ctx context.Context, state *testState) error {
		return errors.New("intentional error")
	})

	wf := New("test-workflow", step)
	result, err := wf.Run(context.Background(), &testState{})

	assert.Error(t, err)
	assert.Equal(t, TerminationError, result.Termination)
	assert.NotNil(t, result.Error)
}

func TestWorkflow_RunStream(t *testing.T) {
	chain := NewChain("inner",
		NewFuncStep[testState]("step1", func(ctx context.Context, state *testState) error {
			return nil
		}),
	)

	wf := New("test-workflow", chain)
	events := wf.RunStream(context.Background(), &testState{})

	var eventTypes []event.Type
	for ev := range events {
		eventTypes = append(eventTypes, ev.Type)
	}

	assert.Contains(t, eventTypes, event.RunStart)
	assert.Contains(t, eventTypes, event.RunEnd)
}

// --- Nested Workflow Tests ---

func TestNestedWorkflows(t *testing.T) {
	innerChain := NewChain("inner-chain",
		NewFuncStep[testState]("inner-step1", func(ctx context.Context, state *testState) error {
			state.Inner1 = "done"
			return nil
		}),
		NewFuncStep[testState]("inner-step2", func(ctx context.Context, state *testState) error {
			state.Inner2 = "done"
			return nil
		}),
	)

	outerChain := NewChain("outer-chain",
		NewFuncStep[testState]("outer-step1", func(ctx context.Context, state *testState) error {
			state.Outer1 = "done"
			return nil
		}),
		innerChain,
		NewFuncStep[testState]("outer-step2", func(ctx context.Context, state *testState) error {
			assert.Equal(t, "done", state.Inner1)
			assert.Equal(t, "done", state.Inner2)
			state.Outer2 = "done"
			return nil
		}),
	)

	wf := New("nested-workflow", outerChain)
	state := &testState{}
	result, err := wf.Run(context.Background(), state)

	require.NoError(t, err)
	assert.Equal(t, "done", result.State.Outer1)
	assert.Equal(t, "done", result.State.Inner1)
	assert.Equal(t, "done", result.State.Inner2)
	assert.Equal(t, "done", result.State.Outer2)
}

// --- Options Tests ---

func TestApplyOptions_Defaults(t *testing.T) {
	opts := ApplyOptions()

	assert.Equal(t, 2*time.Minute, opts.StepTimeout)
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
	step1 := NewFuncStep[testState]("step1", func(ctx context.Context, state *testState) error {
		state.Step1 = "done"
		return nil
	})

	step2 := NewFuncStep[testState]("step2", func(ctx context.Context, state *testState) error {
		return errors.New("step2 error")
	})

	step3 := NewFuncStep[testState]("step3", func(ctx context.Context, state *testState) error {
		state.Step3 = "done"
		return nil
	})

	chain := NewChain("test-chain", step1, step2, step3)

	t.Run("without continue on error", func(t *testing.T) {
		state := &testState{}
		err := chain.Run(context.Background(), state)
		assert.Error(t, err)
		assert.Empty(t, state.Step3)
	})

	t.Run("with continue on error", func(t *testing.T) {
		state := &testState{}
		err := chain.Run(context.Background(), state,
			WithContinueOnError(true),
			WithErrorHandler(func(ctx context.Context, stepName string, err error) error {
				return nil // suppress error
			}),
		)
		require.NoError(t, err)
		assert.Equal(t, "done", state.Step3)
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
