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
- replay / resume / readiness / live evidence

但如果要变成 **可正式提交到 HackQuest 的完整材料**，目前最关键的缺口仍然是：

1. **主网合约地址**
2. **主网 Explorer 链接**
3. **Demo 视频（<= 3 分钟）**
4. **X/Twitter 公开推文证明**
5. **HackQuest 表单文案整理**

---

## 2. HackQuest 必交项清单

### A. Basic Project Information

- [x] 项目名称已明确  
  - 建议使用：`0G OpenClaw Memory Runtime`
- [x] 一句话描述可写  
  - 建议版本：`Durable OpenClaw workflow memory on 0G using Go orchestration, Rust checkpoints, 0G Storage, and on-chain MemoryAnchor verification.`
- [x] 项目方向与赛道匹配  
  - Track 1: **Agentic Infrastructure & OpenClaw Lab**

### B. Code Repository

- [x] GitHub 仓库已存在
- [x] `main` 已更新到最新实现
- [x] 有实质性 commit 历史
  - 当前本地可见提交数：`12`
- [ ] 仓库公开可访问（需手动确认 GitHub visibility）
- [ ] 如仍为私有，需在提交前切为 public 或确认评委权限

### C. 0G Integration Proof（核心门槛）

- [x] 至少集成 1 个 0G 核心组件
  - 当前实际已覆盖：
    - 0G Storage
    - 0G Chain
- [x] 已有链上/存储证据文档
  - `docs/evidence/2026-03-22-live-storage-chain-proof.md`
  - `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`
  - `docs/evidence/2026-03-23-live-http-readiness-proof.md`
- [x] 已有合约代码
  - `contracts/MemoryAnchor.sol`
- [x] 已有测试网 / Galileo 证据
  - 示例证据中的合约地址：`0xc164d2f784f0e23bf380e9367b76e42dfa3c45e7`
- [ ] **主网合约地址**
- [ ] **主网 Explorer 链接**
- [ ] 提交材料里明确说明“调用了哪个 0G API/SDK、解决什么问题”

> 注意：当前最像正式提交 blocker 的是 **“主网证明”**，而不是代码能力。

### D. Demo Video

- [x] 已有 demo 脚本与 demo 流程文档
  - `docs/demo/3min-judge-flow.md`
  - `docs/demo/judge-checklist.md`
- [x] 已有可展示内容
  - HTTP ingest
  - workflow replay / resume
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

- [x] README 已存在且较完整
- [x] QUICKSTART 已存在
- [x] 架构说明已存在
- [x] Demo 路径已存在
- [x] Evidence 文档已存在
- [x] 可说明 0G 组件用途
- [ ] 补一个最终版“提交摘要”段落到 README 顶部（建议）
- [ ] 补主网地址 / Explorer 链接到 README（等部署后）
- [ ] 补 Demo 视频链接到 README（等录制后）

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
- `QUICKSTART.md`
- `docs/demo/3min-judge-flow.md`
- `docs/demo/judge-checklist.md`

### 证据文档

- `docs/evidence/2026-03-22-live-storage-chain-proof.md`
- `docs/evidence/2026-03-23-live-orchestrator-workflow-proof.md`
- `docs/evidence/2026-03-23-live-http-readiness-proof.md`

### 当前最适合提交表单的一句话卖点

> A durable OpenClaw workflow memory runtime on 0G that checkpoints agent execution in Rust, orchestrates persistence in Go, stores checkpoints on 0G Storage, and anchors verification metadata on-chain.

---

## 4. 当前状态总表

| 类别 | 状态 | 说明 |
|---|---|---|
| 赛道匹配 | ✅ | 与 Agentic Infrastructure + OpenClaw 高度匹配 |
| GitHub 仓库 | ✅ | 已存在，`main` 已更新 |
| 有效代码量 | ✅ | 已有完整 Go + Rust + Solidity 主线 |
| README / QUICKSTART | ✅ | 已具备 |
| 0G Storage 集成 | ✅ | 已有代码与 live evidence |
| 0G Chain / Anchor 集成 | ✅ | 已有代码与链上证据 |
| OpenClaw 叙事 | ✅ | 已形成 HTTP ingest / workflow service |
| Replay / Resume / Readiness | ✅ | 已可展示 |
| 测试网 / Galileo 证据 | ✅ | 已具备 |
| 主网地址 | ❌ | 还缺 |
| 主网 Explorer | ❌ | 还缺 |
| Demo 视频 | ❌ | 还缺 |
| X/Twitter 推文 | ❌ | 还缺 |
| HackQuest 表单最终文案 | ⚠️ | 可以现在整理 |

---

## 5. 最小提交路径（建议按顺序）

### 第一步：补齐主网证明

- [ ] 部署 `MemoryAnchor.sol` 到 0G 主网
- [ ] 记录合约地址
- [ ] 记录 Explorer 链接
- [ ] 至少完成一次主网可验证交互

### 第二步：录制 3 分钟 Demo

- [ ] 按 `docs/demo/3min-judge-flow.md` 录制
- [ ] 优先展示：
  1. `serve`
  2. `/health`
  3. `/v1/openclaw/ingest`
  4. `/v1/workflows/{id}/replay`
  5. 0G Storage / anchor evidence

### 第三步：准备公开传播材料

- [ ] 录屏截图 2~3 张
- [ ] 发 X 推文
- [ ] 保存链接

### 第四步：填 HackQuest

- [ ] Project name
- [ ] 30 词内一句话描述
- [ ] GitHub Repo
- [ ] 主网合约地址
- [ ] Explorer 链接
- [ ] Demo 视频链接
- [ ] README / 说明材料
- [ ] 推文链接

---

## 6. 我对你现在的判断

如果只看“**产品和技术完成度**”，这个项目已经进入 **可以包装成正式黑客松提交** 的阶段。  

如果只看“**HackQuest 审核门槛**”，当前最危险的不是代码，而是：

1. **主网证明是否满足**
2. **Demo 视频是否按要求展示真实 0G 调用**
3. **X/Twitter 强制项是否补齐**

---

## 7. 下一步建议（只做一件事）

最推荐下一步：

- **直接准备 HackQuest 表单最终文案 + Demo 视频脚本**

其次：

- **补主网部署和 Explorer 证明**

