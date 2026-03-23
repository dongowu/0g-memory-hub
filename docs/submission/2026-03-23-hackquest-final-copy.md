# 0G APAC Hackathon Final Submission Copy

> Track: **Agentic Infrastructure & OpenClaw Lab**
>
> Project: **0G OpenClaw Memory Runtime**

---

## 1. Project Name

**0G OpenClaw Memory Runtime**

---

## 2. One-Sentence Description

**A durable OpenClaw workflow memory layer on 0G that checkpoints agent execution in Rust, persists state to 0G Storage, and anchors verification metadata on-chain.**

---

## 3. Short Summary

0G OpenClaw Memory Runtime is an infrastructure layer for long-lived agent workflows. It accepts OpenClaw-style events through a Go orchestrator, reconstructs deterministic workflow state in a Rust runtime, writes checkpoint blobs to 0G Storage, and anchors verification metadata on-chain through a MemoryAnchor contract path. The result is a workflow memory system that is replayable, resumable, and externally verifiable instead of being trapped inside a single running process.

---

## 4. Problem

Most agent demos are stateless or operationally fragile. If the process dies, the workflow context disappears. Even when state is persisted, judges and operators often cannot verify what actually happened at each step, whether the workflow can be resumed, or whether the stored state corresponds to a specific execution trace.

For OpenClaw-style orchestration, this creates three infra problems:

1. **No durable workflow memory**
2. **No deterministic replay / resume path**
3. **No verifiable binding between off-chain state and on-chain proof**

---

## 5. Solution

This project turns agent execution into a durable OpenClaw run memory primitive:

- The **Go orchestrator** exposes an OpenClaw-facing HTTP API, workflow CLI, plus the new run context, checkpoint, hydrate, and trace endpoints.
- The **Rust runtime** deterministically replays events and builds checkpoints.
- The **0G Storage path** persists checkpoint blobs for long-term recovery.
- The **0G Chain path** anchors verification metadata such as workflow ID, step index, root hash, and CID hash.
 - The service supports **replay**, **resume**, **health checks**, **run context**, **checkpoint metadata**, **hydrate**, **trace**, **batch ingest**, and **idempotent event ingestion** so workflows behave like real infra rather than a one-shot demo script.

---

## 6. Which 0G Components Are Used

### 0G Storage

Used to upload and download workflow checkpoint blobs. This is the durable memory layer for workflow recovery.

### 0G Chain

Used to anchor checkpoint verification metadata through the MemoryAnchor contract path.

### Optional / Infra Context

The project is positioned as agent infrastructure for the OpenClaw Lab track, with a design that can later attach model inference or orchestration modules without changing the workflow memory layer.

---

## 7. Why 0G Is Necessary Here

This project is not using 0G as a generic brand add-on. It relies on 0G in a workload-shaped way:

- **Storage** holds durable checkpoint state outside the process boundary.
- **Chain** provides a verifiable anchor that others can inspect.
- Together they support a stronger agent infra claim: workflows can be **persisted, resumed, replayed, and audited**.

Without that combination, the system would be only a local runtime and not a credible Web4 / agent memory infra layer.

---

## 8. Key Features Implemented

- OpenClaw-style single-event ingest
- OpenClaw-style batch ingest with per-item results
- Rich OpenClaw event metadata preservation (`runId`, `sessionId`, `traceId`, `parentEventId`, `toolCallId`, `skillName`, `taskId`, `role`)
- Idempotent event handling by `eventId`
- Deterministic Rust checkpoint generation
- Workflow replay, hydrate, and resume
- OpenClaw run context endpoint
- OpenClaw latest checkpoint endpoint
- OpenClaw judge-friendly trace endpoint
- 0G Storage checkpoint upload / download path
- MemoryAnchor chain anchor path
- HTTP health endpoint with live dependency probes
- Per-workflow locking and request-context propagation for safer long-running service behavior

---

## 9. Repository / Demo / Evidence

### Repository

`https://github.com/dongowu/0g-memory-hub`

### Key Paths

- `apps/orchestrator-go`
- `rust/memory-core`
- `contracts/MemoryAnchor.sol`

### Evidence Docs

- `docs/evidence/2026-03-22-live-storage-chain-proof.md`
- `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`
- `docs/evidence/2026-03-23-live-http-readiness-proof.md`

### Demo Docs

- `docs/demo/3min-judge-flow.md`
- `docs/demo/judge-checklist.md`

---

## 10. Fields to Fill Before Submission

- **Mainnet contract address:** `TODO`
- **0G Explorer link:** `TODO`
- **Demo video link:** `TODO`
- **X / Twitter post link:** `TODO`

---

## 11. Suggested 30-Second Pitch

We built a durable memory layer for OpenClaw-style agent workflows on 0G. The Go service accepts workflow events, the Rust runtime deterministically rebuilds state and emits checkpoints, 0G Storage persists those checkpoints, and 0G Chain anchors verification metadata. So instead of an agent losing its context when a process dies, the workflow can be replayed, resumed, and externally verified.
