package dict

import (
	"fmt"
	"sync"
)

// memoryStore is a thread-safe in-memory Store implementation.
type memoryStore struct {
	mu   sync.RWMutex
	data map[uint16]*Dictionary
}

// NewMemoryStore returns a thread-safe in-memory Store.
func NewMemoryStore() Store {
	return &memoryStore{data: make(map[uint16]*Dictionary)}
}

func (m *memoryStore) Get(id uint16) (*Dictionary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("%w: id=%d", ErrNotFound, id)
	}
	return copyDict(d), nil
}

func (m *memoryStore) GetByName(name string) (*Dictionary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, d := range m.data {
		if d.Name == name {
			return copyDict(d), nil
		}
	}
	return nil, fmt.Errorf("%w: name=%q", ErrNotFound, name)
}

func (m *memoryStore) List() ([]*Dictionary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Dictionary, 0, len(m.data))
	for _, d := range m.data {
		out = append(out, copyDict(d))
	}
	return out, nil
}

func (m *memoryStore) Save(d *Dictionary) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d.ID == 0 {
		id, err := m.nextIDLocked()
		if err != nil {
			return err
		}
		d.ID = id
	}
	m.data[d.ID] = copyDict(d)
	return nil
}

func (m *memoryStore) Delete(id uint16) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[id]; !ok {
		return fmt.Errorf("%w: id=%d", ErrNotFound, id)
	}
	delete(m.data, id)
	return nil
}

func (m *memoryStore) NextID() (uint16, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nextIDLocked()
}

func (m *memoryStore) nextIDLocked() (uint16, error) {
	var max uint16
	for id := range m.data {
		if id > max {
			max = id
		}
	}
	if max == ^uint16(0) {
		return 0, ErrIDExhausted
	}
	return max + 1, nil
}

func copyDict(d *Dictionary) *Dictionary {
	cp := *d
	if d.Data != nil {
		cp.Data = make([]byte, len(d.Data))
		copy(cp.Data, d.Data)
	}
	return &cp
}
