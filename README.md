# 0G OpenClaw Memory Runtime

[English](./README.md) | [简体中文](./README.zh-CN.md)

**A durable OpenClaw workflow memory layer on 0G that checkpoints agent execution in Rust, persists state to 0G Storage, and anchors verification metadata on-chain.**

Built for **0G APAC Hackathon — Track 1: Agentic Infrastructure & OpenClaw Lab**.

> **Core claim:** agent memory should survive crashes, resume cleanly, and be verifiable outside the model process.

---

## Submission snapshot

| Item | Value |
|---|---|
| Track | Agentic Infrastructure & OpenClaw Lab |
| Repo | `dongowu/0g-memory-hub` |
| Core stack | Go orchestrator + Rust runtime + Solidity anchor |
| 0G components used | 0G Storage + 0G Chain |
| Current proof | Galileo / 0g-testnet |
| Testnet contract | `0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9` |
| Explorer | `https://chainscan-galileo.0g.ai/address/0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9` |

---

## What this project does

Most agent demos lose workflow context when the process dies. This project turns workflow memory into a durable infra primitive:

- **Go orchestrator** ingests OpenClaw-style workflow events
- **Rust runtime** deterministically replays events and builds checkpoints
- **0G Storage** persists checkpoint blobs outside the process
- **0G Chain / MemoryAnchor** anchors `workflowId`, `stepIndex`, `rootHash`, and `cidHash`
- **Hydrate / verify / trace** proves that a run can be recovered and re-checked after restart

Judge-facing verification statement:

> We not only recover a run after restart, we re-derive the checkpoint and compare it against persisted 0G Storage state and MemoryAnchor-linked metadata.

---

## What is already implemented

- OpenClaw-style single-event ingest
- OpenClaw-style batch ingest
- Rich metadata preservation: `runId`, `sessionId`, `traceId`, `parentEventId`, `toolCallId`, `skillName`, `taskId`, `role`
- Idempotent ingest by `eventId`
- Deterministic Rust replay and checkpoint generation
- Persistent Rust runtime transport for long-lived service mode
- 0G Storage checkpoint upload / download path
- 0G Chain anchor path through `MemoryAnchor`
- `replay`, `resume`, `hydrate`, `verify`, `trace`, and readiness checks
- Judge verify console at `/judge/verify?runId={id}`

---

## Canonical judging paths

Use these paths for review and demo:

| Component | Path |
|---|---|
| Go orchestrator | `apps/orchestrator-go` |
| Rust runtime | `rust/memory-core` |
| Solidity contract | `contracts/MemoryAnchor.sol` |
| Quickstart | `QUICKSTART.md` |
| Demo docs | `docs/demo/` |
| Submission docs | `docs/submission/` |
| Evidence docs | `docs/evidence/` |

Legacy root paths (`main.go`, `cmd/`, `core/`, `sdk/`) are compatibility leftovers and are **not** the main hackathon judging path.

---

## Judge quick links

| What | Path |
|---|---|
| Final HackQuest copy | `docs/submission/2026-03-23-hackquest-final-copy.md` |
| Submission checklist | `docs/submission/2026-03-23-hackquest-submission-checklist.md` |
| 3-minute demo flow | `docs/demo/3min-judge-flow.md` |
| Judge checklist | `docs/demo/judge-checklist.md` |
| Live storage + chain proof | `docs/evidence/2026-03-22-live-storage-chain-proof.md` |
| Live workflow proof | `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md` |
| Live readiness proof | `docs/evidence/2026-03-23-live-http-readiness-proof.md` |
| Galileo deployment proof | `docs/evidence/2026-03-23-0g-testnet-memory-anchor-deployment-proof.md` |

---

## Architecture

```text
OpenClaw-style events
        |
        v
Go orchestrator
  - ingest / batch ingest
  - context / checkpoint / hydrate / verify / trace
        |
        v
Rust runtime
  - deterministic replay
  - checkpoint generation
  - root hash derivation
        |
        +--> 0G Storage
        |      - checkpoint persistence
        |
        +--> 0G Chain / MemoryAnchor
               - workflow anchor metadata
```

---

## Quick start

### Prerequisites

- Go **1.26.x**
- Rust stable
- Node.js **20 - 24**
- npm

### Fast local setup

```bash
npm install
cd rust/memory-core && cargo test --offline && cargo build --bin memory-core-rpc
cd ../../apps/orchestrator-go && go test ./...
```

Start the service:

```bash
export ORCH_RUNTIME_BINARY_PATH="$(pwd)/rust/memory-core/target/debug/memory-core-rpc"
cd apps/orchestrator-go
go run . serve
```

Check readiness:

```bash
curl http://127.0.0.1:8080/health
```

For full setup and manual commands, see `QUICKSTART.md`.

---

## Recommended demo path

For judges, the strongest story is:

1. **ingest** an OpenClaw-style run
2. show **checkpoint/latest**
3. stop and restart the service
4. **hydrate** the run
5. **verify** the restored checkpoint
6. show **trace**
7. close with explorer / evidence proof

Fast local smoke path:

```bash
./scripts/demo_verify_smoke.sh
```

Full narration and manual flow:

- `docs/demo/3min-judge-flow.md`
- `docs/demo/judge-checklist.md`

---

## HTTP surface

### Health and judge surface

- `GET /health`
- `GET /judge/verify?runId={id}`

### OpenClaw ingest

- `POST /v1/openclaw/ingest`
- `POST /v1/openclaw/ingest/batch`

### Workflow and run inspection

- `GET /v1/workflows/{id}`
- `POST /v1/workflows/{id}/resume`
- `GET /v1/workflows/{id}/replay`
- `GET /v1/openclaw/runs/{id}/context`
- `GET /v1/openclaw/runs/{id}/checkpoint/latest`
- `POST /v1/openclaw/runs/{id}/hydrate`
- `GET /v1/openclaw/runs/{id}/verify`
- `GET /v1/openclaw/runs/{id}/trace`

---

## Verification status

Current repository checks:

- `apps/orchestrator-go`: tests pass with **Go 1.26.0**
- `rust/memory-core`: `cargo test --offline` passes
- `contracts/MemoryAnchor.sol`: Hardhat tests pass

Recommended verification commands:

```bash
cd rust/memory-core && cargo test --offline
cd ../../apps/orchestrator-go && go test ./...
cd ../.. && npx hardhat test test/MemoryAnchor.js
```

---

## Current boundaries

- This is a **workflow runtime + verification layer**, not a full consumer AI product
- OpenClaw ingest is synchronous and currently backed by a local file store
- Live storage / chain behavior depends on reachable RPCs and funded credentials
- Current repo proof is **testnet / Galileo**, not mainnet
- Demo video link and X/Twitter submission link still need to be attached manually for final submission

---

## Related docs

- `QUICKSTART.md`
- `0G_INTEGRATION.md`
- `docs/architecture/2026-03-21-openclaw-memory-runtime-design.md`
- `docs/submission/2026-03-23-hackquest-form-answers.md`
