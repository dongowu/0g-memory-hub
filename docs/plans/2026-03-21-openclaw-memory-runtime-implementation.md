# 0G OpenClaw Memory Runtime Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a hackathon-ready AI Infra + OpenClaw project where workflow steps are checkpointed to 0G Storage, anchored on 0G Chain, and recoverable through replay/resume.

**Architecture:** Go owns orchestration, 0G integration, CLI/API, and OpenClaw adaptation. Rust owns deterministic workflow state, checkpoint generation, replay, and root calculation. Solidity owns workflow-centric checkpoint anchoring. Deliver the MVP as a narrow, demo-first vertical slice.

**Tech Stack:** Go, Cobra/Viper, Rust, Solidity, Hardhat, JSON subprocess RPC, 0G Storage RPC, 0G Chain RPC

---

## Product Scope

### Must Have
- workflow run creation
- OpenClaw event adapter
- Rust checkpoint engine
- 0G Storage checkpoint persistence
- workflow-centric chain anchor
- replay + resume
- demo script / CLI
- clean README and submission docs

### Should Have
- HTTP API for viewer/demo
- batched anchoring or async anchor queue
- summary compaction in Rust core

### Won't Have for MVP
- sealed inference
- X402 payments
- consumer-grade UI
- multi-tenant auth

---

## Dependency Graph

```text
Track 0 Repo Cleanup ─┬─> Track 1 Contract & Chain
                      ├─> Track 2 Rust Core
                      ├─> Track 3 Go Orchestrator Skeleton
                      └─> Track 6 Docs & Submission

Track 2 Rust Core ───────┐
                         ├─> Track 4 Go↔Rust Integration
Track 3 Go Skeleton ─────┘

Track 1 Contract & Chain ─┐
Track 4 Integration ──────┼─> Track 5 End-to-End Workflow MVP
Track 0 Cleanup ──────────┘

Track 5 MVP ──────────────> Track 6 Docs & Demo Assets
```

Critical path:
1. repo cleanup
2. Rust core MVP
3. Go orchestrator skeleton
4. Go↔Rust integration
5. contract + chain anchor flow
6. end-to-end demo

---

## Agent Workstream Model

Use a **supervisor + workers** approach.

### Supervisor
Owns:
- sequencing
- interface definitions
- integration review
- keeping docs aligned

### Worker A — Repo Foundation
Owns file hygiene and repo structure.

### Worker B — Rust Core
Owns deterministic workflow state, checkpointing, replay.

### Worker C — Go Orchestrator
Owns CLI/API, OpenClaw adapter, metadata store, replay commands.

### Worker D — 0G Chain + Contract
Owns contract and Go chain adapter updates.

### Worker E — Docs + Demo
Owns README, architecture, demo flow, submission artifacts.

---

## Track 0: Repository Cleanup and Baseline Stabilization

**Objective**
Make the repository buildable and ready for parallel work. Remove leftover mismatches, isolate non-Go assets, and define the new target structure.

**Owned Files**
- `go.mod`
- `main.go`
- `cmd/**`
- `scripts/**`
- `Makefile`
- `README.md`
- `docs/**`
- root-level misplaced files

**Requirements**
- fix current `go test ./...` failure
- remove or relocate files that break Go package discovery
- document target repo layout
- ensure Go build baseline is green

**Acceptance Criteria**
- `go build ./...` exits 0
- `go test ./...` exits 0 or only skips expected packages cleanly
- no Solidity source masquerading as `.go` file
- docs reflect Go+Rust target architecture

**Out of Scope**
- implementing workflow runtime logic

---

## Track 1: Workflow-Centric Contract and Chain Integration

**Objective**
Replace wallet-centric memory anchoring with workflow-centric checkpoint anchoring.

**Owned Files**
- `contracts/MemoryAnchor.sol` (new)
- `contracts/MemoryChain.sol` (read-only unless deprecating)
- `scripts/deploy.js`
- `hardhat.config.js`
- `apps/orchestrator-go/internal/ogchain/**` or current `chain/**`
- contract ABI artifacts used by Go

**Requirements**
- define `anchorCheckpoint(workflowId, stepIndex, rootHash, cid)`
- query latest checkpoint and history by workflow id
- update Go chain adapter to call new contract
- expose tx hash and explorer link

**Acceptance Criteria**
- contract compiles with Hardhat
- Go adapter can anchor and query a checkpoint
- transaction metadata is available to demo layer

**Out of Scope**
- advanced permissions or upgradability

---

## Track 2: Rust Memory Core MVP

**Objective**
Build a deterministic workflow state engine that can initialize a workflow, append step events, build checkpoints, and replay state.

