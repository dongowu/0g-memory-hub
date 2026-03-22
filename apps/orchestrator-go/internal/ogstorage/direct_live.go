package ogstorage

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/0gfoundation/0g-storage-client/contract"
	storagemerkle "github.com/0gfoundation/0g-storage-client/core/merkle"
	"github.com/0gfoundation/0g-storage-client/node"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	defaultTurboIndexerURL        = "https://indexer-storage-testnet-turbo.0g.ai"
	directFallbackFileName        = "checkpoint.bin"
	directFallbackPollAttempts    = 40
	directFallbackPollInterval    = 3 * time.Second
	directFallbackExpectedReplica = 1
)

const (
	flowQueryABIJSON = `[
  {"type":"function","name":"market","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"address"}]},
  {"type":"function","name":"submit","stateMutability":"payable","inputs":[{"name":"submission","type":"tuple","components":[{"name":"data","type":"tuple","components":[{"name":"length","type":"uint256"},{"name":"tags","type":"bytes"},{"name":"nodes","type":"tuple[]","components":[{"name":"root","type":"bytes32"},{"name":"height","type":"uint256"}]}]},{"name":"submitter","type":"address"}]}],"outputs":[{"name":"index","type":"uint256"},{"name":"digest","type":"bytes32"},{"name":"startIndex","type":"uint256"},{"name":"length","type":"uint256"}]}
]`
	marketQueryABIJSON = `[
  {"type":"function","name":"pricePerSector","stateMutability":"view","inputs":[],"outputs":[{"name":"","type":"uint256"}]}
]`
)

type directNodeStatusEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		NetworkIdentity struct {
			ChainID     int64  `json:"chainId"`
			FlowAddress string `json:"flowAddress"`
		} `json:"networkIdentity"`
	} `json:"data"`
}

type directFileInfoEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		UploadedSegNum int64 `json:"uploadedSegNum"`
		Finalized      bool  `json:"finalized"`
	} `json:"data"`
}

type directSegmentUploadEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type directRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type directRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *directRPCError `json:"error,omitempty"`
	ID      int             `json:"id"`
}

type directRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type directSegmentUploadRequest struct {
	Root            common.Hash         `json:"root"`
	Data            []byte              `json:"data"`
	Index           uint64              `json:"index"`
	Proof           storagemerkle.Proof `json:"proof"`
	ExpectedReplica uint                `json:"expectedReplica"`
}

func candidateIndexerBaseURLs(primary string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 2)
	for _, raw := range []string{strings.TrimRight(primary, "/"), defaultTurboIndexerURL} {
		if raw == "" || seen[raw] {
			continue
		}
		seen[raw] = true
		out = append(out, raw)
	}
	return out
}

func uploadDirect(ctx context.Context, cfg SDKConfig, payload []byte) (string, string, error) {
	baseURL, flowAddress, err := resolveDirectBaseAndFlow(ctx, candidateIndexerBaseURLs(cfg.IndexerRPCURL))
	if err != nil {
		return "", "", err
	}

	plan, err := buildDirectUploadPlan(payload)
	if err != nil {
		return "", "", err
	}

	privateKey, from, err := storagePrivateKey(cfg)
	if err != nil {
		return "", "", err
	}
	submission, err := plan.submission(from)
	if err != nil {
		return "", "", err
	}

	root := plan.root().Hex()
	marketAddress, err := queryFlowMarket(ctx, cfg.BlockchainRPCURL, flowAddress)
	if err != nil {
		return "", "", err
	}
	pricePerSector, err := queryPricePerSector(ctx, cfg.BlockchainRPCURL, marketAddress)
	if err != nil {
		return "", "", err
	}
	txHash, err := submitFlowSubmission(ctx, cfg, flowAddress, submission, pricePerSector, privateKey)
	if err != nil {
		return "", "", err
	}
	if err := waitForDirectFilePresence(ctx, baseURL, root); err != nil {
		return "", "", err
	}

	segments, err := plan.segments()
	if err != nil {
		return "", "", err
	}
	expectedReplica := cfg.ExpectedReplica
	if expectedReplica == 0 {
		expectedReplica = directFallbackExpectedReplica
	}
	for _, segment := range segments {
		if err := uploadSegmentToGateway(ctx, baseURL, segment, expectedReplica); err != nil {
			return "", "", err
		}
	}
	if err := waitForDirectUploadComplete(ctx, baseURL, root, plan.data.NumSegments()); err != nil {
		return "", "", err
	}
	return root, txHash, nil
}

