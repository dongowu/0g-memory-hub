package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
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

type ReadinessChecker interface {
	CheckReadiness(ctx context.Context) error
}

type ComponentReadiness struct {
	Ready    bool   `json:"ready"`
	Required bool   `json:"required"`
	Message  string `json:"message,omitempty"`
}

type ReadinessReport struct {
	Ready      bool                          `json:"ready"`
	Components map[string]ComponentReadiness `json:"components"`
}

type AnchorInput struct {
	WorkflowID string
	StepIndex  uint64
	RootHash   string
	CIDHash    string
}

type Service struct {
	depsMu sync.RWMutex
	store  Store

	runtime RuntimeAPI
	storage CheckpointStorage
	anchor  CheckpointAnchor
	nowFn   func() time.Time

	workflowLocksMu sync.Mutex
	workflowLocks   map[string]*workflowLockEntry
}

type workflowDependencies struct {
	runtime RuntimeAPI
	storage CheckpointStorage
	anchor  CheckpointAnchor
}

type workflowLockEntry struct {
	mu   sync.Mutex
	refs int
}

func NewService(store Store) *Service {
	return &Service{
		store:         store,
		nowFn:         func() time.Time { return time.Now().UTC() },
		workflowLocks: make(map[string]*workflowLockEntry),
	}
}

func (s *Service) SetRuntime(runtime RuntimeAPI) {
	s.depsMu.Lock()
	defer s.depsMu.Unlock()
	s.runtime = runtime
}

func (s *Service) SetStorage(storage CheckpointStorage) {
	s.depsMu.Lock()
	defer s.depsMu.Unlock()
	s.storage = storage
}

func (s *Service) SetAnchor(anchor CheckpointAnchor) {
	s.depsMu.Lock()
	defer s.depsMu.Unlock()
	s.anchor = anchor
}

func (s *Service) Start(workflowID string) (types.WorkflowMetadata, error) {
	workflowID = s.ensureWorkflowID(workflowID)
	unlock := s.lockWorkflow(workflowID)
	defer unlock()
	return s.startLocked(workflowID)
}

