package store

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testState struct {
	Name  string   `json:"name"`
	Count int      `json:"count"`
	Items []string `json:"items"`
}

func TestTypedStore_GetSet(t *testing.T) {
	s := NewTyped(testState{Name: "initial"}, nil)

	state := s.Get()
	assert.Equal(t, "initial", state.Name)
	assert.Equal(t, 0, state.Count)

	s.Set(testState{Name: "updated", Count: 42})

	state = s.Get()
	assert.Equal(t, "updated", state.Name)
	assert.Equal(t, 42, state.Count)
}

func TestTypedStore_Update(t *testing.T) {
	s := NewTyped(testState{Items: []string{"a"}}, nil)

	s.Update(func(state *testState) {
		state.Items = append(state.Items, "b", "c")
		state.Count = len(state.Items)
	})

	state := s.Get()
	assert.Equal(t, []string{"a", "b", "c"}, state.Items)
	assert.Equal(t, 3, state.Count)
}

func TestTypedStore_Clone(t *testing.T) {
	s := NewTyped(testState{
		Name:  "original",
		Count: 10,
		Items: []string{"a", "b"},
	}, nil)

	clone, err := s.Clone()
	require.NoError(t, err)

	// Modify original
	s.Set(testState{Name: "modified", Count: 20})

	// Clone should be unchanged
	cloneState := clone.Get()
	assert.Equal(t, "original", cloneState.Name)
	assert.Equal(t, 10, cloneState.Count)
	assert.Equal(t, []string{"a", "b"}, cloneState.Items)

	// Original should be modified
	origState := s.Get()
	assert.Equal(t, "modified", origState.Name)
}

func TestTypedStore_SyncReload(t *testing.T) {
	ctx := context.Background()
	adapter := NewMemoryAdapter()

	// Create and sync
	s1 := NewTyped(testState{Name: "Alice", Count: 42, Items: []string{"x", "y"}}, adapter)
	require.NoError(t, s1.Sync(ctx, "state"))

	// Create new store with same adapter and reload
	s2 := NewTyped(testState{}, adapter)
	require.NoError(t, s2.Reload(ctx, "state"))

	state := s2.Get()
	assert.Equal(t, "Alice", state.Name)
	assert.Equal(t, 42, state.Count)
	assert.Equal(t, []string{"x", "y"}, state.Items)
}

func TestTypedStore_ReloadNotFound(t *testing.T) {
	ctx := context.Background()
	s := NewTyped(testState{}, nil)

	err := s.Reload(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestTypedStore_Concurrent(t *testing.T) {
	s := NewTyped(testState{Count: 0}, nil)
	var wg sync.WaitGroup

	// Concurrent updates
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.Update(func(state *testState) {
				state.Count = n
			})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Get()
		}()
	}

	wg.Wait()
	// Just verify it didn't panic and has some value
	state := s.Get()
	assert.GreaterOrEqual(t, state.Count, 0)
}

func TestTypedStore_WithPrimitiveType(t *testing.T) {
	// TypedStore works with any type
	s := NewTyped([]int{1, 2, 3}, nil)

	slice := s.Get()
	assert.Equal(t, []int{1, 2, 3}, slice)

	s.Update(func(v *[]int) {
		*v = append(*v, 4, 5)
	})

	assert.Equal(t, []int{1, 2, 3, 4, 5}, s.Get())
}
