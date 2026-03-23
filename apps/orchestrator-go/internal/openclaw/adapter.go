package openclaw

import (
	"encoding/json"
	"time"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/pkg/types"
)

// EventInput is the richer input format expected from OpenClaw events.
type EventInput struct {
	EventID       string `json:"eventId"`
	WorkflowID    string `json:"workflowId"`
	RunID         string `json:"runId"`
	SessionID     string `json:"sessionId"`
	TraceID       string `json:"traceId"`
	ParentEventID string `json:"parentEventId"`
	ToolCallID    string `json:"toolCallId"`
	SkillName     string `json:"skillName"`
	TaskID        string `json:"taskId"`
	Role          string `json:"role"`
	EventType     string `json:"eventType"`
	Actor         string `json:"actor"`
	Payload       any    `json:"payload"`
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
	role := in.Role
	if role == "" {
		role = actor
	}
	workflowID := in.WorkflowID
	if workflowID == "" {
		workflowID = in.RunID
	}
	if workflowID == "" {
		workflowID = in.SessionID
	}

	return types.WorkflowStepEvent{
		EventID:       in.EventID,
		WorkflowID:    workflowID,
		RunID:         in.RunID,
		SessionID:     in.SessionID,
		TraceID:       in.TraceID,
		ParentEventID: in.ParentEventID,
		ToolCallID:    in.ToolCallID,
		SkillName:     in.SkillName,
		TaskID:        in.TaskID,
		Role:          role,
		EventType:     eventType,
		Actor:         actor,
		Payload:       normalizePayload(in.Payload),
		CreatedAt:     time.Now().UTC(),
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
