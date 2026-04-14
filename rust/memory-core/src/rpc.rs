use serde::{Deserialize, Serialize};
use std::io::{self, BufRead, Write};

use crate::{
    append_event, build_checkpoint, init_workflow, replay_workflow, resume_from_checkpoint,
    checkpoint::{Checkpoint, DiffCheckpoint},
    event_log::WorkflowEvent,
    merkle::MerkleTree,
    workflow_state::WorkflowState,
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
    /// Generate an inclusion proof for a specific event index.
    MerkleProof {
        events: Vec<WorkflowEvent>,
        leaf_index: usize,
    },
    /// Build a differential checkpoint (only delta events since base).
    DiffCheckpoint {
        state: WorkflowState,
        base_event_count: u64,
        base_root_hash: String,
    },
    /// Apply a differential checkpoint on top of a base checkpoint.
    ApplyDiff {
        base: Checkpoint,
        diff: DiffCheckpoint,
    },
    /// Encode a checkpoint using CBOR + zstd compression.
    EncodeCheckpoint {
        checkpoint: Checkpoint,
    },
    /// Decode a checkpoint from a base64-encoded wire-format buffer.
    DecodeCheckpoint {
        data_base64: String,
    },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "kind", rename_all = "snake_case")]
pub enum Response {
    State {
        state: WorkflowState,
    },
    Checkpoint {
        checkpoint: Checkpoint,
    },
    MerkleProof {
        proof: crate::merkle::HexMerkleProof,
        root: String,
        leaf_count: usize,
    },
    DiffCheckpoint {
        diff: DiffCheckpoint,
        delta_size: usize,
    },
    EncodedCheckpoint {
        data_base64: String,
        wire_size: usize,
        cbor_size: usize,
    },
    Error {
        message: String,
    },
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
        Request::MerkleProof { events, leaf_index } => {
            let tree = MerkleTree::build(&events);
            match tree.proof(leaf_index) {
                Some(proof) => Response::MerkleProof {
                    proof: proof.to_hex_proof(),
                    root: tree.root_hex(),
                    leaf_count: tree.leaf_count(),
                },
                None => Response::Error {
                    message: format!(
                        "leaf index {leaf_index} out of range (tree has {} leaves)",
                        tree.leaf_count()
                    ),
                },
            }
        }
        Request::DiffCheckpoint {
            state,
            base_event_count,
            base_root_hash,
        } => {
            let diff = DiffCheckpoint::from_state(&state, base_event_count, base_root_hash);
            let delta_size = diff.delta_size();
            Response::DiffCheckpoint { diff, delta_size }
        }
        Request::ApplyDiff { base, diff } => match diff.apply_to(&base) {
            Ok(state) => Response::State { state },
            Err(err) => Response::Error {
                message: err.to_string(),
            },
        },
        Request::EncodeCheckpoint { checkpoint } => match crate::codec::encode(&checkpoint) {
            Ok(wire) => {
                let stats = crate::codec::compression_stats(&checkpoint);
                let cbor_size = stats.map(|s| s.cbor_size).unwrap_or(0);
                Response::EncodedCheckpoint {
                    wire_size: wire.len(),
                    cbor_size,
                    data_base64: base64_encode(&wire),
                }
            }
            Err(err) => Response::Error {
                message: err.to_string(),
            },
        },
        Request::DecodeCheckpoint { data_base64 } => {
            match base64_decode(&data_base64) {
                Ok(wire) => match crate::codec::decode::<Checkpoint>(&wire) {
                    Ok(checkpoint) => Response::Checkpoint { checkpoint },
                    Err(err) => Response::Error {
                        message: err.to_string(),
                    },
                },
                Err(msg) => Response::Error { message: msg },
            }
        }
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

// ── Minimal base64 (avoids adding a crate) ──

fn base64_encode(data: &[u8]) -> String {
    const CHARS: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut result = String::with_capacity((data.len() + 2) / 3 * 4);
    for chunk in data.chunks(3) {
        let b0 = chunk[0] as u32;
        let b1 = if chunk.len() > 1 { chunk[1] as u32 } else { 0 };
        let b2 = if chunk.len() > 2 { chunk[2] as u32 } else { 0 };
        let triple = (b0 << 16) | (b1 << 8) | b2;
        result.push(CHARS[((triple >> 18) & 0x3F) as usize] as char);
        result.push(CHARS[((triple >> 12) & 0x3F) as usize] as char);
        if chunk.len() > 1 {
            result.push(CHARS[((triple >> 6) & 0x3F) as usize] as char);
        } else {
            result.push('=');
        }
        if chunk.len() > 2 {
            result.push(CHARS[(triple & 0x3F) as usize] as char);
        } else {
            result.push('=');
        }
    }
    result
}

fn base64_decode(input: &str) -> Result<Vec<u8>, String> {
    fn val(c: u8) -> Result<u32, String> {
        match c {
            b'A'..=b'Z' => Ok((c - b'A') as u32),
            b'a'..=b'z' => Ok((c - b'a' + 26) as u32),
            b'0'..=b'9' => Ok((c - b'0' + 52) as u32),
            b'+' => Ok(62),
            b'/' => Ok(63),
            b'=' => Ok(0),
            _ => Err(format!("invalid base64 character: {c}")),
        }
    }

    let input = input.trim();
    if input.is_empty() {
        return Ok(vec![]);
    }
    if input.len() % 4 != 0 {
        return Err("invalid base64 length".into());
    }

    let mut result = Vec::with_capacity(input.len() / 4 * 3);
    let bytes = input.as_bytes();
    for chunk in bytes.chunks(4) {
        let a = val(chunk[0])?;
        let b = val(chunk[1])?;
        let c = val(chunk[2])?;
        let d = val(chunk[3])?;
        let triple = (a << 18) | (b << 12) | (c << 6) | d;
        result.push(((triple >> 16) & 0xFF) as u8);
        if chunk[2] != b'=' {
            result.push(((triple >> 8) & 0xFF) as u8);
        }
        if chunk[3] != b'=' {
            result.push((triple & 0xFF) as u8);
        }
    }
    Ok(result)
}
