package pack2d

import (
	"fmt"

	"github.com/gemineo/pack2d/internal/codec"
	"github.com/gemineo/pack2d/internal/compress"
	"github.com/gemineo/pack2d/internal/encoding"
	"github.com/gemineo/pack2d/internal/serial"
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
//  3. Decompress (with dictionary if DCT bit is set)
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

	compressed := raw[offset:]

	// 3. Decompress — resolve dictionary if DCT bit is set
	var comp compress.Compressor
	if h.Dictionary {
		if d.cfg.dictStore == nil {
			return nil, Stats{}, fmt.Errorf("%w: no dictionary store configured", ErrDictionaryNotFound)
		}
		entry, derr := d.cfg.dictStore.Get(h.DictionaryID)
		if derr != nil {
			return nil, Stats{}, fmt.Errorf("%w: id=%d: %v", ErrDictionaryNotFound, h.DictionaryID, derr)
		}
		comp, err = compress.NewZstd(3, entry.Data)
		if err != nil {
			return nil, Stats{}, fmt.Errorf("%w: build zstd with dict: %v", ErrDecodeFailed, err)
		}
	} else {
		cmpReg := compress.DefaultRegistry()
		comp, err = cmpReg.Get(h.Compression)
		if err != nil {
			return nil, Stats{}, fmt.Errorf("%w: %v", ErrUnknownCompression, err)
		}
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
		InputBytes:      len(result),
		CompressedBytes: len(compressed),
		EncodedBytes:    len(encoded),
	}
	if len(result) > 0 {
		stats.CompressionRatio = float64(len(compressed)) / float64(len(result))
	}

	return result, stats, nil
}
