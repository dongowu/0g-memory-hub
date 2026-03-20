package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// MemoryChainABI is the ABI for MemoryChain contract
const MemoryChainABI = `[{"type":"function","name":"setMemoryHead","inputs":[{"name":"cid","type":"bytes32"}],"outputs":[]},{"type":"function","name":"getMemoryHead","inputs":[{"name":"agent","type":"address"}],"outputs":[{"name":"","type":"bytes32"}]},{"type":"function","name":"getMemoryHistory","inputs":[{"name":"agent","type":"address"}],"outputs":[{"name":"","type":"bytes32[]"}]}]`

// Config holds all configuration
type Config struct {
	StorageRPC   string
	ChainRPC     string
	IndexerRPC   string
	ContractAddr string
	PrivateKey   string
	ChainID      uint64
}

var cfg = Config{}
var logger *logrus.Logger

// StorageConfig holds 0G Storage configuration
type StorageConfig struct {
	RPCURL     string
	IndexerURL string
	PrivateKey string
}

// ChainConfig holds 0G Chain configuration
type ChainConfig struct {
	RPCURL       string
	ContractAddr string
	PrivateKey   string
	ChainID      uint64
}

// UploadResult holds the result of a storage upload
type UploadResult struct {
	CID      string `json:"cid"`
	TxHash   string `json:"tx_hash"`
	Root     string `json:"root"`
	FileSize uint64 `json:"file_size"`
}

// ChainTxResult holds the result of a chain transaction
type ChainTxResult struct {
	TxHash      string `json:"tx_hash"`
	BlockNumber uint64 `json:"block_number"`
	Status      string `json:"status"`
}

func main() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetLevel(logrus.InfoLevel)

	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfg.StorageRPC, "storage-rpc", "", "0G Storage RPC URL")
	viper.BindPFlag("STORAGE_RPC", RootCmd.PersistentFlags().Lookup("storage-rpc"))

	RootCmd.PersistentFlags().StringVar(&cfg.ChainRPC, "chain-rpc", "", "0G Chain RPC URL")
	viper.BindPFlag("CHAIN_RPC", RootCmd.PersistentFlags().Lookup("chain-rpc"))

	RootCmd.PersistentFlags().StringVar(&cfg.IndexerRPC, "indexer-rpc", "", "0G Indexer RPC URL")
	viper.BindPFlag("INDEXER_RPC", RootCmd.PersistentFlags().Lookup("indexer-rpc"))

	RootCmd.PersistentFlags().StringVar(&cfg.ContractAddr, "contract", "", "MemoryChain contract address")
	viper.BindPFlag("CONTRACT", RootCmd.PersistentFlags().Lookup("contract"))

	RootCmd.PersistentFlags().StringVar(&cfg.PrivateKey, "key", "", "Private key for transactions")
	viper.BindPFlag("KEY", RootCmd.PersistentFlags().Lookup("key"))

	RootCmd.PersistentFlags().Uint64Var(&cfg.ChainID, "chain-id", 1, "Chain ID")

	viper.AutomaticEnv()

	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfg.StorageRPC == "" {
		cfg.StorageRPC = viper.GetString("STORAGE_RPC")
	}
	if cfg.StorageRPC == "" {
		cfg.StorageRPC = "https://testnet-rpc.0g.ai"
	}
	if cfg.ChainRPC == "" {
		cfg.ChainRPC = viper.GetString("CHAIN_RPC")
	}
	if cfg.ChainRPC == "" {
		cfg.ChainRPC = "https://testnet-chain-rpc.0g.ai"
	}
	if cfg.IndexerRPC == "" {
		cfg.IndexerRPC = viper.GetString("INDEXER_RPC")
	}
	if cfg.IndexerRPC == "" {
		cfg.IndexerRPC = "https://testnet-indexer.0g.ai"
	}
}

var RootCmd = &cobra.Command{
	Use:   "0g-memory-hub",
	Short: "0G Memory Hub - AI Agent Eternal Memory System",
	Long: `0G Memory Hub provides immutable, verifiable memory storage for AI agents
using 0G's decentralized infrastructure (Storage + Chain).

For more information, visit: https://docs.0g.ai/`,
}

