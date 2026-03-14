package compress

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

type zlibCompressor struct {
	level int
}

// NewZlib returns a Compressor backed by zlib at the given level.
// level is clamped to [zlib.NoCompression, zlib.BestCompression];
// zlib.DefaultCompression (-1) is also accepted.
func NewZlib(level int) Compressor {
	if level != zlib.DefaultCompression {
		if level < zlib.NoCompression {
			level = zlib.NoCompression
		} else if level > zlib.BestCompression {
			level = zlib.BestCompression
		}
	}
	return &zlibCompressor{level: level}
}

func (z *zlibCompressor) ID() byte     { return 0x00 }
func (z *zlibCompressor) Name() string { return "zlib" }

func (z *zlibCompressor) Compress(dst io.Writer, src io.Reader) (err error) {
	w, werr := zlib.NewWriterLevel(dst, z.level)
	if werr != nil {
		return fmt.Errorf("compress zlib: new writer: %w", werr)
	}
	if _, err = io.Copy(w, src); err != nil {
		return fmt.Errorf("compress zlib: write: %w", err)
	}
	if cerr := w.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("compress zlib: close: %w", err)
	}
	return nil
}

func (z *zlibCompressor) Decompress(dst io.Writer, src io.Reader) error {
	r, err := zlib.NewReader(src)
	if err != nil {
		return fmt.Errorf("compress zlib: new reader: %w", err)
	}
	defer r.Close()
	if _, err = io.Copy(dst, r); err != nil {
		return fmt.Errorf("compress zlib: read: %w", err)
	}
	return nil
}

func (z *zlibCompressor) CompressBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := z.Compress(&buf, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (z *zlibCompressor) DecompressBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := z.Decompress(&buf, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
