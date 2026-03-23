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
	EventID       string    `json:"event_id"`
	WorkflowID    string    `json:"workflow_id"`
	RunID         string    `json:"run_id,omitempty"`
	SessionID     string    `json:"session_id,omitempty"`
	TraceID       string    `json:"trace_id,omitempty"`
	ParentEventID string    `json:"parent_event_id,omitempty"`
	ToolCallID    string    `json:"tool_call_id,omitempty"`
	SkillName     string    `json:"skill_name,omitempty"`
	TaskID        string    `json:"task_id,omitempty"`
	Role          string    `json:"role,omitempty"`
	StepIndex     int64     `json:"step_index"`
	EventType     string    `json:"event_type"`
	Actor         string    `json:"actor"`
	Payload       string    `json:"payload"`
	CreatedAt     time.Time `json:"created_at"`
}
