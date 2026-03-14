package encoding

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RFC 9285 test vectors
func TestBase45RFC9285Vectors(t *testing.T) {
	tests := []struct {
		decoded string
		encoded string
	}{
		{"AB", "BB8"},
		{"Hello!!", "%69 VD92EX0"},
		{"base-45", "UJCLQE7W581"},
		{"ietf!", "QED8WEX0"},
	}
	for _, tt := range tests {
		t.Run(tt.decoded, func(t *testing.T) {
			encoded := Base45Encode([]byte(tt.decoded))
			assert.Equal(t, tt.encoded, encoded)

			decoded, err := Base45Decode(tt.encoded)
			require.NoError(t, err)
			assert.Equal(t, []byte(tt.decoded), decoded)
		})
	}
}

func TestBase45RoundTrip(t *testing.T) {
	tests := [][]byte{
		{},
		{0x00},
		{0xFF},
		[]byte("hello world"),
		[]byte("pack2d encoding test payload"),
	}
	for _, data := range tests {
		encoded := Base45Encode(data)
		decoded, err := Base45Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, data, decoded)
	}
}

func TestBase45DecodeInvalid(t *testing.T) {
	// Characters outside Base45 alphabet
	_, err := Base45Decode("???invalid???")
	assert.Error(t, err)
}

func FuzzBase45Decode(f *testing.F) {
	f.Add("BB8")
	f.Add("%69 VD92EX0")
	f.Add("")
	f.Add("!!!!")
	f.Fuzz(func(t *testing.T, s string) {
		// must never panic
		Base45Decode(s) //nolint:errcheck
	})
}

// helper to keep import used
var _ = hex.EncodeToString
