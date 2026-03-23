# Deep OpenClaw Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Upgrade the current OpenClaw-compatible ingest layer into a deeper OpenClaw workflow memory service with richer event semantics, read/hydrate APIs, and judge-friendly workflow traces.

**Architecture:** Keep the existing Go orchestrator + Rust runtime split. Do not introduce a new database or UI in this phase. Extend the current workflow event model, expose OpenClaw run-centric read APIs, and add a trace view that makes replay / resume / checkpoint / anchor relationships obvious to judges and integrators.

**Tech Stack:** Go (`net/http`, existing workflow/file store modules), Rust runtime unchanged, existing JSON file store, existing tests, existing 0G storage and chain adapters.

---

## Recommended approach

### Option A — Recommended

Add a deeper OpenClaw contract at the HTTP and event-model layer:

- richer OpenClaw event schema
- run-centric context / hydrate / trace endpoints
- no new persistence engine

**Why this is best now:** highest hackathon impact for the least moving parts.

### Option B — Bigger but slower

Build a full OpenClaw execution journal with a separate run/step projection store.

**Why not first:** stronger architecture, but too much scope for the current milestone.

### Option C — Demo-only polish

Keep current data model and only improve docs + demo wording.

**Why not enough:** helps presentation, but does not materially deepen the integration story.

---

## Scope for this phase

### In scope

- richer OpenClaw event metadata
- run-centric read APIs
- hydrate endpoint for recovery flow
- trace endpoint for judge-facing workflow inspection
- tests and docs

### Out of scope

- browser UI
- vector search / semantic retrieval
- replacing the file store
- changing Rust runtime protocol
- adding 0G Compute in this phase

---

## Task 1: Upgrade the OpenClaw event model

**Priority:** P0  
**Why:** Right now the project only accepts OpenClaw-like events. It does not yet preserve enough OpenClaw workflow semantics to feel like native agent infrastructure.

**Files:**
- Modify: `apps/orchestrator-go/internal/openclaw/adapter.go`
- Modify: `apps/orchestrator-go/internal/openclaw/adapter_test.go`
- Modify: `apps/orchestrator-go/pkg/types/workflow.go`
- Modify: `apps/orchestrator-go/internal/server/http.go`
- Modify: `apps/orchestrator-go/internal/server/http_test.go`

**Objective:**
Preserve richer OpenClaw semantics in every stored workflow event.

**Required fields to add:**
- `runId`
- `sessionId`
- `traceId`
- `parentEventId`
- `toolCallId`
- `skillName`
- `taskId`
- `role`

**Recommended event types to support explicitly:**
- `thought`
- `plan`
- `tool_call`
- `tool_result`
- `memory_read`
- `memory_write`
- `final_answer`

**Acceptance Criteria:**
- Single ingest accepts and stores the richer event metadata
- Batch ingest preserves the richer metadata per item
- Existing minimal payloads remain backward-compatible
- Unknown / omitted fields still degrade safely

**TDD Steps:**
1. Extend `adapter_test.go` with failing tests for richer metadata propagation
2. Run `go test ./internal/openclaw -count=1`
3. Extend `http_test.go` with failing ingest response / storage assertions
4. Run `go test ./internal/server -run OpenClaw -count=1`
5. Update `adapter.go` and `pkg/types/workflow.go`
6. Re-run:
   - `go test ./internal/openclaw -count=1`
   - `go test ./internal/server -count=1`

**Suggested commit message:**
`feat(openclaw): preserve richer run and tool trace metadata`

---

## Task 2: Add OpenClaw run context and hydrate APIs

**Priority:** P0  
**Why:** A true memory service must support reads, not only writes.

**Files:**
- Modify: `apps/orchestrator-go/internal/workflow/service.go`
- Modify: `apps/orchestrator-go/internal/workflow/service_test.go`
- Modify: `apps/orchestrator-go/internal/server/http.go`
- Modify: `apps/orchestrator-go/internal/server/http_test.go`

**Objective:**
Expose run-centric APIs that OpenClaw-like callers can use to recover workflow context.

**Endpoints to add:**
- `GET /v1/openclaw/runs/{id}/context`
- `GET /v1/openclaw/runs/{id}/checkpoint/latest`
- `POST /v1/openclaw/runs/{id}/hydrate`

**Recommended behavior:**

### `GET /context`
Return:
- workflow/run identifiers
- latest status / step / root / cid / tx hash
- recent events (start with latest 20; can be static in phase 1)

### `GET /checkpoint/latest`
Return:
- workflow id
- latest step
- latest root
- latest cid
- latest tx hash

### `POST /hydrate`
Return:
- the same workflow metadata after `ResumeWithContext`
- recent events from restored checkpoint

