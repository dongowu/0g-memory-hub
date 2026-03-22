package workflow

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/pkg/types"
)

var ErrWorkflowNotFound = errors.New("workflow not found")

type Store interface {
	Save(meta types.WorkflowMetadata) error
	Get(workflowID string) (types.WorkflowMetadata, error)
	List() ([]types.WorkflowMetadata, error)
}

type FileStore struct {
	path string
	mu   sync.Mutex
}

func NewFileStore(path string) (*FileStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return &FileStore{path: path}, nil
}

func (f *FileStore) Save(meta types.WorkflowMetadata) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.loadUnsafe()
	if err != nil {
		return err
	}

	data[meta.WorkflowID] = meta
	return f.persistUnsafe(data)
}

func (f *FileStore) Get(workflowID string) (types.WorkflowMetadata, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.loadUnsafe()
	if err != nil {
		return types.WorkflowMetadata{}, err
	}

	meta, ok := data[workflowID]
	if !ok {
		return types.WorkflowMetadata{}, ErrWorkflowNotFound
	}
	return meta, nil
}

func (f *FileStore) List() ([]types.WorkflowMetadata, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.loadUnsafe()
	if err != nil {
		return nil, err
	}

	out := make([]types.WorkflowMetadata, 0, len(data))
	for _, meta := range data {
		out = append(out, meta)
	}
	return out, nil
}

func (f *FileStore) loadUnsafe() (map[string]types.WorkflowMetadata, error) {
	b, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string]types.WorkflowMetadata), nil
		}
		return nil, err
	}

	if len(b) == 0 {
		return make(map[string]types.WorkflowMetadata), nil
	}

	var data map[string]types.WorkflowMetadata
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	if data == nil {
		data = make(map[string]types.WorkflowMetadata)
	}
	return data, nil
}

func (f *FileStore) persistUnsafe(data map[string]types.WorkflowMetadata) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(f.path, b, 0o644)
}
