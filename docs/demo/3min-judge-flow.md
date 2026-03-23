# 3-Minute Judge Demo Flow

This flow is optimized for the **0G APAC Hackathon — Agentic Infrastructure & OpenClaw Lab** track and aligned to the current MVP code.

## Core Message

Do **not** pitch this as “just another workflow backend.”

Pitch it as:

> **A durable memory and verification layer for OpenClaw-style agent workflows on 0G.**

The judge should leave with exactly three ideas:

1. Workflow events enter through an OpenClaw-facing interface.
2. Execution state becomes a deterministic checkpoint, not transient process memory.
3. The checkpoint is persisted on 0G and can be replayed, resumed, and verified.

## Demo Objective

Show that workflow execution can be:

- stateful (workflow metadata)
- checkpointed (Rust core)
- persisted (0G Storage path)
- verifiable/anchorable (MemoryAnchor contract path)

## 20-Second Opening Script

Use this wording or stay close to it:

> “This project gives OpenClaw-style agent workflows durable memory on 0G. The Go service accepts workflow events, the Rust runtime deterministically rebuilds state and emits checkpoints, 0G Storage persists those checkpoints, and the chain anchor path makes the execution externally verifiable.”

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
- The point is durable and inspectable agent memory, not just local runtime state.
 - Judges can read the run context, checkpoint metadata, hydrate back into memory, and inspect the trace after each step.

### 0:30 - 1:15 Baseline run

Run:

```bash
cd apps/orchestrator-go
go run . workflow start judge-wf
go run . workflow status judge-wf
```

Explain:

- Workflow is created with persisted local metadata.
- This is the workflow identity that later binds storage and chain proof.

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
- This is where the workflow stops being “just process memory.”

### 2:10 - 2:40 Replay

Run:

```bash
go run . workflow replay judge-wf
```

Explain:

- Replay gives judge-readable execution trace and checkpoint linkage.
- If the process dies, the workflow can still be recovered from persisted state.

### 2:40 - 2:55 Read + Hydrate + Trace

Run:

```bash
curl http://127.0.0.1:8080/v1/openclaw/runs/judge-wf/context
curl http://127.0.0.1:8080/v1/openclaw/runs/judge-wf/checkpoint/latest
curl -X POST http://127.0.0.1:8080/v1/openclaw/runs/judge-wf/hydrate
curl http://127.0.0.1:8080/v1/openclaw/runs/judge-wf/trace
```

Explain:

- `context` returns run metadata plus the last few events with their richer OpenClaw fields.
- `checkpoint/latest` shows the root/cid/tx for the persisted checkpoint.
- `hydrate` demonstrates how the run can recover state and continue.
- `trace` presents the ordered run timeline with event IDs, roles, skills, and tool calls for judges to follow.

### 2:55 - 3:00 Chain proof path

Show:

- `contracts/MemoryAnchor.sol`
- deployment script (`scripts/deploy.js`)
- if available, explorer tx/hash

Explain:

- `anchorCheckpoint(workflowId, stepIndex, rootHash, cidHash)` is the on-chain verification hook.
- This binds off-chain checkpoint state to a public verification path.

## Strong Closing Line

Close with:

> “So the value is not only that an agent can act, but that its workflow memory can survive, be resumed, and be externally verified on 0G.”

## Judge Q&A Short Answers

### “Why is this a fit for Track 1?”

Because it is infrastructure for OpenClaw-style agents: ingest, deterministic execution memory, replay, resume, and durable persistence.

### “Why 0G instead of normal storage?”

Because the project is about durable and inspectable agent memory. 0G Storage gives the checkpoint persistence layer, and the chain anchor path adds verification metadata that can be checked outside the process.

### “What is the core technical novelty?”

The system separates orchestration, deterministic checkpoint generation, durable persistence, and verification into a workflow memory stack instead of treating agent state as ephemeral runtime state.

### “What happens if the process crashes?”

The workflow metadata remains, checkpoints can be downloaded again, and the workflow can be replayed or resumed from persisted state.

## Backup Mode (if live RPC unstable)

If RPC is down, still show:

- local workflow start/status/replay
- Rust RPC binary call:

```bash
printf '{"cmd":"init_workflow","workflow_id":"judge-wf","agent_id":"judge-agent"}\n' \
  | cargo run --quiet --bin memory-core-rpc
```

Be explicit that live storage/chain depends on endpoint availability.

## What to Avoid Saying

- Don’t describe it as only “a CRUD API for workflows.”
- Don’t spend the whole demo on endpoints without restating the agent-memory problem.
- Don’t lead with implementation detail before stating the infra value.
