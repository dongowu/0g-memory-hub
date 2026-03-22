package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/pkg/types"
)

type RuntimeAPI interface {
	ReplayWorkflow(ctx context.Context, workflowID, agentID string, events []RuntimeEvent) (*RuntimeState, error)
	BuildCheckpoint(ctx context.Context, state RuntimeState) (*RuntimeCheckpoint, error)
}

type CheckpointStorage interface {
	UploadCheckpoint(ctx context.Context, payload []byte) (key string, txHash string, err error)
	DownloadCheckpoint(ctx context.Context, key string) ([]byte, error)
}

type CheckpointAnchor interface {
	AnchorCheckpoint(ctx context.Context, in AnchorInput) (string, error)
}

type AnchorInput struct {
	WorkflowID string
	StepIndex  uint64
	RootHash   string
	CIDHash    string
}

type Service struct {
	store   Store
	runtime RuntimeAPI
	storage CheckpointStorage
	anchor  CheckpointAnchor
	nowFn   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		nowFn: func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) SetRuntime(runtime RuntimeAPI) {
	s.runtime = runtime
}

func (s *Service) SetStorage(storage CheckpointStorage) {
	s.storage = storage
}

func (s *Service) SetAnchor(anchor CheckpointAnchor) {
	s.anchor = anchor
}

func (s *Service) Start(workflowID string) (types.WorkflowMetadata, error) {
	if workflowID == "" {
		workflowID = fmt.Sprintf("wf_%d", time.Now().UnixNano())
	}

	meta := types.WorkflowMetadata{
		WorkflowID: workflowID,
		AgentID:    fmt.Sprintf("agent-%s", workflowID),
		Status:     types.WorkflowStatusRunning,
		LatestStep: 0,
		Events:     make([]types.WorkflowStepEvent, 0),
		UpdatedAt:  s.nowFn(),
	}

	if err := s.store.Save(meta); err != nil {
		return types.WorkflowMetadata{}, err
	}
	return meta, nil
}

func (s *Service) Step(ctx context.Context, workflowID string, event types.WorkflowStepEvent) (types.WorkflowMetadata, error) {
	if s.runtime == nil {
		return types.WorkflowMetadata{}, fmt.Errorf("runtime is not configured")
	}
	if s.storage == nil {
		return types.WorkflowMetadata{}, fmt.Errorf("checkpoint storage is not configured")
	}

	meta, err := s.store.Get(workflowID)
	if err != nil {
		return types.WorkflowMetadata{}, err
	}

	event.WorkflowID = workflowID
	event.StepIndex = meta.LatestStep
	if event.EventID == "" {
		event.EventID = fmt.Sprintf("%s-step-%d", workflowID, event.StepIndex)
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = s.nowFn()
	}
	meta.Events = append(meta.Events, event)

	runtimeEvents := make([]RuntimeEvent, 0, len(meta.Events))
	for _, e := range meta.Events {
		runtimeEvents = append(runtimeEvents, RuntimeEvent{
			EventID:   e.EventID,
			StepIndex: uint64(e.StepIndex),
			EventType: e.EventType,
			Actor:     e.Actor,
			Payload:   e.Payload,
		})
	}

	state, err := s.runtime.ReplayWorkflow(ctx, meta.WorkflowID, meta.AgentID, runtimeEvents)
	if err != nil {
		return types.WorkflowMetadata{}, err
	}
	checkpoint, err := s.runtime.BuildCheckpoint(ctx, *state)
	if err != nil {
		return types.WorkflowMetadata{}, err
	}
	checkpointBlob, err := json.Marshal(checkpoint)
	if err != nil {
		return types.WorkflowMetadata{}, err
	}
	key, txHash, err := s.storage.UploadCheckpoint(ctx, checkpointBlob)
	if err != nil {
		return types.WorkflowMetadata{}, err
	}

	meta.Status = types.WorkflowStatusRunning
	meta.LatestStep = int64(checkpoint.LatestStep)
	meta.LatestRoot = checkpoint.RootHash
	meta.LatestCID = key
	meta.LatestTxHash = txHash

	if s.anchor != nil {
		anchorTxHash, err := s.anchor.AnchorCheckpoint(ctx, AnchorInput{
			WorkflowID: hashToBytes32Hex(meta.WorkflowID),
			StepIndex:  checkpoint.LatestStep,
			RootHash:   normalizeBytes32Hex(checkpoint.RootHash),
			CIDHash:    hashToBytes32Hex(key),
		})
		if err != nil {
			return types.WorkflowMetadata{}, err
		}
		if anchorTxHash != "" {
			meta.LatestTxHash = anchorTxHash
		}
	}
	meta.UpdatedAt = s.nowFn()

	if err := s.store.Save(meta); err != nil {
		return types.WorkflowMetadata{}, err
	}
	return meta, nil
}

func (s *Service) Resume(workflowID string) (types.WorkflowMetadata, error) {
	meta, err := s.store.Get(workflowID)
	if err != nil {
		return types.WorkflowMetadata{}, err
	}

	if s.storage != nil && meta.LatestCID != "" {
		payload, err := s.storage.DownloadCheckpoint(context.Background(), meta.LatestCID)
		if err != nil {
			return types.WorkflowMetadata{}, err
		}
		var checkpoint RuntimeCheckpoint
		if err := json.Unmarshal(payload, &checkpoint); err != nil {
			return types.WorkflowMetadata{}, err
		}
		if checkpoint.WorkflowID == meta.WorkflowID {
			meta.LatestStep = int64(checkpoint.LatestStep)
			meta.LatestRoot = checkpoint.RootHash
			meta.Events = fromRuntimeEvents(checkpoint.Events)
		}
	}

	meta.Status = types.WorkflowStatusRunning
	meta.UpdatedAt = s.nowFn()
	if err := s.store.Save(meta); err != nil {
		return types.WorkflowMetadata{}, err
	}
	return meta, nil
}

func (s *Service) Replay(workflowID string) ([]string, error) {
	meta, err := s.store.Get(workflowID)
	if err != nil {
		return nil, err
	}

	lines := make([]string, 0, len(meta.Events)+1)
	lines = append(lines, fmt.Sprintf("workflow=%s status=%s latest_root=%s latest_cid=%s", meta.WorkflowID, meta.Status, meta.LatestRoot, meta.LatestCID))
	for i, evt := range meta.Events {
		lines = append(lines, fmt.Sprintf("step=%d event_id=%s type=%s actor=%s payload=%s", i, evt.EventID, evt.EventType, evt.Actor, evt.Payload))
	}
	return lines, nil
}

func (s *Service) Status(workflowID string) (types.WorkflowMetadata, error) {
	return s.store.Get(workflowID)
}

func hashToBytes32Hex(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

func normalizeBytes32Hex(v string) string {
	trimmed := strings.TrimPrefix(strings.ToLower(v), "0x")
	if len(trimmed) == 64 {
		return trimmed
	}
	return hashToBytes32Hex(v)
}

func fromRuntimeEvents(events []RuntimeEvent) []types.WorkflowStepEvent {
	out := make([]types.WorkflowStepEvent, 0, len(events))
	for _, evt := range events {
		out = append(out, types.WorkflowStepEvent{
			EventID:   evt.EventID,
			StepIndex: int64(evt.StepIndex),
			EventType: evt.EventType,
			Actor:     evt.Actor,
			Payload:   evt.Payload,
		})
	}
	return out
}
