package openclaw

import (
	"encoding/json"
	"time"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/pkg/types"
)

// EventInput is the richer input format expected from OpenClaw events.
type EventInput struct {
	EventID    string
	WorkflowID string
	RunID      string
	SessionID  string
	EventType  string
	Actor      string
	Payload    any
}

// StepInput keeps backward compatibility with older CLI-only inputs.
type StepInput = EventInput

// NormalizeEvent converts an external OpenClaw payload into internal workflow event format.
func NormalizeEvent(in EventInput) types.WorkflowStepEvent {
	eventType := in.EventType
	if eventType == "" {
		eventType = "task_event"
	}
	actor := in.Actor
	if actor == "" {
		actor = "openclaw"
	}
	workflowID := in.WorkflowID
	if workflowID == "" {
		workflowID = in.RunID
	}
	if workflowID == "" {
		workflowID = in.SessionID
	}

	return types.WorkflowStepEvent{
		EventID:    in.EventID,
		WorkflowID: workflowID,
		EventType:  eventType,
		Actor:      actor,
		Payload:    normalizePayload(in.Payload),
		CreatedAt:  time.Now().UTC(),
	}
}

// NormalizeStep converts an external step payload into internal workflow event format.
func NormalizeStep(in StepInput) types.WorkflowStepEvent {
	return NormalizeEvent(in)
}

func normalizePayload(payload any) string {
	switch v := payload.(type) {
	case nil:
		return "{}"
	case string:
		if v == "" {
			return "{}"
		}
		return v
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return "{}"
		}
		return string(raw)
	}
}
