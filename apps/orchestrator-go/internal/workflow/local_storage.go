package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// LocalFileStorage is a CheckpointStorage implementation that stores checkpoints
// on the local filesystem. Used as a fallback when 0G storage is not configured.
type LocalFileStorage struct {
	mu      sync.Mutex
	dataDir string
}

// NewLocalFileStorage creates a LocalFileStorage that stores blobs under dataDir/checkpoints/.
func NewLocalFileStorage(dataDir string) (*LocalFileStorage, error) {
	dir := filepath.Join(dataDir, "checkpoints")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create local storage dir: %w", err)
	}
	return &LocalFileStorage{dataDir: dir}, nil
}

// UploadCheckpoint stores the checkpoint blob to a local file and returns
// a synthetic key (SHA256 of content) and empty txHash.
func (s *LocalFileStorage) UploadCheckpoint(ctx context.Context, payload []byte) (key string, txHash string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := sha256.Sum256(payload)
	key = hex.EncodeToString(hash[:])
	path := filepath.Join(s.dataDir, key+".blob")
	if err := os.WriteFile(path, payload, 0644); err != nil {
		return "", "", fmt.Errorf("write checkpoint file: %w", err)
	}
	return key, "", nil
}

// DownloadCheckpoint reads a checkpoint blob from the local filesystem.
func (s *LocalFileStorage) DownloadCheckpoint(ctx context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dataDir, key+".blob")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint file: %w", err)
	}
	return data, nil
}

// CheckReadiness verifies the local storage directory is writable.
func (s *LocalFileStorage) CheckReadiness(ctx context.Context) error {
	testPath := filepath.Join(s.dataDir, ".readiness_check")
	if err := os.WriteFile(testPath, []byte("ok"), 0644); err != nil {
		return fmt.Errorf("local storage not writable: %w", err)
	}
	_ = os.Remove(testPath)
	return nil
}

// LocalAnchor is a CheckpointAnchor that stores anchor metadata on the local
// filesystem, used when 0G chain is not configured.
type LocalAnchor struct {
	mu      sync.RWMutex
	dataDir string
}

// NewLocalAnchor creates a LocalAnchor that persists anchors to the given data directory.
func NewLocalAnchor(dataDir string) (*LocalAnchor, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create anchor dir: %w", err)
	}
	return &LocalAnchor{dataDir: dataDir}, nil
}

func (a *LocalAnchor) anchorPath() string {
	return filepath.Join(a.dataDir, "anchors.json")
}

func (a *LocalAnchor) loadAll() (map[string]AnchorLatestCheckpoint, error) {
	path := a.anchorPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]AnchorLatestCheckpoint), nil
		}
		return nil, err
	}
	var anchors map[string]AnchorLatestCheckpoint
	if len(data) > 0 {
		if err := json.Unmarshal(data, &anchors); err != nil {
			return nil, err
		}
	}
	if anchors == nil {
		anchors = make(map[string]AnchorLatestCheckpoint)
	}
	return anchors, nil
}

func (a *LocalAnchor) saveAll(anchors map[string]AnchorLatestCheckpoint) error {
	path := a.anchorPath()
	data, err := json.Marshal(anchors)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (a *LocalAnchor) AnchorCheckpoint(ctx context.Context, in AnchorInput) (string, error) {
	anchors, err := a.loadAll()
	if err != nil {
		return "", fmt.Errorf("load anchors: %w", err)
	}

	hash := sha256.Sum256([]byte(in.WorkflowID + in.RootHash))
	txHash := "0x" + hex.EncodeToString(hash[:16])

	key := hashToBytes32Hex(in.WorkflowID)
	anchors[key] = AnchorLatestCheckpoint{
		StepIndex:  in.StepIndex,
		RootHash:   in.RootHash,
		CIDHash:    in.CIDHash,
	}

	if err := a.saveAll(anchors); err != nil {
		return "", fmt.Errorf("save anchors: %w", err)
	}
	return txHash, nil
}

func (a *LocalAnchor) GetLatestCheckpoint(ctx context.Context, workflowID string) (AnchorLatestCheckpoint, error) {
	anchors, err := a.loadAll()
	if err != nil {
		return AnchorLatestCheckpoint{}, fmt.Errorf("load anchors: %w", err)
	}

	key := hashToBytes32Hex(workflowID)
	anchor, ok := anchors[key]
	if !ok {
		return AnchorLatestCheckpoint{}, fmt.Errorf("no anchor found for workflow %s", workflowID)
	}
	return anchor, nil
}

func (a *LocalAnchor) CheckReadiness(ctx context.Context) error {
	testDir := filepath.Join(a.dataDir, ".probe")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return fmt.Errorf("local anchor dir not writable: %w", err)
	}
	_ = os.RemoveAll(testDir)
	return nil
}
