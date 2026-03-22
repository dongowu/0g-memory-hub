package config

import "testing"

func TestLoadDefaultsUseOfficialGalileoEndpoints(t *testing.T) {
	t.Setenv("ORCH_DATA_DIR", "")
	t.Setenv("ORCH_STORAGE_RPC_URL", "")
	t.Setenv("ORCH_CHAIN_RPC_URL", "")
	t.Setenv("ORCH_CHAIN_CONTRACT_ADDRESS", "")
	t.Setenv("ORCH_CHAIN_ID", "")
	t.Setenv("ORCH_RUNTIME_BINARY_PATH", "")

	cfg := Load()

	if cfg.StorageRPCURL != "https://indexer-storage-testnet-standard.0g.ai" {
		t.Fatalf("StorageRPCURL = %s", cfg.StorageRPCURL)
	}
	if cfg.ChainRPCURL != "https://evmrpc-testnet.0g.ai" {
		t.Fatalf("ChainRPCURL = %s", cfg.ChainRPCURL)
	}
	if cfg.ChainID != "16602" {
		t.Fatalf("ChainID = %s", cfg.ChainID)
	}
}
