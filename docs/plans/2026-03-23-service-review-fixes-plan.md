# Service Review Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the highest-value review findings in the OpenClaw-facing service so the project is safer and more honest as a long-running hackathon demo service.

**Architecture:** Keep the current Go orchestrator + Rust runtime split. Prioritize correctness at the HTTP boundary first, then fix context propagation, then reduce service-wide contention, and finally improve readiness from config-only checks toward lightweight live probes.

**Tech Stack:** Go (`net/http`, service/store/runtime modules), Rust runtime unchanged, existing Go tests, existing 0G storage/chain clients.

---

## Recommended execution order

### Critical path

1. **Task 1 — HTTP body size limits**
2. **Task 2 — Honest batch ingest semantics**
3. **Task 3 — Resume context propagation**
4. **Task 4 — Per-workflow locking**
5. **Task 5 — Real readiness probes**

### Parallelization guidance

- **Can run in parallel:** Task 1 and Task 3
- **Should not overlap on same files:** Task 1 and Task 2 both touch `internal/server/http.go`
- **Do after Task 3:** Task 4
- **Do after Task 1/2 are stable:** Task 5

---

## Task 1: Add HTTP request body limits

**Priority:** P0  
**Why:** Public HTTP ingress currently accepts unbounded request bodies.

**Owned Files:**
- Modify: `apps/orchestrator-go/internal/server/http.go`
- Modify: `apps/orchestrator-go/internal/server/http_test.go`

**Objective:**
Reject oversized single-event and batch ingest requests with a clear `413 Payload Too Large` response.

**Requirements:**
- Add request-size limiting for:
  - `POST /v1/openclaw/ingest`
  - `POST /v1/openclaw/ingest/batch`
- Return stable JSON error envelope
- Do not change successful request behavior

**Acceptance Criteria:**
- Oversized body returns HTTP 413
- Small valid body still returns 200
- Invalid JSON inside allowed size still returns 400

**TDD Steps:**
1. Write failing tests in `http_test.go` for oversized single ingest
2. Run `go test ./internal/server -run TestHandlerOpenClawIngestRejectsOversizedBody -count=1`
3. Write failing tests for oversized batch ingest
4. Run `go test ./internal/server -run TestHandlerOpenClawBatchIngestRejectsOversizedBody -count=1`
5. Implement request-size limiting in `http.go`
6. Re-run `go test ./internal/server -count=1`

**Suggested commit message:**
`fix(server): limit openclaw ingest request body size`

---

## Task 2: Make batch ingest semantics explicit and safe

**Priority:** P0  
**Why:** Current batch endpoint can partially commit then return a global 500, which is misleading for callers.

**Owned Files:**
- Modify: `apps/orchestrator-go/internal/server/http.go`
- Modify: `apps/orchestrator-go/internal/server/http_test.go`
- Optional Modify: `README.md`
- Optional Modify: `QUICKSTART.md`

**Objective:**
Change batch ingest from “all-or-nothing looking” behavior to an honest per-item result model.

**Recommended approach:**
Keep batch as **best-effort with per-item results**, not transactional rollback.

**Requirements:**
- Endpoint returns a result object per event
- Each item includes:
  - workflow id if available
  - success/failure
  - latest step on success
  - error code/message on failure
- Top-level response stays 200 if request parsed successfully
- Optional: add top-level summary counts (`successCount`, `failureCount`)

**Acceptance Criteria:**
- If event 2 fails, event 1 success is still visible in response
- Caller can safely retry only failed items
- Tests cover mixed success/failure batch

**TDD Steps:**
1. Add failing test for mixed-result batch behavior
2. Run targeted server test and confirm failure
3. Update response schema in `http.go`
4. Re-run `go test ./internal/server -count=1`
5. Update docs examples if response shape changed

**Suggested commit message:**
`fix(server): return per-item batch ingest results`

---

## Task 3: Propagate request context into resume path

**Priority:** P1  
**Why:** `Resume()` currently uses `context.Background()` for checkpoint download, so cancelled requests can keep running.

**Owned Files:**
- Modify: `apps/orchestrator-go/internal/workflow/service.go`
- Modify: `apps/orchestrator-go/internal/workflow/service_test.go`
- Modify: `apps/orchestrator-go/internal/server/http.go`
- Modify: `apps/orchestrator-go/internal/server/http_test.go`
- Modify: `apps/orchestrator-go/cmd/workflow.go`

**Objective:**
Make resume use caller-provided context from HTTP and CLI entrypoints.

**Requirements:**
- Change `Resume(workflowID string)` to `Resume(ctx context.Context, workflowID string)`
- Pass HTTP request context from `/v1/workflows/{id}/resume`
- Pass `context.Background()` from CLI only at the top layer
- Storage download must honor caller cancellation

**Acceptance Criteria:**
- Tests cover resumed download using passed context
- Existing CLI behavior preserved
- No `context.Background()` remains inside service-level download path

**TDD Steps:**
1. Add failing service test showing resume should use supplied context
2. Add/adjust HTTP test for new method signature path
3. Change service and callers
4. Run:
   - `go test ./internal/workflow -count=1`
   - `go test ./internal/server ./cmd -count=1`

