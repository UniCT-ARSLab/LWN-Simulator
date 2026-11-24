package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

var ErrCheckpointNotFound = errors.New("simulation checkpoint not found")

type SimulationCheckpointStore interface {
	Load() (*types.EngineCheckpoint, error)
	Save(types.EngineCheckpoint) error
	Clear() error
}

type FileSimulationCheckpointStore struct {
	path string
}

func NewFileSimulationCheckpointStore(path string) *FileSimulationCheckpointStore {
	return &FileSimulationCheckpointStore{path: path}
}

func (store *FileSimulationCheckpointStore) Load() (*types.EngineCheckpoint, error) {
	data, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read simulation checkpoint: %w", err)
	}
	var checkpoint types.EngineCheckpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("decode simulation checkpoint: %w", err)
	}
	return &checkpoint, nil
}

func (store *FileSimulationCheckpointStore) Save(checkpoint types.EngineCheckpoint) error {
	data, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("encode simulation checkpoint: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(store.path), 0700); err != nil {
		return fmt.Errorf("create checkpoint directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(store.path), ".simulation-checkpoint-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary checkpoint: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set checkpoint permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary checkpoint: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary checkpoint: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary checkpoint: %w", err)
	}
	if err := os.Rename(temporaryName, store.path); err != nil {
		return fmt.Errorf("replace simulation checkpoint: %w", err)
	}
	return nil
}

func (store *FileSimulationCheckpointStore) Clear() error {
	if err := os.Remove(store.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove simulation checkpoint: %w", err)
	}
	return nil
}
