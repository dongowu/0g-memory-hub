use memory_core::{
    checkpoint::Checkpoint,
    event_log::WorkflowEvent,
    rpc::{Request, Response},
    workflow_state::{WorkflowState, WorkflowStatus},
};

#[test]
fn request_serializes_with_expected_cmd_tag() {
    let req = Request::InitWorkflow {
        workflow_id: "wf-1".to_string(),
        agent_id: "agent-1".to_string(),
    };

    let value = serde_json::to_value(req).expect("serialize request");
    assert_eq!(value["cmd"], "init_workflow");
    assert_eq!(value["workflow_id"], "wf-1");
    assert_eq!(value["agent_id"], "agent-1");
}

#[test]
fn append_event_request_roundtrip() {
    let mut state = WorkflowState::new("wf-2".to_string(), "agent-2".to_string());
    let event = WorkflowEvent::new("evt-0", 0, "tool_call", "planner", "{\"tool\":\"search\"}");
    state.append_event(event.clone()).expect("append event");

    let req = Request::AppendEvent { state, event };
    let payload = serde_json::to_string(&req).expect("serialize");
    let decoded: Request = serde_json::from_str(&payload).expect("deserialize");

    match decoded {
        Request::AppendEvent { state, event } => {
            assert_eq!(state.workflow_id, "wf-2");
            assert_eq!(event.event_id, "evt-0");
            assert_eq!(event.step_index, 0);
        }
        other => panic!("unexpected variant: {:?}", other),
    }
}

#[test]
fn response_error_shape_matches_contract() {
    let resp = Response::Error {
        message: "invalid step index".to_string(),
    };
    let value = serde_json::to_value(resp).expect("serialize response");

    assert_eq!(value["kind"], "error");
    assert_eq!(value["message"], "invalid step index");
}

#[test]
fn checkpoint_response_roundtrip() {
    let checkpoint = Checkpoint {
        workflow_id: "wf-3".to_string(),
        agent_id: "agent-3".to_string(),
        latest_step: 2,
        root_hash: "abc123".to_string(),
        status: WorkflowStatus::Running,
        events: vec![WorkflowEvent::new(
            "evt-1",
            0,
            "tool_call",
            "planner",
            "{\"q\":\"weather\"}",
        )],
    };
    let resp = Response::Checkpoint { checkpoint };
    let payload = serde_json::to_string(&resp).expect("serialize");
    let decoded: Response = serde_json::from_str(&payload).expect("deserialize");

    match decoded {
        Response::Checkpoint { checkpoint } => {
            assert_eq!(checkpoint.workflow_id, "wf-3");
            assert_eq!(checkpoint.latest_step, 2);
        }
        other => panic!("unexpected variant: {:?}", other),
    }
}

