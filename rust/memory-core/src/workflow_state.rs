use serde::{Deserialize, Serialize};

use crate::{event_log::WorkflowEvent, merkle::compute_root, MemoryCoreError};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum WorkflowStatus {
    Running,
    Paused,
    Completed,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct WorkflowState {
    pub workflow_id: String,
    pub agent_id: String,
    pub status: WorkflowStatus,
    pub latest_step: u64,
    pub latest_root: String,
    pub events: Vec<WorkflowEvent>,
}

impl WorkflowState {
    pub fn new(workflow_id: String, agent_id: String) -> Self {
        Self {
            workflow_id,
            agent_id,
            status: WorkflowStatus::Running,
            latest_step: 0,
            latest_root: compute_root(&[]),
            events: Vec::new(),
        }
    }

    pub fn append_event(&mut self, event: WorkflowEvent) -> Result<(), MemoryCoreError> {
        let expected = self.events.len() as u64;
        if event.step_index != expected {
            return Err(MemoryCoreError::InvalidStepOrder {
                expected,
                got: event.step_index,
            });
        }

        self.latest_step = event.step_index;
        self.events.push(event);
        self.latest_root = compute_root(&self.events);
        Ok(())
    }
}

