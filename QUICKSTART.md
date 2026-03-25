# Quickstart — 0G OpenClaw Memory Runtime

This quickstart is optimized for **HackQuest judges, demo recording, and submission prep**.

**Canonical hackathon path:**

- Go orchestrator: `apps/orchestrator-go`
- Rust runtime: `rust/memory-core`
- Anchor contract: `contracts/MemoryAnchor.sol`

If you only need the strongest judge-facing path, do **Section 1** first.

---

## Prerequisites

- Go **1.26.x** on `PATH`
- Rust stable
- Node.js **20 - 24** and `npm` for contract / deploy steps
- A **funded Galileo testnet private key** for the live HTTP ingest / verify demo

All commands below assume you start at the **repo root**.

---

## 1. Fastest judge path: smoke the full service flow

This is the fastest path that proves the main claim:

> ingest -> checkpoint -> hydrate -> verify -> trace

### 1.1 Build the Rust runtime and verify Go tests

```bash
export REPO_ROOT="$(pwd)"

pushd rust/memory-core
cargo test --offline
cargo build --bin memory-core-rpc
popd

pushd apps/orchestrator-go
go test ./...
popd
```

### 1.2 Export the live Galileo demo environment

```bash
export ORCH_RUNTIME_BINARY_PATH="$REPO_ROOT/rust/memory-core/target/debug/memory-core-rpc"
export ORCH_DATA_DIR="$REPO_ROOT/.orchestrator"
export ORCH_STORAGE_RPC_URL="https://indexer-storage-testnet-turbo.0g.ai"
export ORCH_CHAIN_RPC_URL="https://evmrpc-testnet.0g.ai"
export ORCH_CHAIN_CONTRACT_ADDRESS="0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9"
export ORCH_CHAIN_PRIVATE_KEY="0x..."
export ORCH_CHAIN_ID="16602"
export ORCH_HTTP_ADDR="127.0.0.1:8080"
```

> `ORCH_CHAIN_PRIVATE_KEY` must be a funded Galileo key. The current service path anchors workflow checkpoints on each live step, so the HTTP ingest / verify demo is **not** a fully offline flow.

### 1.3 Start the service

```bash
pushd apps/orchestrator-go
go run . serve
```

You should see:

```text
http_addr=127.0.0.1:8080
```

Leave this terminal running.

### 1.4 Run the smoke script from a second terminal

```bash
cd "$REPO_ROOT"
./scripts/demo_verify_smoke.sh
```

Optional: pause in the middle, restart the service, then continue the same run:

```bash
cd "$REPO_ROOT"
PAUSE_FOR_RESTART=1 ./scripts/demo_verify_smoke.sh
```

Success looks like this:

- `GET /health` returns `200`
- ingest succeeds
- `context`, `checkpoint/latest`, `hydrate`, `verify`, and `trace` all return `200`
- the script prints a judge console URL such as:
  - `http://127.0.0.1:8080/judge/verify?runId=<run-id>`

---

## 2. Manual judge walkthrough (same story, slower but clearer on video)

Use this when recording the demo or when a judge wants to see each step explicitly.

### 2.1 Health

```bash
curl http://127.0.0.1:8080/health
```

### 2.2 Ingest one OpenClaw-style run

```bash
curl -X POST http://127.0.0.1:8080/v1/openclaw/ingest/batch \
  -H 'Content-Type: application/json' \
  -d '{
    "events":[
      {
        "workflowId":"wf-judge-01",
        "runId":"run-judge-01",
        "sessionId":"session-judge-01",
        "traceId":"trace-judge-01",
        "eventId":"evt-plan-1",
        "eventType":"tool_call",
        "actor":"planner",
        "role":"planner",
        "toolCallId":"tool-search-1",
        "skillName":"memory_reader",
        "taskId":"task-judge-1",
        "payload":{"goal":"find BTC sentiment"}
      },
      {
        "workflowId":"wf-judge-01",
        "runId":"run-judge-01",
        "sessionId":"session-judge-01",
        "traceId":"trace-judge-01",
        "eventId":"evt-tool-1",
        "eventType":"tool_result",
        "actor":"worker",
        "role":"worker",
        "parentEventId":"evt-plan-1",
        "toolCallId":"tool-search-1",
        "taskId":"task-judge-1",
        "payload":{"ok":true,"summary":"sentiment mildly bullish"}
      }
    ]
  }'
```

### 2.3 Show the run is inspectable

```bash
curl http://127.0.0.1:8080/v1/openclaw/runs/run-judge-01/context
curl http://127.0.0.1:8080/v1/openclaw/runs/run-judge-01/checkpoint/latest
```

### 2.4 Restart, hydrate, verify, trace

Stop the service in terminal A with `Ctrl+C`, start it again with the same env, then run:

```bash
curl -X POST http://127.0.0.1:8080/v1/openclaw/runs/run-judge-01/hydrate
curl http://127.0.0.1:8080/v1/openclaw/runs/run-judge-01/verify
curl http://127.0.0.1:8080/v1/openclaw/runs/run-judge-01/trace
curl "http://127.0.0.1:8080/judge/verify?runId=run-judge-01"
```

CLI fallback for the same verify proof:

