package serial

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXMLRoundTrip(t *testing.T) {
	s := NewXMLSerializer()

	tests := []struct {
		name  string
		input string
	}{
		{"simple element", `<root><child>text</child></root>`},
		{"attributes", `<item id="1" name="test"><value>42</value></item>`},
		{"nested", `<a><b><c>deep</c></b></a>`},
		{"whitespace between tags", "<root>\n  <child>  text  </child>\n</root>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized, err := s.Serialize([]byte(tt.input))
			require.NoError(t, err)
			assert.NotEmpty(t, serialized)

			deserialized, err := s.Deserialize(serialized)
			require.NoError(t, err)
			assert.Equal(t, serialized, deserialized)
		})
	}
}

func TestXMLMinifiesWhitespace(t *testing.T) {
	s := NewXMLSerializer()
	input := "<root>\n  <child>text</child>\n</root>"
	out, err := s.Serialize([]byte(input))
	require.NoError(t, err)
	assert.NotContains(t, string(out), "\n")
	assert.NotContains(t, string(out), "  ")
}

func TestXMLInvalidInput(t *testing.T) {
	s := NewXMLSerializer()
	_, err := s.Serialize([]byte("not xml at all <unclosed"))
	assert.Error(t, err)
}

func TestXMLID(t *testing.T) {
	s := NewXMLSerializer()
	assert.Equal(t, byte(0x02), s.ID())
	assert.Equal(t, "xml", s.Name())
}
