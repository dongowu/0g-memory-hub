# 0G 集成指南

## 0G 组件使用说明

本项目集成了 0G 的两个核心组件：**0G Storage** 和 **0G Chain**。

### 1. 0G Storage 集成

**位置**: `src/storage/mod.rs`

**功能**:
- 文件上传到 0G Storage
- 获取内容标识符 (CID)
- 文件下载和验证
- Merkle 证明验证

**API 调用**:

```rust
// 初始化存储客户端
let storage_config = StorageConfig {
    rpc_url: "https://testnet-rpc.0g.ai".to_string(),
    indexer_url: "https://testnet-indexer.0g.ai".to_string(),
    private_key: "0x...".to_string(),
};
let client = StorageClient::new(storage_config);

// 上传文件
let result = client.upload_file("memory.json").await?;
println!("CID: {}", result.cid);

// 下载文件
client.download_file(&cid, "output.json").await?;

// 验证完整性
let is_valid = client.verify_file(&cid, "output.json").await?;
```

**解决的问题**:
- ✅ 持久化存储：AI 代理的记忆数据永久保存
- ✅ 内容寻址：通过 CID 确保数据不可篡改
- ✅ 数据可用性：0G DA 层保证数据分片的可靠分发
- ✅ 完整性验证：Merkle 证明确保下载的数据完整

### 2. 0G Chain 集成

**位置**: `src/chain/mod.rs`

**智能合约**: `contracts/MemoryChain.sol`

**功能**:
- 在链上记录内存指针 (CID)
- 维护完整的内存历史
- 验证代理身份
- 查询内存链

**合约接口**:

```solidity
// 设置内存头指针
function setMemoryHead(bytes32 cid) external

// 获取当前内存头
function getMemoryHead(address agent) external view returns (bytes32)

// 获取完整历史
function getMemoryHistory(address agent) external view returns (bytes32[] memory)

// 获取特定历史记录
function getMemoryAt(address agent, uint256 index) external view returns (bytes32)
```

**Rust 调用示例**:

```rust
// 初始化链客户端
let chain_config = ChainConfig {
    rpc_url: "https://testnet-chain-rpc.0g.ai".to_string(),
    contract_address: "0x...".to_string(),
    private_key: "0x...".to_string(),
    chain_id: 1,
};
let client = ChainClient::new(chain_config);

// 设置内存指针
let result = client.set_memory_head(agent_addr, &cid).await?;
println!("TX Hash: {}", result.tx_hash);

// 获取当前指针
let head = client.get_memory_head(agent_addr).await?;

// 获取历史
let history = client.get_memory_history(agent_addr).await?;
```

**解决的问题**:
- ✅ 不可篡改性：链上记录无法修改
- ✅ 可验证性：任何人都可以验证内存链的完整性
- ✅ 审计追踪：完整的历史记录便于审计
- ✅ 身份验证：合约确保只有代理所有者可以更新指针

## 0G 主网部署

### 前置条件

1. 获取 0G 主网 RPC 端点
2. 准备部署账户（需要 0G 主网代币用于 Gas）
3. 安装 Foundry 或 Hardhat

### 部署步骤

#### 使用 Foundry

```bash
# 1. 安装 Foundry
curl -L https://foundry.paradigm.xyz | bash
foundryup

# 2. 初始化项目
forge init --no-git

# 3. 复制合约
cp contracts/MemoryChain.sol src/

# 4. 部署
forge create src/MemoryChain.sol:MemoryChain \
  --rpc-url https://rpc.0g.ai \
  --private-key 0x... \
  --etherscan-api-key <API_KEY> \
  --verify
```

#### 使用 Hardhat

```bash
# 1. 初始化项目
npx hardhat init

# 2. 配置 hardhat.config.js
# 添加 0G 网络配置

# 3. 部署
npx hardhat run scripts/deploy.js --network 0g-mainnet
```

### 验证部署

部署后，在 0G Explorer 中验证：

```
https://explorer.0g.ai/address/<CONTRACT_ADDRESS>
```

## 0G 测试网配置

### 测试网信息

| 参数 | 值 |
|------|-----|
| RPC URL | https://testnet-rpc.0g.ai |
| Chain ID | (待确认) |
| Explorer | https://testnet-explorer.0g.ai |
| 水龙头 | https://testnet-faucet.0g.ai |

### 获取测试币

1. 访问 https://testnet-faucet.0g.ai
2. 输入钱包地址
3. 领取测试币

### 环境变量配置

```bash
# .env 文件
OG_STORAGE_RPC=https://testnet-rpc.0g.ai
OG_CHAIN_RPC=https://testnet-chain-rpc.0g.ai
PRIVATE_KEY=0x...
CONTRACT_ADDRESS=0x...
```

## 性能指标

### 0G Storage

- **吞吐量**: 支持高并发上传
- **延迟**: 文件上传至 DA 确认 < 2 秒
- **可靠性**: 纠删编码 + 多副本保证

### 0G Chain

- **TPS**: 11,000+ TPS (每分片)
- **确认时间**: < 1 秒
- **Gas 费用**: 极低（高吞吐系统）

## 故障排查

### 上传失败

```
错误: "RPC connection failed"
解决: 检查 RPC URL 是否正确，网络连接是否正常
```

### 合约交互失败

```
错误: "Insufficient balance"
解决: 确保账户有足够的 0G 代币用于 Gas
```

### CID 验证失败

```
错误: "Merkle proof verification failed"
解决: 确保文件未被修改，重新下载并验证
```

## 最佳实践

1. **私钥管理**
   - 使用环境变量存储私钥
   - 不要在代码中硬编码私钥
   - 使用硬件钱包进行主网交易

2. **错误处理**
   - 实现重试机制（exponential backoff）
   - 记录所有交易哈希用于追踪
   - 验证链上状态后再继续

3. **性能优化**
   - 使用并发上传提高吞吐量
   - 批量处理多个内存更新
   - 缓存常用的 CID 和指针

4. **安全性**
   - 对敏感内存加密
   - 验证所有下载的文件
   - 定期审计链上历史

## 资源链接

- [0G 官方文档](https://docs.0g.ai/)
- [0G Storage SDK](https://github.com/0glabs/0g-storage-client)
- [0G Chain RPC](https://rpc.0g.ai)
- [0G Explorer](https://explorer.0g.ai)
- [0G Discord](https://discord.gg/0g)

## 支持

遇到问题？

1. 查看 [0G 文档](https://docs.0g.ai/)
2. 在 [GitHub Issues](https://github.com/dongowu/0g-memory-hub/issues) 提问
3. 加入 [0G Discord](https://discord.gg/0g) 社区
