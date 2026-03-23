package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/pkg/types"
)

type fakeRuntime struct {
	lastWorkflowID string
	lastAgentID    string
	lastEvents     []RuntimeEvent
}

func (f *fakeRuntime) ReplayWorkflow(_ context.Context, workflowID, agentID string, events []RuntimeEvent) (*RuntimeState, error) {
	f.lastWorkflowID = workflowID
	f.lastAgentID = agentID
	f.lastEvents = append([]RuntimeEvent(nil), events...)
	return &RuntimeState{
		WorkflowID: workflowID,
		AgentID:    agentID,
		Status:     RuntimeStatusRunning,
		LatestStep: uint64(len(events)),
		LatestRoot: "root-from-runtime",
		Events:     append([]RuntimeEvent(nil), events...),
	}, nil
}

func (f *fakeRuntime) BuildCheckpoint(_ context.Context, state RuntimeState) (*RuntimeCheckpoint, error) {
	return &RuntimeCheckpoint{
		WorkflowID: state.WorkflowID,
		AgentID:    state.AgentID,
		LatestStep: state.LatestStep,
		RootHash:   state.LatestRoot,
		Status:     state.Status,
		Events:     append([]RuntimeEvent(nil), state.Events...),
	}, nil
}

type fakeStorage struct {
	key         string
	txHash      string
	lastPayload []byte
	uploadErr   error
	download    []byte
	downloadErr error
	lastKey     string
}

func (f *fakeStorage) UploadCheckpoint(_ context.Context, payload []byte) (string, string, error) {
	if f.uploadErr != nil {
		return "", "", f.uploadErr
	}
	f.lastPayload = append([]byte(nil), payload...)
	txHash := f.txHash
	if txHash == "" {
		txHash = "0xtesttx"
	}
	return f.key, txHash, nil
}

func (f *fakeStorage) DownloadCheckpoint(_ context.Context, key string) ([]byte, error) {
	f.lastKey = key
	if f.downloadErr != nil {
		return nil, f.downloadErr
	}
	return append([]byte(nil), f.download...), nil
}

type fakeAnchor struct {
	txHash string
	err    error
	last   AnchorInput
	called bool
}

func (f *fakeAnchor) AnchorCheckpoint(_ context.Context, in AnchorInput) (string, error) {
	f.called = true
	f.last = in
	if f.err != nil {
		return "", f.err
	}
	if f.txHash == "" {
		return "0xanchortx", nil
	}
	return f.txHash, nil
}

func TestServiceStartAndStep(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "workflows.json")
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(store)
	rt := &fakeRuntime{}
	st := &fakeStorage{key: "cid-1"}
	svc.SetRuntime(rt)
	svc.SetStorage(st)

	meta, err := svc.Start("wf-test")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if meta.Status != types.WorkflowStatusRunning {
		t.Fatalf("status = %s, want %s", meta.Status, types.WorkflowStatusRunning)
	}
	if meta.AgentID == "" {
		t.Fatalf("AgentID must be set")
	}

	event := types.WorkflowStepEvent{
		EventType: "tool_call",
		Actor:     "planner",
		Payload:   `{"q":"hello"}`,
		CreatedAt: time.Now().UTC(),
	}
	meta, err = svc.Step(context.Background(), "wf-test", event)
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if meta.LatestStep != 1 {
		t.Fatalf("LatestStep = %d, want 1", meta.LatestStep)
	}
	if meta.LatestRoot != "root-from-runtime" {
		t.Fatalf("LatestRoot = %s, want root-from-runtime", meta.LatestRoot)
	}
	if meta.LatestCID != "cid-1" {
		t.Fatalf("LatestCID = %s, want cid-1", meta.LatestCID)
	}
	if meta.LatestTxHash != "0xtesttx" {
		t.Fatalf("LatestTxHash = %s, want 0xtesttx", meta.LatestTxHash)
	}
	if len(meta.Events) != 1 {
		t.Fatalf("len(meta.Events) = %d, want 1", len(meta.Events))
	}
	if rt.lastWorkflowID != "wf-test" || rt.lastAgentID == "" {
		t.Fatalf("runtime replay input mismatch: workflow=%s agent=%s", rt.lastWorkflowID, rt.lastAgentID)
	}
	if len(rt.lastEvents) != 1 || rt.lastEvents[0].EventType != "tool_call" {
		t.Fatalf("runtime events mismatch: %+v", rt.lastEvents)
	}

	var cp RuntimeCheckpoint
	if err := json.Unmarshal(st.lastPayload, &cp); err != nil {
		t.Fatalf("storage payload is not checkpoint json: %v", err)
	}
	if cp.WorkflowID != "wf-test" || cp.RootHash != "root-from-runtime" {
		t.Fatalf("checkpoint payload mismatch: %+v", cp)
	}
}

