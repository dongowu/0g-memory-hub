//go:build ignore
// +build ignore

package main

import (
    "bytes"
    "context"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "math/big"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/crypto"
)

type artifact struct { Bytecode string `json:"bytecode"` }
type rpcReq struct { JSONRPC string `json:"jsonrpc"`; Method string `json:"method"`; Params interface{} `json:"params"`; ID int `json:"id"` }
type rpcResp struct { JSONRPC string `json:"jsonrpc"`; Result json.RawMessage `json:"result"`; Error *rpcErr `json:"error,omitempty"`; ID int `json:"id"` }
type rpcErr struct { Code int `json:"code"`; Message string `json:"message"` }

func call(url string, method string, params interface{}) (json.RawMessage, error) {
    body, _ := json.Marshal(rpcReq{JSONRPC:"2.0", Method:method, Params:params, ID:1})
    req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
    if err != nil { return nil, err }
    req.Header.Set("Content-Type", "application/json")
    resp, err := http.DefaultClient.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    raw, _ := io.ReadAll(resp.Body)
    var r rpcResp
    if err := json.Unmarshal(raw, &r); err != nil { return nil, fmt.Errorf("unmarshal rpc response: %w body=%s", err, string(raw)) }
    if r.Error != nil { return nil, fmt.Errorf("rpc error %d: %s", r.Error.Code, r.Error.Message) }
    return r.Result, nil
}
func mustHexToUint64(raw json.RawMessage) uint64 { var s string; if err:=json.Unmarshal(raw,&s); err!=nil { panic(err)}; n:=new(big.Int); n.SetString(strings.TrimPrefix(s,"0x"),16); return n.Uint64() }
func mustHexToBig(raw json.RawMessage) *big.Int { var s string; if err:=json.Unmarshal(raw,&s); err!=nil { panic(err)}; n:=new(big.Int); n.SetString(strings.TrimPrefix(s,"0x"),16); return n }

func main() {
    rpcURL := os.Getenv("OG_CHAIN_RPC")
    pkHex := os.Getenv("PRIVATE_KEY")
    if rpcURL == "" || pkHex == "" { panic("OG_CHAIN_RPC and PRIVATE_KEY required") }
    artBytes, err := os.ReadFile("../../artifacts/contracts/MemoryAnchor.sol/MemoryAnchor.json")
    if err != nil { panic(err) }
    var art artifact
    if err := json.Unmarshal(artBytes, &art); err != nil { panic(err) }
    bytecode := strings.TrimPrefix(art.Bytecode, "0x")
    data, err := hex.DecodeString(bytecode)
    if err != nil { panic(err) }
    key, err := crypto.HexToECDSA(strings.TrimPrefix(pkHex, "0x"))
    if err != nil { panic(err) }
    from := crypto.PubkeyToAddress(key.PublicKey)
    chainID := big.NewInt(16602)
    nonce := mustHexToUint64(must(call(rpcURL, "eth_getTransactionCount", []interface{}{from.Hex(), "pending"})))
    gasPrice := mustHexToBig(must(call(rpcURL, "eth_gasPrice", []interface{}{})))
    gasLimit := uint64(2500000)
    if estRaw, err := call(rpcURL, "eth_estimateGas", []interface{}{map[string]string{"from": from.Hex(), "data": "0x" + bytecode}}); err == nil {
        gasLimit = mustHexToUint64(estRaw)
    }
    tx := types.NewContractCreation(nonce, big.NewInt(0), gasLimit, gasPrice, data)
    signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
    if err != nil { panic(err) }
    rawTx, err := signed.MarshalBinary()
    if err != nil { panic(err) }
    var txHash string
    if err := json.Unmarshal(must(call(rpcURL, "eth_sendRawTransaction", []interface{}{"0x" + hex.EncodeToString(rawTx)})), &txHash); err != nil { panic(err) }
    fmt.Println("TX_HASH=" + txHash)
    for i:=0;i<60;i++ {
        recRaw, err := call(rpcURL, "eth_getTransactionReceipt", []interface{}{txHash})
        if err == nil && string(recRaw) != "null" {
            var receipt map[string]interface{}
            if err := json.Unmarshal(recRaw, &receipt); err != nil { panic(err) }
            if addr, ok := receipt["contractAddress"].(string); ok { fmt.Println("CONTRACT_ADDRESS=" + addr) }
            if bn, ok := receipt["blockNumber"].(string); ok { fmt.Println("BLOCK_NUMBER=" + bn) }
            if st, ok := receipt["status"].(string); ok { fmt.Println("STATUS=" + st) }
            return
        }
        time.Sleep(3 * time.Second)
    }
    panic("timed out waiting for receipt")
}
func must(v json.RawMessage, err error) json.RawMessage { if err!=nil { panic(err)}; return v }