func resolveDirectBaseAndFlow(ctx context.Context, candidates []string) (string, string, error) {
	var errs []string
	for _, baseURL := range candidates {
		flowAddress, err := fetchFlowAddress(ctx, baseURL)
		if err == nil {
			return baseURL, flowAddress, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", baseURL, err))
	}
	return "", "", fmt.Errorf("resolve 0G direct fallback base: %s", strings.Join(errs, "; "))
}

func fetchFlowAddress(ctx context.Context, baseURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/node/status", nil)
	if err != nil {
		return "", fmt.Errorf("create /node/status request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request /node/status: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read /node/status response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("/node/status http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var envelope directNodeStatusEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", fmt.Errorf("decode /node/status response: %w", err)
	}
	if envelope.Code != 0 || envelope.Data.NetworkIdentity.FlowAddress == "" {
		return "", fmt.Errorf("unexpected /node/status payload: %s", string(raw))
	}
	return envelope.Data.NetworkIdentity.FlowAddress, nil
}

func queryFlowMarket(ctx context.Context, rpcURL, flowAddress string) (string, error) {
	contractABI, err := abi.JSON(strings.NewReader(flowQueryABIJSON))
	if err != nil {
		return "", fmt.Errorf("parse flow ABI: %w", err)
	}
	callData, err := contractABI.Pack("market")
	if err != nil {
		return "", fmt.Errorf("pack flow.market call: %w", err)
	}
	encoded, err := ethCall(ctx, rpcURL, flowAddress, "0x"+hex.EncodeToString(callData))
	if err != nil {
		return "", fmt.Errorf("call flow.market: %w", err)
	}
	values, err := contractABI.Unpack("market", hexToBytes(encoded))
	if err != nil {
		return "", fmt.Errorf("unpack flow.market response: %w", err)
	}
	if len(values) != 1 {
		return "", fmt.Errorf("unexpected flow.market outputs: %d", len(values))
	}
	addr, ok := values[0].(common.Address)
	if !ok {
		return "", fmt.Errorf("unexpected flow.market output type %T", values[0])
	}
	return addr.Hex(), nil
}

func queryPricePerSector(ctx context.Context, rpcURL, marketAddress string) (*big.Int, error) {
	contractABI, err := abi.JSON(strings.NewReader(marketQueryABIJSON))
	if err != nil {
		return nil, fmt.Errorf("parse market ABI: %w", err)
	}
	callData, err := contractABI.Pack("pricePerSector")
	if err != nil {
		return nil, fmt.Errorf("pack pricePerSector call: %w", err)
	}
	encoded, err := ethCall(ctx, rpcURL, marketAddress, "0x"+hex.EncodeToString(callData))
	if err != nil {
		return nil, fmt.Errorf("call pricePerSector: %w", err)
	}
	values, err := contractABI.Unpack("pricePerSector", hexToBytes(encoded))
	if err != nil {
		return nil, fmt.Errorf("unpack pricePerSector response: %w", err)
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("unexpected pricePerSector outputs: %d", len(values))
	}
	price, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected pricePerSector output type %T", values[0])
	}
	return price, nil
}

func storagePrivateKey(cfg SDKConfig) (*ecdsa.PrivateKey, common.Address, error) {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("parse storage private key: %w", err)
	}
	return privateKey, crypto.PubkeyToAddress(privateKey.PublicKey), nil
}

