package ogchain

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestGetLatestCheckpoint(t *testing.T) {
	t.Parallel()

	// ABI encoded tuple:
	// step=7, root=0x11..., cid=0x22..., timestamp=1234, submitter=0x...deadbeef
	encodedResult := "0x" +
		"0000000000000000000000000000000000000000000000000000000000000007" +
		"1111111111111111111111111111111111111111111111111111111111111111" +
		"2222222222222222222222222222222222222222222222222222222222222222" +
		"00000000000000000000000000000000000000000000000000000000000004d2" +
		"00000000000000000000000000000000000000000000000000000000deadbeef"

	client := NewJSONRPCClient("http://rpc.local", "", "0x0000000000000000000000000000000000000001", "1", &scriptedHTTPClient{
		t: t,
		handlers: []rpcHandler{
			func(req rpcRequest) rpcResponse {
				if req.Method != "eth_call" {
					t.Fatalf("method = %s, want eth_call", req.Method)
				}
				return rpcResponse{
					JSONRPC: "2.0",
					ID:      1,
					Result:  mustRawJSON(t, encodedResult),
				}
			},
		},
	})
	got, err := client.GetLatestCheckpoint(context.Background(), "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("GetLatestCheckpoint() error = %v", err)
	}
	if got.StepIndex != 7 {
		t.Fatalf("StepIndex = %d, want 7", got.StepIndex)
	}
	if got.Timestamp != 1234 {
		t.Fatalf("Timestamp = %d, want 1234", got.Timestamp)
	}
	if got.RootHash != "0x1111111111111111111111111111111111111111111111111111111111111111" {
		t.Fatalf("RootHash = %s", got.RootHash)
	}
	if got.CIDHash != "0x2222222222222222222222222222222222222222222222222222222222222222" {
		t.Fatalf("CIDHash = %s", got.CIDHash)
	}
	if got.Submitter != "0x00000000000000000000000000000000deadbeef" {
		t.Fatalf("Submitter = %s", got.Submitter)
	}
}

func TestAnchorCheckpointSignsAndSubmits(t *testing.T) {
	t.Parallel()

	const privateKey = "0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae9c645f8af9f4f26"
	const contractAddr = "0x00000000000000000000000000000000000000aa"

	client := NewJSONRPCClient("http://rpc.local", privateKey, contractAddr, "1", &scriptedHTTPClient{
		t: t,
		handlers: []rpcHandler{
			func(req rpcRequest) rpcResponse {
				if req.Method != "eth_getTransactionCount" {
					t.Fatalf("method = %s, want eth_getTransactionCount", req.Method)
				}
				return rpcResponse{JSONRPC: "2.0", ID: 1, Result: mustRawJSON(t, "0x1")}
			},
			func(req rpcRequest) rpcResponse {
				if req.Method != "eth_gasPrice" {
					t.Fatalf("method = %s, want eth_gasPrice", req.Method)
				}
				return rpcResponse{JSONRPC: "2.0", ID: 1, Result: mustRawJSON(t, "0x3b9aca00")}
			},
			func(req rpcRequest) rpcResponse {
				if req.Method != "eth_estimateGas" {
					t.Fatalf("method = %s, want eth_estimateGas", req.Method)
				}
				return rpcResponse{JSONRPC: "2.0", ID: 1, Result: mustRawJSON(t, "0x5208")}
			},
			func(req rpcRequest) rpcResponse {
				if req.Method != "eth_sendRawTransaction" {
					t.Fatalf("method = %s, want eth_sendRawTransaction", req.Method)
				}
				params, ok := req.Params.([]interface{})
				if !ok || len(params) != 1 {
					t.Fatalf("unexpected params: %#v", req.Params)
				}
				rawHex, ok := params[0].(string)
				if !ok || !strings.HasPrefix(rawHex, "0x") {
					t.Fatalf("raw tx must be hex string, got: %#v", params[0])
				}
				raw, err := hex.DecodeString(strings.TrimPrefix(rawHex, "0x"))
				if err != nil {
					t.Fatalf("decode raw tx: %v", err)
				}
				var tx types.Transaction
				if err := tx.UnmarshalBinary(raw); err != nil {
					t.Fatalf("unmarshal raw tx: %v", err)
				}
				if tx.Nonce() != 1 {
					t.Fatalf("nonce = %d, want 1", tx.Nonce())
				}
				if tx.Gas() != 0x5208 {
					t.Fatalf("gas = %d, want 21000", tx.Gas())
				}
				if tx.GasPrice().Cmp(big.NewInt(1000000000)) != 0 {
					t.Fatalf("gas price = %s", tx.GasPrice().String())
				}
				to := tx.To()
				if to == nil || strings.ToLower(to.Hex()) != contractAddr {
					t.Fatalf("to = %v, want %s", to, contractAddr)
				}
				data := tx.Data()
				if len(data) < 4 {
					t.Fatalf("tx data too short: %d", len(data))
				}
				if hex.EncodeToString(data[:4]) != "177df226" {
					t.Fatalf("selector = %s, want 177df226", hex.EncodeToString(data[:4]))
				}
				return rpcResponse{
					JSONRPC: "2.0",
					ID:      1,
					Result:  mustRawJSON(t, "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				}
			},
		},
	})

	result, err := client.AnchorCheckpoint(context.Background(), AnchorInput{
		WorkflowID: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		StepIndex:  1,
		RootHash:   "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		CIDHash:    "0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	})
	if err == nil {
		// continue
	} else {
		t.Fatalf("AnchorCheckpoint() error = %v", err)
	}
	if result == nil || result.CallData == "" {
		t.Fatalf("expected calldata in result, got %+v", result)
	}
	if result.TxHash != "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("TxHash = %s", result.TxHash)
	}
}

func TestAnchorCheckpointRequiresPrivateKey(t *testing.T) {
	t.Parallel()

	client := NewJSONRPCClient("http://rpc.local", "", "0x0000000000000000000000000000000000000001", "1", nil)
	_, err := client.AnchorCheckpoint(context.Background(), AnchorInput{
		WorkflowID: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		StepIndex:  1,
		RootHash:   "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		CIDHash:    "0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	})
	if err == nil {
		t.Fatal("expected private key error, got nil")
	}
	if !strings.Contains(err.Error(), ErrMissingPrivateKey.Error()) {
		t.Fatalf("error = %v, want %v", err, ErrMissingPrivateKey)
	}
}

func TestMethodSelectorKnownValue(t *testing.T) {
	t.Parallel()
	got := methodSelector("getLatestCheckpoint(bytes32)")
	if got != "181b4e65" {
		t.Fatalf("selector = %s, want 181b4e65", got)
	}
}

type rpcHandler func(req rpcRequest) rpcResponse

type scriptedHTTPClient struct {
	t        *testing.T
	handlers []rpcHandler
	index    int
}

func (c *scriptedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.t.Helper()
	if c.index >= len(c.handlers) {
		c.t.Fatalf("unexpected rpc call index=%d", c.index)
	}

	rawReq, err := io.ReadAll(req.Body)
	if err != nil {
		c.t.Fatalf("read body: %v", err)
	}
	var parsed rpcRequest
	if err := json.Unmarshal(rawReq, &parsed); err != nil {
		c.t.Fatalf("decode rpc request: %v", err)
	}

	handler := c.handlers[c.index]
	c.index++
	respObj := handler(parsed)
	rawResp, err := json.Marshal(respObj)
	if err != nil {
		c.t.Fatalf("encode rpc response: %v", err)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(rawResp)),
		Header:     make(http.Header),
	}, nil
}

func mustRawJSON(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal test json: %v", err)
	}
	return raw
}
