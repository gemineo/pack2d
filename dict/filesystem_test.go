package dict

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemStore(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFilesystemStore(dir)
	require.NoError(t, err)

	// Save with auto-assign ID
	d := &Dictionary{
		Name:        "test-dict",
		Description: "A test dictionary",
		Data:        []byte("sample dictionary data"),
		CreatedAt:   time.Now().UTC(),
		SampleCount: 5,
	}
	err = store.Save(d)
	require.NoError(t, err)
	assert.Equal(t, uint16(1), d.ID)

	// Get by ID
	got, err := store.Get(1)
	require.NoError(t, err)
	assert.Equal(t, "test-dict", got.Name)
	assert.Equal(t, []byte("sample dictionary data"), got.Data)
	assert.Equal(t, 5, got.SampleCount)

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

	// Save a second dictionary
	d2 := &Dictionary{
		Name:      "second-dict",
		Data:      []byte("second dict data"),
		CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, store.Save(d2))
	assert.Equal(t, uint16(2), d2.ID)

	list, err = store.List()
	require.NoError(t, err)
	assert.Len(t, list, 2)

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

func TestFilesystemStoreInvalidDir(t *testing.T) {
	_, err := NewFilesystemStore("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
}

func TestFilesystemStoreNotADirectory(t *testing.T) {
	// Pass a regular file path — must fail.
	f, err := os.CreateTemp("", "pack2d-notdir-*.txt")
	require.NoError(t, err)
	f.Close()
	defer os.Remove(f.Name())

	_, err = NewFilesystemStore(f.Name())
	assert.Error(t, err)
}

func TestFilesystemStoreCorruptMeta(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFilesystemStore(dir)
	require.NoError(t, err)

	// Write valid .dict + corrupt .json — Get should fail.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0001_corrupt.dict"), []byte("dictdata"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0001_corrupt.json"), []byte("{not valid json!!"), 0o644))

	_, err = store.Get(1)
	assert.Error(t, err)

	// List silently skips entries with corrupt metadata.
	list, err := store.List()
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestFilesystemStoreGetByNameMissingDictFile(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFilesystemStore(dir)
	require.NoError(t, err)

	// Write valid metadata but omit the .dict file.
	meta := `{"id":1,"name":"nodictfile","description":"","created_at":"2024-01-01T00:00:00Z","sample_count":0}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0001_nodictfile.json"), []byte(meta), 0o644))

	_, err = store.GetByName("nodictfile")
	assert.Error(t, err)
}

func TestFilesystemStoreNextIDSkipsNonNumericFilenames(t *testing.T) {
	dir := t.TempDir()
	// File with non-numeric prefix — NextID must skip it and return 1.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "abc_test.json"), []byte("{}"), 0o644))

	store, err := NewFilesystemStore(dir)
	require.NoError(t, err)
	id, err := store.NextID()
	require.NoError(t, err)
	assert.Equal(t, uint16(1), id)
}

func TestFilesystemStoreSaveExplicitID(t *testing.T) {
	// Save a dictionary with a non-zero ID — NextID must not be called.
	dir := t.TempDir()
	store, err := NewFilesystemStore(dir)
	require.NoError(t, err)

	d := &Dictionary{
		ID:        42,
		Name:      "explicit",
		Data:      []byte("data"),
		CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, store.Save(d))
	assert.Equal(t, uint16(42), d.ID)

	got, err := store.Get(42)
	require.NoError(t, err)
	assert.Equal(t, "explicit", got.Name)
}

func TestFilesystemStoreSaveReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses filesystem permissions")
	}
	dir := t.TempDir()
	store, err := NewFilesystemStore(dir)
	require.NoError(t, err)

	require.NoError(t, os.Chmod(dir, 0o555)) // read + execute, no write
	defer os.Chmod(dir, 0o755)               //nolint:errcheck

	d := &Dictionary{ID: 5, Name: "readonly", Data: []byte("data"), CreatedAt: time.Now().UTC()}
	err = store.Save(d)
	assert.Error(t, err)
}

func TestFilesystemStoreDeleteReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses filesystem permissions")
	}
	dir := t.TempDir()
	store, err := NewFilesystemStore(dir)
	require.NoError(t, err)

	d := &Dictionary{Name: "to-delete", Data: []byte("data"), CreatedAt: time.Now().UTC()}
	require.NoError(t, store.Save(d))

	require.NoError(t, os.Chmod(dir, 0o555)) // make read-only
	defer os.Chmod(dir, 0o755)               //nolint:errcheck

	err = store.Delete(d.ID)
	assert.Error(t, err)
}

func TestFilesystemStoreGetMissingMetaFile(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFilesystemStore(dir)
	require.NoError(t, err)

	// Write only the .dict file, no .json metadata.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "0001_test.dict"), []byte("data"), 0o644))

	// findFiles will find the .dict and infer the .json path.
	// loadMeta will fail because the .json doesn't exist.
	_, err = store.Get(1)
	assert.Error(t, err)
}
