# Verifiable Replay Proof Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a judge-facing verification flow that proves a run can be replayed, recovered from 0G Storage, and matched against the latest on-chain MemoryAnchor checkpoint.

**Architecture:** The Go workflow service gains a new `VerifyRun(...)` read path that compares four perspectives of the same run: local workflow metadata, fresh runtime recomputation, downloaded storage checkpoint, and latest MemoryAnchor state. The HTTP server exposes this as `GET /v1/openclaw/runs/{id}/verify`, and a lightweight same-origin judge page renders the result for demos.

**Tech Stack:** Go HTTP server, Go workflow service, existing Rust runtime RPC process, existing 0G Storage client, existing 0G Chain JSON-RPC client, plain embedded HTML/CSS/JS.

---

## Recommended agent split

- **Agent A (backend owner):** Task 1 + Task 2
- **Agent B (UI owner):** Task 3
- **Agent C (docs/demo owner):** Task 4
- **Agent D (stretch):** Task 5

Dependencies:

- Task 2 depends on Task 1
- Task 3 depends on the response shape from Task 1
- Task 4 depends on Tasks 1–3 being merged
- Task 5 is optional stretch

### Task 1: Workflow verification model and service logic

**Files:**
- Create: `apps/orchestrator-go/internal/workflow/verify_view.go`
- Modify: `apps/orchestrator-go/internal/workflow/service.go`
- Modify: `apps/orchestrator-go/internal/workflow/service_test.go`
- Modify: `apps/orchestrator-go/internal/workflow/run_views_test.go` (only if common helpers are reused)

**Step 1: Write the failing service tests**

Add tests in `apps/orchestrator-go/internal/workflow/service_test.go` for:

- `TestServiceVerifyRunMatchesStorageAndChain`
- `TestServiceVerifyRunReportsStorageMismatch`
- `TestServiceVerifyRunReportsChainMismatch`
- `TestServiceVerifyRunResolvesByRunID`

Test shape should assert fields like:

```go
if !verification.Verified {
    t.Fatalf("Verified = false, want true: %+v", verification)
}
if !verification.Checks.RecomputedRootMatchesStored {
    t.Fatalf("recomputed root mismatch: %+v", verification.Checks)
}
if verification.Chain.StepIndex != 1 {
    t.Fatalf("chain step = %d, want 1", verification.Chain.StepIndex)
}
```

Extend the existing `fakeAnchor` so tests can stub a latest chain checkpoint read.

**Step 2: Run the targeted tests to confirm failure**

Run:

```bash
cd apps/orchestrator-go
go test ./internal/workflow -run 'TestServiceVerifyRun' -count=1
```

Expected: FAIL because `VerifyRun` and the new verification view types do not exist yet.

**Step 3: Implement the minimal verification types**

In `verify_view.go`, add explicit structs such as:

- `RunVerification`
- `VerificationChecks`
- `VerificationExpected`
- `VerificationCheckpoint`
- `VerificationChainCheckpoint`
- `VerificationSourceError`

Keep the first pass small. Required fields only:

- run identity
- expected hashes / step
- recomputed / storage / chain sections
- boolean checks
- top-level `verified`

**Step 4: Implement `VerifyRun(ctx, runID)` in the workflow service**

Implementation outline:

```go
meta, err := s.metadataForRun(runID)
deps := s.dependencies()

runtimeEvents := toRuntimeEvents(meta.Events)
state, err := deps.runtime.ReplayWorkflow(ctx, meta.WorkflowID, meta.AgentID, runtimeEvents)
checkpoint, err := deps.runtime.BuildCheckpoint(ctx, *state)

payload, err := deps.storage.DownloadCheckpoint(ctx, meta.LatestCID)
json.Unmarshal(payload, &downloaded)

expectedWorkflowHash := hashToBytes32Hex(meta.WorkflowID)
expectedCIDHash := hashToBytes32Hex(meta.LatestCID)
expectedRootHash := normalizeBytes32Hex(meta.LatestRoot)
```

If the anchor dependency can read latest checkpoint, compare chain values too.

Do not fail the whole verification on mismatches. Instead, populate result fields and set `Verified=false`.

**Step 5: Add an anchor read interface without breaking the write path**

Do one of these, keeping the write scope contained to workflow service + adapter boundary:

- add a new optional interface for reading latest checkpoints
- or extend the current anchor abstraction carefully

