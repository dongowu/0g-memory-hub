package workflow

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeRuntimeTransport struct {
	lastRequest RuntimeRequest
	response    RuntimeResponse
	err         error
}

func (f *fakeRuntimeTransport) Call(_ context.Context, requestJSON []byte) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	if err := json.Unmarshal(requestJSON, &f.lastRequest); err != nil {
		return nil, err
	}
	return json.Marshal(f.response)
}

func TestRuntimeClientInitWorkflow(t *testing.T) {
	t.Parallel()

	transport := &fakeRuntimeTransport{
		response: RuntimeResponse{
			Kind: "state",
			State: &RuntimeState{
				WorkflowID: "wf-1",
				AgentID:    "agent-1",
				Status:     RuntimeStatusRunning,
			},
		},
	}
	client := NewRuntimeClient(transport)

	state, err := client.InitWorkflow(context.Background(), "wf-1", "agent-1")
	if err != nil {
		t.Fatalf("InitWorkflow() error = %v", err)
	}
	if state.WorkflowID != "wf-1" || state.AgentID != "agent-1" {
		t.Fatalf("unexpected state: %+v", state)
	}
	if transport.lastRequest.Cmd != "init_workflow" {
		t.Fatalf("request cmd = %s, want init_workflow", transport.lastRequest.Cmd)
	}
}

func TestRuntimeClientAppendEventProtocolShape(t *testing.T) {
	t.Parallel()

	transport := &fakeRuntimeTransport{
		response: RuntimeResponse{
			Kind: "state",
			State: &RuntimeState{
				WorkflowID: "wf-2",
				AgentID:    "agent-2",
				Status:     RuntimeStatusRunning,
				LatestStep: 1,
			},
		},
	}
	client := NewRuntimeClient(transport)

	state := RuntimeState{WorkflowID: "wf-2", AgentID: "agent-2", Status: RuntimeStatusRunning}
	event := RuntimeEvent{
		EventID:   "evt-1",
		StepIndex: 0,
		EventType: "tool_call",
		Actor:     "planner",
		Payload:   "{\"tool\":\"search\"}",
	}

	if _, err := client.AppendEvent(context.Background(), state, event); err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}

	if transport.lastRequest.Cmd != "append_event" {
		t.Fatalf("request cmd = %s, want append_event", transport.lastRequest.Cmd)
	}
	if transport.lastRequest.State == nil || transport.lastRequest.Event == nil {
		t.Fatalf("append_event request missing state/event: %+v", transport.lastRequest)
	}
	if transport.lastRequest.Event.EventID != "evt-1" {
		t.Fatalf("event id = %s, want evt-1", transport.lastRequest.Event.EventID)
	}
}

func TestRuntimeProtocolRoundTripJSON(t *testing.T) {
	t.Parallel()

	req := RuntimeRequest{
		Cmd:        "replay_workflow",
		WorkflowID: "wf-3",
		AgentID:    "agent-3",
		Events: []RuntimeEvent{
			{EventID: "evt-0", StepIndex: 0, EventType: "tool_call", Actor: "planner", Payload: "{}"},
		},
	}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var decoded RuntimeRequest
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if decoded.Cmd != "replay_workflow" {
		t.Fatalf("decoded cmd = %s, want replay_workflow", decoded.Cmd)
	}
	if len(decoded.Events) != 1 || decoded.Events[0].EventID != "evt-0" {
		t.Fatalf("decoded events mismatch: %+v", decoded.Events)
	}
}

func TestRuntimeClientErrorResponse(t *testing.T) {
	t.Parallel()

	transport := &fakeRuntimeTransport{
		response: RuntimeResponse{
			Kind:    "error",
			Message: "invalid step index",
		},
	}
	client := NewRuntimeClient(transport)

	_, err := client.InitWorkflow(context.Background(), "wf-err", "agent-err")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "invalid step index" {
		t.Fatalf("unexpected error: %v", err)
	}
}
