package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/openclaw"
	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/workflow"
	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/pkg/types"
)

type Handler struct {
	svc *workflow.Service
	mux *http.ServeMux
}

type successEnvelope struct {
	Data any `json:"data"`
	Meta struct {
		Timestamp string `json:"timestamp"`
	} `json:"meta"`
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type workflowResponse struct {
	WorkflowID   string `json:"workflowId"`
	AgentID      string `json:"agentId"`
	Status       string `json:"status"`
	LatestStep   int64  `json:"latestStep"`
	LatestRoot   string `json:"latestRoot"`
	LatestCID    string `json:"latestCid"`
	LatestTxHash string `json:"latestTxHash"`
}

type openClawBatchIngestRequest struct {
	Events []openclaw.EventInput `json:"events"`
}

type openClawBatchIngestItemError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type openClawBatchIngestItemResult struct {
	WorkflowID   string                        `json:"workflowId,omitempty"`
	AgentID      string                        `json:"agentId,omitempty"`
	Status       string                        `json:"status,omitempty"`
	LatestStep   int64                         `json:"latestStep,omitempty"`
	LatestRoot   string                        `json:"latestRoot,omitempty"`
	LatestCID    string                        `json:"latestCid,omitempty"`
	LatestTxHash string                        `json:"latestTxHash,omitempty"`
	Success      bool                          `json:"success"`
	Error        *openClawBatchIngestItemError `json:"error,omitempty"`
}

const (
	healthProbeTimeout              = 5 * time.Second
	openClawIngestMaxBodyBytes      = 1 << 20 // 1 MiB
	openClawBatchIngestMaxBodyBytes = 4 << 20 // 4 MiB
)

var errPayloadTooLarge = errors.New("payload too large")

func NewHandler(svc *workflow.Service) http.Handler {
	h := &Handler{
		svc: svc,
		mux: http.NewServeMux(),
	}

	h.mux.HandleFunc("/health", h.handleHealth)
	h.mux.HandleFunc("/judge/verify", h.handleJudgeVerifyPage)
	h.mux.HandleFunc("/dashboard", h.handleDashboardPage)
	h.mux.HandleFunc("/v1/openclaw/ingest", h.handleOpenClawIngest)
	h.mux.HandleFunc("/v1/openclaw/ingest/batch", h.handleOpenClawBatchIngest)
	h.mux.HandleFunc("/v1/openclaw/runs/", h.handleOpenClawRunRoutes)
	h.mux.HandleFunc("/v1/openclaw/runs", h.handleOpenClawRunsList)
	h.mux.HandleFunc("/v1/workflows/", h.handleWorkflowRoutes)

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleJudgeVerifyPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(VerifyConsolePageHTML()))
}

func (h *Handler) handleDashboardPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(DashboardPageHTML()))
}

func (h *Handler) handleOpenClawRunsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}
	runs, err := h.svc.ListRuns()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthProbeTimeout)
	defer cancel()

	report := h.svc.Readiness(ctx)
	status := http.StatusOK
	if !report.Ready {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, report)
}

func (h *Handler) handleOpenClawIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var in openclaw.EventInput
	if err := decodeLimitedJSONBody(w, r, openClawIngestMaxBodyBytes, &in); err != nil {
		if errors.Is(err, errPayloadTooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}

	meta, err := h.ingestEvent(r, in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toWorkflowResponse(meta))
}

func (h *Handler) handleOpenClawBatchIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var in openClawBatchIngestRequest
	if err := decodeLimitedJSONBody(w, r, openClawBatchIngestMaxBodyBytes, &in); err != nil {
		if errors.Is(err, errPayloadTooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	if len(in.Events) == 0 {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "events must not be empty")
		return
	}

	results := make([]openClawBatchIngestItemResult, 0, len(in.Events))
	successCount := 0
	failureCount := 0
	for _, eventIn := range in.Events {
		event := openclaw.NormalizeEvent(eventIn)
		result := openClawBatchIngestItemResult{
			WorkflowID: event.WorkflowID,
		}

		meta, err := h.svc.Ingest(r.Context(), event)
		if err != nil {
			result.Success = false
			result.Error = &openClawBatchIngestItemError{
				Code:    "INTERNAL_ERROR",
				Message: err.Error(),
			}
			failureCount++
			results = append(results, result)
			continue
		}

		workflowResult := toWorkflowResponse(meta)
		result.WorkflowID = workflowResult.WorkflowID
		result.AgentID = workflowResult.AgentID
		result.Status = workflowResult.Status
		result.LatestStep = workflowResult.LatestStep
		result.LatestRoot = workflowResult.LatestRoot
		result.LatestCID = workflowResult.LatestCID
		result.LatestTxHash = workflowResult.LatestTxHash
		result.Success = true
		successCount++
		results = append(results, result)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"results":      results,
		"successCount": successCount,
		"failureCount": failureCount,
	})
}

