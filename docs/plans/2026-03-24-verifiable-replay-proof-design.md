# Verifiable Replay / Proof of Recovery Design

## Goal

Add a judge-facing verification capability proving that a persisted OpenClaw-style run can be:

1. reloaded from 0G Storage,
2. recomputed from local workflow events,
3. matched against the latest on-chain MemoryAnchor checkpoint.

The demo story becomes:

**ingest -> checkpoint -> crash/restart -> hydrate -> verify**

instead of only showing hydrate / trace.

## Current context

The current project already has:

- OpenClaw-style ingest endpoints in `apps/orchestrator-go/internal/server/http.go`
- workflow orchestration in `apps/orchestrator-go/internal/workflow/service.go`
- judge-friendly read models in `apps/orchestrator-go/internal/workflow/run_views.go`
- 0G Storage upload/download in `apps/orchestrator-go/internal/ogstorage`
- 0G Chain write path plus `getLatestCheckpoint(...)` client support in `apps/orchestrator-go/internal/ogchain/client.go`
- live Galileo deployment proof in `deployments/0g-testnet/MemoryAnchor.latest.json`

So the missing capability is not “more storage” or “more chain integration”; it is a **single proof view** that ties the existing pieces together.

## Approaches considered

### Option A — Local-only verify

Compare:

- recomputed checkpoint from event log
- downloaded checkpoint from Storage

Pros:

- lowest implementation cost
- no chain read dependency

Cons:

- misses the strongest judge signal: on-chain proof
- does not fully answer “is the recovered state the same state that was anchored?”

### Option B — Three-source verification (**recommended**)

Compare three sources:

- **stored metadata** from local workflow store
- **downloaded checkpoint** from 0G Storage
- **latest on-chain checkpoint** from MemoryAnchor
- plus a **fresh local recomputation** from runtime replay

Pros:

- strongest infra narrative
- reuses current codebase well
- easiest wow moment in demo

Cons:

- moderate implementation effort
- needs one new service view and one new route

### Option C — UI-first only

Ship a pretty page that stitches together existing APIs client-side.

Pros:

- fast visual improvement

Cons:

- weak proof semantics
- not enough if judges ask “where do you actually verify?”

## Recommended design

Adopt **Option B**.

### New service capability

Add a new read/verification API in the workflow layer, for example:

- `VerifyRun(ctx context.Context, runID string) (RunVerification, error)`

The verification flow should:

1. resolve workflow metadata by `runID`
2. rebuild runtime events from `meta.Events`
3. replay + rebuild a fresh checkpoint using the Rust runtime
4. download the persisted checkpoint blob from 0G Storage and decode it
5. read the latest checkpoint from MemoryAnchor using the workflow hash already used during anchoring
6. compare all sources against the expected values
7. return a single structured verdict

### New HTTP route

Add:

- `GET /v1/openclaw/runs/{id}/verify`

This should return a structured result for judge/demo use.

### Response shape

Recommended high-level payload:

- run identity: `workflowId`, `runId`, `sessionId`, `traceId`
- expected values: `expectedWorkflowIDHash`, `expectedCIDHash`, `expectedRootHash`, `expectedStepIndex`
- storage proof section
- recomputed proof section
- chain proof section
- per-check booleans
- top-level `verified`

### Status semantics

Important design decision:

- verification **mismatch** should not be treated as a server crash
- transport / source failures should still be visible in the JSON payload

Recommended behavior:

- `404` only when the run does not exist
- `200` with `verified=false` when verification finishes but any comparison fails
- `500` only for truly unrecoverable internal errors

This keeps the page and demo stable even when a source is red.

## Data comparison rules

Expected values should come from existing project conventions:

- expected workflow hash = `hashToBytes32Hex(meta.WorkflowID)`
- expected cid hash = `hashToBytes32Hex(meta.LatestCID)`
- expected root hash = `normalizeBytes32Hex(meta.LatestRoot)`
- expected step index = `meta.LatestStep`

Checks:

1. **recomputed root matches stored root**
2. **downloaded checkpoint root matches stored root**
3. **downloaded checkpoint latest step matches stored step**
4. **chain root matches expected root**
5. **chain cid hash matches expected cid hash**
6. **chain step index matches expected step**

Top-level `verified=true` only when all required checks pass.

## Judge console

Add a lightweight same-origin page served from the Go HTTP server, e.g.:

- `GET /judge/verify`

This page should:

- accept a `runId`
- call `/v1/openclaw/runs/{id}/verify`
- show a clear green/red summary
- render contract / tx links when available

No framework is required. Plain embedded HTML/CSS/JS is enough.

## Parallelization plan

### Critical path

1. workflow verification model + service logic
2. HTTP route
3. docs/demo update

### Sidecar parallel task

- judge console UI after the verify response contract is stable

## Success criteria

The feature is done when:

1. `GET /v1/openclaw/runs/{id}/verify` returns a stable JSON verdict
2. the verdict includes recomputed, storage, and chain comparisons
3. a judge can see `verified=true` for a live run that was anchored on 0G
4. the page `/judge/verify` can visualize the result
5. demo/docs are updated to use the new proof step
