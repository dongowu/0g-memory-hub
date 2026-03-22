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
