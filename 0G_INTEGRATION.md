# 0G 集成指南（当前黑客松主线）

> 本文档只描述当前 judge-facing 集成路径。
> **Canonical path = `MemoryAnchor` + Galileo + Go orchestrator + Rust runtime**
> 旧的 `MemoryChain` 原型仅作兼容保留，不作为本次提交主路径。

## 1. 当前架构

本项目面向 **Agentic Infrastructure & OpenClaw Lab** 赛道，核心目标是：

1. 接收 OpenClaw / agent workflow 事件
2. 由 Rust runtime 构建可恢复的 checkpoint
3. 把 checkpoint 持久化到 0G Storage
4. 把 `workflowId + stepIndex + rootHash + cidHash` 锚定到 0G Chain
5. 在服务崩溃后恢复运行，并让评委可通过 Explorer 验证链上证据

当前主线目录：

- Go orchestrator: `apps/orchestrator-go`
- Rust runtime: `rust/memory-core`
- 0G Storage integration: `apps/orchestrator-go/internal/ogstorage`
- 0G Chain integration: `apps/orchestrator-go/internal/ogchain`
- Contract: `contracts/MemoryAnchor.sol`

## 2. 0G Storage 集成

### 代码位置

- `apps/orchestrator-go/internal/ogstorage/client.go`
- `apps/orchestrator-go/internal/ogstorage/direct_upload.go`
- `apps/orchestrator-go/internal/ogstorage/direct_live.go`
- `apps/orchestrator-go/internal/ogstorage/direct_fallback.go`

### 做了什么

- 把 runtime 生成的 checkpoint JSON 上传到 0G Storage
- 记录 `rootHash` / `txHash`
- 支持读回 checkpoint 以便 hydrate / replay
- 当标准 indexer 路径不稳定时，走 direct fallback 路径保证 demo 可继续

### 解决的问题

- Agent memory 不再只存在进程内存里
- checkpoint 具备内容寻址和外部可验证性
- Demo 时即使服务重启，也能从已持久化 checkpoint 恢复

## 3. 0G Chain 集成

### 代码位置

- `apps/orchestrator-go/internal/ogchain/client.go`
- `contracts/MemoryAnchor.sol`

### 合约接口

当前提交主线使用 `MemoryAnchor`：

- `anchorCheckpoint(bytes32 workflowId, uint64 stepIndex, bytes32 rootHash, bytes32 cidHash)`
- `getLatestCheckpoint(bytes32 workflowId)`
- `getCheckpointCount(bytes32 workflowId)`
- `getCheckpointAt(bytes32 workflowId, uint256 index)`

### 解决的问题

- 把 workflow checkpoint 的关键摘要上链
- 允许评委通过 Explorer 独立核验
- 让“可恢复 memory”变成“可恢复 + 可验证 memory”

## 4. OpenClaw / 服务层集成

### 代码位置

- `apps/orchestrator-go/internal/openclaw/adapter.go`
- `apps/orchestrator-go/internal/workflow/service.go`
- `apps/orchestrator-go/internal/server/http.go`

### HTTP 能力

当前服务提供：

- `POST /v1/openclaw/ingest`
- `POST /v1/openclaw/ingest/batch`
- `GET /v1/workflows/{id}`
- `POST /v1/workflows/{id}/resume`
- `GET /v1/workflows/{id}/replay`
- `GET /v1/openclaw/runs/{id}/context`
- `GET /v1/openclaw/runs/{id}/checkpoint/latest`
- `POST /v1/openclaw/runs/{id}/hydrate`
- `GET /v1/openclaw/runs/{id}/trace`

这意味着项目不是一个“单点上传 demo”，而是一个可连续运行、可摄入事件、可恢复、可审计的 agent memory service。

## 5. 当前网络参数

### Galileo testnet

- Chain ID: `16602`
- RPC: `https://evmrpc-testnet.0g.ai`
- Explorer: `https://chainscan-galileo.0g.ai`

### Mainnet

- Chain ID: `16661`
- RPC: `https://evmrpc.0g.ai`
- Explorer: `https://chainscan.0g.ai`

## 6. 当前推荐环境变量

### Orchestrator

```bash
export ORCH_STORAGE_RPC_URL=https://indexer-storage-testnet-turbo.0g.ai
export ORCH_CHAIN_RPC_URL=https://evmrpc-testnet.0g.ai
export ORCH_CHAIN_CONTRACT_ADDRESS=0x...
export ORCH_CHAIN_PRIVATE_KEY=0x...
export ORCH_CHAIN_ID=16602
export ORCH_RUNTIME_BINARY_PATH=/path/to/memory-core-rpc
```

### Hardhat / deploy

```bash
export OG_CHAIN_RPC=https://evmrpc-testnet.0g.ai
export OG_TESTNET_CHAIN_ID=16602
export OG_TESTNET_EXPLORER_URL=https://chainscan-galileo.0g.ai
export PRIVATE_KEY=0x...
```

## 7. 当前推荐部署 / 证据命令

```bash
npm run wallet:new
npm run preflight:testnet
npx hardhat compile
npx hardhat test test/MemoryAnchor.js
npm run deploy:proof
npm run evidence:testnet
```

输出物：

- `deployments/0g-testnet/MemoryAnchor.latest.json`
- `docs/evidence/2026-03-23-0g-testnet-memory-anchor-deployment-proof.md`

## 8. 已验证的真实链上证据

- Contract: `0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- Contract explorer: `https://chainscan-galileo.0g.ai/address/0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- Deployment tx: `0x114fd7ebc9f2fcb9aab6780cafbd8964399fcd5b22ba13107a348f0ac5ecd72c`
- Anchor tx: `0xa794dd7aedcf7b7c349005af620f29d8a36557c7b7973f91e358e31287fad1db`

## 9. 提交口径建议

如果评委问“你们到底用了 0G 什么”：

可以直接回答：

> We use 0G Storage to persist workflow checkpoints and 0G Chain to anchor workflow proofs through MemoryAnchor. The Go service ingests OpenClaw-style events, the Rust runtime builds deterministic checkpoints, and the chain proof makes recovery externally verifiable.

## 10. 注意事项

- `MemoryAnchor` 是当前 canonical 合约
- `MemoryChain` 仅是早期原型兼容资产，不要作为主提交路径引用
- 不要把私钥提交进仓库
- Judge-facing 文档请优先引用 `README.md`、`QUICKSTART.md`、`docs/submission/*`、`docs/evidence/*`
