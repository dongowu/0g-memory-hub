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

0G OpenClaw Memory Runtime is an infrastructure layer for long-lived agent workflows. It accepts OpenClaw-style events through a Go orchestrator, reconstructs deterministic workflow state in a Rust runtime, writes checkpoint blobs to 0G Storage, and anchors verification metadata on-chain through a MemoryAnchor contract path. The result is a workflow memory system that is replayable, resumable, re-verifiable, and externally auditable instead of being trapped inside a single running process.

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

- The **Go orchestrator** exposes an OpenClaw-facing HTTP API, workflow CLI, plus run context, checkpoint, hydrate, verify, and trace endpoints.
- The **Go orchestrator** also exposes a run verify surface (`/v1/openclaw/runs/{id}/verify`, `/judge/verify`, and `workflow verify <run-id>`) for judge-facing integrity checks.
- The **Rust runtime** deterministically replays events and builds checkpoints.
- The **0G Storage path** persists checkpoint blobs for long-term recovery.
- The **0G Chain path** anchors verification metadata such as workflow ID, step index, root hash, and CID hash.
- The service supports **replay**, **resume**, **verify**, **health checks**, **run context**, **checkpoint metadata**, **hydrate**, **trace**, **batch ingest**, and **idempotent event ingestion** so workflows behave like real infra rather than a one-shot demo script.

Judge-facing verification statement:

> We not only recover after restart, we re-derive the checkpoint and compare it against 0G Storage and MemoryAnchor-linked metadata.

---

## 6. Which 0G Components Are Used

### 0G Storage

Used to upload and download workflow checkpoint blobs. This is the durable memory layer for workflow recovery.

### 0G Chain

Used to anchor checkpoint verification metadata through the MemoryAnchor contract path.

### Track Scope

This submission intentionally focuses on durable workflow memory infrastructure for the OpenClaw Lab track. 0G Compute and model-serving capabilities are not claimed in the current shipped version.

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
- Run verify endpoint (`/v1/openclaw/runs/{id}/verify`) for checkpoint re-derivation comparison
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

## 10. Current On-Chain Deployment Evidence

- **Galileo testnet contract address:** `0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- **0G Explorer link:** `https://chainscan-galileo.0g.ai/address/0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- **Anchor proof tx:** `https://chainscan-galileo.0g.ai/tx/0xa794dd7aedcf7b7c349005af620f29d8a36557c7b7973f91e358e31287fad1db`
- **Deployment proof doc:** `docs/evidence/2026-03-23-0g-testnet-memory-anchor-deployment-proof.md`

## 11. Manual Submission Fields Still Pending

- **Mainnet contract address (if required):** `TODO`
- **Demo video link (manual field before submission):** `TODO`
- **X / Twitter post link (manual field before submission):** `TODO`

---

## 12. Suggested 30-Second Pitch

We built a durable memory layer for OpenClaw-style agent workflows on 0G. The Go service accepts workflow events, the Rust runtime deterministically rebuilds state and emits checkpoints, 0G Storage persists those checkpoints, and MemoryAnchor anchors verification metadata. After restart, the run is hydrated, checkpoint integrity is re-derived and compared against persisted proof, and the run stays traceable for judges.
