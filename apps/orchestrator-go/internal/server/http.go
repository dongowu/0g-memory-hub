package server

import (
	"context"
	"encoding/json"
	"errors"
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

const healthProbeTimeout = 5 * time.Second

func NewHandler(svc *workflow.Service) http.Handler {
	h := &Handler{
		svc: svc,
		mux: http.NewServeMux(),
	}

	h.mux.HandleFunc("/health", h.handleHealth)
	h.mux.HandleFunc("/v1/openclaw/ingest", h.handleOpenClawIngest)
	h.mux.HandleFunc("/v1/openclaw/ingest/batch", h.handleOpenClawBatchIngest)
	h.mux.HandleFunc("/v1/workflows/", h.handleWorkflowRoutes)

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
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
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	if len(in.Events) == 0 {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "events must not be empty")
		return
	}

	results := make([]workflowResponse, 0, len(in.Events))
	for _, eventIn := range in.Events {
		meta, err := h.ingestEvent(r, eventIn)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		results = append(results, toWorkflowResponse(meta))
	}

	writeJSON(w, http.StatusOK, map[string]any{"results": results})
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
		meta, err := h.svc.Resume(workflowID)
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

func (h *Handler) ingestEvent(r *http.Request, in openclaw.EventInput) (types.WorkflowMetadata, error) {
	return h.svc.Ingest(r.Context(), openclaw.NormalizeEvent(in))
}
