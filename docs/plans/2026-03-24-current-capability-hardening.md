# Current Capability Hardening Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Freeze scope and harden the current 0G OpenClaw Memory Runtime so the existing service capabilities are consistent, demo-ready, and submission-ready without adding new product scope.

**Architecture:** Keep the current service boundary intact: Go orchestrator + Rust runtime + 0G Storage + MemoryAnchor. Focus only on consistency, contract stability, demo operability, and verification UX around existing endpoints (`ingest`, `hydrate`, `verify`, `trace`).

**Tech Stack:** Go HTTP service, Go workflow service, existing Rust runtime process, static HTML judge console, markdown docs, shell smoke scripts.

---

## Scope Freeze

Do **not** add new product directions such as 0G Compute, deeper multi-agent routing, or new business features.

Only improve current capabilities in these four dimensions:

1. API contract consistency
2. Judge/demo usability
3. Local operator ergonomics
4. Submission documentation consistency

---

### Task 1: Normalize the verify response contract

**Files:**
- Modify: `apps/orchestrator-go/internal/workflow/verify_view.go`
- Modify: `apps/orchestrator-go/internal/workflow/service.go`
- Modify: `apps/orchestrator-go/internal/workflow/service_test.go`
- Modify: `apps/orchestrator-go/internal/server/http_test.go`

**Step 1: Write/adjust failing tests for the stable response contract**

Add/adjust tests so verify output exposes a stable judge-facing model:

- `expected`
- `recomputed`
- `storage`
- `chain`
- `checks`
- `verified`

Example assertions:

```go
if verifyOut.Data.Expected.RootHash == "" {
    t.Fatal("expected.rootHash should not be empty")
}
if len(verifyOut.Data.Checks) == 0 {
    t.Fatal("checks should not be empty")
}
```

**Step 2: Run the focused tests and confirm failure**

Run:

```bash
cd apps/orchestrator-go
GOCACHE=/tmp/go-build-0g-memory-hub go test ./internal/workflow ./internal/server -run 'VerifyRun|RunRoutesVerify' -count=1
```

Expected: FAIL because the current contract still uses mixed internal names like `localMetadata`, `recomputedCheckpoint`, `onChainCheckpoint`.

**Step 3: Implement the stable view model**

Refactor `verify_view.go` to expose a cleaner, externally stable response model. Keep internal helper types private if needed.

Required top-level fields:

- `workflowId`
- `runId`
- `sessionId`
- `traceId`
- `verified`
- `expected`
- `recomputed`
- `storage`
- `chain`
- `checks`

**Step 4: Update service assembly logic**

In `service.go`, map internal data into the new stable response object without changing actual verification semantics.

**Step 5: Re-run focused tests**

Run:

```bash
cd apps/orchestrator-go
GOCACHE=/tmp/go-build-0g-memory-hub go test ./internal/workflow ./internal/server -run 'VerifyRun|RunRoutesVerify' -count=1
```

Expected: PASS.

**Step 6: Commit**

```bash
git add apps/orchestrator-go/internal/workflow/verify_view.go apps/orchestrator-go/internal/workflow/service.go apps/orchestrator-go/internal/workflow/service_test.go apps/orchestrator-go/internal/server/http_test.go
git commit -m "refactor(verify): stabilize judge-facing response contract"
```

### Task 2: Improve the judge console for failure readability

**Files:**
- Modify: `apps/orchestrator-go/internal/server/assets/verify_console.html`
- Modify: `apps/orchestrator-go/internal/server/http_test.go` (only if route or content expectations change)

**Step 1: Write a failing expectation or manual checklist**

If UI tests are too expensive, create a manual acceptance checklist in comments or notes for:

- visible expected/actual mismatch information
- visible check failure reasons
- query param auto-run (`?runId=`)
- raw JSON still visible

**Step 2: Upgrade the UI rendering**

Make the page show:

- each check name
- `passed`
- `expected`
- `actual`
- `message`

Prefer a table/list instead of only boolean highlights.

**Step 3: Make status obvious**

Ensure the page makes these three states visually distinct:

- verified success
- verification mismatch
- request/runtime/source failure

**Step 4: Verify manually or via existing route tests**

Run:

