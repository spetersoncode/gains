package store

import (
	"context"
	"encoding/json"
	"sync"

	ai "github.com/spetersoncode/gains"
)

// MessageStore manages conversation history with persistence support.
type MessageStore struct {
	mu       sync.RWMutex
	messages []ai.Message
	adapter  Adapter
}

// NewMessageStore creates a new MessageStore with the given adapter.
// If adapter is nil, a default in-memory adapter is used.
func NewMessageStore(adapter Adapter) *MessageStore {
	if adapter == nil {
		adapter = NewMemoryAdapter()
	}
	return &MessageStore{
		messages: make([]ai.Message, 0),
		adapter:  adapter,
	}
}

// NewMessageStoreFrom creates a MessageStore initialized with existing messages.
func NewMessageStoreFrom(messages []ai.Message, adapter Adapter) *MessageStore {
	ms := NewMessageStore(adapter)
	if len(messages) > 0 {
		ms.messages = make([]ai.Message, len(messages))
		copy(ms.messages, messages)
	}
	return ms
}

// Messages returns a copy of all messages.
func (m *MessageStore) Messages() []ai.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]ai.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// Append adds messages to the store.
func (m *MessageStore) Append(msgs ...ai.Message) {
	if len(msgs) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msgs...)
}

// Len returns the number of messages.
func (m *MessageStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.messages)
}

// Clear removes all messages.
func (m *MessageStore) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([]ai.Message, 0)
}

// Clone creates a deep copy of the MessageStore.
func (m *MessageStore) Clone() *MessageStore {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clone := NewMessageStore(nil)
	if len(m.messages) > 0 {
		clone.messages = make([]ai.Message, len(m.messages))
		copy(clone.messages, m.messages)
	}
	return clone
}

// Last returns the last n messages. If n > Len(), returns all messages.
func (m *MessageStore) Last(n int) []ai.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if n <= 0 {
		return nil
	}

	start := len(m.messages) - n
	if start < 0 {
		start = 0
	}

	result := make([]ai.Message, len(m.messages)-start)
	copy(result, m.messages[start:])
	return result
}

// Sync persists the messages to the adapter under the given key.
func (m *MessageStore) Sync(ctx context.Context, key string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	raw, err := json.Marshal(m.messages)
	if err != nil {
		return &SerializationError{Key: key, Err: err}
	}
	return m.adapter.Set(ctx, key, raw)
}

// Reload loads messages from the adapter using the given key.
func (m *MessageStore) Reload(ctx context.Context, key string) error {
	raw, ok, err := m.adapter.Get(ctx, key)
	if err != nil {
		return err
	}
	if !ok {
		return ErrKeyNotFound
	}

	var messages []ai.Message
	if err := json.Unmarshal(raw, &messages); err != nil {
		return &SerializationError{Key: key, Err: err}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = messages
	return nil
}

// Adapter returns the underlying adapter.
func (m *MessageStore) Adapter() Adapter {
	return m.adapter
}
