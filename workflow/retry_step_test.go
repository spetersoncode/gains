package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	ai "github.com/spetersoncode/gains"
	"github.com/spetersoncode/gains/event"
	"github.com/spetersoncode/gains/internal/retry"
)

type retryState struct {
	Attempts int
	Result   string
}

func TestRetryStep_Run_Success(t *testing.T) {
	attempts := 0
	step := NewFuncStep[retryState]("inner", func(ctx context.Context, s *retryState) error {
		attempts++
		s.Attempts = attempts
		s.Result = "success"
		return nil
	})

	retryStep := NewRetryStep("retry", step)
	state := &retryState{}

	err := retryStep.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
	if state.Result != "success" {
		t.Errorf("expected 'success', got %q", state.Result)
	}
}

func TestRetryStep_Run_TransientError(t *testing.T) {
	attempts := 0
	step := NewFuncStep[retryState]("inner", func(ctx context.Context, s *retryState) error {
		attempts++
		s.Attempts = attempts
		if attempts < 3 {
			return ai.NewTransientError("temporary failure", 500, nil)
		}
		s.Result = "success after retry"
		return nil
	})

	cfg := retry.Config{
		MaxAttempts:  5,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   1.5,
		Jitter:       0,
	}
	retryStep := NewRetryStepWithConfig("retry", step, cfg)
	state := &retryState{}

	err := retryStep.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	if state.Result != "success after retry" {
		t.Errorf("expected 'success after retry', got %q", state.Result)
	}
}

func TestRetryStep_Run_NonTransientError(t *testing.T) {
	attempts := 0
	step := NewFuncStep[retryState]("inner", func(ctx context.Context, s *retryState) error {
		attempts++
		return errors.New("permanent failure") // not wrapped with Transient
	})

	cfg := retry.Config{
		MaxAttempts:  5,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   1.5,
		Jitter:       0,
	}
	retryStep := NewRetryStepWithConfig("retry", step, cfg)
	state := &retryState{}

	err := retryStep.Run(context.Background(), state)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Non-transient errors should not be retried
	if attempts != 1 {
		t.Errorf("expected 1 attempt (no retry for non-transient), got %d", attempts)
	}
}

func TestRetryStep_Run_ExhaustedAttempts(t *testing.T) {
	attempts := 0
	step := NewFuncStep[retryState]("inner", func(ctx context.Context, s *retryState) error {
		attempts++
		return ai.NewTransientError("always fails", 500, nil)
	})

	cfg := retry.Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   1.5,
		Jitter:       0,
	}
	retryStep := NewRetryStepWithConfig("retry", step, cfg)
	state := &retryState{}

	err := retryStep.Run(context.Background(), state)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryStep_RunStream_EmitsRetryEvents(t *testing.T) {
	attempts := 0
	step := NewFuncStep[retryState]("inner", func(ctx context.Context, s *retryState) error {
		attempts++
		if attempts < 2 {
			return ai.NewTransientError("temporary", 500, nil)
		}
		return nil
	})

	cfg := retry.Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   1.5,
		Jitter:       0,
	}
	retryStep := NewRetryStepWithConfig("retry", step, cfg)
	state := &retryState{}

	events := retryStep.RunStream(context.Background(), state)

	var collected []event.Type
	for ev := range events {
		collected = append(collected, ev.Type)
	}

	// Should have: StepStart, RetryAttempt, RetryFailed, RetryScheduled, RetryAttempt, RetrySuccess, StepEnd
	hasRetryAttempt := false
	hasRetryFailed := false
	hasRetrySuccess := false
	hasStepEnd := false

	for _, t := range collected {
		switch t {
		case event.RetryAttempt:
			hasRetryAttempt = true
		case event.RetryFailed:
			hasRetryFailed = true
		case event.RetrySuccess:
			hasRetrySuccess = true
		case event.StepEnd:
			hasStepEnd = true
		}
	}

	if !hasRetryAttempt {
		t.Error("expected RetryAttempt event")
	}
	if !hasRetryFailed {
		t.Error("expected RetryFailed event")
	}
	if !hasRetrySuccess {
		t.Error("expected RetrySuccess event")
	}
	if !hasStepEnd {
		t.Error("expected StepEnd event")
	}
}

func TestRetryStep_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	attempts := 0
	step := NewFuncStep[retryState]("inner", func(ctx context.Context, s *retryState) error {
		attempts++
		if attempts == 1 {
			cancel() // Cancel after first attempt
		}
		return ai.NewTransientError("keep failing", 500, nil)
	})

	cfg := retry.Config{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond, // Long delay to ensure cancellation during wait
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       0,
	}
	retryStep := NewRetryStepWithConfig("retry", step, cfg)
	state := &retryState{}

	err := retryStep.Run(ctx, state)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
