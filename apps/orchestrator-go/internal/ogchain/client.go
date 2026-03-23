package ogchain

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

const MemoryAnchorABI = `[
  {"type":"function","name":"anchorCheckpoint","inputs":[{"name":"workflowId","type":"bytes32"},{"name":"stepIndex","type":"uint64"},{"name":"rootHash","type":"bytes32"},{"name":"cidHash","type":"bytes32"}],"outputs":[]},
  {"type":"function","name":"getLatestCheckpoint","inputs":[{"name":"workflowId","type":"bytes32"}],"outputs":[{"name":"stepIndex","type":"uint64"},{"name":"rootHash","type":"bytes32"},{"name":"cidHash","type":"bytes32"},{"name":"timestamp","type":"uint64"},{"name":"submitter","type":"address"}]}
]`

var (
	ErrMissingPrivateKey  = errors.New("private key is required for signed transaction submission")
	ErrInvalidChainID     = errors.New("invalid chain id")
	ErrInvalidContract    = errors.New("invalid contract address")
	ErrInvalidTransaction = errors.New("invalid signed transaction")
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client interface {
	AnchorCheckpoint(ctx context.Context, in AnchorInput) (*AnchorResult, error)
	GetLatestCheckpoint(ctx context.Context, workflowID string) (*LatestCheckpoint, error)
}

type JSONRPCClient struct {
	rpcURL          string
	privateKey      string
	contractAddress string
	chainID         string
	httpClient      HTTPClient
}

type AnchorInput struct {
	WorkflowID string
	StepIndex  uint64
	RootHash   string
	CIDHash    string
}

type AnchorResult struct {
	CallData string
	TxHash   string
}

type LatestCheckpoint struct {
	StepIndex uint64
	RootHash  string
	CIDHash   string
	Timestamp uint64
	Submitter string
}

func NewJSONRPCClient(rpcURL, privateKey, contractAddress, chainID string, httpClient HTTPClient) *JSONRPCClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &JSONRPCClient{
		rpcURL:          rpcURL,
		privateKey:      privateKey,
		contractAddress: contractAddress,
		chainID:         chainID,
		httpClient:      httpClient,
	}
}

func (c *JSONRPCClient) CheckReadiness(_ context.Context) error {
	if strings.TrimSpace(c.rpcURL) == "" {
		return fmt.Errorf("chain RPC URL is required")
	}
	if strings.TrimSpace(c.privateKey) == "" {
		return ErrMissingPrivateKey
	}
	if _, err := parseChainID(c.chainID); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidChainID, err)
	}
	if !common.IsHexAddress(c.contractAddress) {
		return ErrInvalidContract
	}
	if _, err := crypto.HexToECDSA(strings.TrimPrefix(c.privateKey, "0x")); err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}
	return nil
}

func (c *JSONRPCClient) AnchorCheckpoint(ctx context.Context, in AnchorInput) (*AnchorResult, error) {
	data, err := encodeAnchorCheckpointInput(in.WorkflowID, in.StepIndex, in.RootHash, in.CIDHash)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(c.privateKey) == "" {
		return nil, ErrMissingPrivateKey
	}

	chainID, err := parseChainID(c.chainID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidChainID, err)
	}

	if !common.IsHexAddress(c.contractAddress) {
		return nil, ErrInvalidContract
	}
	contractAddr := common.HexToAddress(c.contractAddress)

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(c.privateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	fromAddr := crypto.PubkeyToAddress(privateKey.PublicKey)

	nonce, err := c.getNonce(ctx, fromAddr)
	if err != nil {
		return nil, err
	}

	gasPrice, err := c.getGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	gasLimit, err := c.estimateGas(ctx, fromAddr, contractAddr, data)
	if err != nil {
		// Keep the request robust across RPC providers with partial method support.
		gasLimit = 300000
	}

	txData, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode call data: %w", err)
	}
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &contractAddr,
		Value:    big.NewInt(0),
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     txData,
	})

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	rawTx, err := signedTx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal signed transaction: %w", err)
	}

	txHash, err := c.sendRawTransaction(ctx, "0x"+hex.EncodeToString(rawTx))
	if err != nil {
		return nil, err
	}

	return &AnchorResult{
		CallData: data,
		TxHash:   txHash,
	}, nil
}

