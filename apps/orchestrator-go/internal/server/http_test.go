package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/workflow"
)

type fakeRuntime struct{}

func (f *fakeRuntime) ReplayWorkflow(_ context.Context, workflowID, agentID string, events []workflow.RuntimeEvent) (*workflow.RuntimeState, error) {
	return &workflow.RuntimeState{
		WorkflowID: workflowID,
		AgentID:    agentID,
		Status:     workflow.RuntimeStatusRunning,
		LatestStep: uint64(len(events)),
		LatestRoot: "root-from-runtime",
		Events:     append([]workflow.RuntimeEvent(nil), events...),
	}, nil
}

func (f *fakeRuntime) BuildCheckpoint(_ context.Context, state workflow.RuntimeState) (*workflow.RuntimeCheckpoint, error) {
	return &workflow.RuntimeCheckpoint{
		WorkflowID: state.WorkflowID,
		AgentID:    state.AgentID,
		LatestStep: state.LatestStep,
		RootHash:   state.LatestRoot,
		Status:     state.Status,
		Events:     append([]workflow.RuntimeEvent(nil), state.Events...),
	}, nil
}

type fakeStorage struct{}

func (f *fakeStorage) UploadCheckpoint(_ context.Context, _ []byte) (string, string, error) {
	return "cid-1", "0xtesttx", nil
}

func (f *fakeStorage) DownloadCheckpoint(_ context.Context, _ string) ([]byte, error) {
	checkpoint := workflow.RuntimeCheckpoint{
		WorkflowID: "run-123",
		AgentID:    "agent-run-123",
		LatestStep: 1,
		RootHash:   "root-from-runtime",
		Status:     workflow.RuntimeStatusRunning,
		Events: []workflow.RuntimeEvent{
			{EventID: "evt-1", StepIndex: 0, EventType: "tool_result", Actor: "worker", Payload: `{"ok":true}`},
		},
	}
	return json.Marshal(checkpoint)
}