func (h *Handler) handleOpenClawRunRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/openclaw/runs/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "openclaw run route not found")
		return
	}

	parts := strings.Split(path, "/")
	runID := parts[0]

	if len(parts) == 2 && parts[1] == "context" && r.Method == http.MethodGet {
		contextView, err := h.svc.RunContext(runID)
		if err != nil {
			handleWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, contextView)
		return
	}

	if len(parts) == 3 && parts[1] == "checkpoint" && parts[2] == "latest" && r.Method == http.MethodGet {
		checkpointView, err := h.svc.LatestCheckpoint(runID)
		if err != nil {
			handleWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, checkpointView)
		return
	}

	if len(parts) == 3 && parts[1] == "checkpoint" && parts[2] == "kv" && r.Method == http.MethodGet {
		summary, err := h.svc.KVLatestCheckpoint(r.Context(), runID)
		if err != nil {
			handleWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, summary)
		return
	}

	if len(parts) == 2 && parts[1] == "hydrate" && r.Method == http.MethodPost {
		contextView, err := h.svc.Hydrate(r.Context(), runID)
		if err != nil {
			handleWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, contextView)
		return
	}

	if len(parts) == 2 && parts[1] == "trace" && r.Method == http.MethodGet {
		traceView, err := h.svc.RunTrace(runID)
		if err != nil {
			handleWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, traceView)
		return
	}

	if len(parts) == 2 && parts[1] == "verify" && r.Method == http.MethodGet {
		verifyView, err := h.svc.VerifyRun(r.Context(), runID)
		if err != nil {
			handleWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, verifyView)
		return
	}

	// Time-travel: GET /v1/openclaw/runs/:id/snapshot/:stepIndex
	if len(parts) == 3 && parts[1] == "snapshot" && r.Method == http.MethodGet {
		stepIndexStr := parts[2]
		var stepIndex int64
		if _, err := fmt.Sscanf(stepIndexStr, "%d", &stepIndex); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_STEP_INDEX", "step index must be an integer")
			return
		}
		snapshotView, err := h.svc.Snapshot(runID, stepIndex)
		if err != nil {
			handleWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, snapshotView)
		return
	}

	writeError(w, http.StatusNotFound, "NOT_FOUND", "openclaw run route not found")
}

func (h *Handler) handleWorkflowRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/workflows/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "workflow route not found")
		return
	}

	parts := strings.Split(path, "/")
	workflowID := parts[0]

	if len(parts) == 1 && r.Method == http.MethodGet {
		meta, err := h.svc.Status(workflowID)
		if err != nil {
			handleWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toWorkflowResponse(meta))
		return
	}

	if len(parts) == 2 && parts[1] == "resume" && r.Method == http.MethodPost {
		meta, err := h.svc.ResumeWithContext(r.Context(), workflowID)
		if err != nil {
			handleWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toWorkflowResponse(meta))
		return
	}

	if len(parts) == 2 && parts[1] == "replay" && r.Method == http.MethodGet {
		lines, err := h.svc.Replay(workflowID)
		if err != nil {
			handleWorkflowError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"lines": lines})
		return
	}

	writeError(w, http.StatusNotFound, "NOT_FOUND", "workflow route not found")
}

func toWorkflowResponse(meta types.WorkflowMetadata) workflowResponse {
	return workflowResponse{
		WorkflowID:   meta.WorkflowID,
		AgentID:      meta.AgentID,
		Status:       string(meta.Status),
		LatestStep:   meta.LatestStep,
		LatestRoot:   meta.LatestRoot,
		LatestCID:    meta.LatestCID,
		LatestTxHash: meta.LatestTxHash,
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	var out successEnvelope
	out.Data = data
	out.Meta.Timestamp = time.Now().UTC().Format(time.RFC3339)
	_ = json.NewEncoder(w).Encode(out)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	var out errorEnvelope
	out.Error.Code = code
	out.Error.Message = message
	_ = json.NewEncoder(w).Encode(out)
}

func handleWorkflowError(w http.ResponseWriter, err error) {
	if errors.Is(err, workflow.ErrWorkflowNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
}

func decodeLimitedJSONBody(w http.ResponseWriter, r *http.Request, maxBytes int64, out any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return errPayloadTooLarge
		}
		return err
	}
	return nil
}

func (h *Handler) ingestEvent(r *http.Request, in openclaw.EventInput) (types.WorkflowMetadata, error) {
	return h.svc.Ingest(r.Context(), openclaw.NormalizeEvent(in))
}
