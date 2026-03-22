//go:build ignore
// +build ignore

package main
import (
  "encoding/hex"
  "encoding/json"
  "fmt"
  "math/big"
  "os"
  "strings"
  "github.com/ethereum/go-ethereum/core/types"
  "github.com/ethereum/go-ethereum/crypto"
)
type artifact struct{ Bytecode string `json:"bytecode"` }
func parseHexBig(s string) *big.Int { n:=new(big.Int); n.SetString(strings.TrimPrefix(s,"0x"),16); return n }
func main(){
  pk:=os.Getenv("PRIVATE_KEY")
  nonce:=parseHexBig(os.Getenv("NONCE"))
  gasPrice:=parseHexBig(os.Getenv("GAS_PRICE"))
  gasLimit:=parseHexBig(os.Getenv("GAS_LIMIT")).Uint64()
  artBytes, err := os.ReadFile("../../artifacts/contracts/MemoryAnchor.sol/MemoryAnchor.json")
  if err != nil { panic(err) }
  var art artifact
  if err := json.Unmarshal(artBytes,&art); err != nil { panic(err) }
  data, err := hex.DecodeString(strings.TrimPrefix(art.Bytecode,"0x"))
  if err != nil { panic(err) }
  key, err := crypto.HexToECDSA(strings.TrimPrefix(pk,"0x"))
  if err != nil { panic(err) }
  tx := types.NewContractCreation(nonce.Uint64(), big.NewInt(0), gasLimit, gasPrice, data)
  signed, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(16602)), key)
  if err != nil { panic(err) }
  raw, err := signed.MarshalBinary()
  if err != nil { panic(err) }
  fmt.Printf("0x%x\n", raw)
}
