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

// ErrIDExhausted is returned when all 65535 dictionary IDs are in use.
var ErrIDExhausted = errors.New("dict: all dictionary IDs exhausted (max 65535)")

// ErrInvalidName is returned when a dictionary name contains path separators or is otherwise invalid.
var ErrInvalidName = errors.New("dict: invalid dictionary name")

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
