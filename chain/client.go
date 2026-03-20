package chain

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

// MemoryChainABI 是 MemoryChain 合约的 ABI
const MemoryChainABI = `[{"type":"function","name":"setMemoryHead","inputs":[{"name":"cid","type":"bytes32"}],"outputs":[]},{"type":"function","name":"getMemoryHead","inputs":[{"name":"agent","type":"address"}],"outputs":[{"name":"","type":"bytes32"}]},{"type":"function","name":"getMemoryHistory","inputs":[{"name":"agent","type":"address"}],"outputs":[{"name":"","type":"bytes32[]"}]},{"type":"function","name":"getMemoryHistoryLength","inputs":[{"name":"agent","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},{"type":"event","name":"MemoryUpdated","inputs":[{"name":"agent","type":"address","indexed":true},{"name":"cid","type":"bytes32","indexed":true},{"name":"timestamp","type":"uint256"}]}]`

// Client 0G Chain 客户端
type Client struct {
	ethClient *ethclient.Client
	parsedABI abi.ABI
	contract  common.Address
	logger    *logrus.Logger
	config    *Config
}

// Config 链配置
type Config struct {
	RPCURL    string
	PrivateKey string
	ChainID   uint64
	GasLimit  uint64
	GasPrice  uint64
}

// NewClient 创建新的链客户端
func NewClient(rpcURL string, contractAddr string) (*Client, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to chain: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(MemoryChainABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return &Client{
		ethClient: client,
		parsedABI: parsedABI,
		contract:  common.HexToAddress(contractAddr),
		logger:    logger,
	}, nil
}

// SetConfig 设置配置
func (c *Client) SetConfig(config *Config) {
	c.config = config
}

// Close 关闭客户端
func (c *Client) Close() {
	c.ethClient.Close()
}

// TxResult 交易结果
type TxResult struct {
	TxHash      string `json:"tx_hash"`
	BlockNumber uint64 `json:"block_number"`
	Status      string `json:"status"`
}

// SetMemoryHead 设置记忆头指针
func (c *Client) SetMemoryHead(ctx context.Context, agent common.Address, cid string, privateKey string) (*TxResult, error) {
	c.logger.WithFields(logrus.Fields{
		"agent": agent.Hex(),
		"cid":   cid,
	}).Info("Setting memory head on-chain")

	// 解析私钥
	key, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// 转换 CID 为 bytes32
	var cidBytes32 [32]byte
	cidHex := strings.TrimPrefix(cid, "0x")
	cidBytes, err := hex.DecodeString(cidHex)
	if err != nil || len(cidBytes) != 32 {
		// 如果不是有效的 32 字节，使用哈希
		hash := crypto.Keccak256Hash([]byte(cid))
		copy(cidBytes32[:], hash[:])
	} else {
		copy(cidBytes32[:], cidBytes)
	}

	// 打包调用数据
	input, err := c.parsedABI.Pack("setMemoryHead", cidBytes32)
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %w", err)
	}

	// 获取发送者地址
	fromAddress := crypto.PubkeyToAddress(key.PublicKey)

	// 获取 nonce
	nonce, err := c.ethClient.NonceAt(ctx, fromAddress, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// 获取 gas price
	gasPrice := big.NewInt(1000000000) // 1 Gwei
	if c.config != nil && c.config.GasPrice > 0 {
		gasPrice = big.NewInt(int64(c.config.GasPrice))
	} else {
		if suggested, err := c.ethClient.SuggestGasPrice(ctx); err == nil {
			gasPrice = suggested
		}
	}

	// 获取 gas limit
	gasLimit := uint64(100000)
	if c.config != nil && c.config.GasLimit > 0 {
		gasLimit = c.config.GasLimit
	}

	// 创建交易
	tx := types.NewTransaction(nonce, c.contract, big.NewInt(0), gasLimit, gasPrice, input)

	// 签名交易
	chainID := big.NewInt(1)
	if c.config != nil && c.config.ChainID > 0 {
		chainID = big.NewInt(int64(c.config.ChainID))
	}
	signer := types.NewEIP155Signer(chainID)
	signedTx, err := types.SignTx(tx, signer, key)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// 发送交易
	err = c.ethClient.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	c.logger.WithField("txHash", signedTx.Hash().Hex()).Info("Transaction sent")

	// 等待确认
	receipt, err := c.ethClient.TransactionReceipt(ctx, signedTx.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to wait for receipt: %w", err)
	}

	c.logger.WithField("blockNumber", receipt.BlockNumber.Uint64()).Info("Transaction confirmed")

	return &TxResult{
		TxHash:      signedTx.Hash().Hex(),
		BlockNumber: receipt.BlockNumber.Uint64(),
		Status:      fmt.Sprintf("%d", receipt.Status),
	}, nil
}

// GetMemoryHead 获取记忆头指针
func (c *Client) GetMemoryHead(ctx context.Context, agent common.Address) (string, error) {
	// 打包调用数据
	input, err := c.parsedABI.Pack("getMemoryHead", agent)
	if err != nil {
		return "", fmt.Errorf("failed to pack call data: %w", err)
	}

	// 调用合约
	result, err := c.ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &c.contract,
		Data: input,
	}, nil)
	if err != nil {
		return "", fmt.Errorf("failed to call contract: %w", err)
	}

	// 解码结果
	var cid [32]byte
	c.parsedABI.UnpackIntoInterface(&cid, "getMemoryHead", result)
	return common.BytesToHash(cid[:]).Hex(), nil
}

// GetMemoryHistory 获取记忆历史
func (c *Client) GetMemoryHistory(ctx context.Context, agent common.Address) ([]string, error) {
	// 打包调用数据
	input, err := c.parsedABI.Pack("getMemoryHistory", agent)
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %w", err)
	}

	// 调用合约
	result, err := c.ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &c.contract,
		Data: input,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	// 解码结果 (bytes32[])
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

// GetMemoryHistoryLength 获取记忆历史长度
func (c *Client) GetMemoryHistoryLength(ctx context.Context, agent common.Address) (uint64, error) {
	input, err := c.parsedABI.Pack("getMemoryHistoryLength", agent)
	if err != nil {
		return 0, fmt.Errorf("failed to pack call data: %w", err)
	}

	result, err := c.ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &c.contract,
		Data: input,
	}, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to call contract: %w", err)
	}

	return new(big.Int).SetBytes(result).Uint64(), nil
}

// WaitForConfirmation 等待交易确认
func (c *Client) WaitForConfirmation(ctx context.Context, txHash string, timeout time.Duration) (*types.Receipt, error) {
	hash := common.HexToHash(txHash)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		receipt, err := c.ethClient.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}

	return nil, fmt.Errorf("timeout waiting for transaction confirmation")
}

// GetNonce 获取下一个 nonce
func (c *Client) GetNonce(ctx context.Context, address common.Address) (uint64, error) {
	return c.ethClient.NonceAt(ctx, address, nil)
}

// GetGasPrice 获取当前 gas price
func (c *Client) GetGasPrice(ctx context.Context) (*big.Int, error) {
	return c.ethClient.SuggestGasPrice(ctx)
}

// GetBlockNumber 获取当前区块号
func (c *Client) GetBlockNumber(ctx context.Context) (uint64, error) {
	return c.ethClient.BlockNumber(ctx)
}
