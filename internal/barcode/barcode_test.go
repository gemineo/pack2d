package barcode

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testData = "HELLO WORLD"

func TestQRCodePNG(t *testing.T) {
	g := NewQRCodeGenerator()
	opts := DefaultOptions()
	opts.ImageFormat = PNG
	data, err := g.Generate(testData, opts)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(data, []byte("\x89PNG")), "expected PNG header")
	assert.Greater(t, len(data), 100)
}

func TestQRCodeSVG(t *testing.T) {
	g := NewQRCodeGenerator()
	opts := DefaultOptions()
	opts.ImageFormat = SVG
	data, err := g.Generate(testData, opts)
	require.NoError(t, err)
	s := string(data)
	assert.True(t, strings.HasPrefix(s, "<svg"), "expected SVG start")
	assert.Contains(t, s, "<rect")
	assert.True(t, strings.HasSuffix(s, "</svg>"), "expected SVG end")
}

func TestDataMatrixPNG(t *testing.T) {
	g := NewDataMatrixGenerator()
	opts := DefaultOptions()
	opts.ImageFormat = PNG
	data, err := g.Generate(testData, opts)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(data, []byte("\x89PNG")), "expected PNG header")
	assert.Greater(t, len(data), 100)
}

func TestDataMatrixSVG(t *testing.T) {
	g := NewDataMatrixGenerator()
	opts := DefaultOptions()
	opts.ImageFormat = SVG
	data, err := g.Generate(testData, opts)
	require.NoError(t, err)
	s := string(data)
	assert.True(t, strings.HasPrefix(s, "<svg"), "expected SVG start")
	assert.Contains(t, s, "<rect")
	assert.True(t, strings.HasSuffix(s, "</svg>"), "expected SVG end")
}

func TestQRCodeFeasibility(t *testing.T) {
	opts := DefaultOptions()
	result := CheckFeasibility(testData, QRCode, opts)
	assert.True(t, result.Feasible)
	assert.Greater(t, result.MaxCapacity, 0)
	assert.Greater(t, result.SymbolVersion, 0)

	// Exceeding capacity
	longData := strings.Repeat("A", 10000)
	result = CheckFeasibility(longData, QRCode, opts)
	assert.False(t, result.Feasible)
}

func TestDataMatrixFeasibility(t *testing.T) {
	opts := DefaultOptions()
	result := CheckFeasibility(testData, DataMatrix, opts)
	assert.True(t, result.Feasible)

	longData := strings.Repeat("A", 10000)
	result = CheckFeasibility(longData, DataMatrix, opts)
	assert.False(t, result.Feasible)
}

func TestQRCodeAllECLevels(t *testing.T) {
	g := NewQRCodeGenerator()
	levels := []ECLevel{ECLow, ECMedium, ECQuarter, ECHigh}
	for _, level := range levels {
		opts := DefaultOptions()
		opts.ErrorCorrection = level
		data, err := g.Generate(testData, opts)
		require.NoError(t, err, "level=%s", level)
		assert.True(t, bytes.HasPrefix(data, []byte("\x89PNG")), "level=%s", level)
	}
}

// Generator interface methods

func TestGeneratorType(t *testing.T) {
	assert.Equal(t, QRCode, NewQRCodeGenerator().Type())
	assert.Equal(t, DataMatrix, NewDataMatrixGenerator().Type())
}

func TestGeneratorMaxCapacity(t *testing.T) {
	opts := DefaultOptions()

	qrCap := NewQRCodeGenerator().MaxCapacity(opts)
	assert.Greater(t, qrCap, 0)

	dmCap := NewDataMatrixGenerator().MaxCapacity(opts)
	assert.Greater(t, dmCap, 0)
}

func TestQRCodeMaxCapacityAllLevels(t *testing.T) {
	g := NewQRCodeGenerator()
	for _, level := range []ECLevel{ECLow, ECMedium, ECQuarter, ECHigh} {
		opts := DefaultOptions()
		opts.ErrorCorrection = level
		cap := g.MaxCapacity(opts)
		assert.Greater(t, cap, 0, "level=%s", level)
	}
	// Unknown level falls back to medium
	opts := DefaultOptions()
	opts.ErrorCorrection = ECLevel("X")
	cap := g.MaxCapacity(opts)
	assert.Equal(t, g.MaxCapacity(DefaultOptions()), cap)
}

// Error paths in Generate

func TestQRCodeUnsupportedFormat(t *testing.T) {
	g := NewQRCodeGenerator()
	opts := DefaultOptions()
	opts.ImageFormat = ImageFormat("webp")
	_, err := g.Generate(testData, opts)
	assert.Error(t, err)
}

func TestQRCodeUnknownECLevel(t *testing.T) {
	g := NewQRCodeGenerator()
	opts := DefaultOptions()
	opts.ErrorCorrection = ECLevel("Z")
	_, err := g.Generate(testData, opts)
	assert.Error(t, err)
}

func TestDataMatrixUnsupportedFormat(t *testing.T) {
	g := NewDataMatrixGenerator()
	opts := DefaultOptions()
	opts.ImageFormat = ImageFormat("webp")
	_, err := g.Generate(testData, opts)
	assert.Error(t, err)
}

// CheckFeasibility edge cases

func TestCheckFeasibilityUnknownBarcodeType(t *testing.T) {
	result := CheckFeasibility(testData, BarcodeType("unknown"), DefaultOptions())
	assert.False(t, result.Feasible)
	assert.Equal(t, 0, result.MaxCapacity)
}

func TestCheckFeasibilityQRUnknownECLevel(t *testing.T) {
	opts := DefaultOptions()
	opts.ErrorCorrection = ECLevel("Z")
	result := CheckFeasibility(testData, QRCode, opts)
	// mapECLevel fails → returns zero FeasibilityResult
	assert.False(t, result.Feasible)
	assert.Equal(t, 0, result.MaxCapacity)
}

func TestCheckFeasibilityCapacityPercent(t *testing.T) {
	opts := DefaultOptions()
	data := strings.Repeat("A", 10)
	result := CheckFeasibility(data, QRCode, opts)
	assert.True(t, result.Feasible)
	assert.Greater(t, result.CapacityUsedPercent, 0.0)
	assert.Less(t, result.CapacityUsedPercent, 100.0)
}

// SVG quiet-zone = 0 (disables border on QR)

func TestQRCodeSVGNoQuietZone(t *testing.T) {
	g := NewQRCodeGenerator()
	opts := DefaultOptions()
	opts.ImageFormat = SVG
	opts.QuietZone = 0
	data, err := g.Generate(testData, opts)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(data), "<svg"))
}

// Small size (forces moduleSize = 1 in SVG generation)

func TestQRCodeSVGTinySize(t *testing.T) {
	g := NewQRCodeGenerator()
	opts := DefaultOptions()
	opts.ImageFormat = SVG
	opts.Size = 1
	data, err := g.Generate(testData, opts)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(data), "<svg"))
}

func TestDataMatrixSVGTinySize(t *testing.T) {
	g := NewDataMatrixGenerator()
	opts := DefaultOptions()
	opts.ImageFormat = SVG
	opts.Size = 1
	data, err := g.Generate(testData, opts)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(data), "<svg"))
}
