package openclaw

import (
	"encoding/json"
	"testing"
)

func TestNormalizeEventPrefersWorkflowIDAndPreservesEventID(t *testing.T) {
	t.Parallel()

	got := NormalizeEvent(EventInput{
		EventID:    "evt-123",
		WorkflowID: "wf-abc",
		RunID:      "run-ignored",
		EventType:  "tool_result",
		Actor:      "worker",
		Payload: map[string]any{
			"ok":   true,
			"tool": "search",
		},
	})

	if got.WorkflowID != "wf-abc" {
		t.Fatalf("WorkflowID = %q, want wf-abc", got.WorkflowID)
	}
	if got.EventID != "evt-123" {
		t.Fatalf("EventID = %q, want evt-123", got.EventID)
	}
	if got.EventType != "tool_result" {
		t.Fatalf("EventType = %q, want tool_result", got.EventType)
	}
	if got.Actor != "worker" {
		t.Fatalf("Actor = %q, want worker", got.Actor)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(got.Payload), &payload); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	if payload["tool"] != "search" {
		t.Fatalf("payload.tool = %v, want search", payload["tool"])
	}
}

func TestNormalizeEventFallsBackToRunIDAndDefaults(t *testing.T) {
	t.Parallel()

	got := NormalizeEvent(EventInput{
		RunID:   "run-123",
		Payload: "",
	})

	if got.WorkflowID != "run-123" {
		t.Fatalf("WorkflowID = %q, want run-123", got.WorkflowID)
	}
	if got.EventType != "task_event" {
		t.Fatalf("EventType = %q, want task_event", got.EventType)
	}
	if got.Actor != "openclaw" {
		t.Fatalf("Actor = %q, want openclaw", got.Actor)
	}
	if got.Payload != "{}" {
		t.Fatalf("Payload = %q, want {}", got.Payload)
	}
}

func TestNormalizeEventPreservesExtendedOpenClawMetadata(t *testing.T) {
	t.Parallel()

	got := NormalizeEvent(EventInput{
		EventID:       "evt-extended",
		WorkflowID:    "wf-extended",
		RunID:         "run-extended",
		SessionID:     "session-extended",
		TraceID:       "trace-extended",
		ParentEventID: "evt-parent",
		ToolCallID:    "tool-call-1",
		SkillName:     "memory_reader",
		TaskID:        "task-42",
		Role:          "planner",
		EventType:     "tool_call",
		Actor:         "coordinator",
		Payload:       map[string]any{"input": "hello"},
	})

	if got.RunID != "run-extended" {
		t.Fatalf("RunID = %q, want run-extended", got.RunID)
	}
	if got.SessionID != "session-extended" {
		t.Fatalf("SessionID = %q, want session-extended", got.SessionID)
	}
	if got.TraceID != "trace-extended" {
		t.Fatalf("TraceID = %q, want trace-extended", got.TraceID)
	}
	if got.ParentEventID != "evt-parent" {
		t.Fatalf("ParentEventID = %q, want evt-parent", got.ParentEventID)
	}
	if got.ToolCallID != "tool-call-1" {
		t.Fatalf("ToolCallID = %q, want tool-call-1", got.ToolCallID)
	}
	if got.SkillName != "memory_reader" {
		t.Fatalf("SkillName = %q, want memory_reader", got.SkillName)
	}
	if got.TaskID != "task-42" {
		t.Fatalf("TaskID = %q, want task-42", got.TaskID)
	}
	if got.Role != "planner" {
		t.Fatalf("Role = %q, want planner", got.Role)
	}
	if got.Actor != "coordinator" {
		t.Fatalf("Actor = %q, want coordinator", got.Actor)
	}
}
