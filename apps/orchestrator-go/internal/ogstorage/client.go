package ogstorage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/0gfoundation/0g-storage-client/common/blockchain"
	"github.com/0gfoundation/0g-storage-client/core"
	"github.com/0gfoundation/0g-storage-client/indexer"
	"github.com/0gfoundation/0g-storage-client/transfer"
)

const (
	defaultExpectedReplica uint          = 1
	defaultUploadTimeout   time.Duration = 5 * time.Minute
	defaultDownloadTimeout time.Duration = 5 * time.Minute
	defaultSelectMethod    string        = "min"
)

type Client interface {
	UploadCheckpoint(ctx context.Context, payload []byte) (*UploadResult, error)
	DownloadCheckpoint(ctx context.Context, key string) ([]byte, error)
}

type UploadResult struct {
	Key    string
	TxHash string
}

type SDKConfig struct {
	IndexerRPCURL    string
	BlockchainRPCURL string
	PrivateKey       string
	ChainID          string
	ExpectedReplica  uint
	UploadTimeout    time.Duration
	DownloadTimeout  time.Duration
}

type transferAdapter interface {
	UploadBytes(ctx context.Context, cfg SDKConfig, payload []byte) (rootHash string, txHash string, err error)
	DownloadBytes(ctx context.Context, cfg SDKConfig, key string) ([]byte, error)
}

type SDKClient struct {
	config  SDKConfig
	adapter transferAdapter
}

func NewSDKClient(cfg SDKConfig, adapter transferAdapter) *SDKClient {
	if cfg.ExpectedReplica == 0 {
		cfg.ExpectedReplica = defaultExpectedReplica
	}
	if cfg.UploadTimeout <= 0 {
		cfg.UploadTimeout = defaultUploadTimeout
	}
	if cfg.DownloadTimeout <= 0 {
		cfg.DownloadTimeout = defaultDownloadTimeout
	}
	if adapter == nil {
		adapter = sdkTransferAdapter{}
	}
	return &SDKClient{
		config:  cfg,
		adapter: adapter,
	}
}

func (c *SDKClient) UploadCheckpoint(ctx context.Context, payload []byte) (*UploadResult, error) {
	if c.config.IndexerRPCURL == "" {
		return nil, fmt.Errorf("0G storage indexer RPC URL is required")
	}
	if c.config.BlockchainRPCURL == "" {
		return nil, fmt.Errorf("0G blockchain RPC URL is required for storage upload")
	}
	if c.config.PrivateKey == "" {
		return nil, fmt.Errorf("0G private key is required for storage upload")
	}

	ctx, cancel := withTimeout(ctx, c.config.UploadTimeout)
	defer cancel()

	rootHash, txHash, err := c.adapter.UploadBytes(ctx, c.config, payload)
	if err != nil {
		return nil, err
	}
	if rootHash == "" {
		return nil, fmt.Errorf("0G storage upload returned empty root hash")
	}
	return &UploadResult{
		Key:    rootHash,
		TxHash: txHash,
	}, nil
}

func (c *SDKClient) DownloadCheckpoint(ctx context.Context, key string) ([]byte, error) {
	if c.config.IndexerRPCURL == "" {
		return nil, fmt.Errorf("0G storage indexer RPC URL is required")
	}
	if key == "" {
		return nil, fmt.Errorf("0G storage key is required")
	}

	ctx, cancel := withTimeout(ctx, c.config.DownloadTimeout)
	defer cancel()

	return c.adapter.DownloadBytes(ctx, c.config, key)
}

func withTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

type sdkTransferAdapter struct{}

func (sdkTransferAdapter) UploadBytes(ctx context.Context, cfg SDKConfig, payload []byte) (string, string, error) {
	root, txHash, directErr := uploadDirect(ctx, cfg, payload)
	if directErr == nil {
		return root, txHash, nil
	}

	web3Client, err := blockchain.NewWeb3(cfg.BlockchainRPCURL, cfg.PrivateKey)
	if err != nil {
		if directErr != nil {
			return "", "", fmt.Errorf("direct fallback: %v; create 0G web3 client: %w", directErr, err)
		}
		return "", "", fmt.Errorf("create 0G web3 client: %w", err)
	}

	indexerClient, err := indexer.NewClient(cfg.IndexerRPCURL, indexer.IndexerClientOption{})
	if err != nil {
		return "", "", fmt.Errorf("create 0G indexer client: %w", err)
	}

	data, err := core.NewDataInMemory(payload)
	if err != nil {
		return "", "", fmt.Errorf("build checkpoint payload for 0G storage: %w", err)
	}

	txHashes, roots, err := indexerClient.SplitableUpload(ctx, web3Client, data, defaultUploadOption(cfg))
	if err != nil {
		if directErr != nil {
			return "", "", fmt.Errorf("direct fallback: %v; upload checkpoint to 0G storage: %w", directErr, err)
		}
		return "", "", fmt.Errorf("upload checkpoint to 0G storage: %w", err)
	}
	if len(roots) == 0 {
		return "", "", fmt.Errorf("0G storage upload returned no roots")
	}

	txHash = ""
	if len(txHashes) > 0 {
		txHash = txHashes[len(txHashes)-1].String()
	}
	return roots[len(roots)-1].String(), txHash, nil
}

func defaultUploadOption(cfg SDKConfig) transfer.UploadOption {
	return transfer.UploadOption{
		FinalityRequired: transfer.FileFinalized,
		ExpectedReplica:  cfg.ExpectedReplica,
		Method:           defaultSelectMethod,
		FullTrusted:      true,
	}
}

func (sdkTransferAdapter) DownloadBytes(ctx context.Context, cfg SDKConfig, key string) ([]byte, error) {
	if payload, err := downloadViaRESTCandidates(ctx, key, candidateIndexerBaseURLs(cfg.IndexerRPCURL)); err == nil {
		return decodeStoredPayload(payload)
	}

	indexerClient, err := indexer.NewClient(cfg.IndexerRPCURL, indexer.IndexerClientOption{})
	if err != nil {
		return nil, fmt.Errorf("create 0G indexer client: %w", err)
	}

	tempPath, cleanup, err := tempDownloadPath()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	if err := indexerClient.Download(ctx, key, tempPath, true); err != nil {
		return nil, fmt.Errorf("download checkpoint from 0G storage: %w", err)
	}

	payload, err := os.ReadFile(tempPath)
	if err != nil {
		return nil, fmt.Errorf("read 0G checkpoint download: %w", err)
	}
	return decodeStoredPayload(payload)
}

func tempDownloadPath() (string, func(), error) {
	dir, err := os.MkdirTemp("", "0g-checkpoint-download-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp download dir: %w", err)
	}
	path := filepath.Join(dir, "checkpoint.json")
	return path, func() {
		_ = os.RemoveAll(dir)
	}, nil
}
