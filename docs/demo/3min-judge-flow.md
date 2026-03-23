# 3-Minute Judge Demo Flow

This flow is optimized for the **0G APAC Hackathon — Agentic Infrastructure & OpenClaw Lab** track and should be recorded as a **story**, not an endpoint tour.

## Recommended Demo Style

### Option A — Crash / Recover / Verify (**Recommended**)

This is the strongest judge-facing path because it proves the core thesis:

> **An OpenClaw-style agent run can survive process loss, recover from durable memory, and stay externally verifiable on 0G.**

### Option B — API Capability Tour

Safer and easier, but weaker. Judges see features, not urgency.

### Option C — Storage / Contract Proof Tour

Useful as supporting evidence, but too infra-heavy if used alone.

**Recommendation:** record **Option A**, then use storage / explorer proof as the final 20-second close.

---

## Core Message

Do **not** pitch this as “a workflow backend” or “a set of APIs.”

Pitch it as:

> **A durable memory runtime for OpenClaw-style agents on 0G: ingest events, checkpoint deterministically, recover after failure, and verify the result.**

The judge should leave with exactly four ideas:

1. OpenClaw-style events enter through a real service interface.
2. Rust turns those events into deterministic checkpoints.
3. 0G Storage + chain anchor make the memory durable and inspectable.
4. Even after a restart, the run can be hydrated and traced back.

---

## Demo Objective

In under 3 minutes, prove all four:

- **agent context exists**
- **checkpoint state is persisted**
- **restart does not lose memory**
- **the run has a public verification path**

---

## 20-Second Opening Script

Use this wording or stay very close:

> “Most agent demos lose memory when the process dies. We built a durable memory layer for OpenClaw-style workflows on 0G. The Go service ingests workflow events, the Rust runtime deterministically rebuilds state and emits checkpoints, 0G Storage persists those checkpoints, and the chain path anchors verification metadata so the run can be recovered and inspected.”

---

## Recording Setup

Use **two terminals** and optionally one browser tab:

- **Terminal A:** `go run . serve`
- **Terminal B:** `curl` commands
- **Browser tab (optional):** explorer or evidence doc

Pre-build before recording so the video stays fast:

```bash
cd rust/memory-core
cargo build --bin memory-core-rpc

cd ../../apps/orchestrator-go
go test ./...
export ORCH_RUNTIME_BINARY_PATH="../../rust/memory-core/target/debug/memory-core-rpc"
go run . serve
```

---

## Timeline (<= 3 minutes)

### 0:00 - 0:20 Problem + architecture

Show repository paths briefly:

- `apps/orchestrator-go`
- `rust/memory-core`
- `contracts/MemoryAnchor.sol`

Say:

- Go is the orchestration and 0G integration layer.
- Rust is the deterministic checkpoint engine.
- MemoryAnchor is the public verification hook.
- The point is not “one more agent API,” but **memory that survives failure**.

### 0:20 - 0:35 Show service readiness

Run:

```bash
curl http://127.0.0.1:8080/health
```

Say:

- The service probes runtime, storage, and optional anchor readiness.
- This is a long-running infra component, not a one-shot script.

### 0:35 - 1:20 Ingest a real OpenClaw-style run

Run:

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

Say:

- This is richer than plain event logging.
- We preserve run/session/trace/tool/task metadata so the workflow remains intelligible.
- The response should show `latestStep`, `latestRoot`, and ideally `latestCid` / `latestTxHash`.

### 1:20 - 1:50 Show the memory is now inspectable

Run:

```bash
curl http://127.0.0.1:8080/v1/openclaw/runs/run-judge-01/context
curl http://127.0.0.1:8080/v1/openclaw/runs/run-judge-01/checkpoint/latest
```

Say:

- `context` shows the recovered run identity plus recent OpenClaw events.
- `checkpoint/latest` shows the checkpoint root and storage / anchor linkage.
- This is where the run stops being “process-local memory.”

### 1:50 - 2:20 Simulate failure and restart

Do this live in Terminal A:

- stop the server with `Ctrl+C`
- restart it with the same `go run . serve`

While restarting, say:

> “Now I’m simulating the exact failure mode most agent demos can’t handle: the process goes away, but the run should still be recoverable.”

### 2:20 - 2:45 Hydrate the run after restart

Run:

```bash
curl -X POST http://127.0.0.1:8080/v1/openclaw/runs/run-judge-01/hydrate
curl http://127.0.0.1:8080/v1/openclaw/runs/run-judge-01/trace
```

Say:

- `hydrate` rebuilds the run from persisted checkpoint state.
- `trace` proves the ordered execution history is still available after restart.
- This is the core product claim: **memory survives process loss**.

### 2:45 - 3:00 Show verification proof

Show one of:

- `latestTxHash` from the API response
- explorer page
- `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`

Say:

- Storage is the durable memory layer.
- MemoryAnchor is the public verification path.
- So the run is not only recoverable, but externally inspectable.

---

## Strong Closing Line

Close with:

> “We’re not just helping an agent act. We’re making its workflow memory durable, recoverable, and verifiable on 0G.”

---

## Judge Q&A Short Answers

### “Why is this a fit for Track 1?”

Because this is OpenClaw-style agent infrastructure: event ingest, deterministic memory, recovery, traceability, and durable persistence.

### “What is the actual wow moment here?”

The wow moment is that after the service restarts, the same run can still be hydrated and traced back using persisted checkpoint state rather than in-memory context.

### “Why 0G instead of ordinary storage?”

Because this project is about durable and inspectable agent memory. 0G Storage persists the checkpoint, and the chain path adds a verification surface outside the agent process.

### “What happens if the process crashes?”

The process can restart, the checkpoint can be reloaded, and the run can be hydrated and traced again. That is the product claim the demo proves.

---

## Backup Mode (if live RPC is unstable)

If live storage / chain is unstable during recording:

1. still show the full ingest → context → restart → hydrate → trace flow
2. then show previously captured proof from:
   - `docs/evidence/2026-03-22-live-storage-chain-proof.md`
   - `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`

Be explicit:

> “The recovery flow is live; the 0G proof page is pre-captured because endpoint stability can vary.”

That is much better than pretending the network issue did not happen.

---

## What to Avoid Saying

- Don’t lead with endpoints before stating the crash-recovery problem.
- Don’t describe this as only “workflow CRUD.”
- Don’t spend most of the video on contract code.
- Don’t claim “AI infra” without proving recovery after restart.
