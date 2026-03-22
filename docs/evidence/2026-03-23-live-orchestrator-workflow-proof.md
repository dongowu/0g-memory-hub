# 2026-03-23 Live Orchestrator Workflow Proof

## Summary

On **March 23, 2026 (Asia/Shanghai)**, the Go orchestrator successfully completed a live workflow step against 0G Galileo by:

1. building a real Rust checkpoint,
2. storing the checkpoint in 0G Storage,
3. anchoring the linked workflow checkpoint in the deployed `MemoryAnchor` contract,
4. resuming and replaying the workflow back from the stored checkpoint.

This run used the following runtime settings:

- Workflow ID: `live-wf-20260322`
- Runtime binary: `/tmp/0g-memory-core-target/debug/memory-core-rpc`
- Chain RPC: `https://evmrpc-testnet.0g.ai`
- Primary storage config: `https://indexer-storage-testnet-standard.0g.ai`
- Effective live storage path: **turbo REST fallback** via `https://indexer-storage-testnet-turbo.0g.ai`
- Anchor contract: `0xc164d2f784f0e23bf380e9367b76e42dfa3c45e7`

## Live workflow step result

Successful CLI output:

```text
workflow_id=live-wf-20260322 status=running latest_step=0 latest_root=910cb69f3a380900ddfa1b56d28cfb269f1bd6695cb76cc427175df12b819eb9 latest_cid=0x9dc55c7b4acdab3246e65ab3cc5263229a819c65a1f3debad48b2e3871f9e7bf latest_tx_hash=0xd684b4ae73926d4fba2fd07d5442c8e12843e8ec72a4071cb1d8b8fc402050bf
```

Meaning:

- `latest_root`: workflow runtime root hash
- `latest_cid`: 0G storage root used as checkpoint key
- `latest_tx_hash`: MemoryAnchor tx hash for the linked chain anchor

## 0G Storage proof

- Storage root: `0x9dc55c7b4acdab3246e65ab3cc5263229a819c65a1f3debad48b2e3871f9e7bf`
- Turbo file info endpoint:
  `https://indexer-storage-testnet-turbo.0g.ai/file/info/0x9dc55c7b4acdab3246e65ab3cc5263229a819c65a1f3debad48b2e3871f9e7bf`

Observed response:

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "tx": {
      "dataMerkleRoot": "0x9dc55c7b4acdab3246e65ab3cc5263229a819c65a1f3debad48b2e3871f9e7bf",
      "size": 229,
      "seq": 27447
    },
    "finalized": true,
    "uploadedSegNum": 1,
    "pruned": false
  }
}
```

Downloaded object evidence:

- Download endpoint:
  `https://indexer-storage-testnet-turbo.0g.ai/file?root=0x9dc55c7b4acdab3246e65ab3cc5263229a819c65a1f3debad48b2e3871f9e7bf&name=checkpoint.bin`
- Downloaded bytes length: `229`
- Downloaded bytes SHA-256:
  `df68591d245c512e9b95c472a96118c664dd44a85a89490f07af0c59f0083ddf`

The stored object is a **compact checkpoint + gzip** payload used by the direct single-segment fallback path.

## Linked chain anchor proof

- Contract: `0xc164d2f784f0e23bf380e9367b76e42dfa3c45e7`
- Anchor tx:
  `0xd684b4ae73926d4fba2fd07d5442c8e12843e8ec72a4071cb1d8b8fc402050bf`
- Explorer:
  `https://chainscan-galileo.0g.ai/tx/0xd684b4ae73926d4fba2fd07d5442c8e12843e8ec72a4071cb1d8b8fc402050bf`
- Status: `1`
- Block: `24194216`

Anchored values:

- Workflow label: `live-wf-20260322`
- `workflowId`:
  `0x9da0769094a1765c47b76e7294e3dbaea48f6872449689f92038cd7b9bc6a8f0`
- `rootHash`:
  `0x910cb69f3a380900ddfa1b56d28cfb269f1bd6695cb76cc427175df12b819eb9`
- `cidHash`:
  `0x791f1e7becedce36e064305fa7e4305c3991e571fce719436c45b4ad5ab1c01d`

## Resume / replay verification

After the live step, the orchestrator successfully resumed and replayed from the stored checkpoint:

```text
workflow_id=live-wf-20260322 status=running latest_step=0
```

```text
workflow=live-wf-20260322 status=running latest_root=910cb69f3a380900ddfa1b56d28cfb269f1bd6695cb76cc427175df12b819eb9 latest_cid=0x9dc55c7b4acdab3246e65ab3cc5263229a819c65a1f3debad48b2e3871f9e7bf
step=0 event_id=live-wf-20260322-step-0 type=tool_result actor=openclaw payload={"task":"live_storage_direct_fallback","ok":true,"ts":"2026-03-22"}
```

This confirms the Go storage client can now:

- write a checkpoint to 0G Storage,
- read it back,
- decode the compact fallback payload,
- restore workflow replay state.