```bash
cd apps/orchestrator-go
GOCACHE=/tmp/go-build-0g-memory-hub go test ./internal/server -run 'TestHandlerJudgeVerifyPage|TestHandlerOpenClawRunRoutesVerify' -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add apps/orchestrator-go/internal/server/assets/verify_console.html apps/orchestrator-go/internal/server/http_test.go
git commit -m "feat(judge): improve verification console readability"
```

### Task 3: Add a CLI verify command for operator/demo fallback

**Files:**
- Modify: `apps/orchestrator-go/cmd/workflow.go`
- Add/Modify tests only if a lightweight command test is practical

**Step 1: Add the failing invocation expectation**

Target command:

```bash
cd apps/orchestrator-go
go run . workflow verify run-judge-01
```

Expected before implementation: command missing.

**Step 2: Implement a thin wrapper**

Add:

- `workflow verify [run-id]`

Behavior:

- calls `svc.VerifyRun(context.Background(), runID)`
- prints JSON to stdout

**Step 3: Verify manually**

Run:

```bash
cd apps/orchestrator-go
GOCACHE=/tmp/go-build-0g-memory-hub go run . workflow verify run-judge-01
```

Expected: structured JSON output.

**Step 4: Commit**

```bash
git add apps/orchestrator-go/cmd/workflow.go
git commit -m "feat(cli): add workflow verify command"
```

### Task 4: Add one reproducible local demo smoke script

**Files:**
- Create: `scripts/demo_verify_smoke.sh`
- Modify: `README.md`
- Modify: `QUICKSTART.md`
- Modify: `docs/demo/3min-judge-flow.md`
- Modify: `docs/demo/judge-checklist.md`

**Step 1: Write the smoke script**

The script should run only the current capability path:

- health
- ingest
- context
- checkpoint/latest
- restart note or separate operator step
- hydrate
- verify
- trace

It can assume the service is already running.

**Step 2: Make the script defensive**

- `set -euo pipefail`
- clear run IDs
- compact curl output
- fail loudly on non-200 responses

**Step 3: Update docs to point to the script**

Add one short command in README/QUICKSTART:

```bash
./scripts/demo_verify_smoke.sh
```

**Step 4: Verify the script syntax**

Run:

```bash
bash -n scripts/demo_verify_smoke.sh
```

Expected: PASS.

**Step 5: Commit**

```bash
git add scripts/demo_verify_smoke.sh README.md QUICKSTART.md docs/demo/3min-judge-flow.md docs/demo/judge-checklist.md
git commit -m "chore(demo): add verify smoke script"
```

### Task 5: Final submission consistency pass

**Files:**
- Modify: `docs/submission/2026-03-23-hackquest-final-copy.md`
- Modify: `docs/submission/2026-03-23-hackquest-form-answers.md`
- Modify: `docs/submission/2026-03-23-hackquest-submission-checklist.md` (if needed)

**Step 1: Reconcile docs with actual implementation**

Ensure wording matches the actual current service, not future intent.

Specifically check:

- `/verify` route exists
- `/judge/verify` exists
- MemoryAnchor/Galileo wording is consistent
- no placeholder wording remains for current features

**Step 2: Search for stale terms**

Run:

```bash
rg -n "placeholder|MemoryChain|testnet-rpc\.0g\.ai|testnet-explorer\.0g\.ai|future work" README.md QUICKSTART.md docs/demo docs/submission
```

Expected: no stale judge-facing wording for current capabilities.

**Step 3: Commit**

```bash
git add docs/submission/2026-03-23-hackquest-final-copy.md docs/submission/2026-03-23-hackquest-form-answers.md docs/submission/2026-03-23-hackquest-submission-checklist.md
git commit -m "docs(submission): align materials with current shipped capability"
```

### Final verification pass

Run all of these fresh before claiming completion:

```bash
cd apps/orchestrator-go
GOCACHE=/tmp/go-build-0g-memory-hub go test ./... -count=1
```

```bash
node --test test/generate-testnet-wallet.test.js test/render-deployment-evidence.test.js
```

```bash
bash -n scripts/demo_verify_smoke.sh
```

Optional live smoke if service is running:

```bash
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8080/v1/openclaw/runs/<run-id>/verify
curl "http://127.0.0.1:8080/judge/verify?runId=<run-id>"
```

### Definition of done

Current capability hardening is done when:

- verify API contract is stable and externally readable
- judge console clearly explains pass/fail and mismatches
- CLI has a verify fallback
- there is one repeatable local demo smoke path
- docs/submission materials describe only shipped functionality
- full tests pass
