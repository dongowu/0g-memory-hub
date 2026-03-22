# 2026-03-22 Live Storage + Chain Proof

## Summary

On **March 22, 2026 (Asia/Shanghai)**, live probing showed:

- `https://indexer-storage-testnet-standard.0g.ai` returned **503**
- `https://indexer-storage-testnet-turbo.0g.ai` responded normally
- Galileo chain RPC remained reachable at `https://evmrpc-testnet.0g.ai`

This evidence run used the **turbo** indexer endpoint.

## Live storage proof

### Endpoints

- Indexer: `https://indexer-storage-testnet-turbo.0g.ai`
- Chain RPC: `https://evmrpc-testnet.0g.ai`
- Flow contract (from `/node/status`): `0x22e03a6a89b950f1c82ec5e74f8eca321a105296`
- Market contract (from `flow.market()`): `0x26c8f001c94b0fd287db5397f05ef8bd8ef2cf4b`

### Payload

- Plaintext payload: `demo-checkpoint-v1`
- Bytes: `18`

### Real Flow submit transaction

- Tx hash: `0x20fdcd388ffe0764b7a778e4b0e73e5cae420b0a06717c5d21ca962f0082b1ad`
- Explorer: `https://chainscan-galileo.0g.ai/tx/0x20fdcd388ffe0764b7a778e4b0e73e5cae420b0a06717c5d21ca962f0082b1ad`
- Block: `24149723`

### Real storage root

- Root: `0xdcd9fbfdca6fc87e5d371b00623a90a0026d25103b13eb236bafe6aa6be844e8`

### File status after segment upload

- `seq`: `27443`
- `uploadedSegNum`: `1`
- `finalized`: `true`

Observed response from:

`GET https://indexer-storage-testnet-turbo.0g.ai/file/info/0xdcd9fbfdca6fc87e5d371b00623a90a0026d25103b13eb236bafe6aa6be844e8`

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "tx": {
      "dataMerkleRoot": "0xdcd9fbfdca6fc87e5d371b00623a90a0026d25103b13eb236bafe6aa6be844e8",
      "size": 18,
      "seq": 27443
    },
    "finalized": true,
    "uploadedSegNum": 1,
    "pruned": false
  }
}
```

### Download verification

Downloaded bytes matched the original plaintext payload:

- Download endpoint:
  `https://indexer-storage-testnet-turbo.0g.ai/file?root=0xdcd9fbfdca6fc87e5d371b00623a90a0026d25103b13eb236bafe6aa6be844e8&name=checkpoint.bin`
- Downloaded plaintext: `demo-checkpoint-v1`

## Linked chain anchor proof

The proven storage root was then anchored into the deployed `MemoryAnchor` contract.

- Contract: `0xc164d2f784f0e23bf380e9367b76e42dfa3c45e7`
- Anchor tx:
  `0x6656012aed1603a2d689a736db0385f097993fad430d7876b71ef9997ffe9fd7`
- Explorer:
  `https://chainscan-galileo.0g.ai/tx/0x6656012aed1603a2d689a736db0385f097993fad430d7876b71ef9997ffe9fd7`
- Block: `24150919`
- Status: `1`

Anchored values:

- `workflowId`: `0xeb0839f4a694b2de078128728f572523c57447850c6d9456881f66643d135235`
- `rootHash`: `0xdcd9fbfdca6fc87e5d371b00623a90a0026d25103b13eb236bafe6aa6be844e8`
- `cidHash`: `0x17f2c483e8ef2e2b5c1828781070e8144fa66de053beca2173022ca245aaaf9a`

## Reproduction scripts

- Storage flow proof:
  `node scripts/live_storage_flow_proof.cjs`
- Chain anchor:
  `OG_STORAGE_ROOT=<root> node scripts/anchor_storage_root.cjs`

## Notes

- The current evidence script intentionally targets a **small payload (<= 256 bytes)** so that the live proof path stays inside a single chunk / single segment case.
- This is not yet wired into the Go orchestrator runtime path.
- It is, however, a **real 0G storage write + real 0G chain transaction + real retrieval verification**.
