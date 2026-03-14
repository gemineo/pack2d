package barcode

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/datamatrix"
)

type dataMatrixGenerator struct{}

// NewDataMatrixGenerator returns a Generator for DataMatrix barcodes.
func NewDataMatrixGenerator() Generator { return &dataMatrixGenerator{} }

func (g *dataMatrixGenerator) Type() BarcodeType { return DataMatrix }

func (g *dataMatrixGenerator) MaxCapacity(_ Options) int {
	// DataMatrix can hold up to 2335 alphanumeric characters (ECC 200, largest symbol)
	return 2335
}

func (g *dataMatrixGenerator) Generate(data string, opts Options) ([]byte, error) {
	bc, err := datamatrix.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("barcode datamatrix: %w", err)
	}

	switch opts.ImageFormat {
	case PNG:
		scaled, err := barcode.Scale(bc, opts.Size, opts.Size)
		if err != nil {
			return nil, fmt.Errorf("barcode datamatrix scale: %w", err)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, scaled); err != nil {
			return nil, fmt.Errorf("barcode datamatrix png: %w", err)
		}
		return buf.Bytes(), nil
	case SVG:
		return dataMatrixToSVG(bc, opts.Size)
	default:
		return nil, fmt.Errorf("barcode datamatrix: unsupported format %q", opts.ImageFormat)
	}
}

func checkDataMatrixFeasibility(data string, opts Options) FeasibilityResult {
	maxCap := 2335
	used := len(data)

	bc, err := datamatrix.Encode(data)
	if err != nil {
		return FeasibilityResult{
			Feasible:            false,
			MaxCapacity:         maxCap,
			UsedCapacity:        used,
			CapacityUsedPercent: float64(used) / float64(maxCap) * 100,
		}
	}

	bounds := bc.Bounds()
	size := bounds.Max.X
	pct := float64(used) / float64(maxCap) * 100
	return FeasibilityResult{
		Feasible:            true,
		MaxCapacity:         maxCap,
		UsedCapacity:        used,
		CapacityUsedPercent: pct,
		SymbolVersion:       size,
	}
}