**Owned Files**
- `rust/memory-core/Cargo.toml`
- `rust/memory-core/src/lib.rs`
- `rust/memory-core/src/workflow_state.rs`
- `rust/memory-core/src/event_log.rs`
- `rust/memory-core/src/checkpoint.rs`
- `rust/memory-core/src/merkle.rs`
- `rust/memory-core/src/replay.rs`
- `rust/memory-core/src/rpc.rs`
- `rust/memory-core/tests/**`

**Requirements**
- `init_workflow`
- `append_event`
- `build_checkpoint`
- `resume_from_checkpoint`
- `replay_workflow`
- stable root hash for same ordered input

**Acceptance Criteria**
- Rust tests cover checkpoint determinism and replay correctness
- same event sequence => same root
- replay reconstructs step state accurately
- subprocess JSON API works locally

**Out of Scope**
- vector search
- complex summarization
- encrypted payloads

---

## Track 3: Go Orchestrator Skeleton

**Objective**
Create the Go application shell that will host CLI commands, local metadata, and OpenClaw adapter interfaces.

**Owned Files**
- `apps/orchestrator-go/go.mod` or current root `go.mod` if incremental
- `apps/orchestrator-go/main.go` or current `main.go`
- `apps/orchestrator-go/cmd/**`
- `apps/orchestrator-go/internal/config/**`
- `apps/orchestrator-go/internal/workflow/**`
- `apps/orchestrator-go/internal/openclaw/**`
- `apps/orchestrator-go/internal/server/**`
- shared types under `pkg/types/**`

**Requirements**
- commands:
  - `workflow start`
  - `workflow step`
  - `workflow resume`
  - `workflow replay`
  - `workflow status`
- local run metadata persistence
- adapter interface for OpenClaw event input

**Acceptance Criteria**
- CLI parses all intended commands
- metadata store records workflow id / status / latest checkpoint
- no 0G or Rust dependency required for skeleton unit tests

**Out of Scope**
- full integration with Rust core

---

## Track 4: Go ↔ Rust Integration Layer

**Objective**
Connect Go orchestrator to Rust memory-core via a narrow protocol.

**Owned Files**
- `apps/orchestrator-go/internal/workflow/runtime_client.go`
- `apps/orchestrator-go/internal/workflow/runtime_protocol.go`
- `apps/orchestrator-go/internal/workflow/runtime_client_test.go`
- `rust/memory-core/src/rpc.rs`
- integration fixtures under `examples/sample-workflows/**`

**Requirements**
- subprocess lifecycle management
- JSON request/response protocol
- typed mapping for init/append/checkpoint/replay/resume
- graceful error handling and timeouts

**Acceptance Criteria**
- Go can start Rust process and receive structured responses
- integration test covers append_event -> build_checkpoint
- protocol errors are surfaced cleanly

**Out of Scope**
- gRPC migration

---

## Track 5: End-to-End 0G Workflow MVP

**Objective**
Wire Rust checkpoint generation to real 0G Storage persistence and chain anchoring.

**Owned Files**
- Go workflow service files
- `storage/**` or `internal/ogstorage/**`
- `chain/**` or `internal/ogchain/**`
- local metadata / checkpoint persistence files
- `scripts/demo.sh`
- demo fixtures in `examples/sample-workflows/**`

**Requirements**
- on each committed step:
  - build checkpoint via Rust
  - upload checkpoint blob to 0G Storage
  - anchor latest checkpoint to chain
  - persist local workflow metadata
- support `resume` from latest checkpoint CID
- support `replay` output for demo

**Acceptance Criteria**
- demo run completes start → step → anchor → resume → replay
- output includes workflow id, cid, root, tx hash
- latest checkpoint can be fetched and replayed

**Out of Scope**
- heavy concurrency tuning

---

## Track 6: README, Demo Assets, and Submission Packaging

**Objective**
Make the project legible, reproducible, and scoreable for judges.

**Owned Files**
- `README.md`
- `0G_INTEGRATION.md`
- `GITHUB_SUBMISSION.md`
- `QUICKSTART.md`
- `docs/architecture/**`
- `docs/demo/**`
- `scripts/demo.sh`

**Requirements**
- one-line pitch aligned to track
- architecture diagram
- exact 0G components used and why
- exact local reproduction steps
- testnet/mainnet notes
- demo recording script / checklist

**Acceptance Criteria**
- judge can follow README to reproduce core flow
- docs clearly show Storage + Chain usage
- demo script matches product narrative

**Out of Scope**
- marketing site

---

## Recommended Parallel Agent Execution Order

### Phase A — unblock foundation
1. Worker A: Track 0 Repo Cleanup
2. Worker E: Track 6 Docs skeleton (parallel, low-risk)

### Phase B — parallel core build
3. Worker B: Track 2 Rust Core
4. Worker C: Track 3 Go Skeleton
5. Worker D: Track 1 Contract & Chain

### Phase C — integration
6. Worker C or dedicated integrator: Track 4 Go↔Rust Integration
7. Worker C + D: Track 5 E2E MVP

