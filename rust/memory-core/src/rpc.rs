use serde::{Deserialize, Serialize};
use std::io::{self, BufRead, Write};

use crate::{
    append_event, build_checkpoint, init_workflow, replay_workflow, resume_from_checkpoint,
    checkpoint::Checkpoint, event_log::WorkflowEvent, workflow_state::WorkflowState,
};

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "cmd", rename_all = "snake_case")]
pub enum Request {
    InitWorkflow {
        workflow_id: String,
        agent_id: String,
    },
    AppendEvent {
        state: WorkflowState,
        event: WorkflowEvent,
    },
    BuildCheckpoint {
        state: WorkflowState,
    },
    ResumeFromCheckpoint {
        checkpoint: Checkpoint,
    },
    ReplayWorkflow {
        workflow_id: String,
        agent_id: String,
        events: Vec<WorkflowEvent>,
    },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "kind", rename_all = "snake_case")]
pub enum Response {
    State { state: WorkflowState },
    Checkpoint { checkpoint: Checkpoint },
    Error { message: String },
}

pub fn execute_request(request: Request) -> Response {
    match request {
        Request::InitWorkflow {
            workflow_id,
            agent_id,
        } => Response::State {
            state: init_workflow(workflow_id, agent_id),
        },
        Request::AppendEvent { mut state, event } => match append_event(&mut state, event) {
            Ok(()) => Response::State { state },
            Err(err) => Response::Error {
                message: err.to_string(),
            },
        },
        Request::BuildCheckpoint { state } => Response::Checkpoint {
            checkpoint: build_checkpoint(&state),
        },
        Request::ResumeFromCheckpoint { checkpoint } => Response::State {
            state: resume_from_checkpoint(&checkpoint),
        },
        Request::ReplayWorkflow {
            workflow_id,
            agent_id,
            events,
        } => match replay_workflow(workflow_id, agent_id, &events) {
            Ok(state) => Response::State { state },
            Err(err) => Response::Error {
                message: err.to_string(),
            },
        },
    }
}

pub fn parse_request_line(line: &str) -> Result<Request, String> {
    serde_json::from_str::<Request>(line).map_err(|err| format!("invalid request JSON: {err}"))
}

pub fn encode_response_line(response: &Response) -> Result<String, String> {
    serde_json::to_string(response).map_err(|err| format!("failed to encode response JSON: {err}"))
}

pub fn run_stdio<R: BufRead, W: Write>(reader: R, mut writer: W) -> io::Result<()> {
    for line_result in reader.lines() {
        let line = line_result?;
        if line.trim().is_empty() {
            continue;
        }

        let response = match parse_request_line(&line) {
            Ok(req) => execute_request(req),
            Err(message) => Response::Error { message },
        };

        let encoded = match encode_response_line(&response) {
            Ok(value) => value,
            Err(message) => {
                let fallback = Response::Error { message };
                serde_json::to_string(&fallback).unwrap_or_else(|_| {
                    "{\"kind\":\"error\",\"message\":\"internal encoding error\"}".to_string()
                })
            }
        };

        writer.write_all(encoded.as_bytes())?;
        writer.write_all(b"\n")?;
        writer.flush()?;
    }

    Ok(())
}
