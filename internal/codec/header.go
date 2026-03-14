package codec

import (
	"errors"
	"fmt"
)

var ErrInvalidHeader = errors.New("codec: invalid header")

// Header represents the decoded pack2d header.
//
// Header byte layout:
//
//	Bits 7-6: VER  (0b00 = v1)
//	Bits 5-4: CMP  (0b00=zlib, 0b01=zstd, 0b10=brotli, 0b11=reserved)
//	Bit  3:   DCT  (0=no dict, 1=dict → 2-byte dict ID follows)
//	Bits 2-0: SER  (0b000=raw, 0b001=json, 0b010=xml, 0b011=cbor)
//	0xFF      = extension marker (unsupported, return error)
type Header struct {
	Version       uint8
	Compression   uint8
	Dictionary    bool
	DictionaryID  uint16
	Serialization uint8
}

// PackHeader serializes h into 1 or 3 bytes.
// Returns 3 bytes when h.Dictionary is true (header byte + big-endian uint16 dict ID).
func PackHeader(h Header) []byte {
	b := (h.Version << 6) | (h.Compression << 4) | (boolToByte(h.Dictionary) << 3) | h.Serialization
	if h.Dictionary {
		return []byte{b, byte(h.DictionaryID >> 8), byte(h.DictionaryID)}
	}
	return []byte{b}
}

// UnpackHeader parses the header from data.
// Returns the header, the number of bytes consumed (1 or 3), and any error.
func UnpackHeader(data []byte) (Header, int, error) {
	if len(data) == 0 {
		return Header{}, 0, fmt.Errorf("%w: empty data", ErrInvalidHeader)
	}
	b := data[0]
	if b == 0xFF {
		return Header{}, 0, fmt.Errorf("%w: extension marker (0xFF) not supported", ErrInvalidHeader)
	}
	ver := (b >> 6) & 0x03
	if ver != 0 {
		return Header{}, 0, fmt.Errorf("%w: unsupported version %d", ErrInvalidHeader, ver)
	}
	cmp := (b >> 4) & 0x03
	dct := (b >> 3) & 0x01
	ser := b & 0x07

	h := Header{
		Version:       ver,
		Compression:   cmp,
		Dictionary:    dct == 1,
		Serialization: ser,
	}

	if h.Dictionary {
		if len(data) < 3 {
			return Header{}, 0, fmt.Errorf("%w: truncated dictionary header", ErrInvalidHeader)
		}
		h.DictionaryID = uint16(data[1])<<8 | uint16(data[2])
		return h, 3, nil
	}
	return h, 1, nil
}

func boolToByte(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}
