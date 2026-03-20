use serde::{Deserialize, Serialize};
use std::time::{SystemTime, UNIX_EPOCH};

/// Represents a memory entry in the 0G Memory Hub
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemoryEntry {
    /// Content identifier (CID) from 0G Storage
    pub cid: String,
    /// Timestamp when memory was created
    pub timestamp: u64,
    /// Agent address that owns this memory
    pub agent: String,
    /// Memory content hash (SHA256)
    pub content_hash: String,
    /// Memory size in bytes
    pub size: u64,
    /// Optional metadata
    pub metadata: Option<serde_json::Value>,
}

impl MemoryEntry {
    pub fn new(cid: String, agent: String, content_hash: String, size: u64) -> Self {
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();

        Self {
            cid,
            timestamp,
            agent,
            content_hash,
            size,
            metadata: None,
        }
    }

    pub fn with_metadata(mut self, metadata: serde_json::Value) -> Self {
        self.metadata = Some(metadata);
        self
    }
}

/// Configuration for 0G Storage connection
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StorageConfig {
    pub rpc_url: String,
    pub indexer_url: String,
    pub private_key: String,
}

/// Configuration for 0G Chain connection
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainConfig {
    pub rpc_url: String,
    pub contract_address: String,
    pub private_key: String,
    pub chain_id: u64,
}

/// Upload result from 0G Storage
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UploadResult {
    pub cid: String,
    pub tx_hash: String,
    pub file_size: u64,
    pub timestamp: u64,
}

/// Chain transaction result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainTxResult {
    pub tx_hash: String,
    pub block_number: u64,
    pub status: String,
}