func TestServiceReplay(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "workflows.json")
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(store)
	svc.SetRuntime(&fakeRuntime{})
	svc.SetStorage(&fakeStorage{key: "cid-replay"})
	if _, err := svc.Start("wf-replay"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if _, err := svc.Step(context.Background(), "wf-replay", types.WorkflowStepEvent{
		EventType: "tool_call",
		Actor:     "planner",
		Payload:   `{"tool":"search"}`,
	}); err != nil {
		t.Fatalf("Step(1) error = %v", err)
	}
	if _, err := svc.Step(context.Background(), "wf-replay", types.WorkflowStepEvent{
		EventType: "tool_result",
		Actor:     "executor",
		Payload:   `{"ok":true}`,
	}); err != nil {
		t.Fatalf("Step(2) error = %v", err)
	}

	lines, err := svc.Replay("wf-replay")
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("len(lines) = %d, want 3", len(lines))
	}
	if lines[1] == "step=1" {
		t.Fatalf("replay output should include event details, got: %s", lines[1])
	}
}

func TestServiceStepIsIdempotentForDuplicateEventID(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "workflows.json")
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(store)
	rt := &fakeRuntime{}
	st := &fakeStorage{key: "cid-dup"}
	svc.SetRuntime(rt)
	svc.SetStorage(st)

	if _, err := svc.Start("wf-dup"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	event := types.WorkflowStepEvent{
		EventID:   "evt-1",
		EventType: "tool_result",
		Actor:     "worker",
		Payload:   `{"ok":true}`,
	}
	first, err := svc.Step(context.Background(), "wf-dup", event)
	if err != nil {
		t.Fatalf("first Step() error = %v", err)
	}
	firstPayload := string(st.lastPayload)
	firstRuntimeCalls := len(rt.lastEvents)

	second, err := svc.Step(context.Background(), "wf-dup", event)
	if err != nil {
		t.Fatalf("second Step() error = %v", err)
	}

	if len(second.Events) != 1 {
		t.Fatalf("len(second.Events) = %d, want 1", len(second.Events))
	}
	if second.LatestStep != first.LatestStep {
		t.Fatalf("LatestStep changed on duplicate = %d, want %d", second.LatestStep, first.LatestStep)
	}
	if second.LatestCID != first.LatestCID {
		t.Fatalf("LatestCID changed on duplicate = %s, want %s", second.LatestCID, first.LatestCID)
	}
	if string(st.lastPayload) != firstPayload {
		t.Fatalf("storage payload changed on duplicate event")
	}
	if len(rt.lastEvents) != firstRuntimeCalls {
		t.Fatalf("runtime replay should not run again for duplicate event")
	}
}

func TestServiceIngestCreatesWorkflowWhenMissing(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "workflows.json")
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(store)
	svc.SetRuntime(&fakeRuntime{})
	svc.SetStorage(&fakeStorage{key: "cid-ingest"})

	meta, err := svc.Ingest(context.Background(), types.WorkflowStepEvent{
		WorkflowID: "wf-ingest",
		EventID:    "evt-1",
		EventType:  "tool_result",
		Actor:      "worker",
		Payload:    `{"ok":true}`,
	})
	if err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}

	if meta.WorkflowID != "wf-ingest" {
		t.Fatalf("WorkflowID = %q, want wf-ingest", meta.WorkflowID)
	}
	if meta.LatestStep != 1 {
		t.Fatalf("LatestStep = %d, want 1", meta.LatestStep)
	}
	if len(meta.Events) != 1 {
		t.Fatalf("len(meta.Events) = %d, want 1", len(meta.Events))
	}
}

func TestServiceIngestConcurrentSameWorkflowPreservesBothEvents(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "workflows.json")
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(store)
	svc.SetRuntime(&fakeRuntime{})
	svc.SetStorage(&fakeStorage{key: "cid-concurrent"})

	events := []types.WorkflowStepEvent{
		{WorkflowID: "wf-concurrent", EventID: "evt-1", EventType: "tool_call", Actor: "planner", Payload: `{"tool":"search"}`},
		{WorkflowID: "wf-concurrent", EventID: "evt-2", EventType: "tool_result", Actor: "worker", Payload: `{"ok":true}`},
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(events))
	for _, event := range events {
		event := event
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Ingest(context.Background(), event)
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("Ingest() concurrent error = %v", err)
		}
	}

	meta, err := svc.Status("wf-concurrent")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if meta.LatestStep != 2 {
		t.Fatalf("LatestStep = %d, want 2", meta.LatestStep)
	}
	if len(meta.Events) != 2 {
		t.Fatalf("len(meta.Events) = %d, want 2", len(meta.Events))
	}
}

