// Package textenc handles text encoding detection and normalization for pack2d.
package textenc

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/ianaindex"
)

// ErrInvalidEncoding is returned when the encoding is unsupported or data is invalid for that encoding.
var ErrInvalidEncoding = errors.New("textenc: unsupported encoding")

// Normalize converts data to UTF-8.
//
// If hint is "" or "utf-8", data is validated as UTF-8 and returned as-is (no allocation).
// Otherwise, hint is used as an IANA charset name to transcode data from that encoding to UTF-8.
//
// BOM detection: 0xFF 0xFE → UTF-16 LE; 0xFE 0xFF → UTF-16 BE (used to override hint).
func Normalize(data []byte, hint string) ([]byte, error) {
	hint = strings.ToLower(strings.TrimSpace(hint))

	// BOM detection overrides hint
	if len(data) >= 2 {
		if data[0] == 0xFF && data[1] == 0xFE {
			hint = "utf-16le"
		} else if data[0] == 0xFE && data[1] == 0xFF {
			hint = "utf-16be"
		}
	}

	if hint == "" || hint == "utf-8" {
		if !utf8.Valid(data) {
			return nil, fmt.Errorf("%w: data is not valid UTF-8", ErrInvalidEncoding)
		}
		return data, nil
	}

	enc, err := ianaindex.IANA.Encoding(hint)
	if err != nil {
		return nil, fmt.Errorf("%w: %q: %v", ErrInvalidEncoding, hint, err)
	}
	out, err := enc.NewDecoder().Bytes(data)
	if err != nil {
		return nil, fmt.Errorf("%w: decode %q: %v", ErrInvalidEncoding, hint, err)
	}
	return out, nil
}

// Detect returns "utf-8" if data is valid UTF-8, "unknown" otherwise.
// Full charset auto-detection is planned for Phase 2.
func Detect(data []byte) (string, error) {
	if utf8.Valid(data) {
		return "utf-8", nil
	}
	return "unknown", nil
}
