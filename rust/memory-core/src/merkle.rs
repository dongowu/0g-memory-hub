use crate::event_log::WorkflowEvent;
use sha2::{Digest, Sha256};

pub fn compute_root(events: &[WorkflowEvent]) -> String {
    let mut hasher = Sha256::new();

    for e in events {
        hasher.update(e.event_id.as_bytes());
        hasher.update([0]);
        hasher.update(e.step_index.to_le_bytes());
        hasher.update([0]);
        hasher.update(e.event_type.as_bytes());
        hasher.update([0]);
        hasher.update(e.actor.as_bytes());
        hasher.update([0]);
        hasher.update(e.payload.as_bytes());
        hasher.update([0xff]);
    }

    format!("{:x}", hasher.finalize())
}

