# HackQuest Form Answers (Copy-Paste Ready)

> Project: **0G OpenClaw Memory Runtime**
>
> Track: **Agentic Infrastructure & OpenClaw Lab**

Use this file as the direct source when filling HackQuest fields.

---

## 1. Project Name

**0G OpenClaw Memory Runtime**

---

## 2. One-Sentence Description

### Recommended version

**Durable OpenClaw workflow memory on 0G using Go orchestration, Rust checkpoints, 0G Storage persistence, and on-chain verification anchors.**

### Shorter backup version

**Durable OpenClaw agent memory on 0G with Rust checkpoints, 0G Storage persistence, and on-chain workflow verification.**

---

## 3. What the Project Does

0G OpenClaw Memory Runtime is a workflow memory layer for long-lived agent execution. It accepts OpenClaw-style events through a Go orchestrator, deterministically rebuilds state in a Rust runtime, writes checkpoints to 0G Storage, and anchors verification metadata on-chain so workflows can be replayed, resumed, and externally verified. The service also exposes run context, checkpoint metadata, hydrate, verify, and trace endpoints so planners and judges can read recovered memory and validate integrity directly.

---

## 4. What Problem It Solves

Most agent demos lose workflow state when the process exits, and even when data is persisted there is usually no clean replay / resume path and no verifiable link between off-chain execution state and on-chain proof. This project solves that by turning agent execution into a durable workflow primitive with read/hydrate/trace contracts instead of transient runtime memory.

---

## 5. Which 0G Components Are Used

### 0G Storage

Used for workflow checkpoint upload and download. This is the durable persistence layer for workflow memory.

### 0G Chain

Used to anchor workflow verification metadata, including workflow ID, step index, root hash, and CID hash, through the MemoryAnchor contract path.

---

## 6. Why This Fits Track 1

This project is infrastructure for OpenClaw-style agent workflows. It provides:

- OpenClaw-facing event ingest
- deterministic workflow checkpointing
- durable memory persistence
- replay and resume
- on-chain verification linkage

That makes it a direct fit for the Agentic Infrastructure & OpenClaw Lab track.

---

## 7. Core Features Implemented

- OpenClaw-style single ingest endpoint
- OpenClaw-style batch ingest endpoint with per-item results
- idempotent event ingestion by `eventId`
- deterministic checkpoint generation in Rust
- workflow replay
- workflow resume
- run context endpoint with richer OpenClaw metadata
- checkpoint metadata endpoint
- hydrate endpoint to recover from persisted state
- verify endpoint (`/v1/openclaw/runs/{id}/verify`, optional `/judge/verify`) to re-derive and compare checkpoint integrity
- trace endpoint that links steps, roles, tools, skills, and checkpoints
- 0G Storage checkpoint upload / download path
- 0G Chain anchor path
- service health/readiness endpoint with live dependency probing

---

## 8. Repository

**GitHub Repo:**  
`https://github.com/dongowu/0g-memory-hub`

---

## 9. Important Repository Paths

- `apps/orchestrator-go`
- `rust/memory-core`
- `contracts/MemoryAnchor.sol`
- `docs/demo/3min-judge-flow.md`
- `docs/evidence/2026-03-22-live-storage-chain-proof.md`
- `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`
- `docs/evidence/2026-03-23-live-http-readiness-proof.md`

---

## 10. Demo Video Description

The demo shows:

1. OpenClaw-style workflow ingestion
2. deterministic checkpoint generation in Rust
3. workflow replay / resume / hydrate after restart
4. 0G Storage checkpoint persistence path
5. checkpoint verify step (`/verify`) that re-derives and compares against Storage + MemoryAnchor-linked metadata
6. run trace continuity after verify

---

## 11. Mainnet / Explorer / Demo / Tweet Fields

Current verified Galileo testnet deployment:

- **Testnet contract address:** `0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- **0G Explorer link:** `https://chainscan-galileo.0g.ai/address/0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- **Anchor proof tx:** `https://chainscan-galileo.0g.ai/tx/0xa794dd7aedcf7b7c349005af620f29d8a36557c7b7973f91e358e31287fad1db`
- **Deployment proof doc:** `docs/evidence/2026-03-23-0g-testnet-memory-anchor-deployment-proof.md`

Still replace these placeholders before final submission if required:

- **Mainnet contract address:** `TODO`
- **Demo video link:** `TODO`
- **X / Twitter post link:** `TODO`

---

## 12. Judge-Facing 30-Second Summary

0G OpenClaw Memory Runtime gives OpenClaw-style agents durable memory on 0G. The Go service accepts workflow events, the Rust runtime deterministically rebuilds state and emits checkpoints, 0G Storage persists those checkpoints, and MemoryAnchor anchors verification metadata. We not only recover after restart; we re-derive the checkpoint and compare it with Storage/MemoryAnchor-linked proof, then show the full run trace.