func submitFlowSubmission(ctx context.Context, cfg SDKConfig, flowAddress string, submission *contract.Submission, pricePerSector *big.Int, privateKey *ecdsa.PrivateKey) (string, error) {
	contractABI, err := abi.JSON(strings.NewReader(flowQueryABIJSON))
	if err != nil {
		return "", fmt.Errorf("parse flow submit ABI: %w", err)
	}
	from := crypto.PubkeyToAddress(privateKey.PublicKey)
	fee := submission.Fee(pricePerSector)

	calldata, err := contractABI.Pack("submit", *submission)
	if err != nil {
		return "", fmt.Errorf("pack flow submit calldata: %w", err)
	}

	chainID, ok := new(big.Int).SetString(cfgChainIDOrDefault(cfg), 10)
	if !ok {
		return "", fmt.Errorf("invalid chain id %q", cfg.ChainID)
	}
	nonce, err := rpcHexToUint64(callRPC(ctx, cfg.BlockchainRPCURL, directRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_getTransactionCount",
		Params:  []interface{}{from.Hex(), "pending"},
		ID:      1,
	}))
	if err != nil {
		return "", fmt.Errorf("get nonce: %w", err)
	}
	gasPrice, err := rpcHexToBig(callRPC(ctx, cfg.BlockchainRPCURL, directRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_gasPrice",
		Params:  []interface{}{},
		ID:      1,
	}))
	if err != nil {
		return "", fmt.Errorf("get gas price: %w", err)
	}
	gasLimit := uint64(500000)
	if estimated, err := rpcHexToUint64(callRPC(ctx, cfg.BlockchainRPCURL, directRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_estimateGas",
		Params: []interface{}{map[string]string{
			"from":  from.Hex(),
			"to":    flowAddress,
			"data":  "0x" + hex.EncodeToString(calldata),
			"value": "0x" + fee.Text(16),
		}},
		ID: 1,
	})); err == nil && estimated > 0 {
		gasLimit = estimated
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       ptrAddress(common.HexToAddress(flowAddress)),
		Value:    fee,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     calldata,
	})
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("sign flow submit tx: %w", err)
	}
	rawTx, err := signed.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("marshal flow submit tx: %w", err)
	}
	txHash, err := rpcHexString(callRPC(ctx, cfg.BlockchainRPCURL, directRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_sendRawTransaction",
		Params:  []interface{}{"0x" + hex.EncodeToString(rawTx)},
		ID:      1,
	}))
	if err != nil {
		return "", fmt.Errorf("send flow submit tx: %w", err)
	}
	if err := waitForReceipt(ctx, cfg.BlockchainRPCURL, txHash); err != nil {
		return "", err
	}
	return txHash, nil
}

func waitForReceipt(ctx context.Context, rpcURL, txHash string) error {
	for i := 0; i < directFallbackPollAttempts; i++ {
		raw, err := callRPC(ctx, rpcURL, directRPCRequest{
			JSONRPC: "2.0",
			Method:  "eth_getTransactionReceipt",
			Params:  []interface{}{txHash},
			ID:      1,
		})
		if err == nil && string(bytes.TrimSpace(raw)) != "null" {
			var receipt struct {
				Status string `json:"status"`
			}
			if err := json.Unmarshal(raw, &receipt); err != nil {
				return fmt.Errorf("decode tx receipt: %w", err)
			}
			if receipt.Status == "0x1" {
				return nil
			}
			return fmt.Errorf("flow submit tx failed with status %s", receipt.Status)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(directFallbackPollInterval):
		}
	}
	return fmt.Errorf("timed out waiting for flow submit receipt %s", txHash)
}

func waitForDirectFilePresence(ctx context.Context, baseURL, root string) error {
	for i := 0; i < directFallbackPollAttempts; i++ {
		info, err := getFileInfo(ctx, baseURL, root)
		if err == nil && info.Code == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(directFallbackPollInterval):
		}
	}
	return fmt.Errorf("timed out waiting for file presence for root %s", root)
}

