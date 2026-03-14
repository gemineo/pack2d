package compress

import (
	"bytes"
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZlibRoundTrip(t *testing.T) {
	c := NewZlib(-1)
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

func TestZlibDecompressCorrupt(t *testing.T) {
	c := NewZlib(-1)
	_, err := c.DecompressBytes([]byte("not zlib data at all"))
	assert.Error(t, err)
}

func BenchmarkCompress(b *testing.B) {
	c := NewZlib(-1)
	data := []byte(strings.Repeat("benchmark data payload\n", 1000))
	b.ResetTimer()
	for b.Loop() {
		_, _ = c.CompressBytes(data)
	}
}

func BenchmarkDecompress(b *testing.B) {
	c := NewZlib(-1)
	data := []byte(strings.Repeat("benchmark data payload\n", 1000))
	compressed, _ := c.CompressBytes(data)
	b.ResetTimer()
	for b.Loop() {
		_, _ = c.DecompressBytes(compressed)
	}
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b) //nolint:gosec
	return b
}

func TestZlibStream(t *testing.T) {
	c := NewZlib(-1)
	data := []byte("stream round trip test")
	var compressed bytes.Buffer
	err := c.Compress(&compressed, bytes.NewReader(data))
	require.NoError(t, err)
	var decompressed bytes.Buffer
	err = c.Decompress(&decompressed, &compressed)
	require.NoError(t, err)
	assert.Equal(t, data, decompressed.Bytes())
}
