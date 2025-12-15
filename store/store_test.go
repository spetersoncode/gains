package store

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_GetSet(t *testing.T) {
	s := New(nil)

	s.Set("key1", "value1")
	s.Set("key2", 42)
	s.Set("key3", true)

	v1, ok := s.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", v1)

	v2, ok := s.Get("key2")
	assert.True(t, ok)
	assert.Equal(t, 42, v2)

	_, ok = s.Get("nonexistent")
	assert.False(t, ok)
}

func TestStore_TypedGetters(t *testing.T) {
	s := New(nil)
	s.Set("str", "hello")
	s.Set("num", 123)
	s.Set("float", 3.14)
	s.Set("bool", true)

	// String
	assert.Equal(t, "hello", s.GetString("str"))
	assert.Equal(t, "", s.GetString("num"))
	assert.Equal(t, "", s.GetString("nonexistent"))

	// Int
	assert.Equal(t, 123, s.GetInt("num"))
	assert.Equal(t, 3, s.GetInt("float")) // truncated
	assert.Equal(t, 0, s.GetInt("nonexistent"))

	// Float
	assert.Equal(t, 3.14, s.GetFloat("float"))
	assert.Equal(t, 123.0, s.GetFloat("num"))
	assert.Equal(t, 0.0, s.GetFloat("nonexistent"))

	// Bool
	assert.True(t, s.GetBool("bool"))
	assert.False(t, s.GetBool("str"))
	assert.False(t, s.GetBool("nonexistent"))
}

func TestStore_Delete(t *testing.T) {
	s := New(nil)
	s.Set("key1", "value1")

	assert.True(t, s.Has("key1"))
	s.Delete("key1")
	assert.False(t, s.Has("key1"))

	// Delete non-existent (should not panic)
	s.Delete("nonexistent")
}

func TestStore_Has(t *testing.T) {
	s := New(nil)

	assert.False(t, s.Has("key1"))
	s.Set("key1", "value1")
	assert.True(t, s.Has("key1"))
}

func TestStore_Keys(t *testing.T) {
	s := New(nil)

	assert.Empty(t, s.Keys())

	s.Set("key1", "v1")
	s.Set("key2", "v2")

	keys := s.Keys()
	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "key1")
	assert.Contains(t, keys, "key2")
}

func TestStore_Len(t *testing.T) {
	s := New(nil)

	assert.Equal(t, 0, s.Len())

	s.Set("key1", "v1")
	s.Set("key2", "v2")

	assert.Equal(t, 2, s.Len())
}

func TestStore_Clone(t *testing.T) {
	s := New(nil)
	s.Set("key1", "value1")
	s.Set("key2", 42)

	clone := s.Clone()

	// Clone has same values
	assert.Equal(t, "value1", clone.GetString("key1"))
	assert.Equal(t, 42, clone.GetInt("key2"))

	// Modifying original doesn't affect clone
	s.Set("key1", "modified")
	assert.Equal(t, "value1", clone.GetString("key1"))

	// Modifying clone doesn't affect original
	clone.Set("key2", 100)
	assert.Equal(t, 42, s.GetInt("key2"))
}

func TestStore_Merge(t *testing.T) {
	s1 := New(nil)
	s1.Set("key1", "value1")
	s1.Set("shared", "from_s1")

	s2 := New(nil)
	s2.Set("key2", "value2")
	s2.Set("shared", "from_s2")

	s1.Merge(s2)

	assert.Equal(t, "value1", s1.GetString("key1"))
	assert.Equal(t, "value2", s1.GetString("key2"))
	assert.Equal(t, "from_s2", s1.GetString("shared")) // overwritten

	// Merge nil should be safe
	s1.Merge(nil)
	assert.Equal(t, 3, s1.Len())
}

func TestStore_Data(t *testing.T) {
	s := New(nil)
	s.Set("key1", "value1")
	s.Set("key2", 42)

	data := s.Data()

	assert.Len(t, data, 2)
	assert.Equal(t, "value1", data["key1"])
	assert.Equal(t, 42, data["key2"])

	// Modifying returned data doesn't affect store
	data["key1"] = "modified"
	assert.Equal(t, "value1", s.GetString("key1"))
}

func TestStore_NewFrom(t *testing.T) {
	data := map[string]any{
		"key1": "value1",
		"key2": 42,
	}

	s := NewFrom(data)

	assert.Equal(t, "value1", s.GetString("key1"))
	assert.Equal(t, 42, s.GetInt("key2"))
}

func TestStore_SyncReload(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	// Create and sync
	s1 := New(adapter)
	s1.Set("name", "Alice")
	s1.Set("age", 30)
	require.NoError(t, s1.Sync(ctx))

	// Create new store with same adapter and reload
	s2 := New(adapter)
	require.NoError(t, s2.Reload(ctx))

	assert.Equal(t, "Alice", s2.GetString("name"))
	// Note: JSON unmarshals numbers as float64
	assert.Equal(t, 30, s2.GetInt("age"))
}

func TestStore_Concurrent(t *testing.T) {
	s := New(nil)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.Set("key", n)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Get("key")
		}()
	}

	wg.Wait()
	assert.True(t, s.Has("key"))
}

func TestStore_IntConversions(t *testing.T) {
	s := New(nil)

	// Test various int types
	s.Set("int", int(42))
	s.Set("int32", int32(42))
	s.Set("int64", int64(42))
	s.Set("float32", float32(42.5))
	s.Set("float64", float64(42.9))

	assert.Equal(t, 42, s.GetInt("int"))
	assert.Equal(t, 42, s.GetInt("int32"))
	assert.Equal(t, 42, s.GetInt("int64"))
	assert.Equal(t, 42, s.GetInt("float32"))
	assert.Equal(t, 42, s.GetInt("float64"))
}
