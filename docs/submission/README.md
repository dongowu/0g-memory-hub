# Submission Pack Index

This folder is the **submission source of truth** for HackQuest / judge-facing materials.

---

## Start here

| Need | File |
|---|---|
| Copy-paste form answers | `docs/submission/2026-03-23-hackquest-form-answers.md` |
| Long-form final copy | `docs/submission/2026-03-23-hackquest-final-copy.md` |
| Submission checklist | `docs/submission/2026-03-23-hackquest-submission-checklist.md` |
| X / Twitter draft | `docs/submission/2026-03-23-x-post-draft.md` |
| Quick setup / demo commands | `QUICKSTART.md` |
| 3-minute demo script | `docs/demo/3min-judge-flow.md` |
| Judge dry-run checklist | `docs/demo/judge-checklist.md` |

---

## Current verified on-chain proof

- **Galileo contract:** `0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- **Explorer:** `https://chainscan-galileo.0g.ai/address/0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- **Anchor tx:** `https://chainscan-galileo.0g.ai/tx/0xa794dd7aedcf7b7c349005af620f29d8a36557c7b7973f91e358e31287fad1db`
- **Deployment proof:** `docs/evidence/2026-03-23-0g-testnet-memory-anchor-deployment-proof.md`

Supporting evidence:

- `docs/evidence/2026-03-22-live-storage-chain-proof.md`
- `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`
- `docs/evidence/2026-03-23-live-http-readiness-proof.md`

---

## Fast submission flow

1. Use `...hackquest-form-answers.md` for all short fields and copy-paste textareas.
2. Use `...hackquest-final-copy.md` when you need a cleaner long-form narrative.
3. Use `QUICKSTART.md` + `docs/demo/3min-judge-flow.md` to rehearse the recording.
4. Use `docs/demo/judge-checklist.md` before recording or submitting.
5. Use `...x-post-draft.md` to publish the required X / Twitter proof.

---

## Manual items still pending before final submit

- **Demo video link**
- **X / Twitter post link**
- **Repo visibility confirmation** (public or judge-accessible)
- **Mainnet address / explorer** only if the final HackQuest form explicitly requires mainnet instead of Galileo testnet proof

---

## Canonical judging paths

- `apps/orchestrator-go`
- `rust/memory-core`
- `contracts/MemoryAnchor.sol`

Legacy root paths (`main.go`, `cmd/`, `core/`, `sdk/`) are compatibility leftovers and are not the main hackathon judging path.
