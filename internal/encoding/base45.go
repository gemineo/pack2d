// Package encoding provides Base45 encoding/decoding for pack2d payloads.
// Base45 uses the QR alphanumeric character set, enabling QR alphanumeric mode
// which is more compact than binary mode for this payload type.
package encoding

import (
	"fmt"

	"github.com/dasio/base45"
)

// Base45Encode encodes data to a Base45 string.
func Base45Encode(data []byte) string {
	return base45.EncodeToString(data)
}

// Base45Decode decodes a Base45-encoded string to bytes.
func Base45Decode(s string) ([]byte, error) {
	b, err := base45.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("encoding base45: %w", err)
	}
	return b, nil
}
