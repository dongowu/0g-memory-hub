package config

import "os"

type Config struct {
	DataDir              string
	StorageRPCURL        string
	ChainRPCURL          string
	ChainPrivateKey      string
	ChainContractAddress string
	ChainID              string
	RuntimeBinaryPath    string
}

func Load() Config {
	dataDir := os.Getenv("ORCH_DATA_DIR")
	if dataDir == "" {
		dataDir = ".orchestrator"
	}

	storageRPCURL := os.Getenv("ORCH_STORAGE_RPC_URL")
	if storageRPCURL == "" {
		storageRPCURL = "https://indexer-storage-testnet-standard.0g.ai"
	}

	chainRPCURL := os.Getenv("ORCH_CHAIN_RPC_URL")
	if chainRPCURL == "" {
		chainRPCURL = "https://evmrpc-testnet.0g.ai"
	}

	chainContractAddress := os.Getenv("ORCH_CHAIN_CONTRACT_ADDRESS")
	if chainContractAddress == "" {
		chainContractAddress = "0x0000000000000000000000000000000000000000"
	}

	chainID := os.Getenv("ORCH_CHAIN_ID")
	if chainID == "" {
		chainID = "16602"
	}

	runtimeBinaryPath := os.Getenv("ORCH_RUNTIME_BINARY_PATH")
	if runtimeBinaryPath == "" {
		runtimeBinaryPath = "memory-core-rpc"
	}

	return Config{
		DataDir:              dataDir,
		StorageRPCURL:        storageRPCURL,
		ChainRPCURL:          chainRPCURL,
		ChainPrivateKey:      os.Getenv("ORCH_CHAIN_PRIVATE_KEY"),
		ChainContractAddress: chainContractAddress,
		ChainID:              chainID,
		RuntimeBinaryPath:    runtimeBinaryPath,
	}
}
