package sdk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dongowu/0g-memory-hub/chain"
	"github.com/dongowu/0g-memory-hub/core"
	"github.com/dongowu/0g-memory-hub/storage"
	"github.com/sirupsen/logrus"
)

// MemoryHub SDK 核心接口
type MemoryHub struct {
	index   *core.MemoryIndex
	storage *storage.Client
	chain   *chain.Client
	config  *Config
	logger  *logrus.Logger
}

// Config SDK 配置
type Config struct {
	DataDir        string // 存储目录
	StorageRPC     string
	IndexerRPC     string
	ChainRPC       string
	ContractAddr   string
	PrivateKey     string
	ChainID        uint64
	BufferSize     int
	BufferBatch    int
	BufferWorkers  int
	LogLevel       string
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		DataDir:       "./data",
		StorageRPC:   "https://testnet-rpc.0g.ai",
		IndexerRPC:   "https://testnet-indexer.0g.ai",
		ChainRPC:     "https://testnet-chain-rpc.0g.ai",
		ContractAddr: "0x0000000000000000000000000000000000000000",
		ChainID:      1,
		BufferSize:    1000,
		BufferBatch:   10,
		BufferWorkers: 4,
		LogLevel:     "info",
	}
}

// New 创建新的 MemoryHub 实例
func New(config *Config) (*MemoryHub, error) {
	if config == nil {
		config = DefaultConfig()
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	switch config.LogLevel {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	os.MkdirAll(config.DataDir, 0755)

	storageClient := storage.NewClient(&storage.Config{
		RPCURL:     config.StorageRPC,
		IndexerURL: config.IndexerRPC,
	})

	chainClient, err := chain.NewClient(config.ChainRPC, config.ContractAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain client: %w", err)
	}
	if config.PrivateKey != "" {
		chainClient.SetConfig(&chain.Config{
			PrivateKey: config.PrivateKey,
			ChainID:    config.ChainID,
		})
	}

	index := core.NewMemoryIndex(filepath.Join(config.DataDir, "index"))

	return &MemoryHub{
		index:   index,
		storage: storageClient,
		chain:   chainClient,
		config:  config,
		logger:  logger,
	}, nil
}

// Close 关闭
func (m *MemoryHub) Close() error {
	m.chain.Close()
	return nil
}

// Write 写入记忆
func (m *MemoryHub) Write(ctx context.Context, agentID string, content []byte, tags []string) (*core.Memory, error) {
	m.logger.WithField("agent", agentID).Info("Writing memory")

	mem := core.NewMemory(agentID, content, tags)

	result, err := m.storage.Upload(ctx, content)
	if err != nil {
		mem.SetStatus("failed")
		return mem, fmt.Errorf("failed to upload: %w", err)
	}

	mem.SetCID(result.CID)
	mem.SetTxHash(result.TxHash)
	mem.SetStatus("uploaded")

	if m.config.PrivateKey != "" {
		mem.SetStatus("anchored")
	}

	if err := m.index.Append(agentID, mem); err != nil {
		return mem, fmt.Errorf("failed to append to chain: %w", err)
	}

	m.logger.WithField("memory", mem.String()).Info("Memory written")
	return mem, nil
}

// WriteString 写入字符串记忆
func (m *MemoryHub) WriteString(ctx context.Context, agentID string, content string, tags []string) (*core.Memory, error) {
	return m.Write(ctx, agentID, []byte(content), tags)
}

// Read 读取记忆
func (m *MemoryHub) Read(ctx context.Context, agentID string, memoryID string) (*core.Memory, error) {
	chain := m.index.GetChain(agentID)
	mem, ok := chain.Get(memoryID)
	if !ok {
		return nil, fmt.Errorf("memory not found: %s", memoryID)
	}

	if len(mem.Content) == 0 && mem.CID != "" {
		data, err := m.storage.Download(ctx, mem.CID)
		if err != nil {
			return mem, fmt.Errorf("failed to download: %w", err)
		}
		mem.Content = data
	}

	return mem, nil
}

// QueryOptions 查询选项
type QueryOptions struct {
	LastN int64 // 最近 N 条
	Since int64 // 起始时间
	Until int64 // 结束时间
	Tag   string
	Tags  []string
}

// Query 查询记忆
func (m *MemoryHub) Query(agentID string, opts *QueryOptions) ([]*core.Memory, error) {
	chain := m.index.GetChain(agentID)

	var memories []*core.Memory

	if opts != nil {
		if opts.LastN > 0 {
			memories = chain.GetLastN(int(opts.LastN))
		} else if opts.Since > 0 || opts.Until > 0 {
			since := opts.Since
			until := opts.Until
			if since == 0 {
				since = 0
			}
			if until == 0 {
				until = time.Now().Unix()
			}
			memories = chain.GetRange(since, until)
		} else if opts.Tag != "" {
			memories = chain.GetByTag(opts.Tag)
		} else {
			memories = chain.GetAll()
		}
	} else {
		memories = chain.GetAll()
	}

	return memories, nil
}

// Replay 恢复记忆
func (m *MemoryHub) Replay(agentID string, opts *QueryOptions) ([]string, error) {
	memories, err := m.Query(agentID, opts)
	if err != nil {
		return nil, err
	}

	lines := make([]string, 0, len(memories))
	for _, mem := range memories {
		timestamp := time.Unix(mem.Timestamp, 0).Format(time.RFC3339)
		tags := ""
		if len(mem.Tags) > 0 {
			tags = fmt.Sprintf(" [%s]", joinStrings(mem.Tags, ", "))
		}
		line := fmt.Sprintf("[%s]%s %s", timestamp, tags, string(mem.Content))
		lines = append(lines, line)
	}

	return lines, nil
}

// GetChain 获取链
func (m *MemoryHub) GetChain(agentID string) *core.MemoryChain {
	return m.index.GetChain(agentID)
}

// ListAgents 列出所有 Agent
func (m *MemoryHub) ListAgents() []string {
	return m.index.ListAgents()
}

// Export 导出
func (m *MemoryHub) Export(agentID string) ([]byte, error) {
	chain := m.index.GetChain(agentID)
	return chain.Export()
}

// Import 导入
func (m *MemoryHub) Import(agentID string, data []byte) error {
	return m.index.ImportAll(data)
}

// GetInfo 获取系统信息
func (m *MemoryHub) GetInfo() *Info {
	return &Info{
		IndexInfo: m.index.GetInfo(),
		Stats:     m.index.GetStats(),
	}
}

// Info 系统信息
type Info struct {
	IndexInfo *core.IndexInfo `json:"index_info"`
	Stats     *core.Stats    `json:"stats"`
}

// Verify 验证链完整性
func (m *MemoryHub) Verify(agentID string) (bool, error) {
	chain := m.index.GetChain(agentID)
	return chain.VerifyChain()
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
