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

/// A differential checkpoint that only stores events since the last checkpoint.
/// This reduces 0G Storage upload size significantly for long-running workflows.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct DiffCheckpoint {
    pub workflow_id: String,
    pub agent_id: String,
    /// Number of events in the base checkpoint (0 means empty base).
    pub base_event_count: u64,
    pub base_root_hash: String,
    pub latest_step: u64,
    pub root_hash: String,
    pub status: WorkflowStatus,
    /// Only the events appended since the base checkpoint.
    pub delta_events: Vec<WorkflowEvent>,
}

impl DiffCheckpoint {
    /// Build a differential checkpoint from a full state.
    /// `base_event_count` is the number of events already in the base checkpoint.
    pub fn from_state(state: &WorkflowState, base_event_count: u64, base_root_hash: String) -> Self {
        let delta_start = (base_event_count as usize).min(state.events.len());
        let delta_events = state.events[delta_start..].to_vec();

        Self {
            workflow_id: state.workflow_id.clone(),
            agent_id: state.agent_id.clone(),
            base_event_count,
            base_root_hash,
            latest_step: state.latest_step,
            root_hash: state.latest_root.clone(),
            status: state.status.clone(),
            delta_events,
        }
    }

    /// Apply this diff on top of a base checkpoint to reconstruct the full state.
    pub fn apply_to(&self, base: &Checkpoint) -> Result<WorkflowState, DiffError> {
        if base.workflow_id != self.workflow_id {
            return Err(DiffError::WorkflowMismatch {
                base: base.workflow_id.clone(),
                diff: self.workflow_id.clone(),
            });
        }
        if base.events.len() as u64 != self.base_event_count {
            return Err(DiffError::StepMismatch {
                base_step: base.events.len() as u64,
                diff_base_step: self.base_event_count,
            });
        }
        if base.root_hash != self.base_root_hash {
            return Err(DiffError::RootMismatch {
                base_root: base.root_hash.clone(),
                diff_base_root: self.base_root_hash.clone(),
            });
        }

        let mut state = base.to_state();
        for event in &self.delta_events {
            state
                .append_event(event.clone())
                .map_err(|e| DiffError::AppendFailed(e.to_string()))?;
        }
        Ok(state)
    }

    /// Number of events in the delta.
    pub fn delta_size(&self) -> usize {
        self.delta_events.len()
    }
}

#[derive(Debug)]
pub enum DiffError {
    WorkflowMismatch { base: String, diff: String },
    StepMismatch { base_step: u64, diff_base_step: u64 },
    RootMismatch { base_root: String, diff_base_root: String },
    AppendFailed(String),
}

impl std::fmt::Display for DiffError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::WorkflowMismatch { base, diff } => {
                write!(f, "workflow mismatch: base={base}, diff={diff}")
            }
            Self::StepMismatch { base_step, diff_base_step } => {
                write!(f, "step mismatch: base.latest_step={base_step}, diff.base_step={diff_base_step}")
            }
            Self::RootMismatch { base_root, diff_base_root } => {
                write!(f, "root mismatch: base={base_root}, diff={diff_base_root}")
            }
            Self::AppendFailed(msg) => write!(f, "append failed: {msg}"),
        }
    }
}

impl std::error::Error for DiffError {}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{init_workflow, append_event, build_checkpoint, merkle::compute_root};

    fn make_event(step: u64) -> WorkflowEvent {
        WorkflowEvent::new(
            format!("evt-{step}"),
            step,
            "task_event",
            "agent",
            format!("{{\"step\":{step}}}"),
        )
    }

    #[test]
    fn full_checkpoint_roundtrip() {
        let mut state = init_workflow("wf-1", "agent-1");
        for i in 0..5 {
            append_event(&mut state, make_event(i)).unwrap();
        }
        let cp = build_checkpoint(&state);
        let restored = cp.to_state();
        assert_eq!(restored.latest_step, 4);
        assert_eq!(restored.events.len(), 5);
        assert_eq!(restored.latest_root, compute_root(&state.events));
    }

    #[test]
    fn diff_checkpoint_basic() {
        let mut state = init_workflow("wf-1", "agent-1");
        for i in 0..3 {
            append_event(&mut state, make_event(i)).unwrap();
        }
        let base_cp = build_checkpoint(&state);

        // Append more events
        for i in 3..7 {
            append_event(&mut state, make_event(i)).unwrap();
        }

        let diff = DiffCheckpoint::from_state(&state, base_cp.events.len() as u64, base_cp.root_hash.clone());
        assert_eq!(diff.delta_size(), 4); // events 3,4,5,6
        assert_eq!(diff.base_event_count, 3);
        assert_eq!(diff.latest_step, 6);

        let restored = diff.apply_to(&base_cp).unwrap();
        assert_eq!(restored.latest_step, 6);
        assert_eq!(restored.events.len(), 7);
        assert_eq!(restored.latest_root, state.latest_root);
    }

    #[test]
    fn diff_rejects_workflow_mismatch() {
        let state = init_workflow("wf-1", "agent-1");
        let base_cp = Checkpoint {
            workflow_id: "wf-other".into(),
            agent_id: "agent-1".into(),
            latest_step: 0,
            root_hash: compute_root(&[]),
            status: WorkflowStatus::Running,
            events: vec![],
        };
        let diff = DiffCheckpoint::from_state(&state, 0, compute_root(&[]));
        assert!(diff.apply_to(&base_cp).is_err());
    }

    #[test]
    fn diff_rejects_step_mismatch() {
        let mut state = init_workflow("wf-1", "agent-1");
        for i in 0..3 {
            append_event(&mut state, make_event(i)).unwrap();
        }
        let base_cp = build_checkpoint(&state);

        // Diff claims base has 5 events, but checkpoint has 3
        let diff = DiffCheckpoint::from_state(&state, 5, base_cp.root_hash.clone());
        assert!(diff.apply_to(&base_cp).is_err());
    }

    #[test]
    fn diff_with_zero_base() {
        let mut state = init_workflow("wf-1", "agent-1");
        for i in 0..3 {
            append_event(&mut state, make_event(i)).unwrap();
        }

        let empty_cp = Checkpoint {
            workflow_id: "wf-1".into(),
            agent_id: "agent-1".into(),
            latest_step: 0,
            root_hash: compute_root(&[]),
            status: WorkflowStatus::Running,
            events: vec![],
        };

        let diff = DiffCheckpoint::from_state(&state, 0, compute_root(&[]));
        assert_eq!(diff.delta_size(), 3);

        let restored = diff.apply_to(&empty_cp).unwrap();
        assert_eq!(restored.latest_step, 2);
        assert_eq!(restored.events.len(), 3);
    }
}