```bash
pushd apps/orchestrator-go
go run . workflow verify run-judge-01
popd
```

For the full spoken flow, use `docs/demo/3min-judge-flow.md`.

---

## 3. Live 0G setup notes

### 3.1 Need a Galileo key?

Install JavaScript dependencies once:

```bash
npm install
```

Generate a wallet:

```bash
npm run wallet:new

# or save it locally
npm run wallet:new:save
```

Fund the printed address from one of:

- `https://faucet.0g.ai`
- `https://cloud.google.com/application/web3/faucet/0g/galileo`

### 3.2 Use the current verified Galileo deployment

Current judge-facing deployment proof:

- Contract: `0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- Explorer: `https://chainscan-galileo.0g.ai/address/0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- Anchor tx: `https://chainscan-galileo.0g.ai/tx/0xa794dd7aedcf7b7c349005af620f29d8a36557c7b7973f91e358e31287fad1db`

If you deploy your own contract, replace `ORCH_CHAIN_CONTRACT_ADDRESS` with the new address.

### 3.3 CLI proof path

This is useful when you want a minimal live proof without the full HTTP demo.

```bash
pushd apps/orchestrator-go
go run . workflow start demo-live
go run . workflow step demo-live \
  --event-type tool_result \
  --actor openclaw \
  --payload '{"task":"fetch_price","ok":true}'
go run . workflow status demo-live
go run . workflow verify demo-live
popd
```

Expected fields after `workflow step`:

- `latest_root`
- `latest_cid`
- `latest_tx_hash`

---

## 4. Deploy / refresh MemoryAnchor proof

Use this path when you want fresh deploy evidence for the submission pack.

```bash
npm install
npm run preflight:testnet
npx hardhat compile
npx hardhat test test/MemoryAnchor.js
npm run deploy:proof
npm run evidence:testnet
```

Main outputs:

- `deployments/0g-testnet/MemoryAnchor.latest.json`
- `docs/evidence/2026-03-23-0g-testnet-memory-anchor-deployment-proof.md`

If you want the orchestrator to use the newly deployed contract immediately:

```bash
export ORCH_CHAIN_CONTRACT_ADDRESS="$(node -e 'const fs=require("fs"); const d=JSON.parse(fs.readFileSync("deployments/0g-testnet/MemoryAnchor.latest.json","utf8")); process.stdout.write(d.contractAddress)')"
```

---

## 5. Submission material pack

Start here when preparing HackQuest materials:

- `docs/submission/README.md`
- `docs/submission/2026-03-23-hackquest-form-answers.md`
- `docs/submission/2026-03-23-hackquest-final-copy.md`
- `docs/submission/2026-03-23-hackquest-submission-checklist.md`
- `docs/submission/2026-03-23-x-post-draft.md`

Demo and proof references:

- `docs/demo/3min-judge-flow.md`
- `docs/demo/judge-checklist.md`
- `docs/evidence/2026-03-22-live-storage-chain-proof.md`
- `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`
- `docs/evidence/2026-03-23-live-http-readiness-proof.md`
- `docs/evidence/2026-03-23-0g-testnet-memory-anchor-deployment-proof.md`

---

## 6. Offline-only sanity path (not a submission proof)

If you only want to sanity-check the local file store and CLI lifecycle without live 0G calls, use this reduced path:

```bash
export REPO_ROOT="$(pwd)"
export ORCH_DATA_DIR="$REPO_ROOT/.orchestrator-offline"

pushd apps/orchestrator-go
go run . workflow start demo-offline
go run . workflow status demo-offline
go run . workflow replay demo-offline
popd
```

This is useful for local inspection only. It does **not** prove the 0G Storage + MemoryAnchor path.

---

## 7. Troubleshooting

### `GET /health` returns `503`

Readiness is live-probed.

Common causes:

- `ORCH_RUNTIME_BINARY_PATH` is wrong
- `ORCH_STORAGE_RPC_URL` is unreachable
- `ORCH_CHAIN_PRIVATE_KEY` is missing or unfunded
- `ORCH_CHAIN_CONTRACT_ADDRESS` is wrong
- `ORCH_CHAIN_ID` does not match the RPC

### Ingest fails with anchor / signing errors

The current live service path anchors checkpoints as part of the workflow step. For the judge-facing HTTP flow, make sure all of these are set correctly:

- `ORCH_CHAIN_PRIVATE_KEY`
- `ORCH_CHAIN_CONTRACT_ADDRESS`
- `ORCH_CHAIN_RPC_URL`
- `ORCH_CHAIN_ID`

### Runtime binary not found

Rebuild the Rust binary and reset the path:

```bash
pushd rust/memory-core
cargo build --bin memory-core-rpc
popd

export ORCH_RUNTIME_BINARY_PATH="$REPO_ROOT/rust/memory-core/target/debug/memory-core-rpc"
```

### Storage readiness / upload fails

Check the indexer manually:

```bash
curl https://indexer-storage-testnet-turbo.0g.ai/node/status
```

### Duplicate-looking ingest events

Ingest is idempotent by `eventId`. Re-sending the same `eventId` is treated as a retry, not a new step.

### Hardhat / Node version warnings

Use Node.js **20 - 24** for the contract toolchain.