func TestServiceStepRequiresDependencies(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "workflows.json")
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	svc := NewService(store)
	if _, err := svc.Start("wf-deps"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	_, err = svc.Step(context.Background(), "wf-deps", types.WorkflowStepEvent{
		EventType: "task_event",
	})
	if err == nil {
		t.Fatal("Step() expected dependency error, got nil")
	}
}

func TestServiceStepStorageError(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "workflows.json")
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	svc := NewService(store)
	svc.SetRuntime(&fakeRuntime{})
	svc.SetStorage(&fakeStorage{uploadErr: errors.New("upload failed")})
	if _, err := svc.Start("wf-storage-err"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	_, err = svc.Step(context.Background(), "wf-storage-err", types.WorkflowStepEvent{
		EventType: "task_event",
		Actor:     "openclaw",
		Payload:   "{}",
	})
	if err == nil {
		t.Fatal("Step() expected error, got nil")
	}
}

func TestServiceStepAnchorsCheckpoint(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "workflows.json")
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(store)
	svc.SetRuntime(&fakeRuntime{})
	svc.SetStorage(&fakeStorage{key: "cid-anchor"})
	anchor := &fakeAnchor{txHash: "0xanchorabc"}
	svc.SetAnchor(anchor)

	if _, err := svc.Start("wf-anchor"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	meta, err := svc.Step(context.Background(), "wf-anchor", types.WorkflowStepEvent{
		EventType: "tool_call",
		Actor:     "planner",
		Payload:   "{}",
	})
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}

	if !anchor.called {
		t.Fatal("anchor was not called")
	}
	if meta.LatestTxHash != "0xanchorabc" {
		t.Fatalf("LatestTxHash = %s, want 0xanchorabc", meta.LatestTxHash)
	}
	if anchor.last.StepIndex != 1 {
		t.Fatalf("anchor stepIndex = %d, want 1", anchor.last.StepIndex)
	}
	if len(anchor.last.WorkflowID) != 64 || len(anchor.last.CIDHash) != 64 {
		t.Fatalf("anchor hashes should be 32-byte hex, got workflow=%d cid=%d", len(anchor.last.WorkflowID), len(anchor.last.CIDHash))
	}
	if strings.HasPrefix(anchor.last.RootHash, "0x") {
		t.Fatalf("root hash should be normalized without 0x prefix: %s", anchor.last.RootHash)
	}
}

func TestServiceResumeFromCheckpointDownload(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "workflows.json")
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := NewService(store)
	downloadCheckpoint := RuntimeCheckpoint{
		WorkflowID: "wf-resume",
		AgentID:    "agent-wf-resume",
		LatestStep: 2,
		RootHash:   "root-from-download",
		Status:     RuntimeStatusRunning,
		Events: []RuntimeEvent{
			{EventID: "evt-0", StepIndex: 0, EventType: "tool_call", Actor: "planner", Payload: "{}"},
			{EventID: "evt-1", StepIndex: 1, EventType: "tool_result", Actor: "executor", Payload: `{"ok":true}`},
		},
	}
	raw, err := json.Marshal(downloadCheckpoint)
	if err != nil {
		t.Fatalf("marshal checkpoint: %v", err)
	}
	st := &fakeStorage{download: raw}
	svc.SetStorage(st)

	if _, err := svc.Start("wf-resume"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	meta, err := svc.Status("wf-resume")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	meta.LatestCID = "cid-from-storage"
	if err := store.Save(meta); err != nil {
		t.Fatalf("store.Save() error = %v", err)
	}

	resumed, err := svc.Resume("wf-resume")
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}

	if st.lastKey != "cid-from-storage" {
		t.Fatalf("download key = %s, want cid-from-storage", st.lastKey)
	}
	if resumed.LatestStep != 2 || resumed.LatestRoot != "root-from-download" {
		t.Fatalf("resume did not refresh checkpoint metadata: %+v", resumed)
	}
	if len(resumed.Events) != 2 || resumed.Events[1].EventType != "tool_result" {
		t.Fatalf("resume did not restore events from checkpoint: %+v", resumed.Events)
	}
}
