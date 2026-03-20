use anyhow::{anyhow, Result};
use ethers::prelude::*;
use std::sync::Arc;

use crate::models::{ChainConfig, ChainTxResult};

/// 0G Chain client for interacting with MemoryChain contract
pub struct ChainClient {
    config: ChainConfig,
}

impl ChainClient {
    pub fn new(config: ChainConfig) -> Self {
        Self { config }
    }

    /// Set memory head pointer on-chain
    pub async fn set_memory_head(&self, agent: &str, cid: &str) -> Result<ChainTxResult> {
        // Validate inputs
        if agent.is_empty() || cid.is_empty() {
            return Err(anyhow!("Agent and CID cannot be empty"));
        }

        // In production, this would:
        // 1. Connect to 0G Chain RPC
        // 2. Load the MemoryChain contract ABI
        // 3. Call setMemoryHead(cid_bytes32)
        // 4. Wait for confirmation

        // Mock implementation
        let tx_hash = format!("0x{}", hex::encode(cid.as_bytes()[..16].to_vec()));
        let block_number = 12345u64;

        Ok(ChainTxResult {
            tx_hash,
            block_number,
            status: "success".to_string(),
        })
    }

    /// Get memory head pointer from chain
    pub async fn get_memory_head(&self, agent: &str) -> Result<String> {
        if agent.is_empty() {
            return Err(anyhow!("Agent address cannot be empty"));
        }

        // In production, this would call getMemoryHead(agent) on the contract
        // Mock implementation
        Ok(format!("0g_memory_head_{}", &agent[..8]))
    }

    /// Get memory history for an agent
    pub async fn get_memory_history(&self, agent: &str) -> Result<Vec<String>> {
        if agent.is_empty() {
            return Err(anyhow!("Agent address cannot be empty"));
        }

        // In production, this would call getMemoryHistory(agent)
        // Mock implementation
        Ok(vec![
            format!("0g_memory_1_{}", &agent[..8]),
            format!("0g_memory_2_{}", &agent[..8]),
        ])
    }

    /// Verify that a CID is anchored on-chain
    pub async fn verify_cid_on_chain(&self, agent: &str, cid: &str) -> Result<bool> {
        let head = self.get_memory_head(agent).await?;
        Ok(head.contains(cid))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_set_memory_head() {
        let config = ChainConfig {
            rpc_url: "http://localhost:8545".to_string(),
            contract_address: "0x0000000000000000000000000000000000000000".to_string(),
            private_key: "0x0000000000000000000000000000000000000000000000000000000000000000"
                .to_string(),
            chain_id: 1,
        };

        let client = ChainClient::new(config);
        let result = client
            .set_memory_head("0x1234567890123456789012345678901234567890", "0g_test_cid")
            .await;

        assert!(result.is_ok());
        let tx = result.unwrap();
        assert!(!tx.tx_hash.is_empty());
    }
}
