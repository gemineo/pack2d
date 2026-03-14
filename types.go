// Package pack2d compresses and encodes textual payloads into compact strings
// optimized for 2D barcodes (QR Code, DataMatrix), then decodes them back.
//
// Pipeline: text → [serialize] → compress → base45-encode → barcode image
// Decode:   base45-decode → decompress → [deserialize] → text
package pack2d

import "errors"

// InputType identifies how the input data should be serialized before compression.
type InputType string

const (
	Raw  InputType = "raw"
	JSON InputType = "json"
	XML  InputType = "xml"
	CBOR InputType = "cbor"
)

// CompressionType identifies the compression algorithm.
type CompressionType string

const (
	Zlib   CompressionType = "zlib"
	Zstd   CompressionType = "zstd"
	Brotli CompressionType = "brotli"
)

// BarcodeType identifies the barcode symbology.
type BarcodeType string

const (
	QRCode     BarcodeType = "qrcode"
	DataMatrix BarcodeType = "datamatrix"
)

// ImageFormat identifies the barcode image output format.
type ImageFormat string

const (
	PNG ImageFormat = "png"
	SVG ImageFormat = "svg"
)

// ECLevel is the error-correction level.
type ECLevel string

const (
	ECLow     ECLevel = "L"
	ECMedium  ECLevel = "M"
	ECQuarter ECLevel = "Q"
	ECHigh    ECLevel = "H"
)

// Stats holds encoding/decoding statistics.
type Stats struct {
	InputBytes       int
	CompressedBytes  int
	EncodedBytes     int
	CompressionRatio float64
}

// InspectResult holds the decoded metadata from an encoded pack2d string.
type InspectResult struct {
	Header             string
	Compression        string
	Serialization      string
	HasDictionary      bool
	DictionaryID       uint16
	Version            uint8
	DataPreview        string
	CompatibleBarcodes []string
}

// Sentinel errors returned by the pack2d root package.
var (
	ErrPayloadTooLarge    = errors.New("pack2d: payload exceeds barcode capacity")
	ErrUnknownCompression = errors.New("pack2d: unknown compression algorithm")
	ErrUnknownSerializer  = errors.New("pack2d: unknown serialization type")
	ErrInvalidHeader      = errors.New("pack2d: invalid header byte")
	ErrDictionaryNotFound = errors.New("pack2d: dictionary not found")
	ErrInvalidEncoding    = errors.New("pack2d: unsupported text encoding")
	ErrDecodeFailed       = errors.New("pack2d: decode failed")
)
