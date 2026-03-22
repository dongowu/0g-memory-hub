//go:build ignore
// +build ignore

package main
import (
  "encoding/hex"
  "fmt"
  "math/big"
  "os"
  "strings"
  "github.com/ethereum/go-ethereum/common"
  "github.com/ethereum/go-ethereum/core/types"
  "github.com/ethereum/go-ethereum/crypto"
  "golang.org/x/crypto/sha3"
)
func sel(sig string) string { h:=sha3.NewLegacyKeccak256(); h.Write([]byte(sig)); return hex.EncodeToString(h.Sum(nil)[:4]) }
func wordHex(s string) string { return strings.TrimPrefix(s,"0x") }
func uintWord(v uint64) string { return fmt.Sprintf("%064x", v) }
func parseHexBig(s string)*big.Int{n:=new(big.Int); n.SetString(strings.TrimPrefix(s,"0x"),16); return n}
func main(){
  pk:=os.Getenv("PRIVATE_KEY")
  nonce:=parseHexBig(os.Getenv("NONCE")).Uint64()
  gasPrice:=parseHexBig(os.Getenv("GAS_PRICE"))
  gasLimit:=parseHexBig(os.Getenv("GAS_LIMIT")).Uint64()
  contract:=common.HexToAddress(os.Getenv("CONTRACT_ADDRESS"))
  workflowID:=os.Getenv("WORKFLOW_ID")
  rootHash:=os.Getenv("ROOT_HASH")
  cidHash:=os.Getenv("CID_HASH")
  calldataHex := sel("anchorCheckpoint(bytes32,uint64,bytes32,bytes32)") + wordHex(workflowID) + uintWord(1) + wordHex(rootHash) + wordHex(cidHash)
  data, err := hex.DecodeString(calldataHex)
  if err != nil { panic(err) }
  key, err := crypto.HexToECDSA(strings.TrimPrefix(pk,"0x"))
  if err != nil { panic(err) }
  tx:= types.NewTransaction(nonce, contract, big.NewInt(0), gasLimit, gasPrice, data)
  signed, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(16602)), key)
  if err != nil { panic(err) }
  raw, err := signed.MarshalBinary(); if err != nil { panic(err) }
  fmt.Printf("CALLDATA=0x%s\n", calldataHex)
  fmt.Printf("RAWTX=0x%x\n", raw)
  fmt.Printf("TXHASH=%s\n", signed.Hash().Hex())
}