func newTestService(t *testing.T) *workflow.Service {
	t.Helper()

	store, err := workflow.NewFileStore(filepath.Join(t.TempDir(), "workflows.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := workflow.NewService(store)
	svc.SetRuntime(&fakeRuntime{})
	svc.SetStorage(&fakeStorage{})
	return svc
}

func TestHandlerHealth(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	NewHandler(newTestService(t)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var out struct {
		Data struct {
			Ready      bool `json:"ready"`
			Components map[string]struct {
				Ready bool `json:"ready"`
			} `json:"components"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !out.Data.Ready {
		t.Fatal("expected ready=true")
	}
	if len(out.Data.Components) == 0 {
		t.Fatal("expected component readiness details")
	}
}

func TestHandlerHealthReturns503WhenDependenciesMissing(t *testing.T) {
	t.Parallel()

	store, err := workflow.NewFileStore(filepath.Join(t.TempDir(), "workflows.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	svc := workflow.NewService(store)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	NewHandler(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 body=%s", rec.Code, rec.Body.String())
	}

	var out struct {
		Data struct {
			Ready      bool `json:"ready"`
			Components map[string]struct {
				Ready   bool   `json:"ready"`
				Message string `json:"message"`
			} `json:"components"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Data.Ready {
		t.Fatal("expected ready=false")
	}
	if out.Data.Components["runtime"].Ready {
		t.Fatal("expected runtime readiness to be false")
	}
}

func TestHandlerOpenClawIngestCreatesAndUpdatesWorkflow(t *testing.T) {
	t.Parallel()

	body := bytes.NewBufferString(`{"runId":"run-123","eventId":"evt-1","eventType":"tool_result","actor":"worker","payload":{"ok":true}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/openclaw/ingest", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	NewHandler(newTestService(t)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}

	var out struct {
		Data struct {
			WorkflowID string `json:"workflowId"`
			LatestCID  string `json:"latestCid"`
			LatestStep int64  `json:"latestStep"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Data.WorkflowID != "run-123" {
		t.Fatalf("WorkflowID = %q, want run-123", out.Data.WorkflowID)
	}
	if out.Data.LatestCID != "cid-1" {
		t.Fatalf("LatestCID = %q, want cid-1", out.Data.LatestCID)
	}
	if out.Data.LatestStep != 1 {
		t.Fatalf("LatestStep = %d, want 1", out.Data.LatestStep)
	}
}

func TestHandlerOpenClawIngestIsIdempotentForDuplicateEventID(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestService(t))
	body := `{"runId":"run-123","eventId":"evt-1","eventType":"tool_result","actor":"worker","payload":{"ok":true}}`

	firstReq := httptest.NewRequest(http.MethodPost, "/v1/openclaw/ingest", bytes.NewBufferString(body))
	firstReq.Header.Set("Content-Type", "application/json")
	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first ingest status = %d, want 200 body=%s", firstRec.Code, firstRec.Body.String())
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/v1/openclaw/ingest", bytes.NewBufferString(body))
	secondReq.Header.Set("Content-Type", "application/json")
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusOK {
		t.Fatalf("second ingest status = %d, want 200 body=%s", secondRec.Code, secondRec.Body.String())
	}

	var out struct {
		Data struct {
			LatestStep int64 `json:"latestStep"`
		} `json:"data"`
	}
	if err := json.Unmarshal(secondRec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Data.LatestStep != 1 {
		t.Fatalf("LatestStep after duplicate ingest = %d, want 1", out.Data.LatestStep)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/v1/workflows/run-123", nil)
	statusRec := httptest.NewRecorder()
	handler.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200 body=%s", statusRec.Code, statusRec.Body.String())
	}

	var statusOut struct {
		Data struct {
			LatestStep int64 `json:"latestStep"`
		} `json:"data"`
	}
	if err := json.Unmarshal(statusRec.Body.Bytes(), &statusOut); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if statusOut.Data.LatestStep != 1 {
		t.Fatalf("persisted LatestStep after duplicate ingest = %d, want 1", statusOut.Data.LatestStep)
	}
}

func TestHandlerOpenClawBatchIngestProcessesMultipleEvents(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestService(t))
	reqBody := bytes.NewBufferString(`{
		"events": [
			{"runId":"run-batch","eventId":"evt-1","eventType":"tool_call","actor":"planner","payload":{"tool":"search"}},
			{"runId":"run-batch","eventId":"evt-2","eventType":"tool_result","actor":"worker","payload":{"ok":true}}
		]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/openclaw/ingest/batch", reqBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}

	var out struct {
		Data struct {
			Results []struct {
				WorkflowID string `json:"workflowId"`
				LatestStep int64  `json:"latestStep"`
			} `json:"results"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(out.Data.Results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(out.Data.Results))
	}
	if out.Data.Results[0].WorkflowID != "run-batch" || out.Data.Results[1].WorkflowID != "run-batch" {
		t.Fatalf("unexpected workflow ids: %+v", out.Data.Results)
	}
	if out.Data.Results[0].LatestStep != 1 || out.Data.Results[1].LatestStep != 2 {
		t.Fatalf("unexpected latest steps: %+v", out.Data.Results)
	}
}

func TestHandlerWorkflowStatusReplayAndResume(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestService(t))

	ingestReq := httptest.NewRequest(http.MethodPost, "/v1/openclaw/ingest", bytes.NewBufferString(`{"runId":"run-123","eventType":"tool_result","payload":{"ok":true}}`))
	ingestReq.Header.Set("Content-Type", "application/json")
	ingestRec := httptest.NewRecorder()
	handler.ServeHTTP(ingestRec, ingestReq)
	if ingestRec.Code != http.StatusOK {
		t.Fatalf("ingest status = %d, want 200 body=%s", ingestRec.Code, ingestRec.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/v1/workflows/run-123", nil)
	statusRec := httptest.NewRecorder()
	handler.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", statusRec.Code)
	}

	replayReq := httptest.NewRequest(http.MethodGet, "/v1/workflows/run-123/replay", nil)
	replayRec := httptest.NewRecorder()
	handler.ServeHTTP(replayRec, replayReq)
	if replayRec.Code != http.StatusOK {
		t.Fatalf("replay code = %d, want 200", replayRec.Code)
	}

	var replayOut struct {
		Data struct {
			Lines []string `json:"lines"`
		} `json:"data"`
	}
	if err := json.Unmarshal(replayRec.Body.Bytes(), &replayOut); err != nil {
		t.Fatalf("decode replay response: %v", err)
	}
	if len(replayOut.Data.Lines) == 0 {
		t.Fatal("replay lines empty")
	}

	resumeReq := httptest.NewRequest(http.MethodPost, "/v1/workflows/run-123/resume", nil)
	resumeRec := httptest.NewRecorder()
	handler.ServeHTTP(resumeRec, resumeReq)
	if resumeRec.Code != http.StatusOK {
		t.Fatalf("resume code = %d, want 200 body=%s", resumeRec.Code, resumeRec.Body.String())
	}
}
