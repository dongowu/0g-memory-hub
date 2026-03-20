package cmd

import (
	"fmt"
	"os"

	"github.com/dongowu/0g-memory-hub/sdk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config 全局配置
type Config struct {
	Wallet         string
	DataDir        string
	StorageRPC     string
	IndexerRPC     string
	ChainRPC       string
	ContractAddr   string
	PrivateKey     string
	ChainID        uint64
	LogLevel       string
}

var cfg = Config{}

// MemoryHub 全局 SDK 实例
var hub *sdk.MemoryHub

// RootCmd 根命令
var RootCmd = &cobra.Command{
	Use:   "memory-hub",
	Short: "0G Memory Hub - AI Agent Eternal Memory System",
	Long: `0G Memory Hub provides immutable, verifiable memory storage for AI agents
using 0G's decentralized infrastructure (Storage + Chain).

Features:
  - Persistent memory storage on 0G
  - Memory chain with hash pointers
  - Query and replay history
  - Agent-native design

For more information, visit: https://docs.0g.ai/`,
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfg.Wallet, "wallet", "", "Wallet address")
	RootCmd.PersistentFlags().StringVar(&cfg.DataDir, "data-dir", "./data", "Data directory")
	RootCmd.PersistentFlags().StringVar(&cfg.StorageRPC, "storage-rpc", "https://testnet-rpc.0g.ai", "0G Storage RPC URL")
	RootCmd.PersistentFlags().StringVar(&cfg.IndexerRPC, "indexer-rpc", "https://testnet-indexer.0g.ai", "0G Indexer RPC URL")
	RootCmd.PersistentFlags().StringVar(&cfg.ChainRPC, "chain-rpc", "https://testnet-chain-rpc.0g.ai", "0G Chain RPC URL")
	RootCmd.PersistentFlags().StringVar(&cfg.ContractAddr, "contract", "", "MemoryChain contract address")
	RootCmd.PersistentFlags().StringVar(&cfg.PrivateKey, "key", "", "Private key for transactions")
	RootCmd.PersistentFlags().Uint64Var(&cfg.ChainID, "chain-id", 1, "Chain ID")
	RootCmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", "info", "Log level (debug, info, warn)")

	viper.BindPFlag("DATA_DIR", RootCmd.PersistentFlags().Lookup("data-dir"))
	viper.BindPFlag("STORAGE_RPC", RootCmd.PersistentFlags().Lookup("storage-rpc"))
	viper.BindPFlag("INDEXER_RPC", RootCmd.PersistentFlags().Lookup("indexer-rpc"))
	viper.BindPFlag("CHAIN_RPC", RootCmd.PersistentFlags().Lookup("chain-rpc"))
	viper.BindPFlag("CONTRACT", RootCmd.PersistentFlags().Lookup("contract"))
	viper.BindPFlag("KEY", RootCmd.PersistentFlags().Lookup("key"))
	viper.BindPFlag("CHAIN_ID", RootCmd.PersistentFlags().Lookup("chain-id"))
	viper.BindPFlag("LOG_LEVEL", RootCmd.PersistentFlags().Lookup("log-level"))

	viper.AutomaticEnv()

	RootCmd.AddCommand(writeCmd)
	RootCmd.AddCommand(readCmd)
	RootCmd.AddCommand(queryCmd)
	RootCmd.AddCommand(replayCmd)
	RootCmd.AddCommand(agentCmd)
	RootCmd.AddCommand(importCmd)
	RootCmd.AddCommand(infoCmd)
	RootCmd.AddCommand(verifyCmd)
	RootCmd.AddCommand(startCmd)
	RootCmd.AddCommand(chatCmd)
	RootCmd.AddCommand(taskCmd)
}

func initConfig() {
	if cfg.DataDir == "" {
		cfg.DataDir = viper.GetString("DATA_DIR")
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "./data"
	}

	if cfg.PrivateKey == "" {
		cfg.PrivateKey = viper.GetString("KEY")
	}
	if cfg.ContractAddr == "" {
		cfg.ContractAddr = viper.GetString("CONTRACT")
	}

	// 初始化 SDK
	var err error
	hub, err = sdk.New(&sdk.Config{
		DataDir:       cfg.DataDir,
		StorageRPC:   cfg.StorageRPC,
		IndexerRPC:   cfg.IndexerRPC,
		ChainRPC:     cfg.ChainRPC,
		ContractAddr: cfg.ContractAddr,
		PrivateKey:   cfg.PrivateKey,
		ChainID:      cfg.ChainID,
		LogLevel:     cfg.LogLevel,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize MemoryHub: %v\n", err)
		os.Exit(1)
	}
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func getHub() *sdk.MemoryHub {
	return hub
}
