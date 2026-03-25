# 0G APAC Hackathon 提交清单（Track 1: Agentic Infrastructure & OpenClaw Lab）

> 截止时间：**2026 年 5 月 9 日 23:59（UTC+8）**
>
> 当前项目：**0G OpenClaw Memory Runtime / 0g-memory-hub**

---

## 1. 一句话结论

当前仓库已经具备了 **Track 1 可讲清楚的技术主线**：

- Go orchestrator
- Rust deterministic runtime
- OpenClaw-style ingest API
- 0G Storage checkpoint path
- on-chain MemoryAnchor anchor path
- replay / hydrate / verify / trace / readiness / live evidence

如果要整理成 **可正式提交到 HackQuest 的材料包**，当前最关键的人工补项仍然是：

1. **Demo 视频（<= 3 分钟）**
2. **X / Twitter 公开推文证明**
3. **Repo visibility 确认**
4. **主网地址 / Explorer**（仅当最终表单明确要求主网，而不接受 Galileo 证明时）

---

## 2. HackQuest 必交项清单

### A. Basic Project Information

- [x] 项目名称已明确  
  - 建议使用：`0G OpenClaw Memory Runtime`
- [x] 一句话描述可写  
  - 建议版本：`Durable OpenClaw workflow memory on 0G using Go orchestration, Rust checkpoints, 0G Storage persistence, and on-chain verification anchors.`
- [x] 项目方向与赛道匹配  
  - Track 1: **Agentic Infrastructure & OpenClaw Lab**

### B. Code Repository

- [x] GitHub 仓库已存在
- [x] `main` 已更新到当前 judge-facing 版本
- [x] 有持续、实质性的 hackathon 期间 commit 历史
  - 最近提交示例：
    - `3c9d777 fix(judge): improve verify console status handling`
    - `b0c13a2 docs(readme): add hackquest-style bilingual summary`
    - `28069a3 docs(demo): align smoke flow and submission copy`
- [ ] 仓库公开可访问（需手动确认 GitHub visibility）
- [ ] 如仍为私有，需在提交前切为 public 或确认评委权限

### C. 0G Integration Proof（核心门槛）

- [x] 至少集成 1 个 0G 核心组件
  - 当前实际已覆盖：
    - 0G Storage
    - 0G Chain
- [x] 已有链上 / 存储证据文档
  - `docs/evidence/2026-03-22-live-storage-chain-proof.md`
  - `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`
  - `docs/evidence/2026-03-23-live-http-readiness-proof.md`
- [x] 已有合约代码
  - `contracts/MemoryAnchor.sol`
- [x] 已有 Galileo 证据
  - 部署地址：`0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
  - Explorer：`https://chainscan-galileo.0g.ai/address/0xE233C1c6f3374bf8F29e6902Ed181b694f6d7BD9`
  - Proof doc：`docs/evidence/2026-03-23-0g-testnet-memory-anchor-deployment-proof.md`
- [ ] 主网地址 / Explorer（仅当最终规则要求主网）
- [x] 提交材料里已明确说明使用了哪个 0G 组件、解决了什么问题

> 注意：当前最像正式提交 blocker 的是 **Demo + X 强制项 + repo visibility**，不是代码能力本身。

### D. Demo Video

- [x] 已有 demo 脚本与 demo 流程文档
  - `docs/demo/3min-judge-flow.md`
  - `docs/demo/judge-checklist.md`
- [x] 已有可展示内容
  - HTTP ingest
  - workflow hydrate / verify / trace
  - readiness
  - storage / anchor proof 文档
- [ ] 录制 **<= 3 分钟** demo 视频
- [ ] 视频中必须展示：
  - [ ] 核心功能
  - [ ] 0G 组件实际调用过程
  - [ ] 不是纯 PPT
- [ ] 上传到 YouTube / Loom
- [ ] 拿到公开访问链接

### E. README / Repo 文档

- [x] README 已改成 judge-first 摘要风格
- [x] QUICKSTART 已整理为 smoke / live 0G / deploy proof / troubleshooting
- [x] 已有 submission pack index
  - `docs/submission/README.md`
- [x] 架构说明已存在
- [x] Demo 路径已存在
- [x] Evidence 文档已存在
- [x] 可说明 0G 组件用途
- [ ] 补 Demo 视频链接到 README / submission docs（录制后）
- [ ] 补 X / Twitter 链接到 submission docs（发布后）

### F. X / Twitter 曝光证明（强制项）

