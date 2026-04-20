package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/swarm-deploy/swarm-deploy/internal/swarm"
)

const storeFileModePrivate = 0o600

// Store persists nodes snapshot in a JSON file.
type Store struct {
	mu   sync.RWMutex
	path string
	rows []swarm.Node
}

// NewNodeStore creates nodes store and loads saved rows from disk.
func NewNodeStore(path string) (*Store, error) {
	s := &Store{
		path: path,
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

// List returns a copy of all saved nodes.
func (s *Store) List() []swarm.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]swarm.Node, len(s.rows))
	copy(out, s.rows)
	return out
}

// Replace replaces nodes snapshot and saves it to disk.
func (s *Store) Replace(nodes []swarm.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rows = nodes
	sortNodes(s.rows)

	return s.flushLocked()
}

func (s *Store) load() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create nodes dir: %w", err)
	}

	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("read nodes file: %w", err)
	}
	if len(payload) == 0 {
		return nil
	}

	var rows []swarm.Node
	if unmarshalErr := json.Unmarshal(payload, &rows); unmarshalErr != nil {
		return fmt.Errorf("decode nodes file: %w", unmarshalErr)
	}

	s.rows = rows

	sortNodes(s.rows)
	return nil
}

func (s *Store) flushLocked() error {
	payload, err := json.Marshal(s.rows)
	if err != nil {
		return fmt.Errorf("encode nodes file: %w", err)
	}

	tmpPath := fmt.Sprintf("%s.tmp", s.path)
	if writeErr := os.WriteFile(tmpPath, payload, storeFileModePrivate); writeErr != nil {
		return fmt.Errorf("write nodes temp file: %w", writeErr)
	}
	if renameErr := os.Rename(tmpPath, s.path); renameErr != nil {
		return fmt.Errorf("replace nodes file: %w", renameErr)
	}

	return nil
}

func sortNodes(nodes []swarm.Node) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Hostname != nodes[j].Hostname {
			return nodes[i].Hostname < nodes[j].Hostname
		}

		return nodes[i].ID < nodes[j].ID
	})
}
