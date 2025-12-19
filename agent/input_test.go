package agent

import (
	"context"
	"testing"
	"time"
)

func TestUserInputBroker_RequestConfirm(t *testing.T) {
	broker := NewUserInputBrokerWith(WithInputTimeout(100 * time.Millisecond))

	t.Run("confirmed", func(t *testing.T) {
		go func() {
			time.Sleep(10 * time.Millisecond)
			// Find the pending request and respond
			if broker.HasPending() {
				broker.Respond(UserInputResponse{
					RequestID: "", // Will be set below
					Confirmed: true,
				})
			}
		}()

		// Use a goroutine to respond since Request blocks
		ctx := context.Background()
		go func() {
			time.Sleep(10 * time.Millisecond)
			broker.mu.Lock()
			for id := range broker.pending {
				broker.mu.Unlock()
				broker.Respond(UserInputResponse{
					RequestID: id,
					Confirmed: true,
				})
				return
			}
			broker.mu.Unlock()
		}()

		confirmed, err := broker.RequestConfirm(ctx, "Delete?", "Are you sure?")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !confirmed {
			t.Error("expected confirmed to be true")
		}
	})
}

func TestUserInputBroker_RequestText(t *testing.T) {
	broker := NewUserInputBrokerWith(WithInputTimeout(100 * time.Millisecond))

	go func() {
		time.Sleep(10 * time.Millisecond)
		broker.mu.Lock()
		for id := range broker.pending {
			broker.mu.Unlock()
			broker.Respond(UserInputResponse{
				RequestID: id,
				Value:     "user input text",
			})
			return
		}
		broker.mu.Unlock()
	}()

	text, err := broker.RequestText(context.Background(), "Enter name", "What is your name?", "John Doe", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "user input text" {
		t.Errorf("expected 'user input text', got %q", text)
	}
}

func TestUserInputBroker_RequestChoice(t *testing.T) {
	broker := NewUserInputBrokerWith(WithInputTimeout(100 * time.Millisecond))

	go func() {
		time.Sleep(10 * time.Millisecond)
		broker.mu.Lock()
		for id := range broker.pending {
			broker.mu.Unlock()
			broker.Respond(UserInputResponse{
				RequestID: id,
				Value:     "option2",
			})
			return
		}
		broker.mu.Unlock()
	}()

	choice, err := broker.RequestChoice(context.Background(), "Select", "Choose one", []string{"option1", "option2", "option3"}, "option1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choice != "option2" {
		t.Errorf("expected 'option2', got %q", choice)
	}
}

func TestUserInputBroker_Timeout(t *testing.T) {
	broker := NewUserInputBrokerWith(WithInputTimeout(50 * time.Millisecond))

	_, err := broker.RequestConfirm(context.Background(), "Timeout test", "Will timeout")
	if err == nil {
		t.Error("expected timeout error")
	}
	if err.Error() != "input request timeout" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestUserInputBroker_ContextCancellation(t *testing.T) {
	broker := NewUserInputBrokerWith(WithInputTimeout(5 * time.Second))

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err := broker.RequestConfirm(ctx, "Cancel test", "Will be cancelled")
	if err == nil {
		t.Error("expected cancellation error")
	}
}

func TestUserInputBroker_Cancelled(t *testing.T) {
	broker := NewUserInputBrokerWith(WithInputTimeout(100 * time.Millisecond))

	go func() {
		time.Sleep(10 * time.Millisecond)
		broker.mu.Lock()
		for id := range broker.pending {
			broker.mu.Unlock()
			broker.Respond(UserInputResponse{
				RequestID: id,
				Cancelled: true,
			})
			return
		}
		broker.mu.Unlock()
	}()

	confirmed, err := broker.RequestConfirm(context.Background(), "Test", "User will cancel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if confirmed {
		t.Error("expected confirmed to be false when cancelled")
	}
}

func TestUserInputBroker_PendingCount(t *testing.T) {
	broker := NewUserInputBroker()

	if broker.PendingCount() != 0 {
		t.Errorf("expected 0 pending, got %d", broker.PendingCount())
	}
	if broker.HasPending() {
		t.Error("expected HasPending to be false")
	}
}

func TestUserInputBroker_RespondNoRequest(t *testing.T) {
	broker := NewUserInputBroker()

	err := broker.Respond(UserInputResponse{
		RequestID: "nonexistent",
		Confirmed: true,
	})
	if err == nil {
		t.Error("expected error for nonexistent request")
	}
}

func TestUserInputBroker_OnInputSubmit(t *testing.T) {
	var submitted UserInputRequest
	broker := NewUserInputBrokerWith(
		WithInputTimeout(100*time.Millisecond),
		WithOnInputSubmit(func(req UserInputRequest) {
			submitted = req
		}),
	)

	go func() {
		time.Sleep(10 * time.Millisecond)
		broker.mu.Lock()
		for id := range broker.pending {
			broker.mu.Unlock()
			broker.Respond(UserInputResponse{
				RequestID: id,
				Confirmed: true,
			})
			return
		}
		broker.mu.Unlock()
	}()

	broker.RequestConfirm(context.Background(), "Test Title", "Test Message")

	if submitted.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got %q", submitted.Title)
	}
	if submitted.Message != "Test Message" {
		t.Errorf("expected message 'Test Message', got %q", submitted.Message)
	}
	if submitted.Type != InputTypeConfirm {
		t.Errorf("expected type InputTypeConfirm, got %q", submitted.Type)
	}
}
