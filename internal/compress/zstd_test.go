package compress

import (
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildZstdDict(samples [][]byte) ([]byte, error) {
	return zstd.BuildDict(zstd.BuildDictOptions{Contents: samples})
}

func TestZstdRoundTrip(t *testing.T) {
	c, err := NewZstd(-1, nil)
	require.NoError(t, err)

	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"single byte", []byte{0x42}},
		{"1KB JSON", []byte(strings.Repeat(`{"key":"value","num":42}`, 40))},
		{"100KB repeated", []byte(strings.Repeat("hello world\n", 9000))},
		{"random bytes", randomBytes(1024)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := c.CompressBytes(tt.data)
			require.NoError(t, err)
			got, err := c.DecompressBytes(compressed)
			require.NoError(t, err)
			assert.Equal(t, tt.data, got)
		})
	}
}

func TestZstdDecompressCorrupt(t *testing.T) {
	c, err := NewZstd(-1, nil)
	require.NoError(t, err)
	_, err = c.DecompressBytes([]byte("not zstd data at all"))
	assert.Error(t, err)
}

func TestZstdWithDictionaryRoundTrip(t *testing.T) {
	// Build a proper zstd dictionary from sample data (requires magic number).
	samples := make([][]byte, 20)
	for i := range samples {
		samples[i] = []byte(strings.Repeat("patient id status active record value", 5))
	}
	dictBytes, err := buildTestDict(samples)
	if err != nil {
		t.Skipf("cannot build test dictionary: %v", err)
	}

	enc, err := NewZstd(-1, dictBytes)
	require.NoError(t, err)
	dec, err := NewZstd(-1, dictBytes)
	require.NoError(t, err)

	data := []byte(`{"patient":"John","id":"12345","status":"active"}`)
	compressed, err := enc.CompressBytes(data)
	require.NoError(t, err)
	got, err := dec.DecompressBytes(compressed)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

// buildTestDict builds a zstd dictionary using the klauspost/compress library.
func buildTestDict(samples [][]byte) ([]byte, error) {
	return buildZstdDict(samples)
}

func TestZstdID(t *testing.T) {
	c, err := NewZstd(-1, nil)
	require.NoError(t, err)
	assert.Equal(t, byte(0x01), c.ID())
	assert.Equal(t, "zstd", c.Name())
}

func FuzzZstdDecompress(f *testing.F) {
	c, _ := NewZstd(-1, nil)
	data := []byte("fuzz seed data for zstd")
	compressed, _ := c.CompressBytes(data)
	f.Add(compressed)
	f.Add([]byte{})
	f.Add([]byte("not zstd"))
	f.Fuzz(func(t *testing.T, b []byte) {
		c.DecompressBytes(b) //nolint:errcheck
	})
}
