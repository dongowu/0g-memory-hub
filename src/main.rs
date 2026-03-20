mod chain;
mod cli;
mod models;
mod storage;

use anyhow::Result;
use clap::Parser;
use cli::{Cli, CliHandler, Commands};
use models::{ChainConfig, StorageConfig};
use tracing_subscriber;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize logging
    tracing_subscriber::fmt::init();

    let cli = Cli::parse();

    // Default configuration (can be overridden by env vars or CLI args)
    let storage_config = StorageConfig {
        rpc_url: cli
            .storage_rpc
            .unwrap_or_else(|| "http://localhost:8545".to_string()),
        indexer_url: "http://localhost:9000".to_string(),
        private_key: cli
            .private_key
            .clone()
            .unwrap_or_else(|| "0x0000000000000000000000000000000000000000000000000000000000000000".to_string()),
    };

    let chain_config = ChainConfig {
        rpc_url: cli
            .chain_rpc
            .unwrap_or_else(|| "http://localhost:8545".to_string()),
        contract_address: cli
            .contract_address
            .unwrap_or_else(|| "0x0000000000000000000000000000000000000000".to_string()),
        private_key: cli
            .private_key
            .unwrap_or_else(|| "0x0000000000000000000000000000000000000000000000000000000000000000".to_string()),
        chain_id: 1,
    };

    let handler = CliHandler::new(storage_config, chain_config);

    match cli.command {
        Commands::Upload { file, replicas } => {
            handler.handle_upload(&file, replicas).await?;
        }
        Commands::Download { cid, output, verify } => {
            handler.handle_download(&cid, &output, verify).await?;
        }
        Commands::SetPointer { agent, cid } => {
            handler.handle_set_pointer(&agent, &cid).await?;
        }
        Commands::GetPointer { agent } => {
            handler.handle_get_pointer(&agent).await?;
        }
        Commands::GetHistory { agent } => {
            handler.handle_get_history(&agent).await?;
        }
        Commands::Demo { file, agent } => {
            handler.handle_demo(&file, &agent).await?;
        }
    }

    Ok(())
}
