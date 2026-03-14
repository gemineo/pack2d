// Package barcode generates QR Code and DataMatrix barcodes from pack2d-encoded strings.
package barcode

import (
	"errors"
)

// BarcodeType identifies the barcode symbology.
type BarcodeType string

const (
	QRCode     BarcodeType = "qrcode"
	DataMatrix BarcodeType = "datamatrix"
)

// ImageFormat identifies the output image format.
type ImageFormat string

const (
	PNG ImageFormat = "png"
	SVG ImageFormat = "svg"
)

// ECLevel is the error-correction level for QR codes.
type ECLevel string

const (
	ECLow     ECLevel = "L"
	ECMedium  ECLevel = "M"
	ECQuarter ECLevel = "Q"
	ECHigh    ECLevel = "H"
)

// Options configures barcode generation.
type Options struct {
	ImageFormat     ImageFormat
	Size            int
	ErrorCorrection ECLevel
	QuietZone       int // modules; 0 = no quiet zone
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		ImageFormat:     PNG,
		Size:            256,
		ErrorCorrection: ECMedium,
		QuietZone:       4,
	}
}

// FeasibilityResult describes whether data fits in the requested barcode.
type FeasibilityResult struct {
	Feasible            bool
	MaxCapacity         int
	UsedCapacity        int
	CapacityUsedPercent float64
	SymbolVersion       int // QR version or DataMatrix symbol size
}

// Generator produces barcode images from encoded strings.
type Generator interface {
	Generate(data string, opts Options) ([]byte, error)
	Type() BarcodeType
	MaxCapacity(opts Options) int
}

// ErrPayloadTooLarge is returned when data exceeds the barcode's capacity.
var ErrPayloadTooLarge = errors.New("barcode: payload exceeds capacity")

// CheckFeasibility reports whether data fits in the requested barcode type with given options.
func CheckFeasibility(data string, barcodeType BarcodeType, opts Options) FeasibilityResult {
	switch barcodeType {
	case QRCode:
		return checkQRFeasibility(data, opts)
	case DataMatrix:
		return checkDataMatrixFeasibility(data, opts)
	default:
		return FeasibilityResult{}
	}
}
