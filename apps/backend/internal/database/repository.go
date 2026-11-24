package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var ErrNotFound = errors.New("item not found")

type JSONRepository[T any] struct {
	path string
	mu   sync.RWMutex
}

func NewJSONRepository[T any](dataDir string, filename string) *JSONRepository[T] {
	return &JSONRepository[T]{
		path: filepath.Join(dataDir, filename),
	}
}

func (r *JSONRepository[T]) GetAll() ([]T, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.load()
}

func (r *JSONRepository[T]) GetByID(
	id string,
	idGetter func(T) string,
) (T, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items, err := r.load()
	if err != nil {
		var zero T
		return zero, err
	}

	for _, item := range items {
		if idGetter(item) == id {
			return item, nil
		}
	}

	var zero T

	return zero, fmt.Errorf("%w: item with id %q not found", ErrNotFound, id)
}

func (r *JSONRepository[T]) Update(
	id string,
	item T,
	idGetter func(T) string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	items, err := r.load()
	if err != nil {
		return err
	}

	for i, existing := range items {
		if idGetter(existing) != id {
			continue
		}

		items[i] = item

		if err := r.save(items); err != nil {
			return fmt.Errorf("update item with id %q: %w", id, err)
		}

		return nil
	}

	return fmt.Errorf("%w: item with id %q not found", ErrNotFound, id)
}

func (r *JSONRepository[T]) Delete(
	id string,
	idGetter func(T) string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	items, err := r.load()
	if err != nil {
		return err
	}

	for i, item := range items {
		if idGetter(item) != id {
			continue
		}

		items = append(items[:i], items[i+1:]...)

		if err := r.save(items); err != nil {
			return fmt.Errorf("delete item with id %q: %w", id, err)
		}

		return nil
	}

	return fmt.Errorf("%w: item with id %q not found", ErrNotFound, id)
}

func (r *JSONRepository[T]) Save(items []T) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.save(items)
}

func (r *JSONRepository[T]) load() ([]T, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", r.path, err)
	}

	var items []T

	if len(data) == 0 {
		return []T{}, nil
	}

	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("decode %s: %w", r.path, err)
	}

	return items, nil
}

func (r *JSONRepository[T]) save(items []T) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", r.path, err)
	}

	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(r.path), 0700); err != nil {
		return fmt.Errorf("create directory for %s: %w", r.path, err)
	}

	temp, err := os.CreateTemp(filepath.Dir(r.path), ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", r.path, err)
	}

	tempPath := temp.Name()

	defer func() {
		_ = os.Remove(tempPath)
	}()

	if err := temp.Chmod(0600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("set permissions for %s: %w", tempPath, err)
	}

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write temporary file %s: %w", tempPath, err)
	}

	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync temporary file %s: %w", tempPath, err)
	}

	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary file %s: %w", tempPath, err)
	}

	if err := os.Rename(tempPath, r.path); err != nil {
		return fmt.Errorf("replace %s: %w", r.path, err)
	}

	return nil
}
