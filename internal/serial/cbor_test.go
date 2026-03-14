package serial

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCBORRoundTrip(t *testing.T) {
	s, err := NewCBORSerializer()
	require.NoError(t, err)

	tests := []struct {
		name  string
		input string
	}{
		{"simple object", `{"key":"value"}`},
		{"number", `{"n":42}`},
		{"nested", `{"a":{"b":{"c":true}}}`},
		{"array", `{"items":[1,2,3]}`},
		{"unicode", `{"name":"héllo"}`},
		{"null value", `{"x":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cbor, err := s.Serialize([]byte(tt.input))
			require.NoError(t, err)
			assert.NotEmpty(t, cbor)

			json, err := s.Deserialize(cbor)
			require.NoError(t, err)
			assert.NotEmpty(t, json)
		})
	}
}

func TestCBORSmallerThanJSON(t *testing.T) {
	s, err := NewCBORSerializer()
	require.NoError(t, err)

	input := `{"patient":"John Smith","id":"12345","status":"active","score":98.6}`
	cbor, err := s.Serialize([]byte(input))
	require.NoError(t, err)
	assert.Less(t, len(cbor), len(input), "CBOR should be smaller than minified JSON")
}

func TestCBORInvalidJSON(t *testing.T) {
	s, err := NewCBORSerializer()
	require.NoError(t, err)
	_, err = s.Serialize([]byte("not json {"))
	assert.Error(t, err)
}

func TestCBORInvalidCBOR(t *testing.T) {
	s, err := NewCBORSerializer()
	require.NoError(t, err)
	_, err = s.Deserialize([]byte("not cbor data"))
	assert.Error(t, err)
}

func TestCBORID(t *testing.T) {
	s, err := NewCBORSerializer()
	require.NoError(t, err)
	assert.Equal(t, byte(0x03), s.ID())
	assert.Equal(t, "cbor", s.Name())
}
