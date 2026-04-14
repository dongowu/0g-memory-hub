package ogkv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/0gfoundation/0g-storage-client/common/blockchain"
	"github.com/0gfoundation/0g-storage-client/kv"
	"github.com/0gfoundation/0g-storage-client/node"
	"github.com/0gfoundation/0g-storage-client/transfer"
	"github.com/ethereum/go-ethereum/common"
)

// StreamID is a fixed stream identifier for the 0G Memory Hub checkpoint KV namespace.
var StreamID = sha256StreamID("0g-memory-hub-checkpoints")

const (
	defaultTimeout = 30 * time.Second
)

// CheckpointSummary is the compact value stored per workflow step in the KV layer.
type CheckpointSummary struct {
	WorkflowID string `json:"w"`
	StepIndex  uint64 `json:"s"`
	RootHash   string `json:"r"`
	CID        string `json:"c"`
	TxHash     string `json:"t"`
	Timestamp  int64  `json:"ts"`
}

// Config for the 0G KV client.
type Config struct {
	KVNodeURL     string // URL of the 0G KV RPC node (for reads)
	ZgsNodeURL    string // URL of the 0G storage node (for writes via batcher)
	BlockchainRPC string // Blockchain RPC URL
	PrivateKey    string // Private key for signing KV write transactions
}

// Client wraps the 0G KV SDK for checkpoint summary storage and retrieval.
type Client struct {
	config Config
}

// NewClient creates a new 0G KV client.
func NewClient(cfg Config) *Client {
	return &Client{config: cfg}
}

// PutCheckpoint writes a checkpoint summary to the 0G KV layer.
func (c *Client) PutCheckpoint(ctx context.Context, summary CheckpointSummary) error {
	if c.config.ZgsNodeURL == "" {
		return fmt.Errorf("0G KV: ZGS node URL is required for writes")
	}
	if c.config.BlockchainRPC == "" {
		return fmt.Errorf("0G KV: blockchain RPC URL is required for writes")
	}
	if c.config.PrivateKey == "" {
		return fmt.Errorf("0G KV: private key is required for writes")
	}

	zgsClient, err := node.NewZgsClient(c.config.ZgsNodeURL, nil)
	if err != nil {
		return fmt.Errorf("0G KV: create ZGS client: %w", err)
	}

	w3Client, err := blockchain.NewWeb3(c.config.BlockchainRPC, c.config.PrivateKey)
	if err != nil {
		return fmt.Errorf("0G KV: create web3 client: %w", err)
	}
	defer w3Client.Close()

	blockchain.CustomGasLimit = 1000000

	batcher := kv.NewBatcher(math.MaxUint64, &transfer.SelectedNodes{
		Trusted:    []*node.ZgsClient{zgsClient},
		Discovered: []*node.ZgsClient{},
	}, w3Client)

	key := checkpointKey(summary.WorkflowID, summary.StepIndex)
	value, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("0G KV: marshal checkpoint summary: %w", err)
	}

	batcher.Set(StreamID, key, value)

	latestKey := latestCheckpointKey(summary.WorkflowID)
	batcher.Set(StreamID, latestKey, value)

	_, err = batcher.Exec(ctx)
	if err != nil {
		return fmt.Errorf("0G KV: exec batcher: %w", err)
	}
	return nil
}

// GetCheckpoint reads a specific checkpoint summary from the 0G KV layer.
func (c *Client) GetCheckpoint(ctx context.Context, workflowID string, stepIndex uint64) (*CheckpointSummary, error) {
	kvClient, err := c.newKVReadClient()
	if err != nil {
		return nil, err
	}
	key := checkpointKey(workflowID, stepIndex)
	return c.readSummary(ctx, kvClient, key)
}

// GetLatestCheckpoint reads the latest checkpoint summary for a workflow.
func (c *Client) GetLatestCheckpoint(ctx context.Context, workflowID string) (*CheckpointSummary, error) {
	kvClient, err := c.newKVReadClient()
	if err != nil {
		return nil, err
	}
	key := latestCheckpointKey(workflowID)
	return c.readSummary(ctx, kvClient, key)
}

// CheckReadiness verifies KV node connectivity.
func (c *Client) CheckReadiness(ctx context.Context) error {
	if c.config.KVNodeURL == "" {
		return fmt.Errorf("0G KV node URL is not configured")
	}
	kvNodeClient, err := node.NewKvClient(c.config.KVNodeURL)
	if err != nil {
		return fmt.Errorf("0G KV: connect to node: %w", err)
	}
	_, err = kvNodeClient.GetHoldingStreamIds(ctx)
	if err != nil {
		return fmt.Errorf("0G KV: probe node: %w", err)
	}
	return nil
}

func (c *Client) newKVReadClient() (*kv.Client, error) {
	if c.config.KVNodeURL == "" {
		return nil, fmt.Errorf("0G KV: node URL is required for reads")
	}
	kvNodeClient, err := node.NewKvClient(c.config.KVNodeURL)
	if err != nil {
		return nil, fmt.Errorf("0G KV: create kv node client: %w", err)
	}
	return kv.NewClient(kvNodeClient), nil
}

func (c *Client) readSummary(ctx context.Context, kvClient *kv.Client, key []byte) (*CheckpointSummary, error) {
	val, err := kvClient.GetValue(ctx, StreamID, key)
	if err != nil {
		return nil, fmt.Errorf("0G KV: get value: %w", err)
	}
	if val == nil || len(val.Data) == 0 {
		return nil, fmt.Errorf("0G KV: key not found")
	}
	var summary CheckpointSummary
	if err := json.Unmarshal(val.Data, &summary); err != nil {
		return nil, fmt.Errorf("0G KV: decode summary: %w", err)
	}
	return &summary, nil
}

func checkpointKey(workflowID string, stepIndex uint64) []byte {
	return []byte(fmt.Sprintf("cp:%s:%d", workflowID, stepIndex))
}

func latestCheckpointKey(workflowID string) []byte {
	return []byte(fmt.Sprintf("cp:%s:latest", workflowID))
}

func sha256StreamID(name string) common.Hash {
	h := sha256.Sum256([]byte(name))
	return common.BytesToHash(h[:])
}

// HexStreamID returns the stream ID as a hex string for logging.
func HexStreamID() string {
	return hex.EncodeToString(StreamID[:])
}
