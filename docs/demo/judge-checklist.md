# Judge Checklist

Use this checklist before recording/submitting demo evidence.

## Required Runtime Checks

- `apps/orchestrator-go` tests pass.
- `rust/memory-core` tests pass.
- `npx hardhat compile` passes.
- `workflow start` works.
- `workflow status` works.
- `workflow replay` works.

## Full 0G Checks (if enabled)

- `workflow step` returns non-empty `latest_root`.
- `workflow step` returns non-empty `latest_cid`.
- MemoryAnchor contract address is recorded.
- At least one chain tx hash is available.
- Explorer link is included in submission notes.

## Demo Recording Checklist (<= 3 min)

- Open with architecture map (Go + Rust + 0G + contract).
- Show live commands, not slides only.
- Show step output with root/cid fields.
- Show replay output.
- Mention fallback mode if RPC instability occurred.

## Submission Assets Checklist

- README updated with architecture and demo path.
- QUICKSTART updated with exact MVP commands.
- `docs/demo/3min-judge-flow.md` included.
- `scripts/demo.sh` present and executable.
- repo has recent substantive commits in hackathon period.
