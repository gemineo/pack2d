package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackUnpackHeader_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		h    Header
	}{
		{"v0 zlib no-dict raw", Header{0, 0x00, false, 0, 0x00}},
		{"v0 zlib no-dict json", Header{0, 0x00, false, 0, 0x01}},
		{"v0 zlib no-dict xml", Header{0, 0x00, false, 0, 0x02}},
		{"v0 zlib no-dict cbor", Header{0, 0x00, false, 0, 0x03}},
		{"v0 zstd no-dict raw", Header{0, 0x01, false, 0, 0x00}},
		{"v0 brotli no-dict raw", Header{0, 0x02, false, 0, 0x00}},
		{"v0 zlib dict raw", Header{0, 0x00, true, 1, 0x00}},
		{"v0 zlib dict json dict=42", Header{0, 0x00, true, 42, 0x01}},
		{"v0 zstd dict xml dict=65535", Header{0, 0x01, true, 65535, 0x02}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packed := PackHeader(tt.h)
			got, n, err := UnpackHeader(packed)
			require.NoError(t, err)
			assert.Equal(t, tt.h, got)
			if tt.h.Dictionary {
				assert.Equal(t, 3, n)
			} else {
				assert.Equal(t, 1, n)
			}
		})
	}
}

func TestUnpackHeader_Errors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty slice", []byte{}},
		{"extension marker 0xFF", []byte{0xFF}},
		{"non-zero version bits", []byte{0x80}},
		{"truncated dict header", []byte{0x08}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := UnpackHeader(tt.data)
			assert.Error(t, err)
		})
	}
}

func FuzzUnpackHeader(f *testing.F) {
	f.Add([]byte{0x00})
	f.Add([]byte{0xFF})
	f.Add([]byte{0x08, 0x00, 0x01})
	f.Fuzz(func(t *testing.T, data []byte) {
		// must never panic
		UnpackHeader(data) //nolint:errcheck
	})
}