func (c *JSONRPCClient) GetLatestCheckpoint(ctx context.Context, workflowID string) (*LatestCheckpoint, error) {
	inputData, err := encodeGetLatestCheckpointInput(workflowID)
	if err != nil {
		return nil, err
	}

	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  "eth_call",
		Params: []interface{}{
			map[string]string{
				"to":   c.contractAddress,
				"data": inputData,
			},
			"latest",
		},
		ID: 1,
	}

	rawResult, err := c.call(ctx, req)
	if err != nil {
		return nil, err
	}

	var encoded string
	if err := json.Unmarshal(rawResult, &encoded); err != nil {
		return nil, fmt.Errorf("decode eth_call result: %w", err)
	}
	return decodeLatestCheckpoint(encoded)
}

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error,omitempty"`
	ID      int             `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *JSONRPCClient) call(ctx context.Context, req rpcRequest) (json.RawMessage, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal json-rpc request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send json-rpc request: %w", err)
	}
	defer resp.Body.Close()

	rawResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read json-rpc response: %w", err)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(rawResp, &rpcResp); err != nil {
		return nil, fmt.Errorf("decode json-rpc response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("json-rpc error (%d): %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

func (c *JSONRPCClient) getNonce(ctx context.Context, from common.Address) (uint64, error) {
	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  "eth_getTransactionCount",
		Params:  []interface{}{from.Hex(), "pending"},
		ID:      1,
	}
	raw, err := c.call(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("get nonce: %w", err)
	}
	return hexToUint64(raw)
}

func (c *JSONRPCClient) getGasPrice(ctx context.Context) (*big.Int, error) {
	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  "eth_gasPrice",
		Params:  []interface{}{},
		ID:      1,
	}
	raw, err := c.call(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get gas price: %w", err)
	}
	return hexToBigInt(raw)
}

func (c *JSONRPCClient) estimateGas(ctx context.Context, from common.Address, to common.Address, data string) (uint64, error) {
	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  "eth_estimateGas",
		Params: []interface{}{
			map[string]string{
				"from": from.Hex(),
				"to":   to.Hex(),
				"data": data,
			},
		},
		ID: 1,
	}
	raw, err := c.call(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("estimate gas: %w", err)
	}
	return hexToUint64(raw)
}

func (c *JSONRPCClient) sendRawTransaction(ctx context.Context, rawTx string) (string, error) {
	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  "eth_sendRawTransaction",
		Params:  []interface{}{rawTx},
		ID:      1,
	}
	raw, err := c.call(ctx, req)
	if err != nil {
		return "", fmt.Errorf("send raw transaction: %w", err)
	}
	var txHash string
	if err := json.Unmarshal(raw, &txHash); err != nil {
		return "", fmt.Errorf("decode tx hash: %w", err)
	}
	if !strings.HasPrefix(txHash, "0x") || len(txHash) != 66 {
		return "", fmt.Errorf("%w: %s", ErrInvalidTransaction, txHash)
	}
	return strings.ToLower(txHash), nil
}

func encodeAnchorCheckpointInput(workflowID string, stepIndex uint64, rootHash string, cidHash string) (string, error) {
	selector := methodSelector("anchorCheckpoint(bytes32,uint64,bytes32,bytes32)")
	wf, err := parseBytes32(workflowID)
	if err != nil {
		return "", fmt.Errorf("parse workflowID: %w", err)
	}
	root, err := parseBytes32(rootHash)
	if err != nil {
		return "", fmt.Errorf("parse rootHash: %w", err)
	}
	cid, err := parseBytes32(cidHash)
	if err != nil {
		return "", fmt.Errorf("parse cidHash: %w", err)
	}
	stepWord := uintToWord(stepIndex)
	return "0x" + selector + wf + stepWord + root + cid, nil
}

func encodeGetLatestCheckpointInput(workflowID string) (string, error) {
	selector := methodSelector("getLatestCheckpoint(bytes32)")
	wf, err := parseBytes32(workflowID)
	if err != nil {
		return "", fmt.Errorf("parse workflowID: %w", err)
	}
	return "0x" + selector + wf, nil
}

func decodeLatestCheckpoint(encoded string) (*LatestCheckpoint, error) {
	data := strings.TrimPrefix(encoded, "0x")
	if len(data) < 64*5 {
		return nil, fmt.Errorf("unexpected encoded length: %d", len(data))
	}

	word := func(i int) string {
		start := i * 64
		return data[start : start+64]
	}

	step := hexWordToUint64(word(0))
	ts := hexWordToUint64(word(3))
	submitterWord := word(4)
	submitter := "0x" + submitterWord[24:]

	return &LatestCheckpoint{
		StepIndex: step,
		RootHash:  "0x" + word(1),
		CIDHash:   "0x" + word(2),
		Timestamp: ts,
		Submitter: strings.ToLower(submitter),
	}, nil
}

func parseBytes32(value string) (string, error) {
	clean := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	if len(clean) != 64 {
		return "", fmt.Errorf("expected 32-byte hex string, got %d hex chars", len(clean))
	}
	if _, err := hex.DecodeString(clean); err != nil {
		return "", fmt.Errorf("invalid hex: %w", err)
	}
	return strings.ToLower(clean), nil
}

func uintToWord(v uint64) string {
	n := new(big.Int).SetUint64(v)
	return fmt.Sprintf("%064x", n)
}

func hexWordToUint64(word string) uint64 {
	n := new(big.Int)
	n.SetString(word, 16)
	return n.Uint64()
}

func methodSelector(signature string) string {
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write([]byte(signature))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:4])
}

func parseChainID(chainID string) (*big.Int, error) {
	clean := strings.TrimSpace(chainID)
	if clean == "" {
		return nil, ErrInvalidChainID
	}
	base := 10
	if strings.HasPrefix(clean, "0x") {
		base = 16
		clean = strings.TrimPrefix(clean, "0x")
	}
	n := new(big.Int)
	if _, ok := n.SetString(clean, base); !ok || n.Sign() <= 0 {
		return nil, ErrInvalidChainID
	}
	return n, nil
}

func hexToUint64(raw json.RawMessage) (uint64, error) {
	n, err := hexToBigInt(raw)
	if err != nil {
		return 0, err
	}
	return n.Uint64(), nil
}

func hexToBigInt(raw json.RawMessage) (*big.Int, error) {
	var hexVal string
	if err := json.Unmarshal(raw, &hexVal); err != nil {
		return nil, fmt.Errorf("decode hex quantity: %w", err)
	}
	hexVal = strings.TrimPrefix(strings.TrimSpace(hexVal), "0x")
	if hexVal == "" {
		hexVal = "0"
	}
	n := new(big.Int)
	if _, ok := n.SetString(hexVal, 16); !ok {
		return nil, fmt.Errorf("invalid hex quantity: %s", hexVal)
	}
	return n, nil
}
