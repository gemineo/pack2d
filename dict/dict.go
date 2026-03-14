// Package dict provides dictionary types and storage for pack2d compression dictionaries.
package dict

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a dictionary is not found.
var ErrNotFound = errors.New("dict: dictionary not found")

// ErrDuplicateID is returned when a dictionary with the same ID already exists.
var ErrDuplicateID = errors.New("dict: duplicate dictionary ID")

// Dictionary holds a compression dictionary and its metadata.
type Dictionary struct {
	ID          uint16
	Name        string
	Description string
	Data        []byte
	CreatedAt   time.Time
	SampleCount int
}

// Store defines the persistence interface for dictionaries.
type Store interface {
	Get(id uint16) (*Dictionary, error)
	GetByName(name string) (*Dictionary, error)
	List() ([]*Dictionary, error)
	Save(d *Dictionary) error
	Delete(id uint16) error
	NextID() (uint16, error)
}
