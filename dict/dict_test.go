package dict

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStoreNextIDExhausted(t *testing.T) {
	store := NewMemoryStore()
	// Save a dictionary with the max ID
	d := &Dictionary{
		ID:   ^uint16(0), // 65535
		Name: "max-id",
		Data: []byte("data"),
	}
	require.NoError(t, store.Save(d))

	_, err := store.NextID()
	assert.ErrorIs(t, err, ErrIDExhausted)

	// Auto-assign should also fail
	d2 := &Dictionary{Name: "overflow", Data: []byte("data")}
	err = store.Save(d2)
	assert.ErrorIs(t, err, ErrIDExhausted)
}

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()

	// Save with auto-assign ID
	d := &Dictionary{
		Name:        "test-dict",
		Description: "A test dictionary",
		Data:        []byte("sample data"),
		CreatedAt:   time.Now(),
		SampleCount: 10,
	}
	err := store.Save(d)
	require.NoError(t, err)
	assert.Equal(t, uint16(1), d.ID)

	// Get by ID
	got, err := store.Get(1)
	require.NoError(t, err)
	assert.Equal(t, "test-dict", got.Name)
	assert.Equal(t, []byte("sample data"), got.Data)

	// Ensure deep copy (mutating returned value doesn't affect store)
	got.Data[0] = 0xFF
	got2, err := store.Get(1)
	require.NoError(t, err)
	assert.Equal(t, byte('s'), got2.Data[0])

	// GetByName
	got, err = store.GetByName("test-dict")
	require.NoError(t, err)
	assert.Equal(t, uint16(1), got.ID)

	// List
	list, err := store.List()
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// NextID
	nextID, err := store.NextID()
	require.NoError(t, err)
	assert.Equal(t, uint16(2), nextID)

	// Delete
	err = store.Delete(1)
	require.NoError(t, err)
	_, err = store.Get(1)
	assert.Error(t, err)

	// Not found errors
	_, err = store.Get(999)
	assert.Error(t, err)
	_, err = store.GetByName("nonexistent")
	assert.Error(t, err)
	err = store.Delete(999)
	assert.Error(t, err)
}
