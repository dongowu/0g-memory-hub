# 3-Minute Judge Demo Flow

This flow is optimized for the AI Infra + OpenClaw track and aligned to the current MVP code.

## Demo Objective

Show that workflow execution can be:

- stateful (workflow metadata)
- checkpointed (Rust core)
- persisted (0G Storage path)
- verifiable/anchorable (MemoryAnchor contract path)

## Timeline (<= 3 minutes)

### 0:00 - 0:30 Architecture context

Show:

- `apps/orchestrator-go`
- `rust/memory-core`
- `contracts/MemoryAnchor.sol`

Say:

- Go handles orchestration and 0G integration.
- Rust handles deterministic workflow state and checkpoints.
- Contract stores workflow-centric anchors.

### 0:30 - 1:15 Baseline run

Run:

```bash
cd apps/orchestrator-go
go run . workflow start judge-wf
go run . workflow status judge-wf
```

Explain:

- Workflow is created with persisted local metadata.

### 1:15 - 2:10 Checkpoint step run

Run:

```bash
go run . workflow step judge-wf \
  --event-type tool_result \
  --actor openclaw \
  --payload '{"task":"price_check","ok":true}'
```

Show output fields:

- `latest_step`
- `latest_root`
- `latest_cid`

Explain:

- Rust runtime computes checkpoint/root.
- Orchestrator uploads checkpoint blob to Storage path.

### 2:10 - 2:40 Replay

Run:

```bash
go run . workflow replay judge-wf
```

Explain:

- Replay gives judge-readable execution trace and checkpoint linkage.

### 2:40 - 3:00 Chain proof path

Show:

- `contracts/MemoryAnchor.sol`
- deployment script (`scripts/deploy.js`)
- if available, explorer tx/hash

Explain:

- `anchorCheckpoint(workflowId, stepIndex, rootHash, cidHash)` is the on-chain verification hook.

## Backup Mode (if live RPC unstable)

If RPC is down, still show:

- local workflow start/status/replay
- Rust RPC binary call:

```bash
printf '{"cmd":"init_workflow","workflow_id":"judge-wf","agent_id":"judge-agent"}\n' \
  | cargo run --quiet --bin memory-core-rpc
```

Be explicit that live storage/chain depends on endpoint availability.
