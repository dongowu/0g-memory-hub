# Persistent Runtime Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace request-per-process Rust runtime calls with a persistent stdio runtime process that handles multiple orchestrator requests with automatic restart on failure.

**Architecture:** Keep the existing JSON request/response protocol and newline framing, but introduce a managed Go-side runtime session that owns one child process with stdin/stdout pipes. Extend the Rust binary from single-request execution to a request loop so the same process can serve multiple requests. The runtime client API stays stable by swapping in a persistent transport implementation.

**Tech Stack:** Go, Rust, os/exec, bufio, context, existing workflow runtime protocol.

---

### Task 1: Prove current transport gap with tests

**Files:**
- Modify: `apps/orchestrator-go/internal/workflow/process_transport_test.go`
- Create: `apps/orchestrator-go/internal/workflow/persistent_transport_test.go`

**Step 1: Write the failing test**
- Add tests that require:
  - reusing one process for multiple `Call` invocations
  - restarting the process after an intentional helper failure
  - explicit close behavior

**Step 2: Run test to verify it fails**
Run: `go test ./internal/workflow -run 'TestPersistent' -count=1`
Expected: FAIL because persistent transport does not exist.

**Step 3: Write minimal implementation target notes in test comments**
- Document newline-delimited request/response assumptions.

**Step 4: Run test to verify it still fails correctly**
Run: `go test ./internal/workflow -run 'TestPersistent' -count=1`
Expected: FAIL with missing type/function errors.

### Task 2: Add Go persistent transport

**Files:**
- Create: `apps/orchestrator-go/internal/workflow/persistent_transport.go`
- Modify: `apps/orchestrator-go/internal/workflow/runtime_client.go` (only if interface helpers are needed)

**Step 1: Write minimal implementation**
- Add a transport that:
  - lazily starts the runtime child process
  - sends one JSON line per request
  - reads one JSON line per response
  - serializes calls with a mutex
  - kills and clears the process on read/write/wait failures
  - supports `Close()`

**Step 2: Run focused tests**
Run: `go test ./internal/workflow -run 'TestPersistent|TestProcessTransport' -count=1`
Expected: PASS.

### Task 3: Update Rust runtime binary to serve multiple requests

**Files:**
- Modify: `rust/memory-core/src/bin/memory-core-rpc.rs`
- Modify: `rust/memory-core/src/rpc.rs` (only if helper extraction is needed)

**Step 1: Write/adjust Rust tests if needed**
- Add a small unit test around request handling helper if the entrypoint becomes thin.

**Step 2: Write minimal implementation**
- Change binary from single line read to loop over stdin lines.
- For each line:
  - parse request
  - execute request
  - print one JSON response line
- Keep process alive until EOF or unrecoverable IO error.

**Step 3: Run Rust tests**
Run: `cargo test`
Expected: PASS.

### Task 4: Wire orchestrator to prefer persistent runtime transport

**Files:**
- Modify: `apps/orchestrator-go/cmd/workflow.go`
- Modify: `apps/orchestrator-go/cmd/serve.go`
- Modify: `apps/orchestrator-go/cmd/root.go` (if transport construction helper is best placed there)

**Step 1: Write failing integration-style Go test**
- Verify dependency wiring uses persistent transport constructor.

**Step 2: Implement minimal wiring**
- Replace direct `NewProcessTransport` construction with persistent transport.
- Ensure long-lived server path can close transport on shutdown if practical; otherwise keep MVP simple.

**Step 3: Run focused tests**
Run: `go test ./cmd ./internal/workflow -count=1`
Expected: PASS.

### Task 5: Full verification

**Step 1: Run Go full suite**
Run: `GOENV=off GOCACHE=/tmp/go-build-0g-storage-proof GOPROXY=https://goproxy.cn,direct GOSUMDB=off /Users/dongowu/.local/share/mise/installs/go/1.26.0/bin/go test ./... -count=1`
Expected: PASS.

**Step 2: Run Rust full suite**
Run: `cargo test`
Expected: PASS.
