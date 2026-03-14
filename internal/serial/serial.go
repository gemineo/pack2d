// Package serial provides serializer implementations for pack2d payloads.
package serial

import (
	"errors"
	"fmt"
)

// ErrUnknownSerializer is returned when no serializer matches the requested ID or name.
var ErrUnknownSerializer = errors.New("serial: unknown serializer")

// Serializer transforms payloads before compression (Serialize) and after decompression (Deserialize).
type Serializer interface {
	Serialize(data []byte) ([]byte, error)
	Deserialize(data []byte) ([]byte, error)
	ID() byte
	Name() string
}

// Registry maps serializer IDs and names to Serializer implementations.
type Registry struct {
	byID   map[byte]Serializer
	byName map[string]Serializer
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		byID:   make(map[byte]Serializer),
		byName: make(map[string]Serializer),
	}
}

// Register adds s to the registry.
func (r *Registry) Register(s Serializer) {
	r.byID[s.ID()] = s
	r.byName[s.Name()] = s
}

// Get returns the Serializer registered under id.
func (r *Registry) Get(id byte) (Serializer, error) {
	s, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("%w: id=%d", ErrUnknownSerializer, id)
	}
	return s, nil
}

// GetByName returns the Serializer registered under name.
func (r *Registry) GetByName(name string) (Serializer, error) {
	s, ok := r.byName[name]
	if !ok {
		return nil, fmt.Errorf("%w: name=%q", ErrUnknownSerializer, name)
	}
	return s, nil
}

// DefaultRegistry returns a Registry pre-loaded with raw, JSON, XML, and CBOR serializers.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(NewRawSerializer())
	r.Register(NewJSONSerializer())
	r.Register(NewXMLSerializer())
	// NewCBORSerializer can only fail if cbor enc/dec mode creation fails, which is not recoverable.
	if cborSer, err := NewCBORSerializer(); err == nil {
		r.Register(cborSer)
	}
	return r
}
