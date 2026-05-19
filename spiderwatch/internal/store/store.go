// Package store manages persistence and retrieval of RunResult records.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/cloud-barista/cb-spider/spiderwatch/internal/model"
	"github.com/sirupsen/logrus"
)

const defaultDataDir = "data/results"

var log = logrus.New()

// Store is a file-backed result store.
type Store struct {
	mu      sync.RWMutex
	dataDir string
}

// New creates a Store using the given directory (created if absent).
func New(dataDir string) (*Store, error) {
	if dataDir == "" {
		dataDir = defaultDataDir
	}
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return nil, fmt.Errorf("store: create data dir %q: %w", dataDir, err)
	}
	return &Store{dataDir: dataDir}, nil
}

// Save persists a RunResult to disk as JSON.
func (s *Store) Save(r *model.RunResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.filePath(r.ID)
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("store: marshal result %q: %w", r.ID, err)
	}
	// Write atomically: temp file then rename
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return fmt.Errorf("store: write temp file %q: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("store: rename %q -> %q: %w", tmp, path, err)
	}
	return nil
}

// Get returns the RunResult for the given ID, or an error if not found.
func (s *Store) Get(id string) (*model.RunResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.readFile(s.filePath(id))
}

// Latest returns the most recent RunResult, or nil if no results exist.
func (s *Store) Latest() (*model.RunResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids, err := s.listIDs()
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return s.readFile(s.filePath(ids[len(ids)-1]))
}

// List returns all RunResults ordered from oldest to newest.
func (s *Store) List() ([]*model.RunResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids, err := s.listIDs()
	if err != nil {
		return nil, err
	}
	results := make([]*model.RunResult, 0, len(ids))
	for _, id := range ids {
		r, err := s.readFile(s.filePath(id))
		if err != nil {
			log.WithError(err).Warnf("store: skipping corrupt file for id=%s", id)
			continue
		}
		results = append(results, r)
	}
	return results, nil
}

// ListIDs returns sorted run IDs (newest last).
func (s *Store) ListIDs() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listIDs()
}

// Delete removes the run file for the given ID.
// Returns nil if the file did not exist (idempotent).
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.filePath(id)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("store: delete %q: %w", path, err)
	}
	return nil
}

func (s *Store) listIDs() ([]string, error) {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("store: read dir %q: %w", s.dataDir, err)
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) == ".json" {
			ids = append(ids, name[:len(name)-5])
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *Store) filePath(id string) string {
	// Sanitize id to prevent path traversal
	base := filepath.Base(id)
	return filepath.Join(s.dataDir, base+".json")
}

func (s *Store) readFile(path string) (*model.RunResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("store: read %q: %w", path, err)
	}
	var r model.RunResult
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("store: unmarshal %q: %w", path, err)
	}
	return &r, nil
}