- [ ] 发布至少 1 条项目介绍推文
- [ ] 包含项目名
- [ ] 包含 Demo 视频或截图
- [ ] 包含 Hashtag：
  - `#0GHackathon`
  - `#BuildOn0G`
- [ ] 包含提及：
  - `@0G_labs`
  - `@0g_CN`
  - `@HackQuest_`
- [ ] 保存推文链接
- [ ] 如有真实用户互动，截图留存（加分项）

---

## 3. 当前仓库里已经可以直接复用的提交资产

### 核心代码

- `apps/orchestrator-go/`
- `rust/memory-core/`
- `contracts/MemoryAnchor.sol`

### 核心文档

- `README.md`
- `README.zh-CN.md`
- `QUICKSTART.md`
- `docs/submission/README.md`
- `docs/demo/3min-judge-flow.md`
- `docs/demo/judge-checklist.md`

### 证据文档

- `docs/evidence/2026-03-22-live-storage-chain-proof.md`
- `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`
- `docs/evidence/2026-03-23-live-http-readiness-proof.md`
- `docs/evidence/2026-03-23-0g-testnet-memory-anchor-deployment-proof.md`

### 当前最适合提交表单的一句话卖点

> A durable OpenClaw workflow memory runtime on 0G that checkpoints agent execution in Rust, orchestrates persistence in Go, stores checkpoints on 0G Storage, and anchors verification metadata on-chain.

---

## 4. 当前状态总表

| 类别 | 状态 | 说明 |
|---|---|---|
| 赛道匹配 | ✅ | 与 Agentic Infrastructure + OpenClaw 高度匹配 |
| GitHub 仓库 | ✅ | 已存在，`main` 已更新 |
| 有效代码量 | ✅ | 已有完整 Go + Rust + Solidity 主线 |
| README / QUICKSTART | ✅ | 已整理成 judge-first 路径 |
| Submission pack | ✅ | `docs/submission/README.md` + form answers 已可直接用 |
| 0G Storage 集成 | ✅ | 已有代码与 live evidence |
| 0G Chain / Anchor 集成 | ✅ | 已有代码与链上证据 |
| OpenClaw 叙事 | ✅ | 已形成 HTTP ingest / workflow service |
| Hydrate / Verify / Trace / Readiness | ✅ | 已可展示 |
| Galileo 证据 | ✅ | 已具备 |
| Demo 视频 | ❌ | 还缺 |
| X / Twitter 推文 | ❌ | 还缺 |
| Repo visibility 确认 | ❌ | 还需手动确认 |
| 主网地址 / Explorer | ⚠️ | 仅在最终规则明确要求主网时需要 |

---

## 5. 最小提交路径（建议按顺序）

### 第一步：录制 3 分钟 Demo

- [ ] 按 `docs/demo/3min-judge-flow.md` 录制
- [ ] 按 `docs/demo/judge-checklist.md` 做 pre-flight
- [ ] 优先展示：
  1. `serve`
  2. `/health`
  3. `/v1/openclaw/ingest` 或 `/v1/openclaw/ingest/batch`
  4. `/v1/openclaw/runs/{id}/checkpoint/latest`
  5. `/v1/openclaw/runs/{id}/hydrate`
  6. `/v1/openclaw/runs/{id}/verify`
  7. `/v1/openclaw/runs/{id}/trace`
  8. `/judge/verify?runId={id}` 或 `workflow verify <run-id>`
  9. 0G Storage / anchor evidence

### 第二步：准备公开传播材料

- [ ] 录屏截图 2 ~ 3 张
- [ ] 发 X 推文
- [ ] 保存链接

### 第三步：确认提交字段

- [ ] Project name
- [ ] 30 词内一句话描述
- [ ] GitHub Repo
- [ ] Galileo 合约地址 / Explorer（或主网，如果最终规则强制）
- [ ] Demo 视频链接
- [ ] README / 说明材料
- [ ] 推文链接

---

## 6. 我对当前项目状态的判断

如果只看“**产品和技术完成度**”，这个项目已经进入 **可以包装成正式黑客松提交** 的阶段。

如果只看“**HackQuest 审核门槛**”，当前最危险的不是代码，而是：

1. **Demo 视频是否按要求展示真实 0G 调用**
2. **X / Twitter 强制项是否补齐**
3. **Repo visibility 是否确认到位**
4. **主网是否被最终规则强制要求**

---

## 7. 下一步建议（只做一件事）

最推荐下一步：

- **先录 Demo 视频并拿到公开视频链接**

其次：

- **发布 X / Twitter 推文并保存链接**
- **如果最终规则要求主网，再补主网部署和 Explorer 证明**
