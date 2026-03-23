use crate::{event_log::WorkflowEvent, init_workflow, MemoryCoreError, WorkflowState};

pub fn replay_workflow(
    workflow_id: impl Into<String>,
    agent_id: impl Into<String>,
    events: &[WorkflowEvent],
) -> Result<WorkflowState, MemoryCoreError> {
    let mut state = init_workflow(workflow_id, agent_id);
    for event in events {
        state.append_event(event.clone())?;
    }
    Ok(state)
}

