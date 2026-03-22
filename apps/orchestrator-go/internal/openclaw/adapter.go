package openclaw

import (
	"time"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/pkg/types"
)

// StepInput is the normalized input format expected from OpenClaw events.
type StepInput struct {
	WorkflowID string
	EventType  string
	Actor      string
	Payload    string
}

// NormalizeStep converts an external step payload into internal workflow event format.
func NormalizeStep(in StepInput) types.WorkflowStepEvent {
	eventType := in.EventType
	if eventType == "" {
		eventType = "task_event"
	}
	actor := in.Actor
	if actor == "" {
		actor = "openclaw"
	}

	return types.WorkflowStepEvent{
		WorkflowID: in.WorkflowID,
		EventType:  eventType,
		Actor:      actor,
		Payload:    in.Payload,
		CreatedAt:  time.Now().UTC(),
	}
}