Preferred API at the workflow layer:

```go
type CheckpointAnchorReader interface {
    GetLatestCheckpoint(ctx context.Context, workflowID string) (*AnchorCheckpointView, error)
}
```

**Step 6: Re-run targeted workflow tests**

Run:

```bash
cd apps/orchestrator-go
go test ./internal/workflow -run 'TestServiceVerifyRun|TestServiceHydrate|TestServiceLatestCheckpointAndRunTrace' -count=1
```

Expected: PASS.

**Step 7: Commit**

```bash
git add apps/orchestrator-go/internal/workflow/verify_view.go apps/orchestrator-go/internal/workflow/service.go apps/orchestrator-go/internal/workflow/service_test.go apps/orchestrator-go/internal/workflow/run_views_test.go
git commit -m "feat(workflow): add verifiable replay proof model"
```

### Task 2: HTTP verify endpoint and route tests

**Files:**
- Modify: `apps/orchestrator-go/internal/server/http.go`
- Modify: `apps/orchestrator-go/internal/server/http_test.go`
- Modify: `apps/orchestrator-go/cmd/workflow.go` (adapter wiring for anchor read support)
- Modify: `apps/orchestrator-go/internal/ogchain/client.go` (only if adapter glue is needed)
- Test: `apps/orchestrator-go/internal/server/http_test.go`

**Step 1: Write the failing HTTP tests**

Add tests in `http_test.go` for:

- `TestHandlerOpenClawRunRoutesVerify`
- `TestHandlerOpenClawRunRoutesVerifyByRunID`
- `TestHandlerOpenClawRunRoutesVerifyNotFound`

Example assertions:

```go
req := httptest.NewRequest(http.MethodGet, "/v1/openclaw/runs/run-openclaw/verify", nil)
rec := httptest.NewRecorder()
handler.ServeHTTP(rec, req)
if rec.Code != http.StatusOK {
    t.Fatalf("verify status = %d, want 200 body=%s", rec.Code, rec.Body.String())
}
```

Decode the JSON and assert `verified`, `checks`, `chain`, and `expected` fields are present.

**Step 2: Run the route tests to verify failure**

Run:

```bash
cd apps/orchestrator-go
go test ./internal/server -run 'TestHandlerOpenClawRunRoutesVerify' -count=1
```

Expected: FAIL because `/verify` route is not registered yet.

**Step 3: Implement the route**

In `http.go`, extend `handleOpenClawRunRoutes` with:

```go
if len(parts) == 2 && parts[1] == "verify" && r.Method == http.MethodGet {
    verification, err := h.svc.VerifyRun(r.Context(), runID)
    if err != nil {
        handleWorkflowError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, verification)
    return
}
```

**Step 4: Wire anchor read support through the existing adapter**

In `cmd/workflow.go`, extend `workflowAnchorAdapter` so the workflow service can call into `ogchain.Client.GetLatestCheckpoint(...)` when available.

Do not rewrite the whole adapter. Keep the patch minimal.

**Step 5: Re-run targeted HTTP tests**

Run:

```bash
cd apps/orchestrator-go
go test ./internal/server -run 'TestHandlerOpenClawRunRoutesVerify|TestHandlerOpenClawRunRoutesContextCheckpointHydrateAndTrace' -count=1
```

Expected: PASS.

**Step 6: Commit**

```bash
git add apps/orchestrator-go/internal/server/http.go apps/orchestrator-go/internal/server/http_test.go apps/orchestrator-go/cmd/workflow.go apps/orchestrator-go/internal/ogchain/client.go
git commit -m "feat(api): add run verification endpoint"
```

### Task 3: Judge console page (same-origin)

**Files:**
- Create: `apps/orchestrator-go/internal/server/assets/verify_console.html`
- Create: `apps/orchestrator-go/internal/server/assets.go`
- Modify: `apps/orchestrator-go/internal/server/http.go`
- Modify: `apps/orchestrator-go/internal/server/http_test.go`

**Step 1: Write the failing page test**

Add a small test such as:

- `TestHandlerJudgeVerifyPage`

Check that `GET /judge/verify` returns `200` and `text/html`.

**Step 2: Run the page test to verify failure**

Run:

```bash
cd apps/orchestrator-go
go test ./internal/server -run TestHandlerJudgeVerifyPage -count=1
```

Expected: FAIL because the route does not exist yet.

**Step 3: Add an embedded static HTML page**