func (s *Service) startLocked(workflowID string) (types.WorkflowMetadata, error) {
	if workflowID == "" {
		workflowID = s.ensureWorkflowID("")
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
	if ctx == nil {
		ctx = context.Background()
	}
	if workflowID == "" {
		return types.WorkflowMetadata{}, fmt.Errorf("workflow id is required")
	}

	unlock := s.lockWorkflow(workflowID)
	defer unlock()
	return s.stepLocked(ctx, s.dependencies(), workflowID, event)
}

func (s *Service) stepLocked(ctx context.Context, deps workflowDependencies, workflowID string, event types.WorkflowStepEvent) (types.WorkflowMetadata, error) {
	if deps.runtime == nil {
		return types.WorkflowMetadata{}, fmt.Errorf("runtime is not configured")
	}
	if deps.storage == nil {
		return types.WorkflowMetadata{}, fmt.Errorf("checkpoint storage is not configured")
	}

	meta, err := s.store.Get(workflowID)
	if err != nil {
		return types.WorkflowMetadata{}, err
	}

	if event.EventID != "" && hasEventID(meta.Events, event.EventID) {
		return meta, nil
	}

	identity := runIdentityFromEvents(meta.Events, workflowID)
	event.WorkflowID = workflowID
	if event.RunID == "" {
		event.RunID = identity.RunID
	}
	if event.SessionID == "" {
		event.SessionID = identity.SessionID
	}
	if event.TraceID == "" {
		event.TraceID = identity.TraceID
	}
	if event.Role == "" && event.Actor != "" {
		event.Role = event.Actor
	}
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

	state, err := deps.runtime.ReplayWorkflow(ctx, meta.WorkflowID, meta.AgentID, runtimeEvents)
	if err != nil {
		return types.WorkflowMetadata{}, err
	}
	checkpoint, err := deps.runtime.BuildCheckpoint(ctx, *state)
	if err != nil {
		return types.WorkflowMetadata{}, err
	}
	checkpointBlob, err := json.Marshal(checkpoint)
	if err != nil {
		return types.WorkflowMetadata{}, err
	}
	key, txHash, err := deps.storage.UploadCheckpoint(ctx, checkpointBlob)
	if err != nil {
		return types.WorkflowMetadata{}, err
	}

	meta.Status = types.WorkflowStatusRunning
	meta.LatestStep = int64(checkpoint.LatestStep)
	meta.LatestRoot = checkpoint.RootHash
	meta.LatestCID = key
	meta.LatestTxHash = txHash

	if deps.anchor != nil {
		anchorTxHash, err := deps.anchor.AnchorCheckpoint(ctx, AnchorInput{
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

func (s *Service) Ingest(ctx context.Context, event types.WorkflowStepEvent) (types.WorkflowMetadata, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	workflowID := s.ensureWorkflowID(event.WorkflowID)
	unlock := s.lockWorkflow(workflowID)
	defer unlock()

	if _, err := s.store.Get(workflowID); err != nil {
		if !errors.Is(err, ErrWorkflowNotFound) {
			return types.WorkflowMetadata{}, err
		}
		if _, err := s.startLocked(workflowID); err != nil {
			return types.WorkflowMetadata{}, err
		}
	}

	return s.stepLocked(ctx, s.dependencies(), workflowID, event)
}

func (s *Service) Resume(workflowID string) (types.WorkflowMetadata, error) {
	return s.ResumeWithContext(context.Background(), workflowID)
}

func (s *Service) ResumeWithContext(ctx context.Context, workflowID string) (types.WorkflowMetadata, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	unlock := s.lockWorkflow(workflowID)
	defer unlock()
	return s.resumeLocked(ctx, s.dependencies(), workflowID)
}

func (s *Service) resumeLocked(ctx context.Context, deps workflowDependencies, workflowID string) (types.WorkflowMetadata, error) {
	meta, err := s.store.Get(workflowID)
	if err != nil {
		return types.WorkflowMetadata{}, err
	}

	if deps.storage != nil && meta.LatestCID != "" {
		payload, err := deps.storage.DownloadCheckpoint(ctx, meta.LatestCID)
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
			meta.Events = fromRuntimeEvents(meta.WorkflowID, checkpoint.Events, meta.Events)
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
	for _, evt := range meta.Events {
		lines = append(lines, fmt.Sprintf("step=%d event_id=%s type=%s actor=%s payload=%s", evt.StepIndex, evt.EventID, evt.EventType, evt.Actor, evt.Payload))
	}
	return lines, nil
}

func (s *Service) Status(workflowID string) (types.WorkflowMetadata, error) {
	return s.store.Get(workflowID)
}

func (s *Service) RunContext(runID string) (RunContext, error) {
	meta, err := s.metadataForRun(runID)
	if err != nil {
		return RunContext{}, err
	}
	return buildRunContext(meta), nil
}

func (s *Service) LatestCheckpoint(runID string) (LatestCheckpoint, error) {
	meta, err := s.metadataForRun(runID)
	if err != nil {
		return LatestCheckpoint{}, err
	}
	return buildLatestCheckpoint(meta), nil
}

func (s *Service) Hydrate(ctx context.Context, runID string) (RunContext, error) {
	meta, err := s.metadataForRun(runID)
	if err != nil {
		return RunContext{}, err
	}
	meta, err = s.ResumeWithContext(ctx, meta.WorkflowID)
	if err != nil {
		return RunContext{}, err
	}
	return buildRunContext(meta), nil
}

func (s *Service) RunTrace(runID string) (RunTrace, error) {
	meta, err := s.metadataForRun(runID)
	if err != nil {
		return RunTrace{}, err
	}
	return buildRunTrace(meta), nil
}

func (s *Service) metadataForRun(runID string) (types.WorkflowMetadata, error) {
	return s.store.FindByRunID(runID)
}

func (s *Service) Readiness(ctx context.Context) ReadinessReport {
	deps := s.dependencies()

	report := ReadinessReport{
		Ready: true,
		Components: map[string]ComponentReadiness{
			"runtime": probeReadiness(ctx, deps.runtime, true),
			"storage": probeReadiness(ctx, deps.storage, true),
			"anchor":  probeReadiness(ctx, deps.anchor, false),
		},
	}

	for _, component := range report.Components {
		if component.Required && !component.Ready {
			report.Ready = false
		}
	}
	return report
}

func (s *Service) ensureWorkflowID(workflowID string) string {
	if workflowID != "" {
		return workflowID
	}
	return fmt.Sprintf("wf_%d", s.nowFn().UnixNano())
}

func (s *Service) dependencies() workflowDependencies {
	s.depsMu.RLock()
	defer s.depsMu.RUnlock()
	return workflowDependencies{
		runtime: s.runtime,
		storage: s.storage,
		anchor:  s.anchor,
	}
}

func (s *Service) lockWorkflow(workflowID string) func() {
	if workflowID == "" {
		return func() {}
	}

	s.workflowLocksMu.Lock()
	if s.workflowLocks == nil {
		s.workflowLocks = make(map[string]*workflowLockEntry)
	}
	entry := s.workflowLocks[workflowID]
	if entry == nil {
		entry = &workflowLockEntry{}
		s.workflowLocks[workflowID] = entry
	}
	entry.refs++
	s.workflowLocksMu.Unlock()

	entry.mu.Lock()

	return func() {
		entry.mu.Unlock()

		s.workflowLocksMu.Lock()
		entry.refs--
		if entry.refs == 0 {
			delete(s.workflowLocks, workflowID)
		}
		s.workflowLocksMu.Unlock()
	}
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

func fromRuntimeEvents(workflowID string, events []RuntimeEvent, prior []types.WorkflowStepEvent) []types.WorkflowStepEvent {
	byEventID := make(map[string]types.WorkflowStepEvent, len(prior))
	for _, event := range prior {
		if event.EventID != "" {
			byEventID[event.EventID] = event
		}
	}

	out := make([]types.WorkflowStepEvent, 0, len(events))
	for _, evt := range events {
		preserved, hasPreserved := byEventID[evt.EventID]
		out = append(out, types.WorkflowStepEvent{
			EventID:       evt.EventID,
			WorkflowID:    workflowID,
			RunID:         preserved.RunID,
			SessionID:     preserved.SessionID,
			TraceID:       preserved.TraceID,
			ParentEventID: preserved.ParentEventID,
			ToolCallID:    preserved.ToolCallID,
			SkillName:     preserved.SkillName,
			TaskID:        preserved.TaskID,
			Role:          preserved.Role,
			StepIndex:     int64(evt.StepIndex),
			EventType:     evt.EventType,
			Actor:         evt.Actor,
			Payload:       evt.Payload,
			CreatedAt:     preserved.CreatedAt,
		})
		if hasPreserved && out[len(out)-1].Role == "" && out[len(out)-1].Actor != "" {
			out[len(out)-1].Role = out[len(out)-1].Actor
		}
		if out[len(out)-1].RunID == "" {
			out[len(out)-1].RunID = workflowID
		}
	}
	return out
}

func hasEventID(events []types.WorkflowStepEvent, eventID string) bool {
	for _, evt := range events {
		if evt.EventID == eventID {
			return true
		}
	}
	return false
}

func probeReadiness(ctx context.Context, dep any, required bool) ComponentReadiness {
	component := ComponentReadiness{
		Ready:    true,
		Required: required,
	}

	if dep == nil {
		if required {
			component.Ready = false
			component.Message = "not configured"
			return component
		}
		component.Message = "not configured (optional)"
		return component
	}

	checker, ok := dep.(ReadinessChecker)
	if !ok {
		component.Message = "configured"
		return component
	}
	if err := checker.CheckReadiness(ctx); err != nil {
		component.Ready = false
		component.Message = err.Error()
		return component
	}
	component.Message = "ok"
	return component
}
