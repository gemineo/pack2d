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
