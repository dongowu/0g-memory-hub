use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

use crate::event_log::WorkflowEvent;

/// A 32-byte SHA-256 hash used as a node in the Merkle tree.
pub type Hash = [u8; 32];

const LEAF_PREFIX: u8 = 0x00;
const NODE_PREFIX: u8 = 0x01;

/// Binary Merkle tree built over a sequence of workflow events.
///
/// Leaves are SHA-256 hashes of each event's canonical encoding.
/// Internal nodes are SHA-256(0x01 || left || right).
/// When the leaf count is not a power of two, the last leaf is promoted.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct MerkleTree {
    /// All nodes stored in level-order (root at index 0 of the top level).
    /// layers[0] = leaves, layers[last] = [root].
    layers: Vec<Vec<Hash>>,
    leaf_count: usize,
}

/// Inclusion proof for a single leaf in the Merkle tree.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct MerkleProof {
    pub leaf_index: usize,
    pub leaf_hash: Hash,
    pub siblings: Vec<ProofNode>,
    pub root: Hash,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct ProofNode {
    pub hash: Hash,
    /// true = sibling is on the right, false = sibling is on the left
    pub is_right: bool,
}

impl MerkleTree {
    /// Build a Merkle tree from a slice of workflow events.
    pub fn build(events: &[WorkflowEvent]) -> Self {
        if events.is_empty() {
            return Self {
                layers: vec![vec![]],
                leaf_count: 0,
            };
        }

        let leaves: Vec<Hash> = events.iter().map(hash_event).collect();
        let leaf_count = leaves.len();
        let mut layers = vec![leaves];

        // Build up from leaves to root
        while layers.last().map_or(true, |l| l.len() > 1) {
            let prev = layers.last().unwrap();
            let mut next = Vec::with_capacity((prev.len() + 1) / 2);
            let mut i = 0;
            while i < prev.len() {
                if i + 1 < prev.len() {
                    next.push(hash_pair(&prev[i], &prev[i + 1]));
                } else {
                    // Odd node: promote without hashing
                    next.push(prev[i]);
                }
                i += 2;
            }
            layers.push(next);
        }

        Self { layers, leaf_count }
    }

    /// Return the Merkle root hash, or a zero hash if the tree is empty.
    pub fn root(&self) -> Hash {
        self.layers
            .last()
            .and_then(|l| l.first().copied())
            .unwrap_or([0u8; 32])
    }

    /// Return the hex-encoded root hash (compatible with the old API).
    pub fn root_hex(&self) -> String {
        hex::encode(self.root())
    }

    /// Number of leaves in the tree.
    pub fn leaf_count(&self) -> usize {
        self.leaf_count
    }

    /// Generate an inclusion proof for the leaf at `index`.
    pub fn proof(&self, index: usize) -> Option<MerkleProof> {
        if index >= self.leaf_count || self.leaf_count == 0 {
            return None;
        }

        let leaf_hash = self.layers[0][index];
        let mut siblings = Vec::new();
        let mut idx = index;

        for layer in &self.layers[..self.layers.len().saturating_sub(1)] {
            let sibling_idx = if idx % 2 == 0 { idx + 1 } else { idx - 1 };
            if sibling_idx < layer.len() {
                siblings.push(ProofNode {
                    hash: layer[sibling_idx],
                    is_right: idx % 2 == 0,
                });
            }
            // else: odd node promoted, no sibling at this level
            idx /= 2;
        }

        Some(MerkleProof {
            leaf_index: index,
            leaf_hash,
            siblings,
            root: self.root(),
        })
    }
}

impl MerkleProof {
    /// Verify this proof against a given root hash.
    pub fn verify(&self, expected_root: &Hash) -> bool {
        let mut current = self.leaf_hash;
        for node in &self.siblings {
            if node.is_right {
                current = hash_pair(&current, &node.hash);
            } else {
                current = hash_pair(&node.hash, &current);
            }
        }
        &current == expected_root
    }

    /// Serialize the proof to a hex-friendly JSON representation.
    pub fn to_hex_proof(&self) -> HexMerkleProof {
        HexMerkleProof {
            leaf_index: self.leaf_index,
            leaf_hash: hex::encode(self.leaf_hash),
            siblings: self
                .siblings
                .iter()
                .map(|s| HexProofNode {
                    hash: hex::encode(s.hash),
                    is_right: s.is_right,
                })
                .collect(),
            root: hex::encode(self.root),
        }
    }
}

/// Hex-encoded version of MerkleProof for JSON serialization.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HexMerkleProof {
    pub leaf_index: usize,
    pub leaf_hash: String,
    pub siblings: Vec<HexProofNode>,
    pub root: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HexProofNode {
    pub hash: String,
    pub is_right: bool,
}

/// Hash a single workflow event into a leaf node.
pub fn hash_event(event: &WorkflowEvent) -> Hash {
    let mut hasher = Sha256::new();
    hasher.update([LEAF_PREFIX]);
    hasher.update(event.event_id.as_bytes());
    hasher.update([0]);
    hasher.update(event.step_index.to_le_bytes());
    hasher.update([0]);
    hasher.update(event.event_type.as_bytes());
    hasher.update([0]);
    hasher.update(event.actor.as_bytes());
    hasher.update([0]);
    hasher.update(event.payload.as_bytes());
    hasher.update([0xff]);
    hasher.finalize().into()
}

/// Hash two child nodes into a parent node.
pub fn hash_pair(left: &Hash, right: &Hash) -> Hash {
    let mut hasher = Sha256::new();
    hasher.update([NODE_PREFIX]);
    hasher.update(left);
    hasher.update(right);
    hasher.finalize().into()
}

