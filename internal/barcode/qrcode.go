package barcode

import (
	"fmt"

	qrlib "github.com/skip2/go-qrcode"
)

// qrAlphanumericCapacity is the maximum alphanumeric character capacity per QR version+level.
// Indexed by version (1-40); version 0 is unused.
// Values for version 40 are used as the upper bound; Phase 1 uses version 40 capacity.
var qrAlphanumericCapacity = map[ECLevel]int{
	ECLow:     4296,
	ECMedium:  3391,
	ECQuarter: 2420,
	ECHigh:    1852,
}

type qrGenerator struct{}

// NewQRCodeGenerator returns a Generator for QR Code barcodes.
func NewQRCodeGenerator() Generator { return &qrGenerator{} }

func (g *qrGenerator) Type() BarcodeType { return QRCode }

func (g *qrGenerator) MaxCapacity(opts Options) int {
	cap, ok := qrAlphanumericCapacity[opts.ErrorCorrection]
	if !ok {
		return qrAlphanumericCapacity[ECMedium]
	}
	return cap
}

func (g *qrGenerator) Generate(data string, opts Options) ([]byte, error) {
	level, err := mapECLevel(opts.ErrorCorrection)
	if err != nil {
		return nil, err
	}
	q, err := qrlib.New(data, level)
	if err != nil {
		return nil, fmt.Errorf("barcode qrcode: %w", err)
	}
	q.DisableBorder = (opts.QuietZone == 0)

	switch opts.ImageFormat {
	case PNG:
		pngData, err := q.PNG(opts.Size)
		if err != nil {
			return nil, fmt.Errorf("barcode qrcode png: %w", err)
		}
		return pngData, nil
	case SVG:
		return qrToSVG(q, opts.Size, opts.QuietZone)
	default:
		return nil, fmt.Errorf("barcode qrcode: unsupported format %q", opts.ImageFormat)
	}
}

func mapECLevel(level ECLevel) (qrlib.RecoveryLevel, error) {
	switch level {
	case ECLow:
		return qrlib.Low, nil
	case ECMedium:
		return qrlib.Medium, nil
	case ECQuarter:
		return qrlib.High, nil // go-qrcode uses High for Q equivalent
	case ECHigh:
		return qrlib.Highest, nil
	default:
		return qrlib.Medium, fmt.Errorf("barcode qrcode: unknown EC level %q", level)
	}
}

func checkQRFeasibility(data string, opts Options) FeasibilityResult {
	level, err := mapECLevel(opts.ErrorCorrection)
	if err != nil {
		return FeasibilityResult{}
	}
	maxCap := qrAlphanumericCapacity[opts.ErrorCorrection]
	used := len(data)

	q, err := qrlib.New(data, level)
	if err != nil {
		return FeasibilityResult{
			Feasible:            false,
			MaxCapacity:         maxCap,
			UsedCapacity:        used,
			CapacityUsedPercent: float64(used) / float64(maxCap) * 100,
		}
	}

	pct := float64(used) / float64(maxCap) * 100
	return FeasibilityResult{
		Feasible:            true,
		MaxCapacity:         maxCap,
		UsedCapacity:        used,
		CapacityUsedPercent: pct,
		SymbolVersion:       q.VersionNumber,
	}
}
