package workflow

import (
	"fmt"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/pkg/types"
)

type AnchorLatestCheckpoint struct {
	StepIndex uint64
	RootHash  string
	CIDHash   string
	Timestamp uint64
	Submitter string
}

type VerifyRunResult struct {
	RunID                string                      `json:"runId"`
	WorkflowID           string                      `json:"workflowId"`
	LocalMetadata        VerifyLocalMetadataView     `json:"localMetadata"`
	RecomputedCheckpoint VerifyCheckpointView        `json:"recomputedCheckpoint"`
	StorageCheckpoint    VerifyCheckpointView        `json:"storageCheckpoint"`
	OnChainCheckpoint    VerifyOnChainCheckpointView `json:"onChainCheckpoint"`
	Checks               []VerifyCheck               `json:"checks"`
	Verified             bool                        `json:"verified"`
}

type VerifyLocalMetadataView struct {
	WorkflowID   string `json:"workflowId"`
	RunID        string `json:"runId"`
	SessionID    string `json:"sessionId,omitempty"`
	TraceID      string `json:"traceId,omitempty"`
	AgentID      string `json:"agentId"`
	Status       string `json:"status"`
	LatestStep   int64  `json:"latestStep"`
	LatestRoot   string `json:"latestRoot"`
	LatestCID    string `json:"latestCid"`
	LatestTxHash string `json:"latestTxHash"`
}

type VerifyCheckpointView struct {
	WorkflowID string `json:"workflowId,omitempty"`
	AgentID    string `json:"agentId,omitempty"`
	StepIndex  uint64 `json:"stepIndex,omitempty"`
	RootHash   string `json:"rootHash,omitempty"`
	Status     string `json:"status,omitempty"`
	CID        string `json:"cid,omitempty"`
	Error      string `json:"error,omitempty"`
}

type VerifyOnChainCheckpointView struct {
	WorkflowIDHash string `json:"workflowIdHash,omitempty"`
	StepIndex      uint64 `json:"stepIndex,omitempty"`
	RootHash       string `json:"rootHash,omitempty"`
	CIDHash        string `json:"cidHash,omitempty"`
	Timestamp      uint64 `json:"timestamp,omitempty"`
	Submitter      string `json:"submitter,omitempty"`
	Error          string `json:"error,omitempty"`
}

type VerifyCheck struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
	Message  string `json:"message,omitempty"`
}

func buildVerifyRunResult(runID string, meta types.WorkflowMetadata) VerifyRunResult {
	identity := runIdentityFromEvents(meta.Events, meta.WorkflowID)
	resolvedRunID := runID
	if resolvedRunID == "" {
		resolvedRunID = identity.RunID
	}
	if resolvedRunID == "" {
		resolvedRunID = meta.WorkflowID
	}

	return VerifyRunResult{
		RunID:      resolvedRunID,
		WorkflowID: meta.WorkflowID,
		LocalMetadata: VerifyLocalMetadataView{
			WorkflowID:   meta.WorkflowID,
			RunID:        identity.RunID,
			SessionID:    identity.SessionID,
			TraceID:      identity.TraceID,
			AgentID:      meta.AgentID,
			Status:       string(meta.Status),
			LatestStep:   meta.LatestStep,
			LatestRoot:   meta.LatestRoot,
			LatestCID:    meta.LatestCID,
			LatestTxHash: meta.LatestTxHash,
		},
		Checks: make([]VerifyCheck, 0, 8),
	}
}

func runtimeCheckpointView(checkpoint RuntimeCheckpoint) VerifyCheckpointView {
	return VerifyCheckpointView{
		WorkflowID: checkpoint.WorkflowID,
		AgentID:    checkpoint.AgentID,
		StepIndex:  checkpoint.LatestStep,
		RootHash:   checkpoint.RootHash,
		Status:     string(checkpoint.Status),
	}
}

func failVerifyCheck(name, message string) VerifyCheck {
	return VerifyCheck{
		Name:    name,
		Passed:  false,
		Message: message,
	}
}

func compareVerifyCheck(name, expected, actual string) VerifyCheck {
	return VerifyCheck{
		Name:     name,
		Passed:   expected == actual,
		Expected: expected,
		Actual:   actual,
	}
}

func uintToString(v uint64) string {
	return fmt.Sprintf("%d", v)
}
