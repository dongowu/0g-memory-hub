# 0G OpenClaw Memory Runtime

[English](./README.md) | [简体中文](./README.zh-CN.md)

**一个构建在 0G 上的 OpenClaw 工作流持久记忆层：在 Rust 中生成 Agent 执行 checkpoint，将状态持久化到 0G Storage，并把验证元数据锚定到链上。**

面向 **0G APAC Hackathon — Track 1: Agentic Infrastructure & OpenClaw Lab**。

> **核心主张：** Agent 记忆应该能在进程崩溃后继续保留、恢复，并能在模型进程之外被验证。

---

## 提交快照

| 项目 | 值 |
|---|---|
| 赛道 | Agentic Infrastructure & OpenClaw Lab |
| 仓库 | `dongowu/0g-memory-hub` |
| 核心技术栈 | Go orchestrator + Rust runtime + Solidity anchor |
| 使用的 0G 组件 | 0G Storage + 0G Chain |
| 当前证明 | Galileo / 0g-testnet |
| 测试网合约 | `0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9` |
| Explorer | `https://chainscan-galileo.0g.ai/address/0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9` |

---

## 这个项目做什么

大多数 Agent Demo 在进程退出后都会丢失工作流上下文。本项目把工作流记忆变成一个可持久化的基础设施能力：

- **Go orchestrator** 接收 OpenClaw 风格工作流事件
- **Rust runtime** 确定性重放事件并构建 checkpoint
- **0G Storage** 将 checkpoint blob 持久化到进程之外
- **0G Chain / MemoryAnchor** 锚定 `workflowId`、`stepIndex`、`rootHash`、`cidHash`
- **hydrate / verify / trace** 证明 run 在重启后可以恢复并再次校验

面向评委的验证表述：

> 我们不仅能在重启后恢复一次 run，还会重新推导 checkpoint，并将其与 0G Storage 中持久化的数据、以及 MemoryAnchor 链上元数据进行比对。

---

## 已实现能力

- OpenClaw 风格单事件 ingest
- OpenClaw 风格批量 ingest
- 丰富元数据保留：`runId`、`sessionId`、`traceId`、`parentEventId`、`toolCallId`、`skillName`、`taskId`、`role`
- 基于 `eventId` 的幂等 ingest
- Rust 中的确定性 replay 与 checkpoint 生成
- 面向长生命周期服务的持久化 Rust runtime transport
- 0G Storage 的 checkpoint 上传 / 下载路径
- 通过 `MemoryAnchor` 实现的 0G Chain anchor 路径
- `replay`、`resume`、`hydrate`、`verify`、`trace` 和 readiness 检查
- `/judge/verify?runId={id}` Judge 验证页

---

## 黑客松主路径

评审和演示时，请优先看这些路径：

| 组件 | 路径 |
|---|---|
| Go orchestrator | `apps/orchestrator-go` |
| Rust runtime | `rust/memory-core` |
| Solidity 合约 | `contracts/MemoryAnchor.sol` |
| Quickstart | `QUICKSTART.md` |
| Demo 文档 | `docs/demo/` |
| 提交文档 | `docs/submission/` |
| 证据文档 | `docs/evidence/` |

legacy root 路径（`main.go`, `cmd/`, `core/`, `sdk/`）仅为兼容保留，**不是** 当前黑客松主要评审路径。

---

## 评委快速入口

| 内容 | 路径 |
|---|---|
| HackQuest 最终提交文案 | `docs/submission/2026-03-23-hackquest-final-copy.md` |
| 提交清单 | `docs/submission/2026-03-23-hackquest-submission-checklist.md` |
| 3 分钟 Demo 流程 | `docs/demo/3min-judge-flow.md` |
| Judge 检查清单 | `docs/demo/judge-checklist.md` |
| Storage + Chain 实时证据 | `docs/evidence/2026-03-22-live-storage-chain-proof.md` |
| 工作流实时证据 | `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md` |
| Readiness 实时证据 | `docs/evidence/2026-03-23-live-http-readiness-proof.md` |
| Galileo 部署证明 | `docs/evidence/2026-03-23-0g-testnet-memory-anchor-deployment-proof.md` |

