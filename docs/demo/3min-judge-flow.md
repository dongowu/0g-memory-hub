# 3-Minute Judge Demo Flow

This flow is optimized for the **0G APAC Hackathon — Agentic Infrastructure & OpenClaw Lab** track and should be recorded as a **story**, not an endpoint tour.

## Recommended Demo Style

### Option A — Crash / Recover / Verify (**Recommended**)

This is the strongest judge-facing path because it proves the core thesis:

> **An OpenClaw-style agent run can survive process loss, recover from durable memory, re-derive its checkpoint, and stay externally verifiable on 0G.**

### Option B — API Capability Tour

Safer and easier, but weaker. Judges see features, not urgency.

### Option C — Storage / Contract Proof Tour

Useful as supporting evidence, but too infra-heavy if used alone.

**Recommendation:** record **Option A**, then use storage / explorer proof as the final 20-second close.

---

## Core Message

Do **not** pitch this as “a workflow backend” or “a set of APIs.”

Pitch it as:

> **A durable memory runtime for OpenClaw-style agents on 0G: ingest events, checkpoint deterministically, recover after failure, verify the restored state, and trace it.**

The judge should leave with exactly four ideas:

1. OpenClaw-style events enter through a real service interface.
2. Rust turns those events into deterministic checkpoints.
3. 0G Storage + chain anchor make the memory durable and inspectable.
4. Even after a restart, the run can be hydrated, verified, and traced back.

---

## Demo Objective

In under 3 minutes, prove all four:

- **agent context exists**
- **checkpoint state is persisted**
- **restart does not lose memory**
- **the run has a public verification path**
- **verification is tied back to run trace, not just a single tx hash**

---

## 20-Second Opening Script

Use this wording or stay very close:

> “Most agent demos lose memory when the process dies. We built a durable memory layer for OpenClaw-style workflows on 0G. The Go service ingests workflow events, the Rust runtime deterministically rebuilds state and emits checkpoints, 0G Storage persists those checkpoints, and the MemoryAnchor path anchors verification metadata so the run can be recovered, re-verified, and inspected.”

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

Fast rehearsal path from the repo root:

```bash
./scripts/demo_verify_smoke.sh
```

Use the script to dry-run the same `health -> ingest -> context -> checkpoint/latest -> hydrate -> verify -> trace` path before recording. For the actual video, keep the manual commands below so judges can follow each step.

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

### 2:20 - 2:35 Hydrate the run after restart

Run:

```bash
curl -X POST http://127.0.0.1:8080/v1/openclaw/runs/run-judge-01/hydrate
```

Say:

- `hydrate` rebuilds the run from persisted checkpoint state.
- This is the recovery half of the claim: **memory survives process loss**.

### 2:35 - 2:50 Verify restored state against persisted proof

Run:

```bash
curl http://127.0.0.1:8080/v1/openclaw/runs/run-judge-01/verify
curl "http://127.0.0.1:8080/judge/verify?runId=run-judge-01"
```

Say:

- `verify` is the judge-facing checkpoint integrity step.
- We do not only recover; we re-derive the checkpoint and compare it with 0G Storage and MemoryAnchor-linked metadata.
- The verify endpoint and judge console should be live in the demo so the proof can be shown directly instead of described verbally.

### 2:50 - 3:00 Close with trace + explorer linkage

Show one of:

- `latestTxHash` from the API response
- explorer page
- `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`
- `GET /v1/openclaw/runs/run-judge-01/trace`

Say:

- `trace` proves ordered execution context is still intact after verify.
- Storage is the durable memory layer and MemoryAnchor is the external verification anchor.
- So the run is recoverable, re-verifiable, and externally inspectable.

---

## Strong Closing Line

Close with:

> “We’re not just helping an agent act. We’re making its workflow memory durable, recoverable, re-verifiable, and traceable on 0G.”

---

## Judge Q&A Short Answers

### “Why is this a fit for Track 1?”

Because this is OpenClaw-style agent infrastructure: event ingest, deterministic memory, recovery, traceability, and durable persistence.

### “What is the actual wow moment here?”

The wow moment is that after restart, the run is hydrated, checkpoint verification is re-derived against persisted proof, and then the run is still traceable end-to-end.

### “Why 0G instead of ordinary storage?”

Because this project is about durable and inspectable agent memory. 0G Storage persists the checkpoint, and the chain path adds a verification surface outside the agent process.

### “What happens if the process crashes?”

The process can restart, the checkpoint can be reloaded, verification can be re-derived against Storage/MemoryAnchor, and the run can still be traced. That is the product claim the demo proves.

---

## Backup Mode (if live RPC is unstable)

If live storage / chain is unstable during recording:

1. still show the full ingest → checkpoint → restart → hydrate → verify → trace flow
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
