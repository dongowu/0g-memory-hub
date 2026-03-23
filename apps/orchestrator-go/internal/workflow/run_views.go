package workflow

import "github.com/dongowu/0g-memory-hub/apps/orchestrator-go/pkg/types"

type RunContext struct {
	WorkflowID   string            `json:"workflowId"`
	RunID        string            `json:"runId"`
	SessionID    string            `json:"sessionId,omitempty"`
	TraceID      string            `json:"traceId,omitempty"`
	AgentID      string            `json:"agentId"`
	Status       string            `json:"status"`
	LatestStep   int64             `json:"latestStep"`
	LatestRoot   string            `json:"latestRoot"`
	LatestCID    string            `json:"latestCid"`
	LatestTxHash string            `json:"latestTxHash"`
	Events       []RunContextEvent `json:"events"`
}

// runContextEventLimit keeps the context view small enough for API responses while
// still exposing the most recent execution history for debugging and hydration.
const runContextEventLimit = 20

type RunContextEvent struct {
	EventID       string `json:"eventId"`
	StepIndex     int64  `json:"stepIndex"`
	EventType     string `json:"eventType"`
	Actor         string `json:"actor"`
	Role          string `json:"role,omitempty"`
	ParentEventID string `json:"parentEventId,omitempty"`
	ToolCallID    string `json:"toolCallId,omitempty"`
	SkillName     string `json:"skillName,omitempty"`
	TaskID        string `json:"taskId,omitempty"`
	Payload       string `json:"payload"`
}

type LatestCheckpoint struct {
	WorkflowID   string `json:"workflowId"`
	RunID        string `json:"runId"`
	SessionID    string `json:"sessionId,omitempty"`
	TraceID      string `json:"traceId,omitempty"`
	LatestStep   int64  `json:"latestStep"`
	LatestRoot   string `json:"latestRoot"`
	LatestCID    string `json:"latestCid"`
	LatestTxHash string `json:"latestTxHash"`
}

type RunTrace struct {
	WorkflowID   string         `json:"workflowId"`
	RunID        string         `json:"runId"`
	SessionID    string         `json:"sessionId,omitempty"`
	TraceID      string         `json:"traceId,omitempty"`
	Status       string         `json:"status"`
	LatestStep   int64          `json:"latestStep"`
	LatestRoot   string         `json:"latestRoot"`
	LatestCID    string         `json:"latestCid"`
	LatestTxHash string         `json:"latestTxHash"`
	Steps        []RunTraceStep `json:"steps"`
}

type RunTraceStep struct {
	EventID       string `json:"eventId"`
	StepIndex     int64  `json:"stepIndex"`
	EventType     string `json:"eventType"`
	Actor         string `json:"actor"`
	Role          string `json:"role,omitempty"`
	ParentEventID string `json:"parentEventId,omitempty"`
	ToolCallID    string `json:"toolCallId,omitempty"`
	SkillName     string `json:"skillName,omitempty"`
	TaskID        string `json:"taskId,omitempty"`
	Payload       string `json:"payload"`
}

type runIdentity struct {
	RunID     string
	SessionID string
	TraceID   string
}

func runIdentityFromEvents(events []types.WorkflowStepEvent, defaultRunID string) runIdentity {
	identity := runIdentity{}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if identity.RunID == "" && event.RunID != "" {
			identity.RunID = event.RunID
		}
		if identity.SessionID == "" && event.SessionID != "" {
			identity.SessionID = event.SessionID
		}
		if identity.TraceID == "" && event.TraceID != "" {
			identity.TraceID = event.TraceID
		}
		if identity.RunID != "" && identity.SessionID != "" && identity.TraceID != "" {
			break
		}
	}
	if identity.RunID == "" {
		identity.RunID = defaultRunID
	}
	return identity
}

func buildRunContextEvent(event types.WorkflowStepEvent) RunContextEvent {
	return RunContextEvent{
		EventID:       event.EventID,
		StepIndex:     event.StepIndex,
		EventType:     event.EventType,
		Actor:         event.Actor,
		Role:          event.Role,
		ParentEventID: event.ParentEventID,
		ToolCallID:    event.ToolCallID,
		SkillName:     event.SkillName,
		TaskID:        event.TaskID,
		Payload:       event.Payload,
	}
}

func buildRunTraceStep(event types.WorkflowStepEvent) RunTraceStep {
	return RunTraceStep{
		EventID:       event.EventID,
		StepIndex:     event.StepIndex,
		EventType:     event.EventType,
		Actor:         event.Actor,
		Role:          event.Role,
		ParentEventID: event.ParentEventID,
		ToolCallID:    event.ToolCallID,
		SkillName:     event.SkillName,
		TaskID:        event.TaskID,
		Payload:       event.Payload,
	}
}

func buildRunContext(meta types.WorkflowMetadata) RunContext {
	identity := runIdentityFromEvents(meta.Events, meta.WorkflowID)
	events := meta.Events
	if len(events) > runContextEventLimit {
		events = events[len(events)-runContextEventLimit:]
	}
	contextEvents := make([]RunContextEvent, 0, len(events))
	for _, event := range events {
		contextEvents = append(contextEvents, buildRunContextEvent(event))
	}
	return RunContext{
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
		Events:       contextEvents,
	}
}

func buildLatestCheckpoint(meta types.WorkflowMetadata) LatestCheckpoint {
	identity := runIdentityFromEvents(meta.Events, meta.WorkflowID)
	return LatestCheckpoint{
		WorkflowID:   meta.WorkflowID,
		RunID:        identity.RunID,
		SessionID:    identity.SessionID,
		TraceID:      identity.TraceID,
		LatestStep:   meta.LatestStep,
		LatestRoot:   meta.LatestRoot,
		LatestCID:    meta.LatestCID,
		LatestTxHash: meta.LatestTxHash,
	}
}

func buildRunTrace(meta types.WorkflowMetadata) RunTrace {
	identity := runIdentityFromEvents(meta.Events, meta.WorkflowID)
	steps := make([]RunTraceStep, 0, len(meta.Events))
	for _, event := range meta.Events {
		steps = append(steps, buildRunTraceStep(event))
	}
	return RunTrace{
		WorkflowID:   meta.WorkflowID,
		RunID:        identity.RunID,
		SessionID:    identity.SessionID,
		TraceID:      identity.TraceID,
		Status:       string(meta.Status),
		LatestStep:   meta.LatestStep,
		LatestRoot:   meta.LatestRoot,
		LatestCID:    meta.LatestCID,
		LatestTxHash: meta.LatestTxHash,
		Steps:        steps,
	}
}
