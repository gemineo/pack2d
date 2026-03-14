package serial

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRawSerializerRoundTrip(t *testing.T) {
	s := NewRawSerializer()
	data := []byte("hello world")
	serialized, err := s.Serialize(data)
	require.NoError(t, err)
	assert.Equal(t, data, serialized)

	deserialized, err := s.Deserialize(serialized)
	require.NoError(t, err)
	assert.Equal(t, data, deserialized)
}

func TestJSONSerializerMinify(t *testing.T) {
	s := NewJSONSerializer()
	input := []byte(`{"a":  1,  "b": "c"}`)
	out, err := s.Serialize(input)
	require.NoError(t, err)
	assert.Equal(t, `{"a":1,"b":"c"}`, string(out))
}

func TestJSONSerializerInvalidInput(t *testing.T) {
	s := NewJSONSerializer()
	_, err := s.Serialize([]byte("not json at all {{{"))
	assert.Error(t, err)
}

func TestJSONSerializerRoundTrip(t *testing.T) {
	s := NewJSONSerializer()
	inputs := []string{
		`{"key":"value","nested":{"a":1}}`,
		`{"unicode":"héllo wörld","emoji":"🎉"}`,
		`[1,2,3,{"a":"b"}]`,
	}
	for _, input := range inputs {
		serialized, err := s.Serialize([]byte(input))
		require.NoError(t, err)
		deserialized, err := s.Deserialize(serialized)
		require.NoError(t, err)
		// deserialized should be valid (same as serialized since Deserialize is pass-through)
		assert.Equal(t, serialized, deserialized)
	}
}

func TestRegistryGetByName(t *testing.T) {
	r := DefaultRegistry()
	s, err := r.GetByName("raw")
	require.NoError(t, err)
	assert.Equal(t, byte(0x00), s.ID())

	s, err = r.GetByName("json")
	require.NoError(t, err)
	assert.Equal(t, byte(0x01), s.ID())

	_, err = r.GetByName("unknown")
	assert.Error(t, err)
}

func TestRegistryGetByID(t *testing.T) {
	r := DefaultRegistry()

	s, err := r.Get(0x00)
	require.NoError(t, err)
	assert.Equal(t, "raw", s.Name())

	s, err = r.Get(0x01)
	require.NoError(t, err)
	assert.Equal(t, "json", s.Name())

	_, err = r.Get(0xFF)
	assert.ErrorIs(t, err, ErrUnknownSerializer)
}

func TestDefaultRegistryContainsAllSerializers(t *testing.T) {
	r := DefaultRegistry()
	for _, tt := range []struct {
		id   byte
		name string
	}{
		{0x00, "raw"},
		{0x01, "json"},
		{0x02, "xml"},
		{0x03, "cbor"},
	} {
		s, err := r.Get(tt.id)
		require.NoError(t, err, "id=0x%02X", tt.id)
		assert.Equal(t, tt.name, s.Name())

		s, err = r.GetByName(tt.name)
		require.NoError(t, err, "name=%q", tt.name)
		assert.Equal(t, tt.id, s.ID())
	}
}

func TestRegistryRegisterOverwrites(t *testing.T) {
	r := NewRegistry()
	r.Register(NewRawSerializer())
	r.Register(NewRawSerializer()) // register twice — must not panic

	s, err := r.Get(0x00)
	require.NoError(t, err)
	assert.Equal(t, "raw", s.Name())
}
