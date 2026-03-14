package compress

import (
	"bytes"
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
)

type brotliCompressor struct {
	level int
}

// NewBrotli returns a Compressor backed by brotli at the given level.
// level is clamped to [0, 11]; -1 selects the default (6).
func NewBrotli(level int) Compressor {
	if level == -1 {
		level = 6
	} else if level < 0 {
		level = 0
	} else if level > 11 {
		level = 11
	}
	return &brotliCompressor{level: level}
}

func (b *brotliCompressor) ID() byte     { return 0x02 }
func (b *brotliCompressor) Name() string { return "brotli" }

func (b *brotliCompressor) Compress(dst io.Writer, src io.Reader) error {
	w := brotli.NewWriterLevel(dst, b.level)
	if _, err := io.Copy(w, src); err != nil {
		return fmt.Errorf("compress brotli: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("compress brotli: close: %w", err)
	}
	return nil
}

func (b *brotliCompressor) Decompress(dst io.Writer, src io.Reader) error {
	r := brotli.NewReader(src)
	if _, err := io.Copy(dst, r); err != nil {
		return fmt.Errorf("compress brotli: read: %w", err)
	}
	return nil
}

func (b *brotliCompressor) CompressBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := b.Compress(&buf, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (b *brotliCompressor) DecompressBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := b.Decompress(&buf, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
