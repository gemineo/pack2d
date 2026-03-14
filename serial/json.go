package serial

import (
	"fmt"

	"github.com/go-json-experiment/json/jsontext"
)

type jsonSerializer struct{}

// NewJSONSerializer returns a Serializer that minifies JSON on Serialize (ID=0x01).
// Deserialize is a pass-through (minified JSON is still valid JSON).
func NewJSONSerializer() Serializer { return &jsonSerializer{} }

func (s *jsonSerializer) ID() byte     { return 0x01 }
func (s *jsonSerializer) Name() string { return "json" }

func (s *jsonSerializer) Serialize(data []byte) ([]byte, error) {
	v := jsontext.Value(data)
	if err := v.Compact(); err != nil {
		return nil, fmt.Errorf("serial json: compact: %w", err)
	}
	out := make([]byte, len(v))
	copy(out, v)
	return out, nil
}

func (s *jsonSerializer) Deserialize(data []byte) ([]byte, error) {
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}