---

## 架构

```text
OpenClaw-style events
        |
        v
Go orchestrator
  - ingest / batch ingest
  - context / checkpoint / hydrate / verify / trace
        |
        v
Rust runtime
  - deterministic replay
  - checkpoint generation
  - root hash derivation
        |
        +--> 0G Storage
        |      - checkpoint persistence
        |
        +--> 0G Chain / MemoryAnchor
               - workflow anchor metadata
```

---

## 快速开始

### 前置要求

- Go **1.26.x**
- Rust stable
- Node.js **20 - 24**
- npm

### 本地快速启动

```bash
npm install
cd rust/memory-core && cargo test --offline && cargo build --bin memory-core-rpc
cd ../../apps/orchestrator-go && go test ./...
```

启动服务：

```bash
export ORCH_RUNTIME_BINARY_PATH="$(pwd)/rust/memory-core/target/debug/memory-core-rpc"
cd apps/orchestrator-go
go run . serve
```

检查 readiness：

```bash
curl http://127.0.0.1:8080/health
```

完整环境配置和手动命令请看 `QUICKSTART.md`。

---

## 推荐 Demo 路径

对评委来说，最强的故事线是：

1. **ingest** 一个 OpenClaw 风格 run
2. 展示 **checkpoint/latest**
3. 停掉并重启服务
4. **hydrate** 这个 run
5. **verify** 恢复后的 checkpoint
6. 展示 **trace**
7. 最后补 explorer / evidence 证明

本地快速 smoke 路径：

```bash
./scripts/demo_verify_smoke.sh
```

完整录屏与讲述方式：

- `docs/demo/3min-judge-flow.md`
- `docs/demo/judge-checklist.md`

---

## HTTP API 面

### Health 与 Judge 页面

- `GET /health`
- `GET /judge/verify?runId={id}`

### OpenClaw ingest

- `POST /v1/openclaw/ingest`
- `POST /v1/openclaw/ingest/batch`

### Workflow / Run 查询

- `GET /v1/workflows/{id}`
- `POST /v1/workflows/{id}/resume`
- `GET /v1/workflows/{id}/replay`
- `GET /v1/openclaw/runs/{id}/context`
- `GET /v1/openclaw/runs/{id}/checkpoint/latest`
- `POST /v1/openclaw/runs/{id}/hydrate`
- `GET /v1/openclaw/runs/{id}/verify`
- `GET /v1/openclaw/runs/{id}/trace`

---

## 验证状态

当前仓库验证结果：

- `apps/orchestrator-go`：在 **Go 1.26.0** 下测试通过
- `rust/memory-core`：`cargo test --offline` 通过
- `contracts/MemoryAnchor.sol`：Hardhat 测试通过

推荐验证命令：

```bash
cd rust/memory-core && cargo test --offline
cd ../../apps/orchestrator-go && go test ./...
cd ../.. && npx hardhat test test/MemoryAnchor.js
```

---

## 当前边界

- 这是一个 **工作流运行时 + 验证层**，不是完整的消费者 AI 产品
- OpenClaw ingest 当前是同步处理，并使用本地文件存储
- 实时 storage / chain 行为依赖 RPC 健康状态和有余额的凭据
- 当前仓库证明是 **testnet / Galileo**，不是 mainnet
- Demo 视频链接和 X/Twitter 提交链接仍需在最终提交时手动补齐

---

## 相关文档

- `QUICKSTART.md`
- `0G_INTEGRATION.md`
- `docs/architecture/2026-03-21-openclaw-memory-runtime-design.md`
- `docs/submission/2026-03-23-hackquest-form-answers.md`
