use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct WorkflowEvent {
    pub event_id: String,
    pub step_index: u64,
    pub event_type: String,
    pub actor: String,
    pub payload: String,
}

impl WorkflowEvent {
    pub fn new(
        event_id: impl Into<String>,
        step_index: u64,
        event_type: impl Into<String>,
        actor: impl Into<String>,
        payload: impl Into<String>,
    ) -> Self {
        Self {
            event_id: event_id.into(),
            step_index,
            event_type: event_type.into(),
            actor: actor.into(),
            payload: payload.into(),
        }
    }
}

