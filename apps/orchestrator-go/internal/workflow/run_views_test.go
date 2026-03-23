package workflow

import (
	"testing"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/pkg/types"
)

func TestRunIdentityFromEventsUsesLatestNonEmptyValues(t *testing.T) {
	t.Parallel()

	identity := runIdentityFromEvents([]types.WorkflowStepEvent{
		{EventID: "evt-1", RunID: "run-1"},
		{EventID: "evt-2", SessionID: "session-2"},
		{EventID: "evt-3", TraceID: "trace-3"},
		{EventID: "evt-4", RunID: "run-4"},
	}, "wf-fallback")

	if identity.RunID != "run-4" {
		t.Fatalf("RunID = %q, want run-4", identity.RunID)
	}
	if identity.SessionID != "session-2" {
		t.Fatalf("SessionID = %q, want session-2", identity.SessionID)
	}
	if identity.TraceID != "trace-3" {
		t.Fatalf("TraceID = %q, want trace-3", identity.TraceID)
	}
}

func TestRunIdentityFromEventsFallsBackToWorkflowID(t *testing.T) {
	t.Parallel()

	identity := runIdentityFromEvents(nil, "wf-fallback")
	if identity.RunID != "wf-fallback" {
		t.Fatalf("RunID = %q, want wf-fallback", identity.RunID)
	}
	if identity.SessionID != "" {
		t.Fatalf("SessionID = %q, want empty", identity.SessionID)
	}
	if identity.TraceID != "" {
		t.Fatalf("TraceID = %q, want empty", identity.TraceID)
	}
}

func TestBuildRunContextAndTraceProjectEventMetadata(t *testing.T) {
	t.Parallel()

	meta := types.WorkflowMetadata{
		WorkflowID:   "wf-projection",
		AgentID:      "agent-wf-projection",
		Status:       types.WorkflowStatusRunning,
		LatestStep:   7,
		LatestRoot:   "root-7",
		LatestCID:    "cid-7",
		LatestTxHash: "tx-7",
		Events: []types.WorkflowStepEvent{
			{
				EventID:       "evt-1",
				WorkflowID:    "wf-projection",
				RunID:         "run-projection",
				SessionID:     "session-projection",
				TraceID:       "trace-projection",
				StepIndex:     7,
				EventType:     "tool_call",
				Actor:         "planner",
				Role:          "planner",
				ParentEventID: "evt-root",
				ToolCallID:    "tool-1",
				SkillName:     "memory_reader",
				TaskID:        "task-1",
				Payload:       `{"q":"hello"}`,
			},
		},
	}

	contextView := buildRunContext(meta)
	traceView := buildRunTrace(meta)
	checkpointView := buildLatestCheckpoint(meta)

	if contextView.RunID != "run-projection" || traceView.RunID != "run-projection" || checkpointView.RunID != "run-projection" {
		t.Fatalf("projected run ids = %#v / %#v / %#v, want run-projection", contextView.RunID, traceView.RunID, checkpointView.RunID)
	}
	if len(contextView.Events) != 1 {
		t.Fatalf("len(contextView.Events) = %d, want 1", len(contextView.Events))
	}
	if len(traceView.Steps) != 1 {
		t.Fatalf("len(traceView.Steps) = %d, want 1", len(traceView.Steps))
	}
	if contextView.Events[0].ToolCallID != "tool-1" {
		t.Fatalf("context ToolCallID = %q, want tool-1", contextView.Events[0].ToolCallID)
	}
	if traceView.Steps[0].SkillName != "memory_reader" {
		t.Fatalf("trace SkillName = %q, want memory_reader", traceView.Steps[0].SkillName)
	}
	if checkpointView.TraceID != "trace-projection" {
		t.Fatalf("checkpoint TraceID = %q, want trace-projection", checkpointView.TraceID)
	}
}
