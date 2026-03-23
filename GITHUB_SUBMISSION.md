# GitHub / HackQuest 提交指南

> 当前黑客松主线请以 **Go orchestrator + Rust runtime + MemoryAnchor** 为准：
> `apps/orchestrator-go` + `rust/memory-core` + `contracts/MemoryAnchor.sol`

## 1. 当前已验证的 0G 证据

- Galileo testnet contract: `0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- Contract explorer: `https://chainscan-galileo.0g.ai/address/0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- Deployment tx: `0x114fd7ebc9f2fcb9aab6780cafbd8964399fcd5b22ba13107a348f0ac5ecd72c`
- Anchor proof tx: `0xa794dd7aedcf7b7c349005af620f29d8a36557c7b7973f91e358e31287fad1db`
- Deployment artifact: `deployments/0g-testnet/MemoryAnchor.latest.json`
- Judge-facing proof doc: `docs/evidence/2026-03-23-0g-testnet-memory-anchor-deployment-proof.md`

## 2. 提交前最重要的判断标准

评委真正会看的是：

1. **是否真的用了 0G**
2. **是否能现场跑通 / 复核**
3. **文档和 Demo 是否清楚**
4. **仓库是否像一个认真维护的产品，而不是拼凑 demo**

所以提交主线应始终围绕：

- `README.md`
- `QUICKSTART.md`
- `docs/submission/2026-03-23-hackquest-form-answers.md`
- `docs/submission/2026-03-23-hackquest-final-copy.md`
- `docs/demo/3min-judge-flow.md`

## 3. 仓库提交检查清单

- [ ] GitHub 仓库可公开访问
- [ ] `main` 已推送最新代码
- [ ] README / QUICKSTART 与当前实现一致
- [x] 已有真实 Galileo 测试网部署证据
- [x] 已有 Explorer 可核验链接
- [ ] Demo 视频链接已补齐
- [ ] X / Twitter 推文链接已补齐
- [ ] 如主办方最终强制要求主网，再补主网合约地址与 Explorer

## 4. 日常提交流程

```bash
git status
git add <files>
git commit -m "feat: ..."
git push origin main
```

如果只是同步当前主分支：

```bash
git push origin main
```

## 5. HackQuest 表单推荐填写口径

### 项目名

`0G Memory Hub`

### 一句话描述（≤30 words）

OpenClaw-style agent memory runtime on 0G that persists checkpoints to 0G Storage and anchors workflow proofs on 0G Chain via MemoryAnchor.

### 代码仓库

`https://github.com/dongowu/0g-memory-hub`

> 提交前确认仓库已设为 Public。

### 0G 集成证明

- Contract address: `0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- Explorer: `https://chainscan-galileo.0g.ai/address/0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
- Proof tx: `https://chainscan-galileo.0g.ai/tx/0xa794dd7aedcf7b7c349005af620f29d8a36557c7b7973f91e358e31287fad1db`

### Demo / Tweet 待补字段

- Demo video: `TODO`
- X post: `TODO`
- Mainnet proof: `TODO`（仅当主办方最终明确要求）

## 6. 推荐提交流程

1. 确认 `origin/main` 是最新
2. 确认仓库为 Public
3. 用 `docs/submission/2026-03-23-hackquest-form-answers.md` 填表
4. 用 `docs/demo/3min-judge-flow.md` 录制 3 分钟视频
5. 发布 X 推文并把链接填回提交材料

## 7. 推荐 X 推文模板

```text
Building OpenClaw-style agent memory on @0G_labs

0G Memory Hub gives AI workflows durable checkpoints, recovery after crashes, and verifiable on-chain memory proofs.

Demo: [VIDEO_LINK]
Repo: https://github.com/dongowu/0g-memory-hub

#0GHackathon #BuildOn0G @0G_labs @0g_CN @HackQuest_
```

## 8. 安全提醒

- **不要提交私钥**
- 任何 `.env` / 本地钱包文件都不要入库
- 当前仓库已忽略 `.wallets/`
- 由于测试私钥曾被明文暴露在聊天里，建议赛后尽快停用并更换该测试钱包
