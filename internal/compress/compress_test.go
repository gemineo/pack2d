package compress

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Registry

func TestRegistryGetByID(t *testing.T) {
	r := NewRegistry()
	r.Register(NewZlib(-1))

	c, err := r.Get(0x00)
	require.NoError(t, err)
	assert.Equal(t, "zlib", c.Name())

	_, err = r.Get(0xFF)
	assert.ErrorIs(t, err, ErrUnknownCompressor)
}

func TestRegistryGetByName(t *testing.T) {
	r := NewRegistry()
	r.Register(NewZlib(-1))

	c, err := r.GetByName("zlib")
	require.NoError(t, err)
	assert.Equal(t, byte(0x00), c.ID())

	_, err = r.GetByName("notexist")
	assert.ErrorIs(t, err, ErrUnknownCompressor)
}

func TestRegistryRegisterOverwrites(t *testing.T) {
	r := NewRegistry()
	r.Register(NewZlib(1))
	r.Register(NewZlib(9)) // same ID and name — must overwrite without panic

	c, err := r.Get(0x00)
	require.NoError(t, err)
	assert.Equal(t, "zlib", c.Name())
}

func TestDefaultRegistryContainsBrotliZstdZlib(t *testing.T) {
	r := DefaultRegistry()
	for _, tt := range []struct {
		id   byte
		name string
	}{
		{0x00, "zlib"},
		{0x01, "zstd"},
		{0x02, "brotli"},
	} {
		c, err := r.Get(tt.id)
		require.NoError(t, err, "id=0x%02X", tt.id)
		assert.Equal(t, tt.name, c.Name())

		c, err = r.GetByName(tt.name)
		require.NoError(t, err, "name=%q", tt.name)
		assert.Equal(t, tt.id, c.ID())
	}

	_, err := r.Get(0x03) // reserved — must not exist
	assert.ErrorIs(t, err, ErrUnknownCompressor)
}

// Zlib

func TestZlibIDName(t *testing.T) {
	z := NewZlib(-1)
	assert.Equal(t, byte(0x00), z.ID())
	assert.Equal(t, "zlib", z.Name())
}

func TestZlibLevelClamping(t *testing.T) {
	data := []byte("level clamping round-trip")
	for _, tt := range []struct {
		level int
		desc  string
	}{
		{-2, "below min → 0"},
		{-1, "default"},
		{0, "no compression"},
		{5, "mid"},
		{9, "best"},
		{100, "above max → 9"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			z := NewZlib(tt.level)
			got := roundTrip(t, z, data)
			assert.Equal(t, data, got)
		})
	}
}

func TestZlibStreamAPI(t *testing.T) {
	z := NewZlib(-1)
	data := []byte("stream API test")

	var buf bytes.Buffer
	require.NoError(t, z.Compress(&buf, bytes.NewReader(data)))

	var out bytes.Buffer
	require.NoError(t, z.Decompress(&out, &buf))
	assert.Equal(t, data, out.Bytes())
}

// Zstd

func TestZstdLevelClamping(t *testing.T) {
	data := []byte("zstd level clamping round-trip test data")
	for _, level := range []int{-1, 0, 1, 2, 3, 4, 5, 100} {
		t.Run(fmt.Sprintf("level_%d", level), func(t *testing.T) {
			c, err := NewZstd(level, nil)
			require.NoError(t, err)
			got := roundTrip(t, c, data)
			assert.Equal(t, data, got)
		})
	}
}

func TestZstdStreamAPI(t *testing.T) {
	c, err := NewZstd(-1, nil)
	require.NoError(t, err)
	data := []byte("zstd stream API test")

	var buf bytes.Buffer
	require.NoError(t, c.Compress(&buf, bytes.NewReader(data)))

	var out bytes.Buffer
	require.NoError(t, c.Decompress(&out, &buf))
	assert.Equal(t, data, out.Bytes())
}

// Brotli

func TestBrotliIDName(t *testing.T) {
	b := NewBrotli(-1)
	assert.Equal(t, byte(0x02), b.ID())
	assert.Equal(t, "brotli", b.Name())
}

func TestBrotliLevelClamping(t *testing.T) {
	data := []byte("brotli level clamping round-trip test data")
	for _, tt := range []struct {
		level int
		desc  string
	}{
		{-2, "negative → 0"},
		{-1, "default → 6"},
		{0, "minimum"},
		{6, "mid"},
		{11, "maximum"},
		{12, "above max → 11"},
		{100, "way above max → 11"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			c := NewBrotli(tt.level)
			got := roundTrip(t, c, data)
			assert.Equal(t, data, got)
		})
	}
}

func TestBrotliStreamAPI(t *testing.T) {
	c := NewBrotli(-1)
	data := []byte("brotli stream API test")

	var buf bytes.Buffer
	require.NoError(t, c.Compress(&buf, bytes.NewReader(data)))

	var out bytes.Buffer
	require.NoError(t, c.Decompress(&out, &buf))
	assert.Equal(t, data, out.Bytes())
}

// roundTrip is a helper: compress then decompress, return the result.
func roundTrip(t *testing.T, c Compressor, data []byte) []byte {
	t.Helper()
	compressed, err := c.CompressBytes(data)
	require.NoError(t, err)
	got, err := c.DecompressBytes(compressed)
	require.NoError(t, err)
	return got
}

// errWriter always returns an error on Write — used to inject dst write failures.
type errWriter struct{}

func (e errWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("injected write error") }

// errReader always returns an error on Read — used to inject src read failures.
type errReader struct{}

func (e errReader) Read([]byte) (int, error) { return 0, fmt.Errorf("injected read error") }

// Zlib error paths

func TestZlibCompressCloseError(t *testing.T) {
	// errWriter causes zlib to fail when it tries to flush on Close.
	z := NewZlib(-1)
	err := z.Compress(errWriter{}, bytes.NewReader([]byte("hello world")))
	assert.Error(t, err)
}

func TestZlibDecompressNewReaderError(t *testing.T) {
	// Truncated input (1 byte) makes zlib.NewReader fail reading the header.
	z := NewZlib(-1)
	_, err := z.DecompressBytes([]byte{0x78}) // incomplete zlib header
	assert.Error(t, err)
}

// Zstd error paths

func TestZstdCompressCloseError(t *testing.T) {
	// errWriter causes the zstd encoder to fail when flushing on Close.
	c, err := NewZstd(-1, nil)
	require.NoError(t, err)
	err = c.Compress(errWriter{}, bytes.NewReader([]byte("hello world")))
	assert.Error(t, err)
}

func TestZstdDecompressResetError(t *testing.T) {
	// errReader causes z.dec.Reset to fail because it cannot read the frame header.
	c, err := NewZstd(-1, nil)
	require.NoError(t, err)
	err = c.Decompress(&bytes.Buffer{}, errReader{})
	assert.Error(t, err)
}

// Brotli error paths

func TestBrotliCompressCloseError(t *testing.T) {
	// errWriter causes brotli to fail when flushing on Close.
	b := NewBrotli(-1)
	err := b.Compress(errWriter{}, bytes.NewReader([]byte("hello world")))
	assert.Error(t, err)
}

func TestBrotliCompressReadError(t *testing.T) {
	// errReader causes io.Copy to fail during brotli compression.
	b := NewBrotli(-1)
	var buf bytes.Buffer
	err := b.Compress(&buf, errReader{})
	assert.Error(t, err)
}