**Suggested commit message:**
`fix(workflow): propagate context through resume path`

---

## Task 4: Replace global service lock with per-workflow locking

**Priority:** P1  
**Why:** Current global mutex serializes all workflows through runtime, storage, and chain I/O.

**Owned Files:**
- Modify: `apps/orchestrator-go/internal/workflow/service.go`
- Modify: `apps/orchestrator-go/internal/workflow/service_test.go`
- Optional Modify: `apps/orchestrator-go/internal/workflow/store.go`

**Objective:**
Reduce contention so unrelated workflows can progress concurrently while preserving same-workflow safety.

**Recommended approach:**
Introduce a lightweight workflow lock manager:
- one mutex map keyed by workflow id
- same workflow serialized
- different workflows can proceed independently

**Requirements:**
- `Start("")` generation path stays safe
- Same workflow still protected against duplicate concurrent mutation
- Different workflows should no longer block each other at service scope

**Acceptance Criteria:**
- Existing same-workflow concurrency tests still pass
- Add test proving two different workflows can execute without a shared global lock bottleneck
- No service-wide lock is held across network/storage/runtime calls

**TDD Steps:**
1. Add failing test for independent workflow concurrency
2. Add failing regression test to preserve same-workflow safety
3. Refactor lock model in `service.go`
4. Run `go test ./internal/workflow -count=1`

**Suggested commit message:**
`refactor(workflow): use per-workflow locking`

---

## Task 5: Upgrade readiness from config checks to lightweight live probes

**Priority:** P2  
**Why:** Current `/health` can report ready when storage/chain are only syntactically configured.

**Owned Files:**
- Modify: `apps/orchestrator-go/internal/workflow/service.go`
- Modify: `apps/orchestrator-go/internal/server/http.go`
- Modify: `apps/orchestrator-go/internal/server/http_test.go`
- Modify: `apps/orchestrator-go/internal/ogstorage/client.go`
- Modify: `apps/orchestrator-go/internal/ogstorage/client_test.go`
- Modify: `apps/orchestrator-go/internal/ogchain/client.go`
- Modify: `apps/orchestrator-go/internal/ogchain/client_test.go`
- Optional Modify: `README.md`

**Objective:**
Make readiness say something closer to “dependencies are actually reachable.”

**Recommended minimal probe design:**
- `runtime`: keep current real probe
- `storage`: perform lightweight HTTP/indexer reachability probe or small metadata endpoint probe
- `anchor`: perform JSON-RPC reachability probe such as `eth_chainId` or `eth_blockNumber`; optionally add contract-level `eth_call`

**Requirements:**
- Keep timeouts bounded
- Avoid expensive write operations in readiness
- Distinguish:
  - configured but unreachable
  - not configured
  - ready

**Acceptance Criteria:**
- Tests cover:
  - ready
  - config missing
  - remote probe failure
- `/health` returns 503 when required components are unreachable
- Documentation explains what readiness really means

**TDD Steps:**
1. Add failing storage readiness tests
2. Add failing chain readiness tests
3. Add/adjust `/health` tests
4. Implement lightweight probe methods
5. Run:
   - `go test ./internal/ogstorage ./internal/ogchain -count=1`
   - `go test ./internal/server ./internal/workflow -count=1`

**Suggested commit message:**
`feat(readiness): add live dependency probes`

---

## Suggested agent task cards

### Agent A — HTTP Safety
**Owns:**
- `apps/orchestrator-go/internal/server/http.go`
- `apps/orchestrator-go/internal/server/http_test.go`

**Do:**
- Task 1
- Task 2

**Do not touch:**
- workflow service internals unless required for interface compatibility

---

### Agent B — Workflow Context + Locking
**Owns:**
- `apps/orchestrator-go/internal/workflow/service.go`
- `apps/orchestrator-go/internal/workflow/service_test.go`
- `apps/orchestrator-go/cmd/workflow.go`
- `apps/orchestrator-go/internal/server/http.go` (only for resume callsite if needed)

**Do:**
- Task 3
- Task 4

**Do not touch:**
- ogstorage / ogchain probe logic

---

### Agent C — Readiness
**Owns:**
- `apps/orchestrator-go/internal/ogstorage/client.go`
- `apps/orchestrator-go/internal/ogstorage/client_test.go`
- `apps/orchestrator-go/internal/ogchain/client.go`
- `apps/orchestrator-go/internal/ogchain/client_test.go`
- `apps/orchestrator-go/internal/server/http.go`
- `apps/orchestrator-go/internal/server/http_test.go`

**Do:**
- Task 5

**Blocked by:**
- ideally after Agent A stabilizes `http.go`

---

## Final verification checklist

After all tasks:

```bash
cd apps/orchestrator-go
GOENV=off GOCACHE=/tmp/go-build-0g-review-fixes GOPROXY=https://goproxy.cn,direct GOSUMDB=off /Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go test ./... -count=1
```

```bash
cd rust/memory-core
cargo test
```

Expected:
- all Go tests pass
- all Rust tests pass
- no API regression in single ingest / batch ingest / replay / resume / readiness

