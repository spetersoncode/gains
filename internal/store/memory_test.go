package store

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryAdapter_GetSet(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	// Set a value
	err := adapter.Set(ctx, "key1", json.RawMessage(`"value1"`))
	require.NoError(t, err)

	// Get the value
	raw, ok, err := adapter.Get(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, json.RawMessage(`"value1"`), raw)

	// Get non-existent key
	_, ok, err = adapter.Get(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestMemoryAdapter_Delete(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	// Set and delete
	err := adapter.Set(ctx, "key1", json.RawMessage(`"value1"`))
	require.NoError(t, err)

	err = adapter.Delete(ctx, "key1")
	require.NoError(t, err)

	_, ok, err := adapter.Get(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, ok)

	// Delete non-existent key (should not error)
	err = adapter.Delete(ctx, "nonexistent")
	require.NoError(t, err)
}

func TestMemoryAdapter_Has(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	has, err := adapter.Has(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, has)

	err = adapter.Set(ctx, "key1", json.RawMessage(`"value1"`))
	require.NoError(t, err)

	has, err = adapter.Has(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, has)
}

func TestMemoryAdapter_Keys(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	keys, err := adapter.Keys(ctx)
	require.NoError(t, err)
	assert.Empty(t, keys)

	_ = adapter.Set(ctx, "key1", json.RawMessage(`"v1"`))
	_ = adapter.Set(ctx, "key2", json.RawMessage(`"v2"`))

	keys, err = adapter.Keys(ctx)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
}

func TestMemoryAdapter_Len(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	length, err := adapter.Len(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, length)

	_ = adapter.Set(ctx, "key1", json.RawMessage(`"v1"`))
	_ = adapter.Set(ctx, "key2", json.RawMessage(`"v2"`))

	length, err = adapter.Len(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, length)
}

func TestMemoryAdapter_Clear(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	_ = adapter.Set(ctx, "key1", json.RawMessage(`"v1"`))
	_ = adapter.Set(ctx, "key2", json.RawMessage(`"v2"`))

	err := adapter.Clear(ctx)
	require.NoError(t, err)

	length, _ := adapter.Len(ctx)
	assert.Equal(t, 0, length)
}

func TestMemoryAdapter_LoadSave(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	// Save data
	data := map[string]json.RawMessage{
		"key1": json.RawMessage(`"value1"`),
		"key2": json.RawMessage(`42`),
	}
	err := adapter.Save(ctx, data)
	require.NoError(t, err)

	// Load data
	loaded, err := adapter.Load(ctx)
	require.NoError(t, err)
	assert.Len(t, loaded, 2)
	assert.Equal(t, json.RawMessage(`"value1"`), loaded["key1"])
	assert.Equal(t, json.RawMessage(`42`), loaded["key2"])
}

func TestMemoryAdapter_Concurrent(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = adapter.Set(ctx, "key", json.RawMessage(`"value"`))
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = adapter.Get(ctx, "key")
		}()
	}

	wg.Wait()

	has, _ := adapter.Has(ctx, "key")
	assert.True(t, has)
}
