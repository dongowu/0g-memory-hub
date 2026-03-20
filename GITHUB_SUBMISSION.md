# GitHub 提交指南

## 仓库创建步骤

### 1. 在 GitHub 上创建私有仓库

1. 访问 https://github.com/new
2. 填写信息：
   - **Repository name**: `0g-memory-hub`
   - **Description**: `AI Agent Eternal Memory System on 0G`
   - **Visibility**: `Private` (先设为私有)
   - **Initialize**: 不勾选任何选项（我们已有本地代码）
3. 点击 "Create repository"

### 2. 配置本地 Git 远程

复制仓库 URL 后，在本地执行：

```bash
cd D:/ownCode/game/0GMemoryHub

# 添加远程
git remote add origin https://github.com/dongowu/0g-memory-hub.git

# 验证
git remote -v
```

### 3. 推送代码到 GitHub

```bash
# 推送主分支
git branch -M main
git push -u origin main
```

### 4. 最终提交前转为公开

在黑客松提交前（5月9日前），将仓库改为公开：
1. 进入仓库 Settings
2. 找到 "Danger Zone"
3. 点击 "Change visibility"
4. 选择 "Public"

## 提交要求检查清单

在提交到黑客松前，确保满足以下所有条件：

- [ ] 代码已推送到 GitHub
- [ ] 仓库已转为 Public
- [ ] 有实质性的 commit 记录（至少 5+ commits）
- [ ] 0G 主网合约已部署
- [ ] 合约地址已在 README 中记录
- [ ] Demo 视频已录制（≤3分钟）
- [ ] 在 X (Twitter) 发布了项目推文
- [ ] 所有 commit 都是 dongowu 账户提交

## 后续开发流程

每次开发完成后：

```bash
# 查看变更
git status

# 添加文件
git add <files>

# 提交
git commit -m "描述性的提交信息"

# 推送
git push origin main
```

## 0G 主网部署

### 1. 获取 0G 主网信息

- RPC URL: https://rpc.0g.ai (待确认)
- Chain ID: (待确认)
- Explorer: https://explorer.0g.ai (待确认)

### 2. 部署合约

使用 Foundry 或 Hardhat：

```bash
# 使用 Foundry
forge create contracts/MemoryChain.sol:MemoryChain \
  --rpc-url <0G_RPC_URL> \
  --private-key <YOUR_PRIVATE_KEY>
```

### 3. 记录合约地址

部署后，将合约地址更新到：
- README.md 中的 "0G Chain Integration" 部分
- 环境变量 `CONTRACT_ADDRESS`

## 推文模板

在 X (Twitter) 上发布：

```
🚀 Building eternal memory for AI agents on @0G_labs!

0G Memory Hub: Immutable, verifiable, on-chain memory system
- Upload to 0G Storage with content addressing
- Anchor on 0G Chain for verifiability
- Concurrent uploads with Rust + Tokio

Demo: [VIDEO_LINK]
Code: https://github.com/dongowu/0g-memory-hub

#0GHackathon #BuildOn0G @0G_labs @0g_CN @HackQuest_
```

## 常见问题

**Q: 如何更改仓库可见性？**
A: Settings → Danger Zone → Change visibility

**Q: 如何确保只有我的账户提交代码？**
A: 所有 commit 都已配置为 dongowu 账户，git 会自动使用该身份

**Q: 如何查看 commit 历史？**
A: `git log --oneline`

**Q: 如何撤销最后一个 commit？**
A: `git reset --soft HEAD~1` (保留更改) 或 `git reset --hard HEAD~1` (丢弃更改)
