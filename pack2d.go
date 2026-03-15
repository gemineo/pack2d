package pack2d

import (
	"fmt"

	"github.com/gemineo/pack2d/internal/barcode"
	"github.com/gemineo/pack2d/internal/codec"
	"github.com/gemineo/pack2d/internal/encoding"
)

// Encode is a package-level convenience wrapper around NewEncoder(opts...).Encode(data).
func Encode(data []byte, opts ...Option) (string, Stats, error) {
	return NewEncoder(opts...).Encode(data)
}

// Decode is a package-level convenience wrapper around NewDecoder(opts...).Decode(encoded).
func Decode(encoded string, opts ...Option) ([]byte, Stats, error) {
	return NewDecoder(opts...).Decode(encoded)
}

// GenerateBarcode encodes data and generates a barcode image in one step.
func GenerateBarcode(data []byte, opts ...Option) ([]byte, Stats, error) {
	enc := NewEncoder(opts...)
	return enc.EncodeToBarcode(data)
}

// Inspect parses a pack2d-encoded string and returns its metadata without fully decoding.
func Inspect(encoded string) (*InspectResult, error) {
	raw, err := encoding.Base45Decode(encoded)
	if err != nil {
		return nil, fmt.Errorf("pack2d inspect: base45: %w", err)
	}
	h, offset, err := codec.UnpackHeader(raw)
	if err != nil {
		return nil, fmt.Errorf("pack2d inspect: header: %w", err)
	}

	compressionNames := map[uint8]string{
		0: "zlib",
		1: "zstd",
		2: "brotli",
	}
	serializationNames := map[uint8]string{
		0: "raw",
		1: "json",
		2: "xml",
		3: "cbor",
	}

	compName, ok := compressionNames[h.Compression]
	if !ok {
		compName = fmt.Sprintf("unknown(%d)", h.Compression)
	}
	serName, ok := serializationNames[h.Serialization]
	if !ok {
		serName = fmt.Sprintf("unknown(%d)", h.Serialization)
	}

	preview := string(raw[offset:])
	if len(preview) > 64 {
		preview = preview[:64] + "..."
	}

	result := &InspectResult{
		Header:             fmt.Sprintf("0x%02X", raw[0]),
		Compression:        compName,
		Serialization:      serName,
		HasDictionary:      h.Dictionary,
		DictionaryID:       h.DictionaryID,
		Version:            h.Version,
		DataPreview:        preview,
		CompatibleBarcodes: []string{"qrcode", "datamatrix"},
	}
	return result, nil
}

// newBarcodeGenerator returns the Generator for the given BarcodeType.
func newBarcodeGenerator(bt BarcodeType) barcode.Generator {
	switch bt {
	case DataMatrix:
		return barcode.NewDataMatrixGenerator()
	default:
		return barcode.NewQRCodeGenerator()
	}
}

// barcodeOptsFromConfig maps pack2d config to barcode.Options.
func barcodeOptsFromConfig(cfg config) barcode.Options {
	return barcode.Options{
		ImageFormat:     barcode.ImageFormat(cfg.imageFormat),
		Size:            cfg.size,
		ErrorCorrection: barcode.ECLevel(cfg.errorCorrection),
		QuietZone:       cfg.quietZone,
	}
}