Create a minimal page with:

- run ID input
- “Verify” button
- green/red summary banner
- cards for expected / storage / recomputed / chain
- raw JSON expander for debugging

No framework. Plain HTML + inline JS only.

The page should call:

```js
fetch(`/v1/openclaw/runs/${encodeURIComponent(runId)}/verify`)
```

**Step 4: Register the page route**

In `http.go`, add a handler for:

- `GET /judge/verify`

Serve the embedded asset. Do not add a separate frontend build system.

**Step 5: Re-run the server tests**

Run:

```bash
cd apps/orchestrator-go
go test ./internal/server -run 'TestHandlerJudgeVerifyPage|TestHandlerOpenClawRunRoutesVerify' -count=1
```

Expected: PASS.

**Step 6: Commit**

```bash
git add apps/orchestrator-go/internal/server/assets/verify_console.html apps/orchestrator-go/internal/server/assets.go apps/orchestrator-go/internal/server/http.go apps/orchestrator-go/internal/server/http_test.go
git commit -m "feat(judge): add verification console"
```

### Task 4: Demo and submission material refresh

**Files:**
- Modify: `README.md`
- Modify: `QUICKSTART.md`
- Modify: `docs/demo/3min-judge-flow.md`
- Modify: `docs/demo/judge-checklist.md`
- Modify: `docs/submission/2026-03-23-hackquest-final-copy.md`
- Modify: `docs/submission/2026-03-23-hackquest-form-answers.md`

**Step 1: Update docs to include the new proof step**

Refresh the main demo path to:

- ingest
- checkpoint
- restart
- hydrate
- verify
- trace

Add example commands:

```bash
curl http://127.0.0.1:8080/v1/openclaw/runs/run-judge-01/verify
open http://127.0.0.1:8080/judge/verify
```

**Step 2: Add judge-facing explanation text**

Add one short paragraph everywhere needed:

> We do not only reload the workflow. We re-derive the checkpoint and compare it with the persisted storage object and the latest on-chain MemoryAnchor state.

**Step 3: Verify docs mention the right contract and route names**

Search for stale names before committing.

Run:

```bash
rg -n "MemoryChain|testnet-rpc\.0g\.ai|testnet-explorer\.0g\.ai|/verify" README.md QUICKSTART.md docs/demo docs/submission
```

Expected: `MemoryAnchor` is the judge-facing path, and `/verify` appears in the updated docs.

**Step 4: Commit**

```bash
git add README.md QUICKSTART.md docs/demo/3min-judge-flow.md docs/demo/judge-checklist.md docs/submission/2026-03-23-hackquest-final-copy.md docs/submission/2026-03-23-hackquest-form-answers.md
git commit -m "docs(demo): add verifiable replay proof flow"
```

### Task 5: Optional stretch — CLI verification command

**Files:**
- Modify: `apps/orchestrator-go/cmd/workflow.go`
- Test: `apps/orchestrator-go/cmd/serve_test.go` or a new command test if needed

**Step 1: Add a failing test or manual invocation note**

If command tests are too expensive, this can be a manual verification task only.

Target command:

```bash
go run . workflow verify run-judge-01
```

**Step 2: Implement a thin wrapper around `svc.VerifyRun(...)`**

Print JSON to stdout so demo operators can quickly inspect proof state without using curl.

**Step 3: Verify manually**

Run:

```bash
cd apps/orchestrator-go
go run . workflow verify run-judge-01
```

Expected: JSON output containing `verified`, `expected`, `recomputed`, `storage`, and `chain`.

**Step 4: Commit**

```bash
git add apps/orchestrator-go/cmd/workflow.go
git commit -m "feat(cli): add workflow verify command"
```

### Final verification pass

After Tasks 1–4 merge, run:

```bash
cd apps/orchestrator-go
go test ./... -count=1
```

Then from repo root run:

```bash
node --test test/generate-testnet-wallet.test.js test/render-deployment-evidence.test.js
```

If live credentials are available, also run a smoke demo:

```bash
cd apps/orchestrator-go
go run . serve
# in another shell:
curl http://127.0.0.1:8080/v1/openclaw/runs/<run-id>/verify
```

### Definition of done

- workflow service can produce a structured proof verdict
- `/v1/openclaw/runs/{id}/verify` works
- `/judge/verify` works
- docs/demo flow is updated to use verify as a wow moment
- full Go test suite passes
