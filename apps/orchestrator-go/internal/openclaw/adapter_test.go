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
