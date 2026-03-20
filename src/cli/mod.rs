use anyhow::Result;
use clap::{Parser, Subcommand};
use std::path::PathBuf;

use crate::chain::ChainClient;
use crate::models::{ChainConfig, StorageConfig};
use crate::storage::StorageClient;

#[derive(Parser)]
#[command(name = "0G Memory Hub")]
#[command(about = "AI Agent Eternal Memory System on 0G", long_about = None)]
pub struct Cli {
    #[command(subcommand)]
    pub command: Commands,

    /// 0G Storage RPC URL
    #[arg(global = true, long, env = "OG_STORAGE_RPC")]
    pub storage_rpc: Option<String>,

    /// 0G Chain RPC URL
    #[arg(global = true, long, env = "OG_CHAIN_RPC")]
    pub chain_rpc: Option<String>,

    /// Private key for signing transactions
    #[arg(global = true, long, env = "PRIVATE_KEY")]
    pub private_key: Option<String>,

    /// Contract address on 0G Chain
    #[arg(global = true, long, env = "CONTRACT_ADDRESS")]
    pub contract_address: Option<String>,
}

#[derive(Subcommand)]
pub enum Commands {
    /// Upload a file to 0G Storage
    Upload {
        /// Path to the file to upload
        #[arg(value_name = "FILE")]
        file: PathBuf,

        /// Number of replicas
        #[arg(short, long, default_value = "2")]
        replicas: u32,
    },

    /// Download a file from 0G Storage
    Download {
        /// Content identifier (CID)
        #[arg(value_name = "CID")]
        cid: String,

        /// Output file path
        #[arg(short, long)]
        output: PathBuf,

        /// Verify Merkle proof
        #[arg(short, long)]
        verify: bool,
    },

    /// Set memory head pointer on-chain
    SetPointer {
        /// Agent address
        #[arg(value_name = "AGENT")]
        agent: String,

        /// Memory CID
        #[arg(value_name = "CID")]
        cid: String,
    },

    /// Get memory head pointer from chain
    GetPointer {
        /// Agent address
        #[arg(value_name = "AGENT")]
        agent: String,
    },

    /// Get full memory history for an agent
    GetHistory {
        /// Agent address
        #[arg(value_name = "AGENT")]
        agent: String,
    },

    /// End-to-end demo: upload file and anchor on-chain
    Demo {
        /// Path to the file to upload
        #[arg(value_name = "FILE")]
        file: PathBuf,

        /// Agent address
        #[arg(value_name = "AGENT")]
        agent: String,
    },
}

pub struct CliHandler {
    storage_client: StorageClient,
    chain_client: ChainClient,
}

impl CliHandler {
    pub fn new(storage_config: StorageConfig, chain_config: ChainConfig) -> Self {
        Self {
            storage_client: StorageClient::new(storage_config),
            chain_client: ChainClient::new(chain_config),
        }
    }

    pub async fn handle_upload(&self, file: &PathBuf, _replicas: u32) -> Result<()> {
        println!("📤 Uploading file to 0G Storage: {:?}", file);

        let result = self.storage_client.upload_file(file.to_str().unwrap()).await?;

        println!("✅ Upload successful!");
        println!("   CID: {}", result.cid);
        println!("   TX Hash: {}", result.tx_hash);
        println!("   File Size: {} bytes", result.file_size);

        Ok(())
    }

    pub async fn handle_download(&self, cid: &str, output: &PathBuf, verify: bool) -> Result<()> {
        println!("📥 Downloading from 0G Storage: {}", cid);

        self.storage_client
            .download_file(cid, output.to_str().unwrap())
            .await?;

        if verify {
            let is_valid = self
                .storage_client
                .verify_file(cid, output.to_str().unwrap())
                .await?;
            println!("✅ Download complete! Verification: {}", is_valid);
        } else {
            println!("✅ Download complete!");
        }

        Ok(())
    }

    pub async fn handle_set_pointer(&self, agent: &str, cid: &str) -> Result<()> {
        println!("🔗 Setting memory pointer on-chain...");
        println!("   Agent: {}", agent);
        println!("   CID: {}", cid);

        let result = self.chain_client.set_memory_head(agent, cid).await?;

        println!("✅ Pointer set successfully!");
        println!("   TX Hash: {}", result.tx_hash);
        println!("   Block: {}", result.block_number);

        Ok(())
    }

    pub async fn handle_get_pointer(&self, agent: &str) -> Result<()> {
        println!("🔍 Fetching memory pointer from chain...");

        let cid = self.chain_client.get_memory_head(agent).await?;

        println!("✅ Current memory head: {}", cid);

        Ok(())
    }

    pub async fn handle_get_history(&self, agent: &str) -> Result<()> {
        println!("📚 Fetching memory history...");

        let history = self.chain_client.get_memory_history(agent).await?;

        println!("✅ Memory history ({} entries):", history.len());
        for (i, cid) in history.iter().enumerate() {
            println!("   [{}] {}", i + 1, cid);
        }

        Ok(())
    }

    pub async fn handle_demo(&self, file: &PathBuf, agent: &str) -> Result<()> {
        println!("\n🚀 Starting end-to-end demo...\n");

        // Step 1: Upload
        println!("Step 1️⃣  Upload file to 0G Storage");
        let upload_result = self.storage_client.upload_file(file.to_str().unwrap()).await?;
        println!("   ✅ CID: {}\n", upload_result.cid);

        // Step 2: Set pointer
        println!("Step 2️⃣  Anchor CID on 0G Chain");
        let chain_result = self
            .chain_client
            .set_memory_head(agent, &upload_result.cid)
            .await?;
        println!("   ✅ TX Hash: {}\n", chain_result.tx_hash);

        // Step 3: Verify
        println!("Step 3️⃣  Verify on-chain pointer");
        let head = self.chain_client.get_memory_head(agent).await?;
        println!("   ✅ Memory head: {}\n", head);

        println!("🎉 Demo complete! Memory is now eternal on 0G.\n");

        Ok(())
    }
}
