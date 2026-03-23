# 0G OpenClaw Memory Runtime

This repository now contains two tracks:

- Legacy root implementation (`main.go`, `cmd/`, `core/`, `sdk/`) kept for compatibility.
- New hackathon track implementation for AI Infra + OpenClaw in:
  - `apps/orchestrator-go`
  - `rust/memory-core`
  - `contracts/MemoryAnchor.sol`

The hackathon target is a workflow runtime where each step can be checkpointed, persisted to 0G Storage, and anchored on 0G Chain.

## Architecture (Hackathon Track)

### Go orchestrator (`apps/orchestrator-go`)

- CLI commands:
  - `serve`
  - `workflow start`
  - `workflow step`
  - `workflow status`
  - `workflow replay`
  - `workflow resume`
- Responsibilities:
  - Expose an OpenClaw-facing HTTP API
  - Accept normalized OpenClaw-like step events
  - Support single-event and batched ingest
  - Keep ingest idempotent by `eventId`
  - Execute workflow creation + step append atomically inside the service
  - Call Rust runtime over persistent stdio JSON transport
  - Use HTTP timeouts and graceful shutdown for long-running service mode
  - Upload checkpoint blobs through 0G Storage adapter
  - Anchor checkpoint metadata through MemoryAnchor chain adapter
  - Persist local workflow metadata

### Rust core (`rust/memory-core`)

- Deterministic workflow state machine
- Event append / replay
- Checkpoint and root hash generation
- Stdio JSON RPC binary: `memory-core-rpc`

### Solidity contract (`contracts/MemoryAnchor.sol`)

- Workflow-centric anchoring by `workflowId`
- Stores latest checkpoint and full history
- Anchor fields include `stepIndex`, `rootHash`, `cidHash`, `timestamp`, `submitter`

## Repository Layout (Relevant Paths)

```text
apps/orchestrator-go/
  cmd/
  internal/config/
  internal/openclaw/
  internal/workflow/
  internal/ogstorage/
  internal/ogchain/
rust/memory-core/
  src/
  tests/
contracts/
  MemoryAnchor.sol
scripts/
  deploy.js
  demo.sh
docs/demo/
```

## Environment

### Orchestrator env vars

```bash
ORCH_DATA_DIR=.orchestrator
ORCH_RUNTIME_BINARY_PATH=memory-core-rpc
ORCH_STORAGE_RPC_URL=https://indexer-storage-testnet-turbo.0g.ai
ORCH_CHAIN_RPC_URL=https://evmrpc-testnet.0g.ai
ORCH_CHAIN_CONTRACT_ADDRESS=0x0000000000000000000000000000000000000000
ORCH_CHAIN_PRIVATE_KEY=0x...
ORCH_CHAIN_ID=16602
ORCH_HTTP_ADDR=127.0.0.1:8080
```

### Hardhat env vars

```bash
OG_CHAIN_RPC=https://evmrpc-testnet.0g.ai
PRIVATE_KEY=0x...
```

## Build and Test

### Rust core

```bash
cd rust/memory-core
cargo test
cargo run --bin memory-core-rpc
```

### Go orchestrator

```bash
cd apps/orchestrator-go
/Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go test ./...
/Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go run . serve
/Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go run . workflow start demo-wf
```

> Note: on March 22, 2026, live probing showed the standard indexer root RPC was unstable while turbo REST endpoints were healthy. The orchestrator keeps the official SDK path first and now includes a generalized direct fallback path for checkpoint uploads when the root RPC path is unhealthy.

## OpenClaw HTTP API

The orchestrator can now run as a long-lived service:

```bash
cd apps/orchestrator-go
/Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go run . serve
```

Available endpoints:

- `GET /health`
- `POST /v1/openclaw/ingest`
- `POST /v1/openclaw/ingest/batch`
- `GET /v1/workflows/{id}`
- `POST /v1/workflows/{id}/resume`
- `GET /v1/workflows/{id}/replay`

Example single ingest:

```bash
curl -X POST http://127.0.0.1:8080/v1/openclaw/ingest \
  -H 'Content-Type: application/json' \
  -d '{"runId":"demo-wf","eventId":"evt-1","eventType":"tool_result","actor":"worker","payload":{"ok":true}}'
```

Example batch ingest:

```bash
curl -X POST http://127.0.0.1:8080/v1/openclaw/ingest/batch \
  -H 'Content-Type: application/json' \
  -d '{"events":[
    {"runId":"demo-wf","eventId":"evt-1","eventType":"tool_call","actor":"planner","payload":{"tool":"search"}},
    {"runId":"demo-wf","eventId":"evt-2","eventType":"tool_result","actor":"worker","payload":{"ok":true}}
  ]}'
```

Duplicate `eventId` submissions are treated as idempotent retries and will not append duplicate workflow steps.

`GET /health` now returns a structured readiness report:

- `200 OK` when required components are ready
- `503 Service Unavailable` when a required component is missing or unhealthy

Current readiness behavior:

- `runtime`: active probe through the Rust stdio runtime client
- `storage`: configuration-level readiness for upload-capable 0G storage settings
- `anchor`: configuration-level readiness for chain anchor settings, reported as optional in the current MVP

### Contract

```bash
npx hardhat compile
```

## Live Evidence

- Storage + chain proof record:
  `docs/evidence/2026-03-22-live-storage-chain-proof.md`
- Live orchestrator workflow proof:
  `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`
- Live HTTP readiness proof:
  `docs/evidence/2026-03-23-live-http-readiness-proof.md`
- Reproduction scripts:
  - `node scripts/live_storage_flow_proof.cjs`
  - `OG_STORAGE_ROOT=<root> node scripts/anchor_storage_root.cjs`

## Demo Paths

### Local non-RPC demo (always available)

- Start workflow
- Show status
- Replay local metadata trace
- Or run the HTTP API and ingest local OpenClaw events

This does not require live 0G RPC.

### Full 0G demo (requires reachable RPC and contract)

- Build `memory-core-rpc` and expose it via `ORCH_RUNTIME_BINARY_PATH`
- Call `workflow step` to create checkpoint and upload to Storage
- Anchor checkpoint using chain client path
- Show tx hash / status / replay

See:

- `QUICKSTART.md`
- `docs/demo/3min-judge-flow.md`
- `scripts/demo.sh`

## Current MVP Boundaries

- Runtime subprocess transport is persistent for long-lived server use.
- Cancelled HTTP/runtime requests force a fresh Rust child on the next call to avoid stale stdio reads.
- Storage and chain adapters are real client code, but live network behavior depends on your RPC endpoints.
- Contract deployment and explorer verification depend on your configured network and account.
- OpenClaw ingest is synchronous and currently uses a local file-backed workflow store.
