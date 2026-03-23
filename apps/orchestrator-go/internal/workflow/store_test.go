package workflow

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/pkg/types"
)

func TestFileStoreSaveAndGet(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "workflows.json")
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	want := types.WorkflowMetadata{
		WorkflowID: "wf_1",
		Status:     types.WorkflowStatusRunning,
		LatestStep: 2,
		LatestRoot: "root-2",
		LatestCID:  "cid-2",
		UpdatedAt:  time.Now().UTC(),
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Get("wf_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.WorkflowID != want.WorkflowID {
		t.Fatalf("WorkflowID = %s, want %s", got.WorkflowID, want.WorkflowID)
	}
	if got.LatestStep != want.LatestStep {
		t.Fatalf("LatestStep = %d, want %d", got.LatestStep, want.LatestStep)
	}
	if got.LatestRoot != want.LatestRoot {
		t.Fatalf("LatestRoot = %s, want %s", got.LatestRoot, want.LatestRoot)
	}
	if got.LatestCID != want.LatestCID {
		t.Fatalf("LatestCID = %s, want %s", got.LatestCID, want.LatestCID)
	}
}

func TestFileStoreFindByRunID(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "workflows.json")
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	older := types.WorkflowMetadata{
		WorkflowID: "wf_old",
		Status:     types.WorkflowStatusRunning,
		Events: []types.WorkflowStepEvent{
			{WorkflowID: "wf_old", RunID: "run-shared", EventID: "evt-old"},
		},
		UpdatedAt: time.Now().UTC().Add(-time.Minute),
	}
	newer := types.WorkflowMetadata{
		WorkflowID: "wf_new",
		Status:     types.WorkflowStatusRunning,
		Events: []types.WorkflowStepEvent{
			{WorkflowID: "wf_new", RunID: "run-shared", EventID: "evt-new"},
		},
		UpdatedAt: time.Now().UTC(),
	}

	if err := store.Save(older); err != nil {
		t.Fatalf("Save(older) error = %v", err)
	}
	if err := store.Save(newer); err != nil {
		t.Fatalf("Save(newer) error = %v", err)
	}

	got, err := store.FindByRunID("run-shared")
	if err != nil {
		t.Fatalf("FindByRunID() error = %v", err)
	}
	if got.WorkflowID != "wf_new" {
		t.Fatalf("WorkflowID = %q, want wf_new", got.WorkflowID)
	}
}

func TestFileStoreFindByRunIDReturnsErrForEmptyRunID(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "workflows.json")
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	_, err = store.FindByRunID("")
	if err == nil {
		t.Fatal("FindByRunID(\"\") error = nil, want ErrWorkflowNotFound")
	}
	if err != ErrWorkflowNotFound {
		t.Fatalf("FindByRunID(\"\") error = %v, want %v", err, ErrWorkflowNotFound)
	}
}

func TestFileStoreFindByRunIDUsesWorkflowIDTieBreakForEqualUpdatedAt(t *testing.T) {
	t.Parallel()

	storePath := filepath.Join(t.TempDir(), "workflows.json")
	store, err := NewFileStore(storePath)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	updatedAt := time.Now().UTC()
	olderWorkflowID := types.WorkflowMetadata{
		WorkflowID: "wf_a",
		Status:     types.WorkflowStatusRunning,
		Events: []types.WorkflowStepEvent{
			{WorkflowID: "wf_a", RunID: "run-shared", EventID: "evt-a"},
		},
		UpdatedAt: updatedAt,
	}
	newerWorkflowID := types.WorkflowMetadata{
		WorkflowID: "wf_b",
		Status:     types.WorkflowStatusRunning,
		Events: []types.WorkflowStepEvent{
			{WorkflowID: "wf_b", RunID: "run-shared", EventID: "evt-b"},
		},
		UpdatedAt: updatedAt,
	}

	if err := store.Save(olderWorkflowID); err != nil {
		t.Fatalf("Save(olderWorkflowID) error = %v", err)
	}
	if err := store.Save(newerWorkflowID); err != nil {
		t.Fatalf("Save(newerWorkflowID) error = %v", err)
	}

	got, err := store.FindByRunID("run-shared")
	if err != nil {
		t.Fatalf("FindByRunID() error = %v", err)
	}
	if got.WorkflowID != "wf_b" {
		t.Fatalf("WorkflowID = %q, want wf_b", got.WorkflowID)
	}
}
