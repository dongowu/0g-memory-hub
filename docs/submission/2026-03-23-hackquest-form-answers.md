# HackQuest Form Answers (Copy-Paste Ready)

> Project: **0G OpenClaw Memory Runtime**
>
> Track: **Agentic Infrastructure & OpenClaw Lab**

Use this file when filling the HackQuest form. Short fields are grouped first, then longer textareas.

---

## A. Short fields

| HackQuest field | Paste this |
|---|---|
| Project name | `0G OpenClaw Memory Runtime` |
| Track | `Agentic Infrastructure & OpenClaw Lab` |
| One-sentence description (recommended) | `Durable OpenClaw workflow memory on 0G using Go orchestration, Rust checkpoints, 0G Storage persistence, and on-chain verification anchors.` |
| One-sentence description (short backup) | `Durable OpenClaw agent memory on 0G with Rust checkpoints, 0G Storage persistence, and on-chain workflow verification.` |
| GitHub repo | `https://github.com/dongowu/0g-memory-hub` |
| Galileo contract address | `0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9` |
| 0G Explorer link | `https://chainscan-galileo.0g.ai/address/0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9` |
| Anchor proof tx | `https://chainscan-galileo.0g.ai/tx/0xa794dd7aedcf7b7c349005af620f29d8a36557c7b7973f91e358e31287fad1db` |
| Deployment proof doc | `docs/evidence/2026-03-23-0g-testnet-memory-anchor-deployment-proof.md` |
| Demo video link | `TODO` |
| X / Twitter post link | `TODO` |
| Mainnet contract address (only if required) | `TODO` |
| Mainnet explorer link (only if required) | `TODO` |

---

## B. Long-form copy-paste answers

### 1. What does the project do?

0G OpenClaw Memory Runtime is a workflow memory layer for long-lived agent execution. It accepts OpenClaw-style events through a Go orchestrator, deterministically rebuilds state in a Rust runtime, writes checkpoints to 0G Storage, and anchors verification metadata on-chain so workflows can be replayed, resumed, and externally verified. The service also exposes run context, checkpoint metadata, hydrate, verify, and trace endpoints so planners, operators, and judges can read recovered memory and validate integrity directly.

### 2. What problem does it solve?

Most agent demos lose workflow state when the process exits, and even when data is persisted there is usually no clean replay / resume path and no verifiable link between off-chain execution state and on-chain proof. This project solves that by turning agent execution into a durable workflow primitive with read / hydrate / verify / trace contracts instead of transient runtime memory.

### 3. Which 0G components are used?

**0G Storage** is used for workflow checkpoint upload and download. This is the durable persistence layer for workflow memory.

**0G Chain** is used to anchor workflow verification metadata, including workflow ID, step index, root hash, and CID hash, through the `MemoryAnchor` contract path.

### 4. Why does this fit Track 1?

This project is infrastructure for OpenClaw-style agent workflows. It provides OpenClaw-facing event ingest, deterministic workflow checkpointing, durable memory persistence, replay / hydrate / resume, and on-chain verification linkage. That makes it a direct fit for **Agentic Infrastructure & OpenClaw Lab**.

### 5. Core features implemented

- OpenClaw-style single ingest endpoint
- OpenClaw-style batch ingest endpoint with per-item results
- Idempotent event ingestion by `eventId`
- Deterministic checkpoint generation in Rust
- Workflow replay
- Workflow resume
- Run context endpoint with richer OpenClaw metadata
- Checkpoint metadata endpoint
- Hydrate endpoint to recover from persisted state
- Verify endpoint (`/v1/openclaw/runs/{id}/verify`), judge console (`/judge/verify`), and CLI fallback (`workflow verify <run-id>`) to re-derive and compare checkpoint integrity
- Trace endpoint that links steps, roles, tools, skills, and checkpoints
- 0G Storage checkpoint upload / download path
- 0G Chain anchor path
- Service health / readiness endpoint with live dependency probing

### 6. Demo video description

The demo should show:

1. OpenClaw-style workflow ingestion
2. deterministic checkpoint generation in Rust
3. workflow replay / hydrate after restart
4. 0G Storage checkpoint persistence
5. checkpoint verify step (`/verify`) that re-derives and compares against Storage + MemoryAnchor-linked metadata
6. run trace continuity after verify

### 7. Judge-facing 30-second summary

0G OpenClaw Memory Runtime gives OpenClaw-style agents durable memory on 0G. The Go service accepts workflow events, the Rust runtime deterministically rebuilds state and emits checkpoints, 0G Storage persists those checkpoints, and MemoryAnchor anchors verification metadata. We not only recover after restart; we re-derive the checkpoint and compare it with Storage / MemoryAnchor-linked proof, then show the full run trace.

---

## C. Repository and judging paths

- `apps/orchestrator-go`
- `rust/memory-core`
- `contracts/MemoryAnchor.sol`
- `docs/demo/3min-judge-flow.md`
- `docs/demo/judge-checklist.md`
- `docs/evidence/2026-03-22-live-storage-chain-proof.md`
- `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`
- `docs/evidence/2026-03-23-live-http-readiness-proof.md`

---

## D. Final manual fields to fill before submit

- Demo video URL
- X / Twitter proof URL
- Mainnet address / explorer only if the final form explicitly requires mainnet rather than Galileo proof