/// Legacy-compatible function: compute a hex-encoded root from events.
pub fn compute_root(events: &[WorkflowEvent]) -> String {
    MerkleTree::build(events).root_hex()
}

// ── Inline hex helpers (avoids adding the `hex` crate) ──

mod hex {
    const HEX_CHARS: &[u8; 16] = b"0123456789abcdef";

    pub fn encode(bytes: impl AsRef<[u8]>) -> String {
        let bytes = bytes.as_ref();
        let mut s = String::with_capacity(bytes.len() * 2);
        for &b in bytes {
            s.push(HEX_CHARS[(b >> 4) as usize] as char);
            s.push(HEX_CHARS[(b & 0x0f) as usize] as char);
        }
        s
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::event_log::WorkflowEvent;

    fn make_event(step: u64) -> WorkflowEvent {
        WorkflowEvent::new(
            format!("evt-{step}"),
            step,
            "task_event",
            "agent",
            format!("{{\"step\":{step}}}"),
        )
    }

    #[test]
    fn empty_tree_returns_zero_root() {
        let tree = MerkleTree::build(&[]);
        assert_eq!(tree.root(), [0u8; 32]);
        assert_eq!(tree.leaf_count(), 0);
    }

    #[test]
    fn single_event_tree() {
        let events = vec![make_event(0)];
        let tree = MerkleTree::build(&events);
        assert_ne!(tree.root(), [0u8; 32]);
        assert_eq!(tree.leaf_count(), 1);

        let proof = tree.proof(0).unwrap();
        assert!(proof.verify(&tree.root()));
    }

    #[test]
    fn two_event_tree() {
        let events = vec![make_event(0), make_event(1)];
        let tree = MerkleTree::build(&events);
        assert_eq!(tree.leaf_count(), 2);

        let expected_root = hash_pair(&hash_event(&events[0]), &hash_event(&events[1]));
        assert_eq!(tree.root(), expected_root);

        for i in 0..2 {
            let proof = tree.proof(i).unwrap();
            assert!(proof.verify(&tree.root()), "proof failed for leaf {i}");
        }
    }

    #[test]
    fn three_event_tree_proofs() {
        let events: Vec<_> = (0..3).map(make_event).collect();
        let tree = MerkleTree::build(&events);
        assert_eq!(tree.leaf_count(), 3);

        for i in 0..3 {
            let proof = tree.proof(i).unwrap();
            assert!(proof.verify(&tree.root()), "proof failed for leaf {i}");
        }
    }

    #[test]
    fn power_of_two_events() {
        let events: Vec<_> = (0..8).map(make_event).collect();
        let tree = MerkleTree::build(&events);
        assert_eq!(tree.leaf_count(), 8);

        for i in 0..8 {
            let proof = tree.proof(i).unwrap();
            assert!(proof.verify(&tree.root()), "proof failed for leaf {i}");
        }
    }

    #[test]
    fn large_tree_proofs() {
        let events: Vec<_> = (0..100).map(make_event).collect();
        let tree = MerkleTree::build(&events);
        assert_eq!(tree.leaf_count(), 100);

        // Spot check several indices
        for &i in &[0, 1, 49, 50, 98, 99] {
            let proof = tree.proof(i).unwrap();
            assert!(proof.verify(&tree.root()), "proof failed for leaf {i}");
        }
    }

    #[test]
    fn proof_fails_with_wrong_root() {
        let events: Vec<_> = (0..4).map(make_event).collect();
        let tree = MerkleTree::build(&events);
        let proof = tree.proof(0).unwrap();

        let wrong_root = [0xab; 32];
        assert!(!proof.verify(&wrong_root));
    }

    #[test]
    fn out_of_bounds_proof_returns_none() {
        let events: Vec<_> = (0..3).map(make_event).collect();
        let tree = MerkleTree::build(&events);
        assert!(tree.proof(3).is_none());
        assert!(tree.proof(100).is_none());
    }

    #[test]
    fn deterministic_root() {
        let events: Vec<_> = (0..5).map(make_event).collect();
        let tree1 = MerkleTree::build(&events);
        let tree2 = MerkleTree::build(&events);
        assert_eq!(tree1.root(), tree2.root());
    }

    #[test]
    fn different_events_different_roots() {
        let events_a: Vec<_> = (0..3).map(make_event).collect();
        let events_b: Vec<_> = (10..13).map(|i| {
            WorkflowEvent::new(format!("other-{i}"), i - 10, "other", "bot", "{}")
        }).collect();
        let tree_a = MerkleTree::build(&events_a);
        let tree_b = MerkleTree::build(&events_b);
        assert_ne!(tree_a.root(), tree_b.root());
    }

    #[test]
    fn hex_proof_roundtrip() {
        let events: Vec<_> = (0..4).map(make_event).collect();
        let tree = MerkleTree::build(&events);
        let proof = tree.proof(2).unwrap();
        let hex_proof = proof.to_hex_proof();
        assert_eq!(hex_proof.leaf_index, 2);
        assert_eq!(hex_proof.root.len(), 64);
        assert_eq!(hex_proof.leaf_hash.len(), 64);
    }

    #[test]
    fn compute_root_backward_compat() {
        let events: Vec<_> = (0..3).map(make_event).collect();
        let root = compute_root(&events);
        let tree = MerkleTree::build(&events);
        assert_eq!(root, tree.root_hex());
    }
}
