use anyhow::{anyhow, Result};
use sha2::{Digest, Sha256};
use std::fs;
use std::path::Path;

use crate::models::{StorageConfig, UploadResult};

/// 0G Storage client for uploading and downloading files
pub struct StorageClient {
    config: StorageConfig,
}

impl StorageClient {
    pub fn new(config: StorageConfig) -> Self {
        Self { config }
    }

    /// Calculate SHA256 hash of a file
    pub fn calculate_file_hash(file_path: &str) -> Result<String> {
        let data = fs::read(file_path)?;
        let mut hasher = Sha256::new();
        hasher.update(&data);
        Ok(format!("{:x}", hasher.finalize()))
    }

    /// Upload a file to 0G Storage
    /// In production, this would call the actual 0G Storage SDK
    pub async fn upload_file(&self, file_path: &str) -> Result<UploadResult> {
        let path = Path::new(file_path);
        if !path.exists() {
            return Err(anyhow!("File not found: {}", file_path));
        }

        let file_size = fs::metadata(file_path)?.len();
        let content_hash = Self::calculate_file_hash(file_path)?;

        // Generate a mock CID (in production, this comes from 0G Storage)
        let cid = format!("0g_{}", &content_hash[..16]);

        // Mock transaction hash
        let tx_hash = format!("0x{}", hex::encode(&content_hash.as_bytes()[..16]));

        let timestamp = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)?
            .as_secs();

        Ok(UploadResult {
            cid,
            tx_hash,
            file_size,
            timestamp,
        })
    }

    /// Download a file from 0G Storage by CID
    pub async fn download_file(&self, cid: &str, output_path: &str) -> Result<()> {
        // In production, this would fetch from 0G Storage
        // For now, we'll create a mock response
        let mock_content = format!("Mock content for CID: {}", cid);
        fs::write(output_path, mock_content)?;
        Ok(())
    }

    /// Verify file integrity using Merkle proof
    pub async fn verify_file(&self, cid: &str, file_path: &str) -> Result<bool> {
        let content_hash = Self::calculate_file_hash(file_path)?;
        // In production, verify against 0G DA layer
        Ok(cid.contains(&content_hash[..8]))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_calculate_file_hash() {
        // Create a temporary test file
        let test_file = "/tmp/test_memory.txt";
        fs::write(test_file, "test content").unwrap();

        let hash = StorageClient::calculate_file_hash(test_file).unwrap();
        assert!(!hash.is_empty());
        assert_eq!(hash.len(), 64); // SHA256 hex is 64 chars

        fs::remove_file(test_file).ok();
    }
}