**Acceptance Criteria:**
- Existing workflow routes stay compatible
- `hydrate` uses request context and existing resume path
- `context` makes it easy for a planner/executor to reconstruct recent state

**TDD Steps:**
1. Add failing service tests for context / checkpoint view behavior
2. Run `go test ./internal/workflow -count=1`
3. Add failing handler tests for `/context`, `/checkpoint/latest`, `/hydrate`
4. Run `go test ./internal/server -count=1`
5. Implement minimal service helpers and handler routes
6. Re-run targeted tests, then full package tests

**Suggested commit message:**
`feat(openclaw): add run context and hydrate endpoints`

---

## Task 3: Add a judge-friendly OpenClaw trace endpoint

**Priority:** P1  
**Why:** Judges need to see that this is agent memory infrastructure, not only a workflow CRUD service.

**Files:**
- Modify: `apps/orchestrator-go/internal/workflow/service.go`
- Modify: `apps/orchestrator-go/internal/workflow/service_test.go`
- Modify: `apps/orchestrator-go/internal/server/http.go`
- Modify: `apps/orchestrator-go/internal/server/http_test.go`

**Objective:**
Expose a run timeline that links workflow steps to memory, checkpoint, and verification state.

**Endpoint to add:**
- `GET /v1/openclaw/runs/{id}/trace`

**Recommended response shape:**
- top-level run/workflow metadata
- ordered `steps[]`
- per step:
  - `eventId`
  - `eventType`
  - `role`
  - `skillName`
  - `toolCallId`
  - `parentEventId`
  - `payload`
  - `stepIndex`
- top-level latest:
  - `latestRoot`
  - `latestCid`
  - `latestTxHash`

**Acceptance Criteria:**
- The trace is readable without replaying raw storage files manually
- It is obvious which run is being inspected
- It is demo-friendly for judges and future UI work

**TDD Steps:**
1. Add failing service test for trace projection shape
2. Run `go test ./internal/workflow -count=1`
3. Add failing handler test for `GET /v1/openclaw/runs/{id}/trace`
4. Run `go test ./internal/server -count=1`
5. Implement minimal projection logic in service + handler
6. Re-run targeted tests and then full Go suite

**Suggested commit message:**
`feat(openclaw): expose run trace timeline`

---

## Task 4: Align docs and demo around “OpenClaw memory service”

**Priority:** P1  
**Why:** The code change only matters if the judge can immediately understand the new integration depth.

**Files:**
- Modify: `README.md`
- Modify: `QUICKSTART.md`
- Modify: `docs/demo/3min-judge-flow.md`
- Modify: `docs/submission/2026-03-23-hackquest-final-copy.md`
- Modify: `docs/submission/2026-03-23-hackquest-form-answers.md`

**Objective:**
Update docs to describe the project as an OpenClaw run-memory service rather than only an event ingest backend.

**Required doc updates:**
- list the new run context / hydrate / trace endpoints
- explain richer event fields
- explain how OpenClaw-like planners/executors can read memory back
- update demo flow to show one read endpoint, not only write endpoints

**Acceptance Criteria:**
- README top section reflects the stronger integration story
- Demo script includes at least one read/hydrate call
- Submission copy mentions OpenClaw run context and recovery path

**TDD / Verification Steps:**
1. Update docs after code paths are merged
2. Run `git diff --check`
3. Re-read docs for consistency with actual endpoint names

**Suggested commit message:**
`docs(openclaw): position service as run memory layer`

---

## Task 5: Full verification and submission readiness check

**Priority:** P0  
**Why:** This phase changes API shape, stored metadata, and judge-facing docs.

**Files:**
- No new feature files required
- Re-verify all files changed in Tasks 1–4

**Verification Commands:**

### Go

```bash
cd apps/orchestrator-go
GOENV=off GOCACHE=/tmp/go-build-openclaw-deep /Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go test ./... -count=1
```

### Rust

```bash
cd rust/memory-core
cargo test
```

### Docs sanity

```bash
git diff --check
```

**Acceptance Criteria:**
- Go tests pass
- Rust tests still pass
- New OpenClaw endpoints are covered by tests
- Docs and demo wording match actual routes and behavior

**Suggested commit message:**
`test(openclaw): verify deep integration milestone`

---

## Recommended execution order

1. **Task 1 — Upgrade the OpenClaw event model**
2. **Task 2 — Add OpenClaw run context and hydrate APIs**
3. **Task 3 — Add judge-friendly trace endpoint**
4. **Task 4 — Align docs and demo**
5. **Task 5 — Full verification**

---

## Expected outcome

After this plan, the project should no longer look like only an OpenClaw-compatible ingest API. It should look like a real **OpenClaw run memory and recovery layer on 0G**, which is a much stronger fit for the Agentic Infrastructure & OpenClaw Lab track.
