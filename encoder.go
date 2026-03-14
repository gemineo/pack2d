package pack2d

import (
	"fmt"

	"github.com/gemineo/pack2d/internal/codec"
	"github.com/gemineo/pack2d/internal/compress"
	"github.com/gemineo/pack2d/internal/encoding"
	"github.com/gemineo/pack2d/internal/serial"
	"github.com/gemineo/pack2d/internal/textenc"
)

// Encoder compresses and encodes data into a pack2d base45 string.
type Encoder struct {
	cfg config
}

// NewEncoder creates an Encoder with the given options applied over defaults.
func NewEncoder(opts ...Option) *Encoder {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}
	return &Encoder{cfg: cfg}
}

// Encode runs the full encode pipeline and returns the base45-encoded string.
//
// Pipeline:
//  1. Normalize to UTF-8 via textenc
//  2. Serialize (e.g. JSON minification)
//  3. Compress
//  4. Prepend header bytes
//  5. Base45 encode
func (e *Encoder) Encode(data []byte) (string, Stats, error) {
	// 1. Normalize encoding (skip for raw binary input)
	var normalized []byte
	if e.cfg.inputType == Raw && (e.cfg.encoding == "" || e.cfg.encoding == "utf-8") {
		normalized = data
	} else {
		var nerr error
		normalized, nerr = textenc.Normalize(data, e.cfg.encoding)
		if nerr != nil {
			return "", Stats{}, fmt.Errorf("%w: %v", ErrInvalidEncoding, nerr)
		}
	}

	// 2. Serialize
	serReg := serial.DefaultRegistry()
	ser, err := serReg.GetByName(string(e.cfg.inputType))
	if err != nil {
		return "", Stats{}, fmt.Errorf("%w: %v", ErrUnknownSerializer, err)
	}
	serialized, err := ser.Serialize(normalized)
	if err != nil {
		return "", Stats{}, fmt.Errorf("pack2d encode: serialize: %w", err)
	}

	// 3. Compress
	cmpReg := compress.DefaultRegistry()
	comp, err := cmpReg.GetByName(string(e.cfg.compression))
	if err != nil {
		return "", Stats{}, fmt.Errorf("%w: %v", ErrUnknownCompression, err)
	}
	compressed, err := comp.CompressBytes(serialized)
	if err != nil {
		return "", Stats{}, fmt.Errorf("pack2d encode: compress: %w", err)
	}

	// 4. Build and prepend header
	h := codec.Header{
		Version:       0,
		Compression:   comp.ID(),
		Dictionary:    false,
		Serialization: ser.ID(),
	}
	header := codec.PackHeader(h)
	payload := append(header, compressed...)

	// 5. Base45 encode
	encoded := encoding.Base45Encode(payload)

	stats := Stats{
		InputBytes:      len(data),
		CompressedBytes: len(compressed),
		EncodedBytes:    len(encoded),
	}
	if len(data) > 0 {
		stats.CompressionRatio = float64(len(compressed)) / float64(len(data))
	}

	return encoded, stats, nil
}

// EncodeToBarcode encodes data and generates a barcode image.
func (e *Encoder) EncodeToBarcode(data []byte) ([]byte, Stats, error) {
	encoded, stats, err := e.Encode(data)
	if err != nil {
		return nil, stats, err
	}

	barcodeOpts := barcodeOptsFromConfig(e.cfg)
	barcodeGen := newBarcodeGenerator(e.cfg.barcodeType)
	imgData, err := barcodeGen.Generate(encoded, barcodeOpts)
	if err != nil {
		return nil, stats, fmt.Errorf("pack2d encode: barcode: %w", err)
	}
	return imgData, stats, nil
}
