package types

import "time"

type WorkflowStatus string

const (
	WorkflowStatusRunning WorkflowStatus = "running"
	WorkflowStatusPaused  WorkflowStatus = "paused"
)

type WorkflowMetadata struct {
	WorkflowID   string              `json:"workflow_id"`
	AgentID      string              `json:"agent_id"`
	Status       WorkflowStatus      `json:"status"`
	LatestStep   int64               `json:"latest_step"`
	LatestRoot   string              `json:"latest_root"`
	LatestCID    string              `json:"latest_cid"`
	LatestTxHash string              `json:"latest_tx_hash"`
	Events       []WorkflowStepEvent `json:"events"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

type WorkflowStepEvent struct {
	EventID    string    `json:"event_id"`
	WorkflowID string    `json:"workflow_id"`
	StepIndex  int64     `json:"step_index"`
	EventType  string    `json:"event_type"`
	Actor      string    `json:"actor"`
	Payload    string    `json:"payload"`
	CreatedAt  time.Time `json:"created_at"`
}
