package serial

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

type xmlSerializer struct{}

// NewXMLSerializer returns a Serializer that minifies XML on Serialize (ID=0x02).
// Deserialize is a pass-through (minified XML is still valid XML).
func NewXMLSerializer() Serializer { return &xmlSerializer{} }

func (s *xmlSerializer) ID() byte     { return 0x02 }
func (s *xmlSerializer) Name() string { return "xml" }

// Serialize minifies XML by stripping whitespace between tags using a token copy loop.
func (s *xmlSerializer) Serialize(data []byte) ([]byte, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("serial xml: decode token: %w", err)
		}

		// Skip the XML processing instruction (<?xml ...?>) to avoid duplication
		if pi, ok := tok.(xml.ProcInst); ok && pi.Target == "xml" {
			continue
		}

		// Strip whitespace-only CharData between tags
		if cd, ok := tok.(xml.CharData); ok {
			if len(bytes.TrimSpace(cd)) == 0 {
				continue
			}
		}

		if err := enc.EncodeToken(tok); err != nil {
			return nil, fmt.Errorf("serial xml: encode token: %w", err)
		}
	}

	if err := enc.Flush(); err != nil {
		return nil, fmt.Errorf("serial xml: flush: %w", err)
	}

	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, nil
}

func (s *xmlSerializer) Deserialize(data []byte) ([]byte, error) {
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}