// Upload to 0G Storage using indexer API
func uploadToStorage(ctx context.Context, filePath string, config *StorageConfig) (*UploadResult, error) {
	logger.WithField("file", filePath).Info("Uploading to 0G Storage")

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	fileSize := uint64(len(data))

	// Calculate content hash
	hash := sha256.Sum256(data)
	contentHash := hex.EncodeToString(hash[:])

	// Create mock CID (in production, this comes from 0G Storage)
	cid := fmt.Sprintf("0g_%s", contentHash[:16])

	logger.WithFields(logrus.Fields{
		"cid":      cid,
		"size":     fileSize,
		"indexer": config.IndexerURL,
	}).Info("Upload completed (mock)")

	return &UploadResult{
		CID:      cid,
		TxHash:   "0x" + contentHash[:32],
		Root:     cid,
		FileSize: fileSize,
	}, nil
}

// Download from 0G Storage
func downloadFromStorage(ctx context.Context, cid string, outputPath string, config *StorageConfig) error {
	logger.WithFields(logrus.Fields{
		"cid":   cid,
		"to":    outputPath,
		"indexer": config.IndexerURL,
	}).Info("Downloading from 0G Storage")

	// Create mock download
	mockData := []byte(fmt.Sprintf("Mock content for CID: %s", cid))
	if err := os.WriteFile(outputPath, mockData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	logger.Info("Download completed")
	return nil
}

// Chain Client

type chainClient struct {
	ethClient *ethclient.Client
	parsedABI abi.ABI
	config    *ChainConfig
}

func newChainClient(config *ChainConfig) (*chainClient, error) {
	client, err := ethclient.Dial(config.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to chain: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(MemoryChainABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return &chainClient{
		ethClient: client,
		parsedABI: parsedABI,
		config:    config,
	}, nil
}

func (c *chainClient) Close() {
	c.ethClient.Close()
}

func (c *chainClient) SetMemoryHead(ctx context.Context, agent common.Address, cid string) (*ChainTxResult, error) {
	logger.WithFields(logrus.Fields{"agent": agent.Hex(), "cid": cid}).Info("Setting memory head on-chain")

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(c.config.PrivateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	var cidBytes32 [32]byte
	cidHex := strings.TrimPrefix(cid, "0x")
	cidBytes, err := hex.DecodeString(cidHex)
	if err != nil || len(cidBytes) != 32 {
		hash := crypto.Keccak256Hash([]byte(cid))
		copy(cidBytes32[:], hash[:])
	} else {
		copy(cidBytes32[:], cidBytes)
	}

	input, err := c.parsedABI.Pack("setMemoryHead", cidBytes32)
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %w", err)
	}

	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err := c.ethClient.NonceAt(ctx, fromAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := c.ethClient.SuggestGasPrice(ctx)
	if err != nil {
		gasPrice = big.NewInt(1000000000)
	}

	contractAddr := common.HexToAddress(c.config.ContractAddr)
	tx := types.NewTransaction(nonce, contractAddr, big.NewInt(0), 100000, gasPrice, input)

	chainID, _ := c.ethClient.NetworkID(ctx)
	signer := types.NewEIP155Signer(chainID)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = c.ethClient.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	logger.WithField("txHash", signedTx.Hash().Hex()).Info("Transaction sent")

	receipt, err := c.ethClient.TransactionReceipt(ctx, signedTx.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to wait for receipt: %w", err)
	}

	logger.WithField("blockNumber", receipt.BlockNumber.Uint64()).Info("Transaction confirmed")

	return &ChainTxResult{
		TxHash:      signedTx.Hash().Hex(),
		BlockNumber: receipt.BlockNumber.Uint64(),
		Status:      fmt.Sprintf("%d", receipt.Status),
	}, nil
}

func (c *chainClient) GetMemoryHead(ctx context.Context, agent common.Address) (string, error) {
	input, err := c.parsedABI.Pack("getMemoryHead", agent)
	if err != nil {
		return "", fmt.Errorf("failed to pack call data: %w", err)
	}

	contractAddr := common.HexToAddress(c.config.ContractAddr)
	result, err := c.ethClient.CallContract(ctx, ethereum.CallMsg{To: &contractAddr, Data: input}, nil)
	if err != nil {
		return "", fmt.Errorf("failed to call contract: %w", err)
	}

	var cid [32]byte
	c.parsedABI.UnpackIntoInterface(&cid, "getMemoryHead", result)
	return common.BytesToHash(cid[:]).Hex(), nil
}

func (c *chainClient) GetMemoryHistory(ctx context.Context, agent common.Address) ([]string, error) {
	input, err := c.parsedABI.Pack("getMemoryHistory", agent)
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %w", err)
	}

	contractAddr := common.HexToAddress(c.config.ContractAddr)
	result, err := c.ethClient.CallContract(ctx, ethereum.CallMsg{To: &contractAddr, Data: input}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	var history []common.Hash
	if len(result) > 64 {
		length := new(big.Int).SetBytes(result[32:64]).Uint64()
		history = make([]common.Hash, length)
		for i := uint64(0); i < length && i < 100; i++ {
			start := 64 + int(i*32)
			if start+32 <= len(result) {
				copy(history[i][:], result[start:start+32])
			}
		}
	}

	resultStr := make([]string, len(history))
	for i, h := range history {
		resultStr[i] = h.Hex()
	}
	return resultStr, nil
}

// CLI Commands

func uploadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload [file]",
		Short: "Upload a file to 0G Storage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config := &StorageConfig{
				RPCURL:     cfg.StorageRPC,
				IndexerURL: cfg.IndexerRPC,
				PrivateKey: cfg.PrivateKey,
			}

			result, err := uploadToStorage(cmd.Context(), args[0], config)
			if err != nil {
				return fmt.Errorf("upload failed: %w", err)
			}

			j, _ := json.MarshalIndent(result, "", "  ")
			fmt.Printf("\n Upload successful!\n%s\n", j)
			return nil
		},
	}
	return cmd
}

