# 0G Memory Hub 🧠

**AI Agent Eternal Memory System on 0G**

An innovative solution for building immutable, verifiable, and eternally accessible memory systems for AI agents using 0G's decentralized infrastructure.

## 🎯 Project Overview

0G Memory Hub addresses a critical challenge in AI agent development: **how to give agents permanent, tamper-proof memory that persists across sessions and is verifiable on-chain**.

### Core Features

- **Persistent Storage**: Upload agent memories to 0G Storage with content addressing (CID)
- **On-Chain Anchoring**: Anchor memory pointers on 0G Chain for immutability and verifiability
- **Concurrent Uploads**: Rust + Tokio for high-throughput parallel uploads
- **Memory History**: Full audit trail of all memory updates on-chain
- **End-to-End Verification**: Merkle proof verification for data integrity

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    AI Agent                                  │
└────────────────────┬────────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
        ▼                         ▼
┌──────────────────┐      ┌──────────────────┐
│  0G Storage      │      │  0G Chain        │
│  (Persistence)   │      │  (Anchoring)     │
│                  │      │                  │
│ - Upload files   │      │ - MemoryChain    │
│ - CID addressing │      │   contract       │
│ - DA layer       │      │ - Pointer record │
│ - Merkle proof   │      │ - History log    │
└──────────────────┘      └──────────────────┘
        ▲                         ▲
        └────────────┬────────────┘
                     │
        ┌────────────▼────────────┐
        │   0G Memory Hub CLI     │
        │   (Rust + Tokio)        │
        └────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Rust 1.70+
- Cargo
- 0G Testnet RPC endpoint
- Private key for transactions

### Installation

```bash
git clone https://github.com/dongowu/0g-memory-hub.git
cd 0g-memory-hub
cargo build --release
```

### Configuration

Set environment variables:

```bash
export OG_STORAGE_RPC="https://testnet-rpc.0g.ai"
export OG_CHAIN_RPC="https://testnet-chain-rpc.0g.ai"
export PRIVATE_KEY="0x..."
export CONTRACT_ADDRESS="0x..."
```

Or pass as CLI arguments:

```bash
cargo run -- --storage-rpc <URL> --chain-rpc <URL> --private-key <KEY> --contract-address <ADDR> <COMMAND>
```

## 📝 Usage

### 1. Upload a Memory File

```bash
cargo run -- upload ./memory.json --replicas 2
```

Output:
```
📤 Uploading file to 0G Storage: "./memory.json"
✅ Upload successful!
   CID: 0g_a1b2c3d4e5f6g7h8
   TX Hash: 0x1234567890abcdef
   File Size: 1024 bytes
```

### 2. Set Memory Pointer On-Chain

```bash
cargo run -- set-pointer 0x1234567890123456789012345678901234567890 0g_a1b2c3d4e5f6g7h8
```

Output:
```
🔗 Setting memory pointer on-chain...
   Agent: 0x1234567890123456789012345678901234567890
   CID: 0g_a1b2c3d4e5f6g7h8
✅ Pointer set successfully!
   TX Hash: 0x9876543210fedcba
   Block: 12345
```

### 3. Get Current Memory Head

```bash
cargo run -- get-pointer 0x1234567890123456789012345678901234567890
```

### 4. View Memory History

```bash
cargo run -- get-history 0x1234567890123456789012345678901234567890
```

### 5. Download Memory File

```bash
cargo run -- download 0g_a1b2c3d4e5f6g7h8 --output ./memory_restored.json --verify
```

### 6. End-to-End Demo

```bash
cargo run -- demo ./memory.json 0x1234567890123456789012345678901234567890
```

This runs the complete workflow:
1. Upload file to 0G Storage
2. Anchor CID on 0G Chain
3. Verify on-chain pointer

## 🔗 0G Components Integration

### 0G Storage

- **Purpose**: Persistent, content-addressed storage for memory files
- **Integration**: `src/storage/mod.rs`
- **API Used**: 0G Storage SDK for upload/download with CID generation
- **Benefits**:
  - Immutable content addressing
  - Data availability guarantees via DA layer
  - Merkle proof verification

### 0G Chain

- **Purpose**: On-chain anchoring of memory pointers
- **Integration**: `src/chain/mod.rs`
- **Contract**: `contracts/MemoryChain.sol`
- **Functions**:
  - `setMemoryHead(bytes32 cid)` - Update memory pointer
  - `getMemoryHead(address agent)` - Retrieve current pointer
  - `getMemoryHistory(address agent)` - Full audit trail
- **Benefits**:
  - Immutable record of memory updates
  - Verifiable on-chain history
  - EVM-compatible smart contracts

## 📊 Performance Targets

- **Upload Throughput**: 500+ TPS (with 100 concurrent uploads)
- **Chain Confirmation**: <1 second (0G Chain finality)
- **Storage Latency**: <2 seconds (end-to-end)
- **Memory Size**: 100KB - 1MB per entry (scalable)

## 🧪 Testing

Run the test suite:

```bash
cargo test
```

Run with logging:

```bash
RUST_LOG=debug cargo test -- --nocapture
```

## 📋 Project Structure

```
0g-memory-hub/
├── src/
│   ├── main.rs              # CLI entry point
│   ├── cli/mod.rs           # Command-line interface
│   ├── storage/mod.rs       # 0G Storage client
│   ├── chain/mod.rs         # 0G Chain client
│   └── models/mod.rs        # Data structures
├── contracts/
│   └── MemoryChain.sol      # Smart contract
├── Cargo.toml               # Rust dependencies
└── README.md                # This file
```

## 🔐 Security Considerations

1. **Private Key Management**: Store private keys securely (use environment variables or hardware wallets)
2. **Memory Encryption**: Sensitive memories can be encrypted before upload
3. **Access Control**: Contract enforces that only the agent owner can update pointers
4. **Data Verification**: Merkle proofs ensure data integrity

## 🛠️ Deployment Steps

### 1. Deploy Smart Contract

```bash
# Compile Solidity contract
solc --optimize --bin --abi contracts/MemoryChain.sol

# Deploy to 0G Testnet
# (Use Hardhat, Foundry, or web3.py)
```

### 2. Configure Contract Address

Update `CONTRACT_ADDRESS` environment variable with deployed contract address.

### 3. Run CLI

```bash
cargo run --release -- demo ./test_memory.json 0x<agent_address>
```

### 4. Verify on Explorer

Check transaction on 0G Explorer:
- Storage upload: View CID in 0G Storage Explorer
- Chain anchor: View transaction in 0G Chain Explorer

## 📈 Roadmap

- [x] Basic storage upload/download
- [x] On-chain pointer anchoring
- [x] CLI tool
- [ ] Concurrent upload optimization (Tokio)
- [ ] iNFT integration for encrypted metadata
- [ ] X402 payment protocol for agent services
- [ ] Performance benchmarking (500+ TPS)
- [ ] Web dashboard for memory visualization
- [ ] Multi-agent coordination

## 🤝 Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details

## 🔗 Resources

- [0G Documentation](https://docs.0g.ai/)
- [0G Storage SDK](https://github.com/0glabs/0g-storage-client)
- [0G Chain RPC](https://testnet-rpc.0g.ai)
- [0G Explorer](https://testnet-explorer.0g.ai)

## 📞 Support

For issues and questions:
- GitHub Issues: [0g-memory-hub/issues](https://github.com/dongowu/0g-memory-hub/issues)
- Discord: [0G Community](https://discord.gg/0g)

---

**Built for the 0G APAC Hackathon 2026** 🚀

*Code is law, but AI is the future of that law.*
