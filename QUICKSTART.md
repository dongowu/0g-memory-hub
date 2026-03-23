# Quickstart (OpenClaw Memory Runtime MVP)

This quickstart targets the new hackathon runtime:

- Go orchestrator: `apps/orchestrator-go`
- Rust core: `rust/memory-core`
- Workflow anchor contract: `contracts/MemoryAnchor.sol`

## 1. Build Rust runtime binary

```bash
cd rust/memory-core
cargo test
cargo build --bin memory-core-rpc
```

Set binary path for orchestrator:

```bash
export ORCH_RUNTIME_BINARY_PATH="$PWD/target/debug/memory-core-rpc"
```

## 2. Configure orchestrator env

```bash
export ORCH_DATA_DIR=.orchestrator
export ORCH_STORAGE_RPC_URL=https://indexer-storage-testnet-turbo.0g.ai
export ORCH_CHAIN_RPC_URL=https://evmrpc-testnet.0g.ai
export ORCH_CHAIN_CONTRACT_ADDRESS=0x0000000000000000000000000000000000000000
export ORCH_CHAIN_PRIVATE_KEY=0x...
export ORCH_CHAIN_ID=16602
export ORCH_HTTP_ADDR=127.0.0.1:8080
```

If you only want a local demo without live 0G calls, `ORCH_STORAGE_RPC_URL` can stay set but you should skip `workflow step`.

## 3. Run orchestrator checks

```bash
cd apps/orchestrator-go
/Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go test ./...
```

## 4. Local baseline flow (no live RPC required)

```bash
/Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go run . workflow start demo-wf
/Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go run . workflow status demo-wf
/Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go run . workflow replay demo-wf
```

## 5. Run the HTTP service for OpenClaw-style ingest

```bash
/Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go run . serve
```

In a second shell:

```bash
curl http://127.0.0.1:8080/health

curl -X POST http://127.0.0.1:8080/v1/openclaw/ingest \
  -H 'Content-Type: application/json' \
  -d '{"runId":"demo-http","eventId":"evt-1","eventType":"tool_result","actor":"worker","payload":{"ok":true}}'

curl -X POST http://127.0.0.1:8080/v1/openclaw/ingest/batch \
  -H 'Content-Type: application/json' \
  -d '{"events":[
    {"runId":"demo-http","eventId":"evt-2","eventType":"tool_call","actor":"planner","payload":{"tool":"search"}},
    {"runId":"demo-http","eventId":"evt-3","eventType":"tool_result","actor":"worker","payload":{"ok":true}}
  ]}'

curl http://127.0.0.1:8080/v1/workflows/demo-http
curl http://127.0.0.1:8080/v1/workflows/demo-http/replay
curl -X POST http://127.0.0.1:8080/v1/workflows/demo-http/resume
```

The HTTP path is retry-safe for duplicate `eventId` values and uses the persistent Rust runtime transport under the hood.

`/health` returns:

- `200` when required service dependencies are ready
- `503` when runtime or storage readiness fails

The response includes per-component readiness so you can tell whether the issue is runtime, storage, or optional anchor configuration.

## 6. Full step flow (requires current official storage integration)

```bash
/Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go run . workflow step demo-wf \
  --event-type tool_result \
  --actor openclaw \
  --payload '{"task":"fetch_price","ok":true}'
```

Expected output includes:

- `latest_step`
- `latest_root`
- `latest_cid`

## 7. Compile contract and deploy

```bash
npx hardhat compile
npx hardhat run scripts/deploy.js --network 0g-testnet
```

Optionally deploy legacy contract:

```bash
CONTRACT_NAME=MemoryChain npx hardhat run scripts/deploy.js --network 0g-testnet
```

## 8. Judge demo script

Use the helper script at repository root:

```bash
bash scripts/demo.sh
```

For full 0G attempt:

```bash
DEMO_ENABLE_0G=1 bash scripts/demo.sh
```

## 9. Live storage proof scripts

For the independently verified small-payload storage proof path:

```bash
PRIVATE_KEY=0x... node scripts/live_storage_flow_proof.cjs
OG_STORAGE_ROOT=<root> PRIVATE_KEY=0x... node scripts/anchor_storage_root.cjs
```

For the live Go orchestrator proof path, see:

- `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`

## 10. Common issues

### `workflow step` fails with runtime error

- Check `ORCH_RUNTIME_BINARY_PATH` points to `memory-core-rpc`.
- Verify binary is executable.

### `workflow step` fails with storage RPC error

- Check `ORCH_STORAGE_RPC_URL` points to a live 0G indexer endpoint.
- The current branch uses the official SDK path first, then falls back to a generalized direct upload path when the indexer root RPC is unhealthy.

### HTTP ingest returns duplicate-looking payloads

- Re-sending the same `eventId` is treated as an idempotent retry and will not create a second workflow step.
- If you want a new workflow step, send a fresh `eventId`.

### hardhat warning about Node version

- Use Node LTS for reliable local behavior.
