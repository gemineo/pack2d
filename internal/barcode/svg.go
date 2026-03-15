package barcode

import (
	"bytes"
	"fmt"
	"image"
	"image/color"

	qrlib "github.com/skip2/go-qrcode"
)

// qrToSVG generates an SVG image from a QR code using q.Bitmap().
// Each dark module becomes a <rect> element.
func qrToSVG(q *qrlib.QRCode, size, quietZone int) ([]byte, error) {
	bm := q.Bitmap() // [row][col]bool, true=dark module
	if len(bm) == 0 || len(bm[0]) == 0 {
		return nil, fmt.Errorf("barcode qrcode svg: empty bitmap")
	}
	rows := len(bm)
	cols := len(bm[0])

	moduleSize := size / cols
	if moduleSize < 1 {
		moduleSize = 1
	}
	totalSize := cols * moduleSize

	var buf bytes.Buffer
	fmt.Fprintf(&buf, `<svg xmlns="http://www.w3.org/2000/svg" version="1.1" viewBox="0 0 %d %d" width="%d" height="%d">`,
		totalSize, totalSize, size, size)
	buf.WriteString(`<rect width="100%" height="100%" fill="white"/>`)

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if bm[r][c] {
				x := c * moduleSize
				y := r * moduleSize
				fmt.Fprintf(&buf, `<rect x="%d" y="%d" width="%d" height="%d" fill="black"/>`,
					x, y, moduleSize, moduleSize)
			}
		}
	}
	buf.WriteString(`</svg>`)
	return buf.Bytes(), nil
}

// dataMatrixToSVG generates an SVG image from a DataMatrix barcode image.
// Dark pixels become <rect> elements.
func dataMatrixToSVG(img image.Image, size int) ([]byte, error) {
	bounds := img.Bounds()
	cols := bounds.Max.X - bounds.Min.X
	rows := bounds.Max.Y - bounds.Min.Y
	if cols == 0 || rows == 0 {
		return nil, fmt.Errorf("barcode datamatrix svg: empty image")
	}

	moduleSize := size / cols
	if moduleSize < 1 {
		moduleSize = 1
	}
	totalSize := cols * moduleSize

	var buf bytes.Buffer
	fmt.Fprintf(&buf, `<svg xmlns="http://www.w3.org/2000/svg" version="1.1" viewBox="0 0 %d %d" width="%d" height="%d">`,
		totalSize, totalSize, size, size)
	buf.WriteString(`<rect width="100%" height="100%" fill="white"/>`)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			if isDark(c) {
				px := (x - bounds.Min.X) * moduleSize
				py := (y - bounds.Min.Y) * moduleSize
				fmt.Fprintf(&buf, `<rect x="%d" y="%d" width="%d" height="%d" fill="black"/>`,
					px, py, moduleSize, moduleSize)
			}
		}
	}
	buf.WriteString(`</svg>`)
	return buf.Bytes(), nil
}

// isDark returns true if the color is closer to black than white.
func isDark(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	// Luminance threshold: dark if all channels are below 50%
	return r < 0x8000 && g < 0x8000 && b < 0x8000
}
