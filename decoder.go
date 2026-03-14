package pack2d

import (
	"fmt"

	"github.com/gemineo/pack2d/codec"
	"github.com/gemineo/pack2d/compress"
	"github.com/gemineo/pack2d/encoding"
	"github.com/gemineo/pack2d/serial"
)

// Decoder decodes pack2d base45-encoded strings back to original data.
type Decoder struct {
	cfg config
}

// NewDecoder creates a Decoder with the given options applied over defaults.
func NewDecoder(opts ...Option) *Decoder {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}
	return &Decoder{cfg: cfg}
}

// Decode runs the full decode pipeline.
//
// Pipeline:
//  1. Base45 decode
//  2. Unpack header
//  3. Decompress
//  4. Deserialize
func (d *Decoder) Decode(encoded string) ([]byte, Stats, error) {
	// 1. Base45 decode
	raw, err := encoding.Base45Decode(encoded)
	if err != nil {
		return nil, Stats{}, fmt.Errorf("%w: base45: %v", ErrDecodeFailed, err)
	}

	// 2. Unpack header
	h, offset, err := codec.UnpackHeader(raw)
	if err != nil {
		return nil, Stats{}, fmt.Errorf("%w: header: %v", ErrDecodeFailed, err)
	}

	// Dictionary support is Phase 2
	if h.Dictionary {
		return nil, Stats{}, fmt.Errorf("%w: dictionary-encoded payloads not supported in Phase 1", ErrDictionaryNotFound)
	}

	compressed := raw[offset:]

	// 3. Decompress
	cmpReg := compress.DefaultRegistry()
	comp, err := cmpReg.Get(h.Compression)
	if err != nil {
		return nil, Stats{}, fmt.Errorf("%w: %v", ErrUnknownCompression, err)
	}
	decompressed, err := comp.DecompressBytes(compressed)
	if err != nil {
		return nil, Stats{}, fmt.Errorf("%w: decompress: %v", ErrDecodeFailed, err)
	}

	// 4. Deserialize
	serReg := serial.DefaultRegistry()
	ser, err := serReg.Get(h.Serialization)
	if err != nil {
		return nil, Stats{}, fmt.Errorf("%w: %v", ErrUnknownSerializer, err)
	}
	result, err := ser.Deserialize(decompressed)
	if err != nil {
		return nil, Stats{}, fmt.Errorf("%w: deserialize: %v", ErrDecodeFailed, err)
	}

	stats := Stats{
		InputBytes:      len(encoded),
		CompressedBytes: len(compressed),
		EncodedBytes:    len(raw),
	}
	if len(encoded) > 0 {
		stats.CompressionRatio = float64(len(compressed)) / float64(len(result))
	}

	return result, stats, nil
}
