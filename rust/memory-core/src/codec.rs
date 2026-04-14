//! Checkpoint encoding: CBOR serialization with optional zstd compression.
//!
//! Wire format (compressed):
//!   [0x5A]          – magic byte indicating zstd-compressed CBOR
//!   [4 bytes LE]    – uncompressed CBOR length
//!   [zstd frame]    – compressed CBOR data
//!
//! Wire format (uncompressed):
//!   [0xCB]          – magic byte indicating raw CBOR
//!   [CBOR data]     – raw CBOR payload

use serde::{de::DeserializeOwned, Serialize};

const MAGIC_COMPRESSED: u8 = 0x5A; // 'Z' for zstd
const MAGIC_RAW_CBOR: u8 = 0xCB; // 'C' for CBOR
const ZSTD_COMPRESSION_LEVEL: i32 = 3;
const COMPRESSION_THRESHOLD: usize = 256; // Only compress if CBOR > 256 bytes

/// Encode a value as CBOR, optionally compressed with zstd.
pub fn encode<T: Serialize>(value: &T) -> Result<Vec<u8>, CodecError> {
    let cbor_bytes = cbor_encode(value)?;

    if cbor_bytes.len() < COMPRESSION_THRESHOLD {
        // Not worth compressing small payloads
        let mut out = Vec::with_capacity(1 + cbor_bytes.len());
        out.push(MAGIC_RAW_CBOR);
        out.extend_from_slice(&cbor_bytes);
        return Ok(out);
    }

    let compressed = zstd_compress(&cbor_bytes)?;

    // Only use compression if it actually shrinks the data
    if compressed.len() + 5 < cbor_bytes.len() {
        let uncompressed_len = cbor_bytes.len() as u32;
        let mut out = Vec::with_capacity(1 + 4 + compressed.len());
        out.push(MAGIC_COMPRESSED);
        out.extend_from_slice(&uncompressed_len.to_le_bytes());
        out.extend_from_slice(&compressed);
        Ok(out)
    } else {
        let mut out = Vec::with_capacity(1 + cbor_bytes.len());
        out.push(MAGIC_RAW_CBOR);
        out.extend_from_slice(&cbor_bytes);
        Ok(out)
    }
}

/// Decode a value from a wire-format buffer (compressed or raw CBOR).
pub fn decode<T: DeserializeOwned>(data: &[u8]) -> Result<T, CodecError> {
    if data.is_empty() {
        return Err(CodecError::EmptyInput);
    }

    match data[0] {
        MAGIC_COMPRESSED => {
            if data.len() < 5 {
                return Err(CodecError::TruncatedHeader);
            }
            let _uncompressed_len =
                u32::from_le_bytes([data[1], data[2], data[3], data[4]]) as usize;
            let cbor_bytes = zstd_decompress(&data[5..])?;
            cbor_decode(&cbor_bytes)
        }
        MAGIC_RAW_CBOR => cbor_decode(&data[1..]),
        other => Err(CodecError::UnknownMagic(other)),
    }
}

/// Return the CBOR-encoded bytes without the wire-format envelope (for testing).
pub fn cbor_only<T: Serialize>(value: &T) -> Result<Vec<u8>, CodecError> {
    cbor_encode(value)
}

/// Return the compressed size and uncompressed size for analysis.
pub fn compression_stats<T: Serialize>(value: &T) -> Result<CompressionStats, CodecError> {
    let cbor_bytes = cbor_encode(value)?;
    let compressed = zstd_compress(&cbor_bytes)?;
    let wire = encode(value)?;
    Ok(CompressionStats {
        cbor_size: cbor_bytes.len(),
        compressed_size: compressed.len(),
        wire_size: wire.len(),
        ratio: if cbor_bytes.is_empty() {
            1.0
        } else {
            wire.len() as f64 / cbor_bytes.len() as f64
        },
    })
}

#[derive(Debug, Clone)]
pub struct CompressionStats {
    pub cbor_size: usize,
    pub compressed_size: usize,
    pub wire_size: usize,
    pub ratio: f64,
}

#[derive(Debug)]
pub enum CodecError {
    EmptyInput,
    TruncatedHeader,
    UnknownMagic(u8),
    CborEncode(String),
    CborDecode(String),
    ZstdCompress(String),
    ZstdDecompress(String),
}

