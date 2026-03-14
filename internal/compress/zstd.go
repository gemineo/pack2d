package compress

import (
	"bytes"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

type zstdCompressor struct {
	enc  *zstd.Encoder
	dec  *zstd.Decoder
	dict []byte
}

// NewZstd returns a Compressor backed by zstd at the given level.
// level is clamped to [1, 4] where 1=fastest, 4=best; -1 selects the default (2).
// dictionary may be nil for no-dictionary compression.
func NewZstd(level int, dictionary []byte) (Compressor, error) {
	var speedLevel zstd.EncoderLevel
	switch {
	case level == -1:
		speedLevel = zstd.SpeedDefault
	case level <= 1:
		speedLevel = zstd.SpeedFastest
	case level == 2:
		speedLevel = zstd.SpeedDefault
	case level == 3:
		speedLevel = zstd.SpeedBetterCompression
	default:
		speedLevel = zstd.SpeedBestCompression
	}

	encOpts := []zstd.EOption{zstd.WithEncoderLevel(speedLevel)}
	if len(dictionary) > 0 {
		encOpts = append(encOpts, zstd.WithEncoderDict(dictionary))
	}
	enc, err := zstd.NewWriter(nil, encOpts...)
	if err != nil {
		return nil, fmt.Errorf("compress zstd: new encoder: %w", err)
	}

	decOpts := []zstd.DOption{}
	if len(dictionary) > 0 {
		decOpts = append(decOpts, zstd.WithDecoderDicts(dictionary))
	}
	dec, err := zstd.NewReader(nil, decOpts...)
	if err != nil {
		enc.Close()
		return nil, fmt.Errorf("compress zstd: new decoder: %w", err)
	}

	var dictCopy []byte
	if len(dictionary) > 0 {
		dictCopy = make([]byte, len(dictionary))
		copy(dictCopy, dictionary)
	}

	return &zstdCompressor{enc: enc, dec: dec, dict: dictCopy}, nil
}

func (z *zstdCompressor) ID() byte     { return 0x01 }
func (z *zstdCompressor) Name() string { return "zstd" }

func (z *zstdCompressor) Compress(dst io.Writer, src io.Reader) error {
	z.enc.Reset(dst)
	if _, err := io.Copy(z.enc, src); err != nil {
		return fmt.Errorf("compress zstd: write: %w", err)
	}
	if err := z.enc.Close(); err != nil {
		return fmt.Errorf("compress zstd: close: %w", err)
	}
	return nil
}

func (z *zstdCompressor) Decompress(dst io.Writer, src io.Reader) error {
	if err := z.dec.Reset(src); err != nil {
		return fmt.Errorf("compress zstd: reset reader: %w", err)
	}
	if _, err := io.Copy(dst, z.dec); err != nil {
		return fmt.Errorf("compress zstd: read: %w", err)
	}
	return nil
}

func (z *zstdCompressor) CompressBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := z.Compress(&buf, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	if buf.Len() == 0 {
		return []byte{}, nil
	}
	return buf.Bytes(), nil
}

func (z *zstdCompressor) DecompressBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := z.Decompress(&buf, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	if buf.Len() == 0 {
		return []byte{}, nil
	}
	return buf.Bytes(), nil
}
