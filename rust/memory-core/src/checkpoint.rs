use serde::{Deserialize, Serialize};

use crate::{event_log::WorkflowEvent, workflow_state::WorkflowStatus, WorkflowState};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Checkpoint {
    pub workflow_id: String,
    pub agent_id: String,
    pub latest_step: u64,
    pub root_hash: String,
    pub status: WorkflowStatus,
    pub events: Vec<WorkflowEvent>,
}

impl Checkpoint {
    pub fn from_state(state: &WorkflowState) -> Self {
        Self {
            workflow_id: state.workflow_id.clone(),
            agent_id: state.agent_id.clone(),
            latest_step: state.latest_step,
            root_hash: state.latest_root.clone(),
            status: state.status.clone(),
            events: state.events.clone(),
        }
    }

    pub fn to_state(&self) -> WorkflowState {
        WorkflowState {
            workflow_id: self.workflow_id.clone(),
            agent_id: self.agent_id.clone(),
            status: self.status.clone(),
            latest_step: self.latest_step,
            latest_root: self.root_hash.clone(),
            events: self.events.clone(),
        }
    }
}

