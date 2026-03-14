// Package compress provides an abstraction layer for data compression algorithms.
// The Registry is not concurrent-write-safe: it is intended to be populated once
// at construction time and used read-only thereafter.
package compress

import (
	"errors"
	"fmt"
	"io"
)

// ErrUnknownCompressor is returned when a compressor with the given ID or name is not registered.
var ErrUnknownCompressor = errors.New("compress: unknown compressor")

// Compressor defines the compress/decompress interface used by pack2d.
type Compressor interface {
	Compress(dst io.Writer, src io.Reader) error
	Decompress(dst io.Writer, src io.Reader) error
	CompressBytes(data []byte) ([]byte, error)
	DecompressBytes(data []byte) ([]byte, error)
	ID() byte
	Name() string
}

// Registry maps compressor IDs and names to Compressor implementations.
// Populate via Register before first use; do not call Register concurrently.
type Registry struct {
	byID   map[byte]Compressor
	byName map[string]Compressor
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		byID:   make(map[byte]Compressor),
		byName: make(map[string]Compressor),
	}
}

// Register adds c to the registry. Overwrites any previous registration with the same ID or name.
func (r *Registry) Register(c Compressor) {
	r.byID[c.ID()] = c
	r.byName[c.Name()] = c
}

// Get returns the Compressor registered under id.
func (r *Registry) Get(id byte) (Compressor, error) {
	c, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("%w: id=%d", ErrUnknownCompressor, id)
	}
	return c, nil
}

// GetByName returns the Compressor registered under name.
func (r *Registry) GetByName(name string) (Compressor, error) {
	c, ok := r.byName[name]
	if !ok {
		return nil, fmt.Errorf("%w: name=%q", ErrUnknownCompressor, name)
	}
	return c, nil
}

// DefaultRegistry returns a Registry pre-loaded with zlib, zstd, and brotli compressors.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(NewZlib(-1))
	// NewZstd can only fail if the zstd library fails to initialize, which is not recoverable.
	if zstdComp, err := NewZstd(3, nil); err == nil {
		r.Register(zstdComp)
	}
	r.Register(NewBrotli(6))
	return r
}
