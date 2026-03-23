package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type RuntimeTransport interface {
	Call(ctx context.Context, requestJSON []byte) ([]byte, error)
}

type RuntimeClient struct {
	transport RuntimeTransport
}

func NewRuntimeClient(transport RuntimeTransport) *RuntimeClient {
	return &RuntimeClient{transport: transport}
}

func (c *RuntimeClient) CheckReadiness(ctx context.Context) error {
	_, err := c.InitWorkflow(ctx, "readiness-probe", "readiness-probe")
	return err
}

func (c *RuntimeClient) InitWorkflow(ctx context.Context, workflowID, agentID string) (*RuntimeState, error) {
	req := RuntimeRequest{
		Cmd:        "init_workflow",
		WorkflowID: workflowID,
		AgentID:    agentID,
	}
	resp, err := c.call(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.State == nil {
		return nil, errors.New("runtime response missing state")
	}
	return resp.State, nil
}

func (c *RuntimeClient) AppendEvent(ctx context.Context, state RuntimeState, event RuntimeEvent) (*RuntimeState, error) {
	req := RuntimeRequest{
		Cmd:   "append_event",
		State: &state,
		Event: &event,
	}
	resp, err := c.call(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.State == nil {
		return nil, errors.New("runtime response missing state")
	}
	return resp.State, nil
}

func (c *RuntimeClient) BuildCheckpoint(ctx context.Context, state RuntimeState) (*RuntimeCheckpoint, error) {
	req := RuntimeRequest{
		Cmd:   "build_checkpoint",
		State: &state,
	}
	resp, err := c.call(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.Checkpoint == nil {
		return nil, errors.New("runtime response missing checkpoint")
	}
	return resp.Checkpoint, nil
}

func (c *RuntimeClient) ResumeFromCheckpoint(ctx context.Context, checkpoint RuntimeCheckpoint) (*RuntimeState, error) {
	req := RuntimeRequest{
		Cmd:        "resume_from_checkpoint",
		Checkpoint: &checkpoint,
	}
	resp, err := c.call(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.State == nil {
		return nil, errors.New("runtime response missing state")
	}
	return resp.State, nil
}

func (c *RuntimeClient) ReplayWorkflow(ctx context.Context, workflowID, agentID string, events []RuntimeEvent) (*RuntimeState, error) {
	req := RuntimeRequest{
		Cmd:        "replay_workflow",
		WorkflowID: workflowID,
		AgentID:    agentID,
		Events:     events,
	}
	resp, err := c.call(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.State == nil {
		return nil, errors.New("runtime response missing state")
	}
	return resp.State, nil
}

func (c *RuntimeClient) call(ctx context.Context, req RuntimeRequest) (*RuntimeResponse, error) {
	rawReq, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal runtime request: %w", err)
	}
	rawResp, err := c.transport.Call(ctx, rawReq)
	if err != nil {
		return nil, fmt.Errorf("runtime transport call: %w", err)
	}
	var resp RuntimeResponse
	if err := json.Unmarshal(rawResp, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal runtime response: %w", err)
	}
	if resp.Kind == "error" {
		if resp.Message == "" {
			return nil, errors.New("runtime error response without message")
		}
		return nil, errors.New(resp.Message)
	}
	return &resp, nil
}
