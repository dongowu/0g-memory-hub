# Contract Toolchain Fix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the Hardhat contract toolchain credible for hackathon judging by compiling and deploying `MemoryAnchor`, with docs/scripts aligned to the actual contract.

**Architecture:** Keep the fix minimal. Remove conflicting Hardhat plugin wiring, modernize the deploy script for the active ethers runtime, center the toolchain on `MemoryAnchor`, and add a small regression test that proves the contract’s core anchoring behavior.

**Tech Stack:** Hardhat, Solidity, ethers, Node.js, repo docs.

---

### Task 1: Lock the target contract and write a regression test

**Files:**
- Create: `test/MemoryAnchor.js`
- Read: `contracts/MemoryAnchor.sol`

**Step 1:** Add a failing Hardhat test covering `anchorCheckpoint()` success and monotonic step enforcement.

**Step 2:** Run `npx hardhat test test/MemoryAnchor.js` and confirm it fails under the current toolchain.

### Task 2: Fix Hardhat plugin/config wiring

**Files:**
- Modify: `hardhat.config.js`
- Modify: `package.json` (only if script/config cleanup is needed)

**Step 1:** Remove task/plugin wiring that causes the `verify` task conflict.

**Step 2:** Make network/explorer values look like a real project by preferring environment-based configuration over stale placeholders.

**Step 3:** Run `npx hardhat compile` and `npx hardhat test test/MemoryAnchor.js`.

### Task 3: Fix deploy script and docs alignment

**Files:**
- Modify: `scripts/deploy.js`
- Modify: `QUICKSTART.md`
- Modify: `README.md` (if contract/deploy section needs alignment)

**Step 1:** Make the deploy script deploy `MemoryAnchor` by default.

**Step 2:** Update ethers usage to match the currently loaded runtime.

**Step 3:** Print judge-useful deployment metadata.

**Step 4:** Update docs so deploy instructions match the actual contract and script behavior.

### Task 4: Verify end to end

**Files:** none

**Step 1:** Run `npx hardhat compile`.

**Step 2:** Run `npx hardhat test test/MemoryAnchor.js`.

**Step 3:** Re-run relevant repo checks if docs/scripts changed materially.