### Phase D — polish
8. Worker E: Final demo docs, scripts, submission package

---

## Suggested Agent Task Cards

### Agent Task Card A1 — Repo Cleanup
**Objective:** Fix repository hygiene and green the baseline build/test.
**Owned Files:** repo root, `scripts/**`, `Makefile`, docs, mislocated files.
**Acceptance:** `go build ./...` and `go test ./...` are green.

### Agent Task Card B1 — Rust Workflow State
**Objective:** Implement workflow state + event log + checkpoint structs.
**Owned Files:** `rust/memory-core/src/workflow_state.rs`, `event_log.rs`, `checkpoint.rs`.
**Acceptance:** Rust unit tests for append and serialize pass.

### Agent Task Card B2 — Rust Replay & Root
**Objective:** Implement root hashing + replay logic.
**Owned Files:** `rust/memory-core/src/merkle.rs`, `replay.rs`, tests.
**Acceptance:** deterministic root and replay tests pass.

### Agent Task Card C1 — Go CLI Skeleton
**Objective:** Add workflow subcommands and local metadata model.
**Owned Files:** Go CLI command tree and workflow service files.
**Acceptance:** CLI commands compile and metadata tests pass.

### Agent Task Card C2 — OpenClaw Adapter
**Objective:** Define the event adapter interface and sample adapter.
**Owned Files:** `internal/openclaw/**`, `pkg/types/**`.
**Acceptance:** sample workflow emits normalized step events.

### Agent Task Card D1 — MemoryAnchor Contract
**Objective:** Create workflow-centric anchor contract and deployment script.
**Owned Files:** `contracts/MemoryAnchor.sol`, `scripts/deploy.js`.
**Acceptance:** Hardhat compile passes; deployment script references new contract.

### Agent Task Card D2 — Go Chain Adapter Update
**Objective:** Make Go able to anchor/query workflow checkpoints.
**Owned Files:** chain adapter files only.
**Acceptance:** mocked or real integration test returns latest checkpoint metadata.

### Agent Task Card I1 — Go↔Rust Runtime Client
**Objective:** Implement JSON subprocess protocol.
**Owned Files:** runtime client/protocol files.
**Acceptance:** Go integration test exercises Rust commands end-to-end locally.

### Agent Task Card E1 — Demo & Docs
**Objective:** Produce hackathon-grade README, quickstart, and demo script.
**Owned Files:** docs and demo files only.
**Acceptance:** judge can reproduce core flow from docs.

---

## Immediate First 3 Tasks

### Task 1: Baseline Cleanup
**Files:**
- Modify: `scripts/**`, `Makefile`, `README.md`, misplaced source files
- Test: current Go packages

**Step 1: Identify files breaking `go test ./...`**
Run: `go test ./...`
Expected: identify failing package/file list

**Step 2: Remove or relocate invalid Go package assets**
Example target: move non-Go assets out of `scripts/` package path or rename appropriately

**Step 3: Re-run baseline checks**
Run: `go build ./... && go test ./...`
Expected: both exit 0

### Task 2: Scaffold Rust Core
**Files:**
- Create: `rust/memory-core/Cargo.toml`
- Create: `rust/memory-core/src/lib.rs`
- Create: `rust/memory-core/src/workflow_state.rs`
- Create: `rust/memory-core/src/event_log.rs`
- Create: `rust/memory-core/src/checkpoint.rs`
- Test: `rust/memory-core/tests/workflow_state.rs`

**Step 1: Write failing Rust tests for init + append_event**
Run: `cargo test -p memory-core` (inside Rust workspace)
Expected: FAIL because implementation missing

**Step 2: Implement minimal structs and methods**
Expected: init workflow and append ordered events

**Step 3: Re-run Rust tests**
Expected: PASS

### Task 3: Create Go Workflow CLI Skeleton
**Files:**
- Create/Modify: Go command files for `workflow start|step|resume|replay|status`
- Create: workflow service + local metadata store
- Test: CLI/service unit tests

**Step 1: Write failing tests for metadata persistence**
Run: `go test ./... -run Workflow`
Expected: FAIL initially

**Step 2: Implement minimal workflow metadata model**
Expected: store workflow_id, status, latest checkpoint refs

**Step 3: Add CLI command wiring**
Expected: help text and command routing compile cleanly

---

## Supervisor Rules for Agent Delegation

- Give each agent exclusive file ownership where possible
- Do not let two agents modify the same Go package simultaneously
- Integrate only after each worker provides verification evidence
- Require every worker to report:
  - files changed
  - commands run
  - exact output summary
  - remaining risks

## Done Definition

The MVP is complete when all of the following are true:
- workflow steps can be ingested through Go
- Rust builds deterministic checkpoints
- checkpoints are stored in 0G Storage
- latest checkpoint is anchored on chain
- workflow can resume after interruption
- workflow can replay for demo
- README + demo assets are judge-ready
