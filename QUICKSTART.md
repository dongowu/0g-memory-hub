# 快速开始指南

## 5 分钟快速上手

### 1. 克隆并进入项目

```bash
cd D:/ownCode/game/0GMemoryHub
```

### 2. 配置环境变量

```bash
# 复制示例配置
cp .env.example .env

# 编辑 .env 文件，填入你的配置
# - OG_STORAGE_RPC: 0G Storage RPC 端点
# - OG_CHAIN_RPC: 0G Chain RPC 端点
# - PRIVATE_KEY: 你的私钥
# - CONTRACT_ADDRESS: 部署的合约地址
```

### 3. 编译项目

```bash
cargo build --release
```

### 4. 运行演示

```bash
# 查看所有命令
cargo run --release -- --help

# 运行端到端演示
cargo run --release -- demo ./test_memory.json 0x1234567890123456789012345678901234567890
```

## 常用命令

### 上传文件到 0G Storage

```bash
cargo run --release -- upload ./memory.json --replicas 2
```

### 在链上设置内存指针

```bash
cargo run --release -- set-pointer 0x1234567890123456789012345678901234567890 0g_cid_hash
```

### 获取当前内存指针

```bash
cargo run --release -- get-pointer 0x1234567890123456789012345678901234567890
```

### 获取完整内存历史

```bash
cargo run --release -- get-history 0x1234567890123456789012345678901234567890
```

### 从 0G Storage 下载文件

```bash
cargo run --release -- download 0g_cid_hash --output ./restored.json --verify
```

## 使用 Makefile 快捷命令

```bash
# 查看所有命令
make help

# 编译
make build

# 运行测试
make test

# 运行演示
make demo

# 清理
make clean

# 代码格式化
make fmt

# 代码检查
make lint
```

## 部署智能合约

### 使用 Foundry

```bash
# 1. 安装 Foundry
curl -L https://foundry.paradigm.xyz | bash
foundryup

# 2. 部署合约
forge create contracts/MemoryChain.sol:MemoryChain \
  --rpc-url https://testnet-rpc.0g.ai \
  --private-key 0x... \
  --verify
```

### 使用 Hardhat

```bash
# 1. 初始化 Hardhat 项目
npx hardhat init

# 2. 配置网络
# 在 hardhat.config.js 中添加 0G 网络

# 3. 部署
npx hardhat run scripts/deploy.js --network 0g-testnet
```

## 测试网信息

| 参数 | 值 |
|------|-----|
| RPC URL | https://testnet-rpc.0g.ai |
| Chain ID | (待确认) |
| Explorer | https://testnet-explorer.0g.ai |
| 水龙头 | https://testnet-faucet.0g.ai |

## 获取测试币

1. 访问 https://testnet-faucet.0g.ai
2. 输入你的钱包地址
3. 领取测试币

## 常见问题

**Q: 编译失败？**
A: 确保安装了 Rust 1.70+
```bash
rustup update
```

**Q: 如何查看日志？**
A: 设置日志级别
```bash
RUST_LOG=debug cargo run --release -- <command>
```

**Q: 如何测试本地？**
A: 使用本地 RPC 端点
```bash
export OG_STORAGE_RPC=http://localhost:8545
export OG_CHAIN_RPC=http://localhost:8546
```

**Q: 私钥安全吗？**
A: 使用环境变量存储，不要在代码中硬编码。生产环境建议使用硬件钱包。

## 下一步

1. ✅ 部署智能合约到 0G 主网
2. ✅ 录制 Demo 视频
3. ✅ 发布 Twitter 推文
4. ✅ 提交到黑客松平台

## 获取帮助

- 📖 [完整文档](README.md)
- 🔗 [0G 集成指南](0G_INTEGRATION.md)
- 📝 [GitHub 提交指南](GITHUB_SUBMISSION.md)
- 🎯 [项目总结](PROJECT_SUMMARY.md)

---

**准备好了吗？让我们开始构建永恒的 AI 记忆！** 🚀
