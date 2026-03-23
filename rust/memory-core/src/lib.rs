pub mod checkpoint;
pub mod event_log;
pub mod merkle;
pub mod replay;
pub mod rpc;
pub mod workflow_state;

pub use checkpoint::Checkpoint;
pub use event_log::WorkflowEvent;
pub use replay::replay_workflow;
pub use workflow_state::{WorkflowState, WorkflowStatus};

use thiserror::Error;

#[derive(Debug, Error)]
pub enum MemoryCoreError {
    #[error("step index is out of order: expected {expected}, got {got}")]
    InvalidStepOrder { expected: u64, got: u64 },
}

pub fn init_workflow(workflow_id: impl Into<String>, agent_id: impl Into<String>) -> WorkflowState {
    WorkflowState::new(workflow_id.into(), agent_id.into())
}

pub fn append_event(state: &mut WorkflowState, event: WorkflowEvent) -> Result<(), MemoryCoreError> {
    state.append_event(event)
}

pub fn build_checkpoint(state: &WorkflowState) -> Checkpoint {
    Checkpoint::from_state(state)
}

pub fn resume_from_checkpoint(checkpoint: &Checkpoint) -> WorkflowState {
    checkpoint.to_state()
}

