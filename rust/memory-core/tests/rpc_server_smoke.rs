use std::io::Cursor;

use memory_core::{
    event_log::WorkflowEvent,
    rpc::{Request, Response},
};

#[test]
fn execute_request_init_workflow_returns_state() {
    let req = Request::InitWorkflow {
        workflow_id: "wf-smoke".to_string(),
        agent_id: "agent-smoke".to_string(),
    };

    let resp = memory_core::rpc::execute_request(req);
    match resp {
        Response::State { state } => {
            assert_eq!(state.workflow_id, "wf-smoke");
            assert_eq!(state.agent_id, "agent-smoke");
        }
        Response::Error { message } => panic!("unexpected error response: {message}"),
        other => panic!("unexpected response variant: {other:?}"),
    }
}

#[test]
fn execute_request_append_event_out_of_order_returns_error() {
    let state = memory_core::init_workflow("wf-o3", "agent-o3");
    let req = Request::AppendEvent {
        state,
        event: WorkflowEvent::new("evt-9", 9, "tool_call", "planner", "{}"),
    };

    let resp = memory_core::rpc::execute_request(req);
    match resp {
        Response::Error { message } => assert!(message.contains("step index is out of order")),
        other => panic!("unexpected response variant: {other:?}"),
    }
}

#[test]
fn run_stdio_processes_newline_delimited_requests() {
    let input = "{\"cmd\":\"init_workflow\",\"workflow_id\":\"wf-stdio\",\"agent_id\":\"agent-stdio\"}\n";
    let mut output: Vec<u8> = Vec::new();

    memory_core::rpc::run_stdio(Cursor::new(input.as_bytes()), &mut output)
        .expect("stdio handler should process input");

    let line = String::from_utf8(output).expect("output should be valid utf8");
    let trimmed = line.trim();
    let response: Response = serde_json::from_str(trimmed).expect("response should be valid JSON");

    match response {
        Response::State { state } => {
            assert_eq!(state.workflow_id, "wf-stdio");
            assert_eq!(state.agent_id, "agent-stdio");
        }
        other => panic!("unexpected response variant: {other:?}"),
    }
}
