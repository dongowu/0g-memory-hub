use memory_core::{
    append_event, build_checkpoint, event_log::WorkflowEvent, init_workflow, replay_workflow,
    resume_from_checkpoint, workflow_state::WorkflowStatus,
};

fn sample_events() -> Vec<WorkflowEvent> {
    vec![
        WorkflowEvent::new("evt-0", 0, "tool_call", "planner", "{\"tool\":\"search\"}"),
        WorkflowEvent::new("evt-1", 1, "tool_result", "executor", "{\"ok\":true}"),
        WorkflowEvent::new("evt-2", 2, "summary", "memory", "{\"tokens\":42}"),
    ]
}

#[test]
fn deterministic_root_for_same_ordered_events() {
    let events = sample_events();

    let mut a = init_workflow("wf-1", "agent-1");
    let mut b = init_workflow("wf-1", "agent-1");

    for event in &events {
        append_event(&mut a, event.clone()).expect("append into state a");
        append_event(&mut b, event.clone()).expect("append into state b");
    }

    assert_eq!(a.latest_root, b.latest_root);
    assert_eq!(a.latest_step, 2);
}

#[test]
fn replay_reconstructs_state() {
    let events = sample_events();
    let replayed = replay_workflow("wf-2", "agent-2", &events).expect("replay should succeed");

    assert_eq!(replayed.workflow_id, "wf-2");
    assert_eq!(replayed.agent_id, "agent-2");
    assert_eq!(replayed.latest_step, 2);
    assert_eq!(replayed.events, events);
    assert_eq!(replayed.status, WorkflowStatus::Running);
}

#[test]
fn checkpoint_roundtrip_serialization_and_resume() {
    let events = sample_events();
    let mut state = init_workflow("wf-3", "agent-3");
    for event in events {
        append_event(&mut state, event).expect("append should succeed");
    }

    let checkpoint = build_checkpoint(&state);
    let payload = serde_json::to_string(&checkpoint).expect("serialize checkpoint");
    let decoded: memory_core::Checkpoint =
        serde_json::from_str(&payload).expect("deserialize checkpoint");
    let resumed = resume_from_checkpoint(&decoded);

    assert_eq!(decoded.workflow_id, "wf-3");
    assert_eq!(decoded.latest_step, 2);
    assert_eq!(decoded.root_hash, state.latest_root);
    assert_eq!(resumed, state);
}

