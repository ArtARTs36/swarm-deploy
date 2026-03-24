package inspector

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const nodeStoreFileModePrivate = 0o600

// NodeStore persists nodes snapshot in a JSON file.
type NodeStore struct {
	mu   sync.RWMutex
	path string
	rows []NodeInfo
}

// NewNodeStore creates nodes store and loads saved rows from disk.
func NewNodeStore(path string) (*NodeStore, error) {
	s := &NodeStore{
		path: path,
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

// List returns a copy of all saved nodes.
func (s *NodeStore) List() []NodeInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]NodeInfo, len(s.rows))
	copy(out, s.rows)
	return out
}

// Replace replaces nodes snapshot and saves it to disk.
func (s *NodeStore) Replace(nodes []NodeInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	updated := make([]NodeInfo, 0, len(nodes))
	for _, node := range nodes {
		normalized := normalizeNodeInfo(node)
		if normalized.ID == "" {
			continue
		}
		updated = append(updated, normalized)
	}

	sortNodeInfos(updated)
	s.rows = updated

	return s.flushLocked()
}

func (s *NodeStore) load() error {
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

	var rows []NodeInfo
	if unmarshalErr := json.Unmarshal(payload, &rows); unmarshalErr != nil {
		return fmt.Errorf("decode nodes file: %w", unmarshalErr)
	}

	s.rows = make([]NodeInfo, 0, len(rows))
	for _, row := range rows {
		normalized := normalizeNodeInfo(row)
		if normalized.ID == "" {
			continue
		}
		s.rows = append(s.rows, normalized)
	}

	sortNodeInfos(s.rows)
	return nil
}

func (s *NodeStore) flushLocked() error {
	payload, err := json.Marshal(s.rows)
	if err != nil {
		return fmt.Errorf("encode nodes file: %w", err)
	}

	tmpPath := fmt.Sprintf("%s.tmp", s.path)
	if writeErr := os.WriteFile(tmpPath, payload, nodeStoreFileModePrivate); writeErr != nil {
		return fmt.Errorf("write nodes temp file: %w", writeErr)
	}
	if renameErr := os.Rename(tmpPath, s.path); renameErr != nil {
		return fmt.Errorf("replace nodes file: %w", renameErr)
	}

	return nil
}