impl std::fmt::Display for CodecError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::EmptyInput => write!(f, "empty input"),
            Self::TruncatedHeader => write!(f, "truncated header"),
            Self::UnknownMagic(b) => write!(f, "unknown magic byte: 0x{b:02x}"),
            Self::CborEncode(e) => write!(f, "CBOR encode error: {e}"),
            Self::CborDecode(e) => write!(f, "CBOR decode error: {e}"),
            Self::ZstdCompress(e) => write!(f, "zstd compress error: {e}"),
            Self::ZstdDecompress(e) => write!(f, "zstd decompress error: {e}"),
        }
    }
}

impl std::error::Error for CodecError {}

// ── Internal helpers ──

fn cbor_encode<T: Serialize>(value: &T) -> Result<Vec<u8>, CodecError> {
    let mut buf = Vec::new();
    ciborium::into_writer(value, &mut buf).map_err(|e| CodecError::CborEncode(e.to_string()))?;
    Ok(buf)
}

fn cbor_decode<T: DeserializeOwned>(data: &[u8]) -> Result<T, CodecError> {
    ciborium::from_reader(data).map_err(|e| CodecError::CborDecode(e.to_string()))
}

fn zstd_compress(data: &[u8]) -> Result<Vec<u8>, CodecError> {
    zstd::encode_all(data, ZSTD_COMPRESSION_LEVEL)
        .map_err(|e| CodecError::ZstdCompress(e.to_string()))
}

fn zstd_decompress(data: &[u8]) -> Result<Vec<u8>, CodecError> {
    zstd::decode_all(data).map_err(|e| CodecError::ZstdDecompress(e.to_string()))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::checkpoint::Checkpoint;
    use crate::event_log::WorkflowEvent;
    use crate::workflow_state::WorkflowStatus;

    fn sample_checkpoint(event_count: usize) -> Checkpoint {
        let events: Vec<_> = (0..event_count)
            .map(|i| {
                WorkflowEvent::new(
                    format!("evt-{i}"),
                    i as u64,
                    "task_event",
                    "agent",
                    format!("{{\"data\":\"value-{i}\",\"extra\":\"padding to make payload bigger for compression test {i}\"}}"),
                )
            })
            .collect();
        Checkpoint {
            workflow_id: "wf-test".into(),
            agent_id: "agent-01".into(),
            latest_step: event_count.saturating_sub(1) as u64,
            root_hash: "abc123".into(),
            status: WorkflowStatus::Running,
            events,
        }
    }

    #[test]
    fn roundtrip_small_checkpoint() {
        let cp = sample_checkpoint(2);
        let encoded = encode(&cp).unwrap();
        // Small payload → should be raw CBOR (0xCB) or compressed (0x5A)
        assert!(encoded[0] == MAGIC_RAW_CBOR || encoded[0] == MAGIC_COMPRESSED);
        let decoded: Checkpoint = decode(&encoded).unwrap();
        assert_eq!(cp, decoded);
    }

    #[test]
    fn roundtrip_large_checkpoint_compressed() {
        let cp = sample_checkpoint(50);
        let encoded = encode(&cp).unwrap();
        // Large enough to trigger compression
        let stats = compression_stats(&cp).unwrap();
        if stats.wire_size < stats.cbor_size {
            assert_eq!(encoded[0], MAGIC_COMPRESSED);
        }
        let decoded: Checkpoint = decode(&encoded).unwrap();
        assert_eq!(cp, decoded);
    }

    #[test]
    fn cbor_smaller_than_json() {
        let cp = sample_checkpoint(20);
        let cbor = cbor_only(&cp).unwrap();
        let json = serde_json::to_vec(&cp).unwrap();
        assert!(
            cbor.len() < json.len(),
            "CBOR ({}) should be smaller than JSON ({})",
            cbor.len(),
            json.len()
        );
    }

    #[test]
    fn empty_input_returns_error() {
        let result = decode::<Checkpoint>(&[]);
        assert!(result.is_err());
    }

    #[test]
    fn unknown_magic_returns_error() {
        let result = decode::<Checkpoint>(&[0xFF, 0x00]);
        assert!(result.is_err());
    }

    #[test]
    fn truncated_compressed_header_returns_error() {
        let result = decode::<Checkpoint>(&[MAGIC_COMPRESSED, 0x00, 0x00]);
        assert!(result.is_err());
    }

    #[test]
    fn compression_stats_work() {
        let cp = sample_checkpoint(30);
        let stats = compression_stats(&cp).unwrap();
        assert!(stats.cbor_size > 0);
        assert!(stats.wire_size > 0);
        assert!(stats.ratio > 0.0);
    }
}
