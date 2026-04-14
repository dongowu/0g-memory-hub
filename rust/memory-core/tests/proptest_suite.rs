//! Property-based tests for the Merkle tree, diff checkpoints, and codec.

use memory_core::{
    append_event, build_checkpoint, init_workflow,
    checkpoint::DiffCheckpoint,
    event_log::WorkflowEvent,
    merkle::MerkleTree,
};
use proptest::prelude::*;

fn arb_event(step: u64) -> WorkflowEvent {
    WorkflowEvent::new(
        format!("evt-{step}"),
        step,
        "task_event",
        "agent",
        format!("{{\"v\":{step}}}"),
    )
}

fn arb_event_rich(step: u64, actor: String, payload: String) -> WorkflowEvent {
    WorkflowEvent::new(
        format!("evt-{step}"),
        step,
        "task_event",
        actor,
        payload,
    )
}

// ── Merkle tree properties ──

proptest! {
    #![proptest_config(ProptestConfig::with_cases(200))]

    #[test]
    fn merkle_root_is_deterministic(n in 1usize..50) {
        let events: Vec<_> = (0..n as u64).map(arb_event).collect();
        let tree1 = MerkleTree::build(&events);
        let tree2 = MerkleTree::build(&events);
        prop_assert_eq!(tree1.root(), tree2.root());
    }

    #[test]
    fn merkle_all_proofs_verify(n in 1usize..50) {
        let events: Vec<_> = (0..n as u64).map(arb_event).collect();
        let tree = MerkleTree::build(&events);
        for i in 0..n {
            let proof = tree.proof(i).unwrap();
            prop_assert!(proof.verify(&tree.root()), "proof failed at index {}", i);
        }
    }

    #[test]
    fn merkle_proof_fails_wrong_root(n in 1usize..30) {
        let events: Vec<_> = (0..n as u64).map(arb_event).collect();
        let tree = MerkleTree::build(&events);
        let proof = tree.proof(0).unwrap();
        let wrong_root = [0xAB; 32];
        prop_assert!(!proof.verify(&wrong_root));
    }

    #[test]
    fn merkle_different_data_different_roots(
        n in 1usize..20,
        actor in "[a-z]{3,8}",
        payload in "[a-z0-9]{5,50}",
    ) {
        let events_a: Vec<_> = (0..n as u64).map(arb_event).collect();
        let events_b: Vec<_> = (0..n as u64)
            .map(|i| arb_event_rich(i, actor.clone(), payload.clone()))
            .collect();
        let root_a = MerkleTree::build(&events_a).root();
        let root_b = MerkleTree::build(&events_b).root();
        // Different input should (almost certainly) produce different roots
        // Only skip assertion if by cosmic coincidence they match
        if actor != "agent" || payload != format!("{{\"v\":{}}}", 0) {
            prop_assert_ne!(root_a, root_b);
        }
    }
}

// ── Diff checkpoint properties ──

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    #[test]
    fn diff_roundtrip(base_count in 0usize..10, extra_count in 1usize..10) {
        let mut state = init_workflow("wf-prop", "agent-prop");
        for i in 0..(base_count as u64) {
            append_event(&mut state, arb_event(i)).unwrap();
        }
        let base_cp = build_checkpoint(&state);

        for i in (base_count as u64)..((base_count + extra_count) as u64) {
            append_event(&mut state, arb_event(i)).unwrap();
        }

        let diff = DiffCheckpoint::from_state(
            &state,
            base_cp.events.len() as u64,
            base_cp.root_hash.clone(),
        );

        prop_assert_eq!(diff.delta_size(), extra_count);

        let restored = diff.apply_to(&base_cp).unwrap();
        prop_assert_eq!(restored.latest_root, state.latest_root);
        prop_assert_eq!(restored.events.len(), state.events.len());
    }
}

// ── Codec properties ──

proptest! {
    #![proptest_config(ProptestConfig::with_cases(50))]

    #[test]
    fn codec_roundtrip(n in 0usize..40) {
        let mut state = init_workflow("wf-codec", "agent-codec");
        for i in 0..n as u64 {
            append_event(&mut state, arb_event(i)).unwrap();
        }
        let cp = build_checkpoint(&state);

        let encoded = memory_core::codec::encode(&cp).unwrap();
        let decoded: memory_core::Checkpoint = memory_core::codec::decode(&encoded).unwrap();
        prop_assert_eq!(cp, decoded);
    }

    #[test]
    fn cbor_smaller_than_json(n in 5usize..30) {
        let mut state = init_workflow("wf-size", "agent-size");
        for i in 0..n as u64 {
            append_event(&mut state, arb_event(i)).unwrap();
        }
        let cp = build_checkpoint(&state);

        let cbor = memory_core::codec::cbor_only(&cp).unwrap();
        let json = serde_json::to_vec(&cp).unwrap();
        prop_assert!(cbor.len() <= json.len(),
            "CBOR ({}) should be <= JSON ({})", cbor.len(), json.len());
    }
}
