package compress

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrotliRoundTrip(t *testing.T) {
	c := NewBrotli(-1)

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

func TestBrotliDecompressCorrupt(t *testing.T) {
	c := NewBrotli(-1)
	_, err := c.DecompressBytes([]byte("not brotli data at all"))
	assert.Error(t, err)
}

func TestBrotliID(t *testing.T) {
	c := NewBrotli(-1)
	assert.Equal(t, byte(0x02), c.ID())
	assert.Equal(t, "brotli", c.Name())
}

func FuzzBrotliDecompress(f *testing.F) {
	c := NewBrotli(-1)
	data := []byte("fuzz seed data for brotli")
	compressed, _ := c.CompressBytes(data)
	f.Add(compressed)
	f.Add([]byte{})
	f.Add([]byte("not brotli"))
	f.Fuzz(func(t *testing.T, b []byte) {
		c.DecompressBytes(b) //nolint:errcheck
	})
}