func uploadSegmentToGateway(ctx context.Context, baseURL string, segment node.SegmentWithProof, expectedReplica uint) error {
	body, err := json.Marshal(directSegmentUploadRequest{
		Root:            segment.Root,
		Data:            segment.Data,
		Index:           segment.Index,
		Proof:           segment.Proof,
		ExpectedReplica: expectedReplica,
	})
	if err != nil {
		return fmt.Errorf("marshal /file/segment payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/file/segment", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create /file/segment request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request /file/segment: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read /file/segment response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("/file/segment http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var envelope directSegmentUploadEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode /file/segment response: %w", err)
	}
	if envelope.Code != 0 {
		return fmt.Errorf("/file/segment code=%d message=%s", envelope.Code, envelope.Message)
	}
	return nil
}

func waitForDirectUploadComplete(ctx context.Context, baseURL, root string, expectedSegments uint64) error {
	for i := 0; i < directFallbackPollAttempts; i++ {
		info, err := getFileInfo(ctx, baseURL, root)
		if err == nil && info.Code == 0 && uint64(info.Data.UploadedSegNum) >= expectedSegments {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(directFallbackPollInterval):
		}
	}
	return fmt.Errorf("timed out waiting for uploadedSegNum >= %d for root %s", expectedSegments, root)
}

func downloadViaRESTCandidates(ctx context.Context, key string, candidates []string) ([]byte, error) {
	var errs []string
	for _, baseURL := range candidates {
		payload, err := downloadViaREST(ctx, baseURL, key)
		if err == nil {
			return payload, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", baseURL, err))
	}
	return nil, fmt.Errorf("download via 0G REST failed: %s", strings.Join(errs, "; "))
}

func downloadViaREST(ctx context.Context, baseURL, key string) ([]byte, error) {
	u := strings.TrimRight(baseURL, "/") + "/file?root=" + url.QueryEscape(key) + "&name=" + url.QueryEscape(directFallbackFileName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create /file request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request /file: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read /file response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("/file http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return raw, nil
}

func getFileInfo(ctx context.Context, baseURL, root string) (*directFileInfoEnvelope, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/file/info/"+root, nil)
	if err != nil {
		return nil, fmt.Errorf("create /file/info request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request /file/info: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read /file/info response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("/file/info http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var envelope directFileInfoEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode /file/info response: %w", err)
	}
	return &envelope, nil
}

func ethCall(ctx context.Context, rpcURL, to, data string) (string, error) {
	return rpcHexString(callRPC(ctx, rpcURL, directRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_call",
		Params: []interface{}{
			map[string]string{
				"to":   to,
				"data": data,
			},
			"latest",
		},
		ID: 1,
	}))
}

func callRPC(ctx context.Context, rpcURL string, req directRPCRequest) (json.RawMessage, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal rpc request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create rpc request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send rpc request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read rpc response: %w", err)
	}

	var rpcResp directRPCResponse
	if err := json.Unmarshal(raw, &rpcResp); err != nil {
		return nil, fmt.Errorf("decode rpc response: %w body=%s", err, string(raw))
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error (%d): %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

func rpcHexString(raw json.RawMessage, err error) (string, error) {
	if err != nil {
		return "", err
	}
	var out string
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("decode hex string: %w", err)
	}
	return out, nil
}

func rpcHexToUint64(raw json.RawMessage, err error) (uint64, error) {
	encoded, err := rpcHexString(raw, err)
	if err != nil {
		return 0, err
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(encoded, "0x"), 16)
	if !ok {
		return 0, fmt.Errorf("invalid hex uint64 %q", encoded)
	}
	return n.Uint64(), nil
}

func rpcHexToBig(raw json.RawMessage, err error) (*big.Int, error) {
	encoded, err := rpcHexString(raw, err)
	if err != nil {
		return nil, err
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(encoded, "0x"), 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex big integer %q", encoded)
	}
	return n, nil
}

func cfgChainIDOrDefault(cfg SDKConfig) string {
	if strings.TrimSpace(cfg.ChainID) == "" {
		return "16602"
	}
	return cfg.ChainID
}

func hexToBytes(encoded string) []byte {
	out, _ := hex.DecodeString(strings.TrimPrefix(encoded, "0x"))
	return out
}

func ptrAddress(addr common.Address) *common.Address {
	return &addr
}
