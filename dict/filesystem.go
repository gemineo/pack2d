package dict

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gojson "github.com/go-json-experiment/json"
)

type filesystemStore struct {
	dir string
}

// filesystemMeta is the on-disk JSON metadata for a dictionary.
type filesystemMeta struct {
	ID          uint16    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	SampleCount int       `json:"sample_count"`
}

// NewFilesystemStore returns a Store backed by the given directory.
// The directory must already exist.
func NewFilesystemStore(dir string) (Store, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("dict filesystem: stat dir: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("dict filesystem: %q is not a directory", dir)
	}
	return &filesystemStore{dir: dir}, nil
}

func (s *filesystemStore) dictPath(id uint16, name string) string {
	return filepath.Join(s.dir, fmt.Sprintf("%04d_%s.dict", id, name))
}

func (s *filesystemStore) metaPath(id uint16, name string) string {
	return filepath.Join(s.dir, fmt.Sprintf("%04d_%s.json", id, name))
}

func (s *filesystemStore) findFiles(id uint16) (dictFile, metaFile string, err error) {
	pattern := filepath.Join(s.dir, fmt.Sprintf("%04d_*.dict", id))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", "", fmt.Errorf("dict filesystem: glob: %w", err)
	}
	if len(matches) == 0 {
		return "", "", fmt.Errorf("%w: id=%d", ErrNotFound, id)
	}
	dictFile = matches[0]
	metaFile = strings.TrimSuffix(dictFile, ".dict") + ".json"
	return dictFile, metaFile, nil
}

func (s *filesystemStore) loadMeta(metaFile string) (*filesystemMeta, error) {
	data, err := os.ReadFile(metaFile)
	if err != nil {
		return nil, fmt.Errorf("dict filesystem: read meta: %w", err)
	}
	var m filesystemMeta
	if err := gojson.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("dict filesystem: unmarshal meta: %w", err)
	}
	return &m, nil
}

func (s *filesystemStore) Get(id uint16) (*Dictionary, error) {
	dictFile, metaFile, err := s.findFiles(id)
	if err != nil {
		return nil, err
	}
	m, err := s.loadMeta(metaFile)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(dictFile)
	if err != nil {
		return nil, fmt.Errorf("dict filesystem: read dict: %w", err)
	}
	return &Dictionary{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		Data:        data,
		CreatedAt:   m.CreatedAt,
		SampleCount: m.SampleCount,
	}, nil
}

func (s *filesystemStore) GetByName(name string) (*Dictionary, error) {
	pattern := filepath.Join(s.dir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("dict filesystem: glob meta: %w", err)
	}
	for _, f := range matches {
		m, err := s.loadMeta(f)
		if err != nil {
			continue
		}
		if m.Name == name {
			dictFile := strings.TrimSuffix(f, ".json") + ".dict"
			data, err := os.ReadFile(dictFile)
			if err != nil {
				return nil, fmt.Errorf("dict filesystem: read dict: %w", err)
			}
			return &Dictionary{
				ID:          m.ID,
				Name:        m.Name,
				Description: m.Description,
				Data:        data,
				CreatedAt:   m.CreatedAt,
				SampleCount: m.SampleCount,
			}, nil
		}
	}
	return nil, fmt.Errorf("%w: name=%q", ErrNotFound, name)
}

func (s *filesystemStore) List() ([]*Dictionary, error) {
	pattern := filepath.Join(s.dir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("dict filesystem: glob meta: %w", err)
	}
	out := make([]*Dictionary, 0, len(matches))
	for _, f := range matches {
		m, err := s.loadMeta(f)
		if err != nil {
			continue
		}
		dictFile := strings.TrimSuffix(f, ".json") + ".dict"
		data, err := os.ReadFile(dictFile)
		if err != nil {
			continue
		}
		out = append(out, &Dictionary{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Data:        data,
			CreatedAt:   m.CreatedAt,
			SampleCount: m.SampleCount,
		})
	}
	return out, nil
}

func (s *filesystemStore) Save(d *Dictionary) error {
	if d.ID == 0 {
		id, err := s.NextID()
		if err != nil {
			return err
		}
		d.ID = id
	}

	dictFile := s.dictPath(d.ID, d.Name)
	metaFile := s.metaPath(d.ID, d.Name)

	if err := os.WriteFile(dictFile, d.Data, 0o644); err != nil {
		return fmt.Errorf("dict filesystem: write dict: %w", err)
	}

	m := filesystemMeta{
		ID:          d.ID,
		Name:        d.Name,
		Description: d.Description,
		CreatedAt:   d.CreatedAt,
		SampleCount: d.SampleCount,
	}
	metaBytes, err := gojson.Marshal(m)
	if err != nil {
		return fmt.Errorf("dict filesystem: marshal meta: %w", err)
	}
	if err := os.WriteFile(metaFile, metaBytes, 0o644); err != nil {
		return fmt.Errorf("dict filesystem: write meta: %w", err)
	}
	return nil
}

func (s *filesystemStore) Delete(id uint16) error {
	dictFile, metaFile, err := s.findFiles(id)
	if err != nil {
		return err
	}
	if err := os.Remove(dictFile); err != nil {
		return fmt.Errorf("dict filesystem: remove dict: %w", err)
	}
	if err := os.Remove(metaFile); err != nil {
		return fmt.Errorf("dict filesystem: remove meta: %w", err)
	}
	return nil
}

func (s *filesystemStore) NextID() (uint16, error) {
	pattern := filepath.Join(s.dir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return 0, fmt.Errorf("dict filesystem: glob for next id: %w", err)
	}
	var maxID uint16
	for _, f := range matches {
		base := filepath.Base(f)
		parts := strings.SplitN(base, "_", 2)
		if len(parts) < 2 {
			continue
		}
		n, err := strconv.ParseUint(parts[0], 10, 16)
		if err != nil {
			continue
		}
		if uint16(n) > maxID {
			maxID = uint16(n)
		}
	}
	return maxID + 1, nil
}
