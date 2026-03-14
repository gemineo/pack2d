package serial

import (
	"fmt"

	gojson "github.com/go-json-experiment/json"
	"github.com/fxamacker/cbor/v2"
)

type cborSerializer struct {
	encMode cbor.EncMode
	decMode cbor.DecMode
}

// NewCBORSerializer returns a Serializer that converts JSON → CBOR on Serialize
// and CBOR → JSON on Deserialize (ID=0x03).
// CBOR encoding is deterministic (sorted keys via CoreDetEncOptions).
func NewCBORSerializer() (Serializer, error) {
	encMode, err := cbor.CoreDetEncOptions().EncMode()
	if err != nil {
		return nil, fmt.Errorf("serial cbor: create enc mode: %w", err)
	}
	decMode, err := cbor.DecOptions{}.DecMode()
	if err != nil {
		return nil, fmt.Errorf("serial cbor: create dec mode: %w", err)
	}
	return &cborSerializer{encMode: encMode, decMode: decMode}, nil
}

func (s *cborSerializer) ID() byte     { return 0x03 }
func (s *cborSerializer) Name() string { return "cbor" }

// Serialize converts JSON input to CBOR binary output.
func (s *cborSerializer) Serialize(data []byte) ([]byte, error) {
	var v any
	if err := gojson.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("serial cbor: unmarshal json: %w", err)
	}
	out, err := s.encMode.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("serial cbor: marshal cbor: %w", err)
	}
	return out, nil
}

// Deserialize converts CBOR binary input back to JSON output.
func (s *cborSerializer) Deserialize(data []byte) ([]byte, error) {
	var v any
	if err := s.decMode.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("serial cbor: unmarshal cbor: %w", err)
	}
	out, err := gojson.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("serial cbor: marshal json: %w", err)
	}
	return out, nil
}
