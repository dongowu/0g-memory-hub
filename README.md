# 0G OpenClaw Memory Runtime

**0G OpenClaw Memory Runtime** is a durable memory and verification layer for agent workflows. It accepts OpenClaw-style events, rebuilds deterministic workflow state in Rust, persists checkpoints through 0G Storage, and anchors verification metadata on-chain through 0G Chain.

For the **0G APAC Hackathon Track 1: Agentic Infrastructure & OpenClaw Lab**, the core claim of this project is simple:

> **Agents need memory that survives crashes, resumes cleanly, and can be verified outside the model process.**

This repository demonstrates that claim with:

- **OpenClaw-style ingest** for long-lived workflow execution
- **Deterministic Rust checkpoints** for replayable agent state
- **0G Storage persistence** for durable checkpoint blobs
- **0G Chain anchoring** for public verification metadata
- **Replay / resume / verify / readiness** for operator-facing workflow reliability

Every stored event keeps the richer OpenClaw semantics judges expect — `runId`, `sessionId`, `traceId`, `parentEventId`, `toolCallId`, `skillName`, `taskId`, and `role` — so the workflow trace can map directly back to what the agent planned, executed, or remembered.

Judge-facing verification claim:

> We not only recover the run after restart, we re-derive the checkpoint and compare it against 0G Storage and MemoryAnchor-linked metadata.

## Why this matters for 0G

- **0G Storage** gives the workflow a durable place to persist checkpoint state outside the running process.
- **0G Chain** gives the workflow a verifiable anchor for `workflowId`, `stepIndex`, `rootHash`, and `cidHash`.
- Together, they turn agent execution from an in-memory interaction into a **recoverable and inspectable infra primitive**.

## Judge-Facing Assets

- Final HackQuest submission copy:
  `docs/submission/2026-03-23-hackquest-final-copy.md`
- Submission checklist:
  `docs/submission/2026-03-23-hackquest-submission-checklist.md`
- 3-minute judge flow:
  `docs/demo/3min-judge-flow.md`

## Repository Context

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
OG_TESTNET_CHAIN_ID=16602
PRIVATE_KEY=0x...
```

### Need a fresh testnet wallet?

Generate one locally:

```bash
npm run wallet:new
```

Generate and also save it to `.wallets/` locally:

```bash
npm run wallet:new:save
```

Then fund the printed address from:

- https://faucet.0g.ai
- https://cloud.google.com/application/web3/faucet/0g/galileo

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
- `GET /v1/openclaw/runs/{id}/context` (run metadata + recent events)
- `GET /v1/openclaw/runs/{id}/checkpoint/latest` (latest checkpoint root/cid/tx)
- `POST /v1/openclaw/runs/{id}/hydrate` (resume from the persisted checkpoint)
- `GET /v1/openclaw/runs/{id}/verify` (re-derive + compare against persisted checkpoint metadata)
- `GET /v1/openclaw/runs/{id}/trace` (judge-friendly run timeline)
- `GET /judge/verify?runId={id}` (judge-facing verify console)

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
- `storage`: lightweight live probe against the configured 0G indexer `/node/status` endpoint, with bounded timeout and turbo indexer fallback for reachability checks
- `anchor`: optional lightweight live probe against the configured chain RPC using `eth_chainId` and `eth_blockNumber`

This means `/health` can now return `503` not only when a required dependency is missing, but also when it is configured yet currently unreachable.

### Contract

```bash
npm run wallet:new
npm run preflight:testnet
npx hardhat compile
npx hardhat test test/MemoryAnchor.js
npx hardhat run scripts/deploy.js --network 0g-testnet
npm run deploy:proof
npm run evidence:testnet
```

After deployment, point the orchestrator at the deployed anchor contract:

```bash
export ORCH_CHAIN_CONTRACT_ADDRESS=$(node -e "const fs=require('fs');const d=JSON.parse(fs.readFileSync('deployments/0g-testnet/MemoryAnchor.latest.json','utf8'));process.stdout.write(d.contractAddress)")
```

`scripts/deploy.js` now deploys `MemoryAnchor` by default for the judge path on Galileo.

When `RUN_ANCHOR_PROOF=1` (or `npm run deploy:proof`) is enabled, the deploy script also:

- submits one sample `anchorCheckpoint(...)` transaction,
- reads the checkpoint back from chain,
- prints explorer-ready transaction links when configured,
- writes deployment metadata to `deployments/<network>/MemoryAnchor.latest.json`.

`npm run preflight:testnet` prints the exact environment and orchestrator exports needed before a live 0G Galileo deployment.

`npm run evidence:testnet` converts the latest deployment artifact JSON into a judge-facing markdown file under `docs/evidence/`.

## Live Evidence

- Storage + chain proof record:
  `docs/evidence/2026-03-22-live-storage-chain-proof.md`
- Live orchestrator workflow proof:
  `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`
- Live HTTP readiness proof:
  `docs/evidence/2026-03-23-live-http-readiness-proof.md`
- Live Galileo deployment proof:
  `docs/evidence/2026-03-23-0g-testnet-memory-anchor-deployment-proof.md`
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
- Show `ingest -> checkpoint -> restart -> hydrate -> verify -> trace`
- Use `/v1/openclaw/runs/{id}/verify` to prove checkpoint re-derivation and Storage / MemoryAnchor linkage

See:

- `QUICKSTART.md`
- `docs/demo/3min-judge-flow.md`
- `docs/demo/judge-checklist.md`

## Current MVP Boundaries

- Runtime subprocess transport is persistent for long-lived server use.
- Cancelled HTTP/runtime requests force a fresh Rust child on the next call to avoid stale stdio reads.
- Storage and chain adapters are real client code, but live network behavior depends on your RPC endpoints.
- Contract deployment and explorer verification depend on your configured network and account.
- OpenClaw ingest is synchronous and currently uses a local file-backed workflow store.