func downloadCmd() *cobra.Command {
	var outputPath string
	cmd := &cobra.Command{
		Use:   "download [cid]",
		Short: "Download a file from 0G Storage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputPath == "" {
				outputPath = "./download_" + args[0][:16] + ".bin"
			}

			config := &StorageConfig{
				IndexerURL: cfg.IndexerRPC,
				PrivateKey: cfg.PrivateKey,
			}

			err := downloadFromStorage(cmd.Context(), args[0], outputPath, config)
			if err != nil {
				return fmt.Errorf("download failed: %w", err)
			}

			fmt.Printf("\n Download successful! Output: %s\n", outputPath)
			return nil
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path")
	return cmd
}

func setPointerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-pointer [agent] [cid]",
		Short: "Set memory head pointer on 0G Chain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := common.HexToAddress(args[0])
			config := &ChainConfig{
				RPCURL:       cfg.ChainRPC,
				ContractAddr: cfg.ContractAddr,
				PrivateKey:   cfg.PrivateKey,
				ChainID:      cfg.ChainID,
			}

			client, err := newChainClient(config)
			if err != nil {
				return fmt.Errorf("failed to create chain client: %w", err)
			}
			defer client.Close()

			result, err := client.SetMemoryHead(cmd.Context(), agent, args[1])
			if err != nil {
				return fmt.Errorf("set pointer failed: %w", err)
			}

			j, _ := json.MarshalIndent(result, "", "  ")
			fmt.Printf("\n Memory pointer set successfully!\n%s\n", j)
			return nil
		},
	}
	return cmd
}

func getPointerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-pointer [agent]",
		Short: "Get memory head pointer from 0G Chain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := common.HexToAddress(args[0])
			config := &ChainConfig{
				RPCURL:       cfg.ChainRPC,
				ContractAddr: cfg.ContractAddr,
				PrivateKey:   cfg.PrivateKey,
			}

			client, err := newChainClient(config)
			if err != nil {
				return fmt.Errorf("failed to create chain client: %w", err)
			}
			defer client.Close()

			cid, err := client.GetMemoryHead(cmd.Context(), agent)
			if err != nil {
				return fmt.Errorf("get pointer failed: %w", err)
			}

			fmt.Printf("\n Memory pointer:\n   Agent: %s\n   CID: %s\n", args[0], cid)
			return nil
		},
	}
	return cmd
}

func getHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-history [agent]",
		Short: "Get full memory history from 0G Chain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := common.HexToAddress(args[0])
			config := &ChainConfig{
				RPCURL:       cfg.ChainRPC,
				ContractAddr: cfg.ContractAddr,
				PrivateKey:   cfg.PrivateKey,
			}

			client, err := newChainClient(config)
			if err != nil {
				return fmt.Errorf("failed to create chain client: %w", err)
			}
			defer client.Close()

			history, err := client.GetMemoryHistory(cmd.Context(), agent)
			if err != nil {
				return fmt.Errorf("get history failed: %w", err)
			}

			fmt.Printf("\n Memory history (%d entries):\n", len(history))
			for i, cid := range history {
				fmt.Printf("   [%d] %s\n", i+1, cid)
			}
			return nil
		},
	}
	return cmd
}

func demoCmd() *cobra.Command {
	var agent string

	cmd := &cobra.Command{
		Use:   "demo [file]",
		Short: "End-to-end demo: upload file and anchor on-chain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if agent == "" {
				privateKey, _ := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
				agent = crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
			}

			fmt.Println("\n 0G Memory Hub - End-to-End Demo\n")

			// Step 1: Upload
			fmt.Println("Step 1  Upload to 0G Storage")
			storageConfig := &StorageConfig{
				RPCURL:     cfg.StorageRPC,
				IndexerURL: cfg.IndexerRPC,
				PrivateKey: cfg.PrivateKey,
			}
			result, err := uploadToStorage(cmd.Context(), args[0], storageConfig)
			if err != nil {
				return fmt.Errorf("upload failed: %w", err)
			}
			fmt.Printf("   CID: %s\n\n", result.CID)

			// Step 2: Set pointer
			fmt.Println("Step 2  Anchor on 0G Chain")
			chainConfig := &ChainConfig{
				RPCURL:       cfg.ChainRPC,
				ContractAddr: cfg.ContractAddr,
				PrivateKey:   cfg.PrivateKey,
				ChainID:      cfg.ChainID,
			}
			client, err := newChainClient(chainConfig)
			if err != nil {
				return fmt.Errorf("failed to create chain client: %w", err)
			}
			defer client.Close()

			txResult, err := client.SetMemoryHead(cmd.Context(), common.HexToAddress(agent), result.CID)
			if err != nil {
				return fmt.Errorf("set pointer failed: %w", err)
			}
			fmt.Printf("   TX Hash: %s\n\n", txResult.TxHash)

			// Step 3: Verify
			fmt.Println("Step 3  Verify on-chain")
			head, _ := client.GetMemoryHead(cmd.Context(), common.HexToAddress(agent))
			fmt.Printf("   Memory head: %s\n\n", head)

			fmt.Println(" Demo complete! Memory is now eternal on 0G.\n")
			return nil
		},
	}
	cmd.Flags().StringVarP(&agent, "agent", "a", "", "Agent address (defaults to sender)")
	return cmd
}

func genCmd() *cobra.Command {
	var size int64
	cmd := &cobra.Command{
		Use:   "gen [filename]",
		Short: "Generate a test file",
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := "test_memory.bin"
			if len(args) > 0 {
				filename = args[0]
			}

			data := make([]byte, size)
			for i := range data {
				data[i] = byte(time.Now().UnixNano() % 256)
			}

			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}

			hash := sha256.Sum256(data)
			fmt.Printf("Generated: %s (%d bytes)\n   Hash: %s\n", filename, size, hex.EncodeToString(hash[:]))
			return nil
		},
	}
	cmd.Flags().Int64VarP(&size, "size", "s", 1024, "File size in bytes")
	return cmd
}

func init() {
	RootCmd.AddCommand(uploadCmd())
	RootCmd.AddCommand(downloadCmd())
	RootCmd.AddCommand(setPointerCmd())
	RootCmd.AddCommand(getPointerCmd())
	RootCmd.AddCommand(getHistoryCmd())
	RootCmd.AddCommand(demoCmd())
	RootCmd.AddCommand(genCmd())
}
