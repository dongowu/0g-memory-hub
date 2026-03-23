# OpenClaw Ingest Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Turn the Go orchestrator into a callable OpenClaw-facing HTTP service that can ingest OpenClaw-style events and map them into durable workflows.

**Architecture:** Add a thin HTTP layer on top of the existing workflow service. Extend the OpenClaw adapter to normalize richer event inputs and keep the workflow domain logic unchanged where possible. The server will expose health, workflow read APIs, and an OpenClaw ingest endpoint that creates or advances workflow runs.

**Tech Stack:** Go, net/http, encoding/json, existing workflow service/store/runtime/storage/chain adapters.

---

### Task 1: Extend OpenClaw normalization model

**Files:**
- Modify: `apps/orchestrator-go/internal/openclaw/adapter.go`
- Create: `apps/orchestrator-go/internal/openclaw/adapter_test.go`

**Step 1: Write the failing test**
- Add tests for OpenClaw event normalization with:
  - explicit `workflowId`
  - fallback from `runId` to workflow id
  - fallback actor/event type defaults
  - payload serialization for structured payloads

**Step 2: Run test to verify it fails**
Run: `go test ./internal/openclaw -run TestNormalizeEvent -count=1`
Expected: FAIL because richer OpenClaw event support does not exist yet.

**Step 3: Write minimal implementation**
- Add a richer event input type that supports:
  - `workflowId`
  - `runId`
  - `sessionId`
  - `eventType`
  - `actor`
  - `payload`
- Normalize to `types.WorkflowStepEvent` and derive workflow id from `workflowId || runId`.

**Step 4: Run test to verify it passes**
Run: `go test ./internal/openclaw -run TestNormalizeEvent -count=1`
Expected: PASS.

**Step 5: Commit**
```bash
git add apps/orchestrator-go/internal/openclaw/adapter.go apps/orchestrator-go/internal/openclaw/adapter_test.go
git commit -m "feat(openclaw): normalize richer ingest events"
```

### Task 2: Add HTTP server handlers

**Files:**
- Create: `apps/orchestrator-go/internal/server/http.go`
- Create: `apps/orchestrator-go/internal/server/http_test.go`
- Modify: `apps/orchestrator-go/internal/workflow/service.go` (only if a helper is needed)

**Step 1: Write the failing test**
- Add handler tests for:
  - `GET /health`
  - `POST /v1/openclaw/ingest`
  - `GET /v1/workflows/{id}`
  - `POST /v1/workflows/{id}/resume`
  - `GET /v1/workflows/{id}/replay`

**Step 2: Run test to verify it fails**
Run: `go test ./internal/server -count=1`
Expected: FAIL because server package does not exist.

**Step 3: Write minimal implementation**
- Build a small `net/http` server with JSON endpoints.
- Use a consistent JSON envelope.
- Reuse the existing workflow service and OpenClaw adapter.

**Step 4: Run test to verify it passes**
Run: `go test ./internal/server -count=1`
Expected: PASS.

**Step 5: Commit**
```bash
git add apps/orchestrator-go/internal/server/http.go apps/orchestrator-go/internal/server/http_test.go apps/orchestrator-go/internal/workflow/service.go
git commit -m "feat(server): add openclaw ingest http api"
```

### Task 3: Wire a serve command

**Files:**
- Modify: `apps/orchestrator-go/cmd/root.go`
- Create: `apps/orchestrator-go/cmd/serve.go`
- Modify: `apps/orchestrator-go/internal/config/config.go`
- Modify: `apps/orchestrator-go/internal/config/config_test.go`

**Step 1: Write the failing test**
- Add config test for server bind address default.
- Add a small command construction test if needed.

**Step 2: Run test to verify it fails**
Run: `go test ./cmd ./internal/config -count=1`
Expected: FAIL because serve command / config field is missing.

**Step 3: Write minimal implementation**
- Add `ORCH_HTTP_ADDR` config.
- Add `serve` command to boot the HTTP API.
- Reuse the same dependency wiring currently used by workflow CLI commands.

**Step 4: Run test to verify it passes**
Run: `go test ./cmd ./internal/config -count=1`
Expected: PASS.

**Step 5: Commit**
```bash
git add apps/orchestrator-go/cmd/root.go apps/orchestrator-go/cmd/serve.go apps/orchestrator-go/internal/config/config.go apps/orchestrator-go/internal/config/config_test.go
git commit -m "feat(cmd): add orchestrator http serve command"
```

### Task 4: Verify full package

**Files:**
- Optional docs touch later, not required for this capability increment.

**Step 1: Run focused packages**
Run: `go test ./internal/openclaw ./internal/server ./cmd ./internal/config -count=1`
Expected: PASS.

**Step 2: Run full suite**
Run: `GOENV=off GOCACHE=/tmp/go-build-0g-storage-proof GOPROXY=https://goproxy.cn,direct GOSUMDB=off /Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go test ./... -count=1`
Expected: PASS.

**Step 3: Commit**
```bash
git add -A
git commit -m "test(orchestrator): verify openclaw ingest service"
```
