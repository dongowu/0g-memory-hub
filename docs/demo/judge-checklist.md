# Judge Checklist

Use this checklist before recording/submitting demo evidence.

## Required Pre-Flight Checks

- `apps/orchestrator-go` tests pass.
- `rust/memory-core` tests pass.
- `npx hardhat compile` passes.
- `go run . serve` starts cleanly.
- `GET /health` works.
- `POST /v1/openclaw/ingest/batch` works.
- `GET /v1/openclaw/runs/{id}/context` works.
- `POST /v1/openclaw/runs/{id}/hydrate` works.
- `GET /v1/openclaw/runs/{id}/verify` works and returns structured verification data.
- `GET /v1/openclaw/runs/{id}/trace` works.
- `GET /judge/verify?runId={id}` opens and shows the judge console.

## Full 0G Checks (if enabled)

- `workflow step` returns non-empty `latest_root`.
- `workflow step` returns non-empty `latest_cid`.
- MemoryAnchor contract address is recorded.
- At least one chain tx hash is available.
- Explorer link is included in submission notes.

## Demo Recording Checklist (<= 3 min)

- Open with the **problem**: agent memory usually dies with the process.
- Use the **Crash / Recover / Verify / Trace** story, not a generic feature tour.
- Show architecture briefly: Go + Rust + 0G Storage + MemoryAnchor.
- Show live commands, not slides only.
- Show richer OpenClaw metadata (`runId`, `traceId`, tool/task linkage).
- Show checkpoint output with `latestRoot` and `latestCid`.
- Stop and restart the service at least once during the recording.
- Show `hydrate` after restart.
- Show `verify` after hydrate.
- Say explicitly: "we re-derive the checkpoint and compare it against Storage + MemoryAnchor-linked metadata."
- Show `trace` after verify.
- Show explorer or evidence doc for the verification close.
- Mention fallback mode explicitly if RPC instability occurred.

## Submission Assets Checklist

- README updated with architecture and demo path.
- QUICKSTART updated with exact MVP commands.
- `docs/demo/3min-judge-flow.md` included.
- demo commands verified end-to-end before recording.
- repo has recent substantive commits in hackathon period.
