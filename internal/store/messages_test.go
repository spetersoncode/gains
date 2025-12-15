package store

import (
	"context"
	"sync"
	"testing"

	ai "github.com/spetersoncode/gains"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageStore_Append(t *testing.T) {
	ms := NewMessageStore(nil)

	assert.Equal(t, 0, ms.Len())

	ms.Append(ai.Message{Role: ai.RoleUser, Content: "Hello"})
	assert.Equal(t, 1, ms.Len())

	ms.Append(
		ai.Message{Role: ai.RoleAssistant, Content: "Hi there"},
		ai.Message{Role: ai.RoleUser, Content: "How are you?"},
	)
	assert.Equal(t, 3, ms.Len())
}

func TestMessageStore_Messages(t *testing.T) {
	ms := NewMessageStore(nil)

	ms.Append(
		ai.Message{Role: ai.RoleUser, Content: "Hello"},
		ai.Message{Role: ai.RoleAssistant, Content: "Hi"},
	)

	messages := ms.Messages()
	assert.Len(t, messages, 2)
	assert.Equal(t, "Hello", messages[0].Content)
	assert.Equal(t, "Hi", messages[1].Content)

	// Verify it's a copy - modifying returned slice doesn't affect store
	messages[0].Content = "Modified"
	storeMessages := ms.Messages()
	assert.Equal(t, "Hello", storeMessages[0].Content)
}

func TestMessageStore_Clear(t *testing.T) {
	ms := NewMessageStore(nil)

	ms.Append(
		ai.Message{Role: ai.RoleUser, Content: "Hello"},
		ai.Message{Role: ai.RoleAssistant, Content: "Hi"},
	)

	ms.Clear()
	assert.Equal(t, 0, ms.Len())
	assert.Empty(t, ms.Messages())
}

func TestMessageStore_Clone(t *testing.T) {
	ms := NewMessageStore(nil)

	ms.Append(
		ai.Message{Role: ai.RoleUser, Content: "Hello"},
		ai.Message{Role: ai.RoleAssistant, Content: "Hi"},
	)

	clone := ms.Clone()

	// Clone has same messages
	assert.Equal(t, 2, clone.Len())
	assert.Equal(t, "Hello", clone.Messages()[0].Content)

	// Modifying original doesn't affect clone
	ms.Append(ai.Message{Role: ai.RoleUser, Content: "New"})
	assert.Equal(t, 3, ms.Len())
	assert.Equal(t, 2, clone.Len())

	// Modifying clone doesn't affect original
	clone.Clear()
	assert.Equal(t, 3, ms.Len())
}

func TestMessageStore_Last(t *testing.T) {
	ms := NewMessageStore(nil)

	ms.Append(
		ai.Message{Role: ai.RoleUser, Content: "1"},
		ai.Message{Role: ai.RoleAssistant, Content: "2"},
		ai.Message{Role: ai.RoleUser, Content: "3"},
		ai.Message{Role: ai.RoleAssistant, Content: "4"},
	)

	// Get last 2
	last := ms.Last(2)
	assert.Len(t, last, 2)
	assert.Equal(t, "3", last[0].Content)
	assert.Equal(t, "4", last[1].Content)

	// Get more than available
	all := ms.Last(10)
	assert.Len(t, all, 4)

	// Get 0 or negative
	assert.Nil(t, ms.Last(0))
	assert.Nil(t, ms.Last(-1))
}

func TestMessageStore_NewFrom(t *testing.T) {
	initial := []ai.Message{
		{Role: ai.RoleUser, Content: "Hello"},
		{Role: ai.RoleAssistant, Content: "Hi"},
	}

	ms := NewMessageStoreFrom(initial, nil)

	assert.Equal(t, 2, ms.Len())
	assert.Equal(t, "Hello", ms.Messages()[0].Content)

	// Verify it's a copy
	initial[0].Content = "Modified"
	assert.Equal(t, "Hello", ms.Messages()[0].Content)
}

func TestMessageStore_SyncReload(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	// Create and sync
	ms1 := NewMessageStore(adapter)
	ms1.Append(
		ai.Message{Role: ai.RoleUser, Content: "Hello"},
		ai.Message{Role: ai.RoleAssistant, Content: "Hi there"},
	)
	require.NoError(t, ms1.Sync(ctx, "conversation"))

	// Create new store with same adapter and reload
	ms2 := NewMessageStore(adapter)
	require.NoError(t, ms2.Reload(ctx, "conversation"))

	assert.Equal(t, 2, ms2.Len())
	messages := ms2.Messages()
	assert.Equal(t, ai.RoleUser, messages[0].Role)
	assert.Equal(t, "Hello", messages[0].Content)
	assert.Equal(t, ai.RoleAssistant, messages[1].Role)
	assert.Equal(t, "Hi there", messages[1].Content)
}

func TestMessageStore_ReloadNotFound(t *testing.T) {
	ctx := context.Background()
	ms := NewMessageStore(nil)

	err := ms.Reload(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestMessageStore_Concurrent(t *testing.T) {
	ms := NewMessageStore(nil)
	var wg sync.WaitGroup

	// Concurrent appends
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ms.Append(ai.Message{Role: ai.RoleUser, Content: "msg"})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ms.Messages()
		}()
	}

	wg.Wait()
	assert.Equal(t, 100, ms.Len())
}

func TestMessageStore_EmptyAppend(t *testing.T) {
	ms := NewMessageStore(nil)

	// Empty append should be a no-op
	ms.Append()
	assert.Equal(t, 0, ms.Len())
}
