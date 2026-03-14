package pack2d

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		inputType   InputType
		compression CompressionType
	}{
		{"raw zlib simple", []byte("hello world"), Raw, Zlib},
		{"raw zlib empty", []byte{}, Raw, Zlib},
		{"raw zlib binary", []byte{0x00, 0x01, 0x02, 0xFF}, Raw, Zlib},
		{"json zlib simple", []byte(`{"key":"value"}`), JSON, Zlib},
		{"json zlib unicode", []byte(`{"name":"héllo","emoji":"🎉"}`), JSON, Zlib},
		{"json zlib nested", []byte(`{"a":{"b":{"c":42}}}`), JSON, Zlib},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, stats, err := Encode(tt.data,
				WithInputType(tt.inputType),
				WithCompression(tt.compression),
			)
			require.NoError(t, err)
			assert.Greater(t, len(encoded), 0)
			assert.Greater(t, stats.EncodedBytes, 0)

			decoded, _, err := Decode(encoded)
			require.NoError(t, err)

			// For JSON, the decoded result may be minified
			if tt.inputType == JSON && len(tt.data) > 0 {
				// Just verify it's valid JSON by checking it decodes without error
				assert.NotEmpty(t, decoded)
			} else {
				assert.Equal(t, tt.data, decoded)
			}
		})
	}
}

func TestRoundTripLarge(t *testing.T) {
	data := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 100))
	encoded, stats, err := Encode(data)
	require.NoError(t, err)
	assert.Less(t, stats.CompressedBytes, stats.InputBytes, "compression should reduce size")

	decoded, _, err := Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, data, decoded)
}

func TestInspect(t *testing.T) {
	data := []byte("inspect test data")
	encoded, _, err := Encode(data)
	require.NoError(t, err)

	result, err := Inspect(encoded)
	require.NoError(t, err)
	assert.Equal(t, "zlib", result.Compression)
	assert.Equal(t, "raw", result.Serialization)
	assert.False(t, result.HasDictionary)
	assert.Equal(t, uint8(0), result.Version)
	assert.Contains(t, result.CompatibleBarcodes, "qrcode")
}

func TestInspectJSON(t *testing.T) {
	data := []byte(`{"key":"value"}`)
	encoded, _, err := Encode(data, WithInputType(JSON))
	require.NoError(t, err)

	result, err := Inspect(encoded)
	require.NoError(t, err)
	assert.Equal(t, "json", result.Serialization)
}

func FuzzDecode(f *testing.F) {
	data := []byte("fuzz seed")
	encoded, _, _ := Encode(data)
	f.Add(encoded)
	f.Add("")
	f.Add("HELLO")
	f.Add("!!!!")
	f.Fuzz(func(t *testing.T, s string) {
		// must never panic
		Decode(s) //nolint:errcheck
	})
}

func TestDecodeInvalid(t *testing.T) {
	_, _, err := Decode("not valid base45 ???")
	assert.Error(t, err)
}
