package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/workflow"
	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/pkg/types"
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

type failableRuntime struct {
	failOnEventType string
}

func (f *failableRuntime) ReplayWorkflow(_ context.Context, workflowID, agentID string, events []workflow.RuntimeEvent) (*workflow.RuntimeState, error) {
	if len(events) > 0 && events[len(events)-1].EventType == f.failOnEventType {
		return nil, fmt.Errorf("runtime rejected event type %s", f.failOnEventType)
	}
	return (&fakeRuntime{}).ReplayWorkflow(context.Background(), workflowID, agentID, events)
}

func (f *failableRuntime) BuildCheckpoint(_ context.Context, state workflow.RuntimeState) (*workflow.RuntimeCheckpoint, error) {
	return (&fakeRuntime{}).BuildCheckpoint(context.Background(), state)
}

type fakeStorage struct {
	lastDownloadCtx context.Context
}

func (f *fakeStorage) UploadCheckpoint(_ context.Context, _ []byte) (string, string, error) {
	return "cid-1", "0xtesttx", nil
}

func (f *fakeStorage) DownloadCheckpoint(ctx context.Context, _ string) ([]byte, error) {
	f.lastDownloadCtx = ctx
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

type readinessFailingStorage struct {
	fakeStorage
	err error
}

func (f *readinessFailingStorage) CheckReadiness(_ context.Context) error {
	return f.err
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

func newTestServiceWithRuntime(t *testing.T, runtime workflow.RuntimeAPI) *workflow.Service {
	t.Helper()

	store, err := workflow.NewFileStore(filepath.Join(t.TempDir(), "workflows.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	svc := workflow.NewService(store)
	svc.SetRuntime(runtime)
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

func TestHandlerHealthReturns503WhenConfiguredDependencyIsUnreachable(t *testing.T) {
	t.Parallel()

	store, err := workflow.NewFileStore(filepath.Join(t.TempDir(), "workflows.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	svc := workflow.NewService(store)
	svc.SetRuntime(&fakeRuntime{})
	svc.SetStorage(&readinessFailingStorage{err: errors.New("dial tcp 127.0.0.1:1234: i/o timeout")})

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
	if out.Data.Components["storage"].Ready {
		t.Fatal("expected storage readiness to be false")
	}
	if !strings.Contains(out.Data.Components["storage"].Message, "i/o timeout") {
		t.Fatalf("storage message = %q, want timeout detail", out.Data.Components["storage"].Message)
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
				Success    bool   `json:"success"`
				LatestStep int64  `json:"latestStep"`
			} `json:"results"`
			SuccessCount int `json:"successCount"`
			FailureCount int `json:"failureCount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(out.Data.Results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(out.Data.Results))
	}
	if out.Data.SuccessCount != 2 || out.Data.FailureCount != 0 {
		t.Fatalf("unexpected summary success=%d failure=%d", out.Data.SuccessCount, out.Data.FailureCount)
	}
	if out.Data.Results[0].WorkflowID != "run-batch" || out.Data.Results[1].WorkflowID != "run-batch" {
		t.Fatalf("unexpected workflow ids: %+v", out.Data.Results)
	}
	if !out.Data.Results[0].Success || !out.Data.Results[1].Success {
		t.Fatalf("expected success=true for all results: %+v", out.Data.Results)
	}
	if out.Data.Results[0].LatestStep != 1 || out.Data.Results[1].LatestStep != 2 {
		t.Fatalf("unexpected latest steps: %+v", out.Data.Results)
	}
}

func TestHandlerOpenClawBatchIngestPreservesExtendedMetadata(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestService(t))
	reqBody := bytes.NewBufferString(`{
		"events": [
			{
				"workflowId":"wf-batch-openclaw",
				"runId":"run-batch-openclaw",
				"sessionId":"session-batch-openclaw",
				"traceId":"trace-batch-openclaw",
				"eventId":"evt-batch-1",
				"eventType":"tool_call",
				"actor":"coordinator",
				"role":"planner",
				"toolCallId":"tool-batch-1",
				"skillName":"search_skill",
				"taskId":"task-batch-1",
				"payload":{"tool":"search"}
			}
		]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/openclaw/ingest/batch", reqBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("batch ingest status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}

	contextReq := httptest.NewRequest(http.MethodGet, "/v1/openclaw/runs/run-batch-openclaw/context", nil)
	contextRec := httptest.NewRecorder()
	handler.ServeHTTP(contextRec, contextReq)
	if contextRec.Code != http.StatusOK {
		t.Fatalf("context status = %d, want 200 body=%s", contextRec.Code, contextRec.Body.String())
	}

	var contextOut struct {
		Data struct {
			WorkflowID string `json:"workflowId"`
			RunID     string `json:"runId"`
			SessionID string `json:"sessionId"`
			TraceID   string `json:"traceId"`
			Events    []struct {
				Role       string `json:"role"`
				ToolCallID string `json:"toolCallId"`
				SkillName  string `json:"skillName"`
				TaskID     string `json:"taskId"`
			} `json:"events"`
		} `json:"data"`
	}
	if err := json.Unmarshal(contextRec.Body.Bytes(), &contextOut); err != nil {
		t.Fatalf("decode context response: %v", err)
	}
	if contextOut.Data.RunID != "run-batch-openclaw" {
		t.Fatalf("runId = %q, want run-batch-openclaw", contextOut.Data.RunID)
	}
	if contextOut.Data.WorkflowID != "wf-batch-openclaw" {
		t.Fatalf("workflowId = %q, want wf-batch-openclaw", contextOut.Data.WorkflowID)
	}
	if contextOut.Data.SessionID != "session-batch-openclaw" || contextOut.Data.TraceID != "trace-batch-openclaw" {
		t.Fatalf("unexpected session/trace values: session=%q trace=%q", contextOut.Data.SessionID, contextOut.Data.TraceID)
	}
	if len(contextOut.Data.Events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(contextOut.Data.Events))
	}
	if contextOut.Data.Events[0].Role != "planner" {
		t.Fatalf("role = %q, want planner", contextOut.Data.Events[0].Role)
	}
	if contextOut.Data.Events[0].ToolCallID != "tool-batch-1" {
		t.Fatalf("toolCallId = %q, want tool-batch-1", contextOut.Data.Events[0].ToolCallID)
	}
	if contextOut.Data.Events[0].SkillName != "search_skill" {
		t.Fatalf("skillName = %q, want search_skill", contextOut.Data.Events[0].SkillName)
	}
	if contextOut.Data.Events[0].TaskID != "task-batch-1" {
		t.Fatalf("taskId = %q, want task-batch-1", contextOut.Data.Events[0].TaskID)
	}
}

func TestHandlerOpenClawIngestRejectsOversizedBody(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestService(t))
	oversizedPayload := strings.Repeat("a", int(openClawIngestMaxBodyBytes))
	body := `{"runId":"run-oversized","eventType":"tool_result","payload":"` + oversizedPayload + `"}`

	req := httptest.NewRequest(http.MethodPost, "/v1/openclaw/ingest", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413 body=%s", rec.Code, rec.Body.String())
	}

	var out struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Error.Code != "PAYLOAD_TOO_LARGE" {
		t.Fatalf("error.code = %q, want PAYLOAD_TOO_LARGE", out.Error.Code)
	}
}

func TestHandlerOpenClawBatchIngestRejectsOversizedBody(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestService(t))
	oversizedPayload := strings.Repeat("a", int(openClawBatchIngestMaxBodyBytes))
	body := `{"events":[{"runId":"run-batch-oversized","eventType":"tool_result","payload":"` + oversizedPayload + `"}]}`

	req := httptest.NewRequest(http.MethodPost, "/v1/openclaw/ingest/batch", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413 body=%s", rec.Code, rec.Body.String())
	}

	var out struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Error.Code != "PAYLOAD_TOO_LARGE" {
		t.Fatalf("error.code = %q, want PAYLOAD_TOO_LARGE", out.Error.Code)
	}
}

func TestHandlerOpenClawIngestReturnsBadRequestForInvalidJSONWithinLimit(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestService(t))
	req := httptest.NewRequest(http.MethodPost, "/v1/openclaw/ingest", bytes.NewBufferString(`{"runId":"broken"`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlerOpenClawBatchIngestReturnsPerItemResultsOnPartialFailure(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServiceWithRuntime(t, &failableRuntime{failOnEventType: "fail_event"}))
	reqBody := bytes.NewBufferString(`{
		"events": [
			{"runId":"run-mixed","eventId":"evt-1","eventType":"tool_result","actor":"worker","payload":{"ok":true}},
			{"runId":"run-mixed","eventId":"evt-2","eventType":"fail_event","actor":"worker","payload":{"ok":false}}
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
				Success    bool   `json:"success"`
				LatestStep int64  `json:"latestStep"`
				Error      *struct {
					Code string `json:"code"`
				} `json:"error"`
			} `json:"results"`
			SuccessCount int `json:"successCount"`
			FailureCount int `json:"failureCount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(out.Data.Results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(out.Data.Results))
	}
	if out.Data.SuccessCount != 1 || out.Data.FailureCount != 1 {
		t.Fatalf("unexpected summary success=%d failure=%d", out.Data.SuccessCount, out.Data.FailureCount)
	}
	if !out.Data.Results[0].Success || out.Data.Results[0].LatestStep != 1 {
		t.Fatalf("first result should be success with latestStep=1: %+v", out.Data.Results[0])
	}
	if out.Data.Results[1].Success {
		t.Fatalf("second result should be failure: %+v", out.Data.Results[1])
	}
	if out.Data.Results[1].Error == nil || out.Data.Results[1].Error.Code != "INTERNAL_ERROR" {
		t.Fatalf("second result error = %+v, want INTERNAL_ERROR", out.Data.Results[1].Error)
	}
	if out.Data.Results[1].WorkflowID != "run-mixed" {
		t.Fatalf("second result workflowId = %q, want run-mixed", out.Data.Results[1].WorkflowID)
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

func TestHandlerWorkflowResumePropagatesRequestContext(t *testing.T) {
	t.Parallel()

	store, err := workflow.NewFileStore(filepath.Join(t.TempDir(), "workflows.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	storage := &fakeStorage{}
	svc := workflow.NewService(store)
	svc.SetRuntime(&fakeRuntime{})
	svc.SetStorage(storage)

	if _, err := svc.Ingest(context.Background(), types.WorkflowStepEvent{
		WorkflowID: "run-123",
		EventID:    "evt-1",
		EventType:  "tool_result",
		Actor:      "worker",
		Payload:    `{"ok":true}`,
	}); err != nil {
		t.Fatalf("Ingest() error = %v", err)
	}

	handler := NewHandler(svc)

	type ctxKey string
	traceKey := ctxKey("trace-id")
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows/run-123/resume", nil)
	req = req.WithContext(context.WithValue(req.Context(), traceKey, "trace-123"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("resume code = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}
	if storage.lastDownloadCtx == nil {
		t.Fatal("download context was not captured")
	}
	if got := storage.lastDownloadCtx.Value(traceKey); got != "trace-123" {
		t.Fatalf("download context value = %v, want trace-123", got)
	}
}

func TestHandlerOpenClawRunRoutesContextCheckpointHydrateAndTrace(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestService(t))
	body := bytes.NewBufferString(`{
		"workflowId":"wf-openclaw",
		"runId":"run-openclaw",
		"sessionId":"session-openclaw",
		"traceId":"trace-openclaw",
		"eventId":"evt-oc-1",
		"eventType":"tool_call",
		"actor":"coordinator",
		"role":"planner",
		"toolCallId":"tool-oc-1",
		"skillName":"search_skill",
		"taskId":"task-oc-1",
		"payload":{"tool":"search"}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/openclaw/ingest", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ingest status = %d, want 200 body=%s", rec.Code, rec.Body.String())
	}

	contextReq := httptest.NewRequest(http.MethodGet, "/v1/openclaw/runs/run-openclaw/context", nil)
	contextRec := httptest.NewRecorder()
	handler.ServeHTTP(contextRec, contextReq)
	if contextRec.Code != http.StatusOK {
		t.Fatalf("context status = %d, want 200 body=%s", contextRec.Code, contextRec.Body.String())
	}
	var contextOut struct {
		Data struct {
			WorkflowID string `json:"workflowId"`
			RunID      string `json:"runId"`
			SessionID  string `json:"sessionId"`
			TraceID    string `json:"traceId"`
			Events     []struct {
				EventID    string `json:"eventId"`
				Role       string `json:"role"`
				ToolCallID string `json:"toolCallId"`
				SkillName  string `json:"skillName"`
			} `json:"events"`
		} `json:"data"`
	}
	if err := json.Unmarshal(contextRec.Body.Bytes(), &contextOut); err != nil {
		t.Fatalf("decode context response: %v", err)
	}
	if contextOut.Data.WorkflowID != "wf-openclaw" {
		t.Fatalf("workflowId = %q, want wf-openclaw", contextOut.Data.WorkflowID)
	}
	if contextOut.Data.RunID != "run-openclaw" {
		t.Fatalf("runId = %q, want run-openclaw", contextOut.Data.RunID)
	}
	if contextOut.Data.SessionID != "session-openclaw" || contextOut.Data.TraceID != "trace-openclaw" {
		t.Fatalf("unexpected session/trace values: session=%q trace=%q", contextOut.Data.SessionID, contextOut.Data.TraceID)
	}
	if len(contextOut.Data.Events) != 1 || contextOut.Data.Events[0].ToolCallID != "tool-oc-1" {
		t.Fatalf("unexpected context events: %+v", contextOut.Data.Events)
	}

	checkpointReq := httptest.NewRequest(http.MethodGet, "/v1/openclaw/runs/run-openclaw/checkpoint/latest", nil)
	checkpointRec := httptest.NewRecorder()
	handler.ServeHTTP(checkpointRec, checkpointReq)
	if checkpointRec.Code != http.StatusOK {
		t.Fatalf("checkpoint status = %d, want 200 body=%s", checkpointRec.Code, checkpointRec.Body.String())
	}
	var checkpointOut struct {
		Data struct {
			WorkflowID string `json:"workflowId"`
			LatestCID  string `json:"latestCid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(checkpointRec.Body.Bytes(), &checkpointOut); err != nil {
		t.Fatalf("decode checkpoint response: %v", err)
	}
	if checkpointOut.Data.WorkflowID != "wf-openclaw" || checkpointOut.Data.LatestCID == "" {
		t.Fatalf("unexpected checkpoint response: %+v", checkpointOut.Data)
	}

	traceReq := httptest.NewRequest(http.MethodGet, "/v1/openclaw/runs/run-openclaw/trace", nil)
	traceRec := httptest.NewRecorder()
	handler.ServeHTTP(traceRec, traceReq)
	if traceRec.Code != http.StatusOK {
		t.Fatalf("trace status = %d, want 200 body=%s", traceRec.Code, traceRec.Body.String())
	}
	var traceOut struct {
		Data struct {
			WorkflowID string `json:"workflowId"`
			Steps      []struct {
				EventID string `json:"eventId"`
				Role    string `json:"role"`
			} `json:"steps"`
		} `json:"data"`
	}
	if err := json.Unmarshal(traceRec.Body.Bytes(), &traceOut); err != nil {
		t.Fatalf("decode trace response: %v", err)
	}
	if traceOut.Data.WorkflowID != "wf-openclaw" || len(traceOut.Data.Steps) != 1 {
		t.Fatalf("unexpected trace response: %+v", traceOut.Data)
	}
	if traceOut.Data.Steps[0].Role != "planner" {
		t.Fatalf("trace step role = %q, want planner", traceOut.Data.Steps[0].Role)
	}

	hydrateReq := httptest.NewRequest(http.MethodPost, "/v1/openclaw/runs/run-openclaw/hydrate", nil)
	hydrateRec := httptest.NewRecorder()
	handler.ServeHTTP(hydrateRec, hydrateReq)
	if hydrateRec.Code != http.StatusOK {
		t.Fatalf("hydrate status = %d, want 200 body=%s", hydrateRec.Code, hydrateRec.Body.String())
	}
}
