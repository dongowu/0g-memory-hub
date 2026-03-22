package workflow

// RuntimeEvent mirrors rust/memory-core WorkflowEvent JSON.
type RuntimeEvent struct {
	EventID   string `json:"event_id"`
	StepIndex uint64 `json:"step_index"`
	EventType string `json:"event_type"`
	Actor     string `json:"actor"`
	Payload   string `json:"payload"`
}

// RuntimeState mirrors rust/memory-core WorkflowState JSON.
type RuntimeState struct {
	WorkflowID string         `json:"workflow_id"`
	AgentID    string         `json:"agent_id"`
	Status     RuntimeStatus  `json:"status"`
	LatestStep uint64         `json:"latest_step"`
	LatestRoot string         `json:"latest_root"`
	Events     []RuntimeEvent `json:"events"`
}

// RuntimeCheckpoint mirrors rust/memory-core Checkpoint JSON.
type RuntimeCheckpoint struct {
	WorkflowID string         `json:"workflow_id"`
	AgentID    string         `json:"agent_id"`
	LatestStep uint64         `json:"latest_step"`
	RootHash   string         `json:"root_hash"`
	Status     RuntimeStatus  `json:"status"`
	Events     []RuntimeEvent `json:"events"`
}

type RuntimeStatus string

const (
	RuntimeStatusRunning   RuntimeStatus = "Running"
	RuntimeStatusPaused    RuntimeStatus = "Paused"
	RuntimeStatusCompleted RuntimeStatus = "Completed"
)

// RuntimeRequest is a tagged command envelope compatible with Rust Request enum.
type RuntimeRequest struct {
	Cmd        string             `json:"cmd"`
	WorkflowID string             `json:"workflow_id,omitempty"`
	AgentID    string             `json:"agent_id,omitempty"`
	State      *RuntimeState      `json:"state,omitempty"`
	Event      *RuntimeEvent      `json:"event,omitempty"`
	Checkpoint *RuntimeCheckpoint `json:"checkpoint,omitempty"`
	Events     []RuntimeEvent     `json:"events,omitempty"`
}

// RuntimeResponse is a tagged response envelope compatible with Rust Response enum.
type RuntimeResponse struct {
	Kind       string             `json:"kind"`
	State      *RuntimeState      `json:"state,omitempty"`
	Checkpoint *RuntimeCheckpoint `json:"checkpoint,omitempty"`
	Message    string             `json:"message,omitempty"`
}
