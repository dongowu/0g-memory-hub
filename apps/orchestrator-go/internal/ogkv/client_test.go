package ogkv

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestStreamIDIsDeterministic(t *testing.T) {
	id1 := sha256StreamID("0g-memory-hub-checkpoints")
	id2 := sha256StreamID("0g-memory-hub-checkpoints")
	if id1 != id2 {
		t.Fatalf("stream ID should be deterministic, got %s and %s", id1.Hex(), id2.Hex())
	}
	if id1 == (common.Hash{}) {
		t.Fatal("stream ID should not be zero")
	}
}

func TestStreamIDMatchesGlobalVar(t *testing.T) {
	expected := sha256StreamID("0g-memory-hub-checkpoints")
	if StreamID != expected {
		t.Fatalf("StreamID global does not match expected: got %s, want %s", StreamID.Hex(), expected.Hex())
	}
}

func TestCheckpointKeyFormat(t *testing.T) {
	key := checkpointKey("wf-demo-01", 5)
	expected := "cp:wf-demo-01:5"
	if string(key) != expected {
		t.Fatalf("checkpoint key: got %q, want %q", string(key), expected)
	}
}

func TestLatestCheckpointKeyFormat(t *testing.T) {
	key := latestCheckpointKey("wf-demo-01")
	expected := "cp:wf-demo-01:latest"
	if string(key) != expected {
		t.Fatalf("latest checkpoint key: got %q, want %q", string(key), expected)
	}
}

func TestHexStreamIDNonEmpty(t *testing.T) {
	hex := HexStreamID()
	if hex == "" {
		t.Fatal("HexStreamID should not be empty")
	}
	if len(hex) != 64 {
		t.Fatalf("HexStreamID should be 64 hex chars, got %d", len(hex))
	}
}

func TestNewClientCreation(t *testing.T) {
	client := NewClient(Config{
		KVNodeURL:     "http://localhost:6789",
		ZgsNodeURL:    "http://localhost:5678",
		BlockchainRPC: "http://localhost:8545",
		PrivateKey:    "0xdeadbeef",
	})
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
}
