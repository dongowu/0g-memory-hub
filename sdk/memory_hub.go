package sdk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dongowu/0g-memory-hub/chain"
	"github.com/dongowu/0g-memory-hub/core"
	"github.com/dongowu/0g-memory-hub/storage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
)

// MemoryHub SDK 核心接口
type MemoryHub struct {
	sessions *core.SessionManager
	storage  *storage.Client
	chain    *chain.Client
	config   *Config
	logger   *logrus.Logger
}

// Config SDK 配置
type Config struct {
	DataDir       string
	StorageRPC   string
	IndexerRPC   string
	ChainRPC     string
	ContractAddr string
	PrivateKey  string
	ChainID      uint64
	LogLevel     string
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

	return &MemoryHub{
		sessions: core.NewSessionManager(filepath.Join(config.DataDir, "sessions")),
		storage:  storageClient,
		chain:    chainClient,
		config:   config,
		logger:   logger,
	}, nil
}

// Close 关闭
func (m *MemoryHub) Close() error {
	m.chain.Close()
	return nil
}

// ==================== Session Management ====================

// GetOrCreateSession 获取或创建会话
func (m *MemoryHub) GetOrCreateSession(wallet string) *core.Session {
	return m.sessions.GetOrCreate(wallet)
}

// GetSession 获取会话
func (m *MemoryHub) GetSession(wallet string) *core.Session {
	return m.sessions.Get(wallet)
}

// SaveSession 保存会话
func (m *MemoryHub) SaveSession(session *core.Session) error {
	return m.sessions.Save(session)
}

// ==================== Message Operations ====================

// AddMessage 添加对话消息
func (m *MemoryHub) AddMessage(session *core.Session, role, content string) *core.Message {
	return session.AddMessage(role, content)
}

// GetContext 获取对话上下文
func (m *MemoryHub) GetContext(session *core.Session) []*core.Message {
	return session.GetContext()
}

// ==================== Memory Operations ====================

// SaveMemory 保存记忆到 0G
func (m *MemoryHub) SaveMemory(ctx context.Context, session *core.Session) (*core.Memory, error) {
	m.logger.WithField("wallet", session.Wallet).Info("Saving memory")

	// 序列化会话为 JSON
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize session: %w", err)
	}

	// 上传到 0G Storage
	result, err := m.storage.Upload(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to upload: %w", err)
	}

	// 创建记忆
	mem := &core.Memory{
		ID:        fmt.Sprintf("mem_%d", time.Now().UnixNano()),
		CID:       result.CID,
		TxHash:    result.TxHash,
		AgentID:   session.Wallet,
		Timestamp: time.Now().Unix(),
		Status:    "uploaded",
	}

	// 设置内容哈希
	hash := sha256.Sum256(data)
	mem.Content = data
	mem.Hash = hex.EncodeToString(hash[:])

	// 添加到会话
	session.AddMemory(data, []string{"session"})

	// 锚定到链上
	if m.config.PrivateKey != "" {
		session.SetRootHash(result.CID)
	}

	// 保存会话
	if err := m.SaveSession(session); err != nil {
		m.logger.WithError(err).Warn("Failed to save session locally")
	}

	m.logger.WithFields(logrus.Fields{
		"cid": result.CID,
	}).Info("Memory saved successfully")

	return mem, nil
}

// LoadMemory 从 0G 加载记忆
func (m *MemoryHub) LoadMemory(ctx context.Context, cid string) (*core.Session, error) {
	m.logger.WithField("cid", cid).Info("Loading memory")

	data, err := m.storage.Download(ctx, cid)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}

	var session core.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to deserialize session: %w", err)
	}

	m.logger.Info("Memory loaded successfully")
	return &session, nil
}

// ==================== Task Operations ====================

// AddTask 添加任务
func (m *MemoryHub) AddTask(session *core.Session, description, frequency string) *core.Task {
	return session.AddTask(description, frequency)
}

// ListTasks 列出任务
func (m *MemoryHub) ListTasks(session *core.Session) []*core.Task {
	return session.Tasks
}

// GetPendingTasks 获取待处理任务
func (m *MemoryHub) GetPendingTasks(session *core.Session) []*core.Task {
	return session.GetPendingTasks()
}

// UpdateTask 更新任务
func (m *MemoryHub) UpdateTask(session *core.Session, taskID, content string) (*core.TaskUpdate, error) {
	return session.UpdateTask(taskID, content)
}

// CompleteTask 完成任务
func (m *MemoryHub) CompleteTask(session *core.Session, taskID string) error {
	return session.SetTaskStatus(taskID, "done")
}

// ==================== Chain Operations ====================

// AnchorToChain 锚定到链上
func (m *MemoryHub) AnchorToChain(ctx context.Context, session *core.Session) (*chain.TxResult, error) {
	if m.config.PrivateKey == "" {
		return nil, fmt.Errorf("private key not configured")
	}

	m.logger.WithField("wallet", session.Wallet).Info("Anchoring to chain")

	// 转换钱包地址
	walletAddr := common.HexToAddress(session.Wallet)

	// 从链上获取最新的 memory CID
	result, err := m.chain.SetMemoryHead(ctx, walletAddr, session.RootHash, m.config.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to anchor: %w", err)
	}

	m.logger.WithField("txHash", result.TxHash).Info("Anchored successfully")
	return result, nil
}

// GetChainHead 从链上获取最新 CID
func (m *MemoryHub) GetChainHead(ctx context.Context, wallet string) (string, error) {
	walletAddr := common.HexToAddress(wallet)
	return m.chain.GetMemoryHead(ctx, walletAddr)
}

// ==================== Export/Import ====================

// ExportSession 导出会话
func (m *MemoryHub) ExportSession(session *core.Session) ([]byte, error) {
	return session.Export()
}

// ImportSession 导入会话
func (m *MemoryHub) ImportSession(wallet string, data []byte) (*core.Session, error) {
	return m.sessions.Import(wallet, data)
}

// ==================== Verification ====================

// VerifyMemory 验证记忆完整性
func (m *MemoryHub) VerifyMemory(ctx context.Context, session *core.Session) (bool, error) {
	m.logger.WithField("wallet", session.Wallet).Info("Verifying memory integrity")

	// 验证上下文中的消息
	for _, msg := range session.Context {
		hash := sha256.Sum256([]byte(msg.Content))
		expected := hex.EncodeToString(hash[:])
		if msg.Hash != "" && msg.Hash != expected {
			return false, fmt.Errorf("message %s has been tampered", msg.ID)
		}
	}

	// 验证记忆链
	for i, mem := range session.Memories {
		if !mem.VerifyIntegrity() {
			return false, fmt.Errorf("memory %d integrity check failed", i)
		}
	}

	m.logger.Info("Memory integrity verified")
	return true, nil
}

// ==================== Stats ====================

// GetSessionStats 获取会话统计
type SessionStats struct {
	Wallet        string `json:"wallet"`
	MemoryCount   int    `json:"memory_count"`
	TaskCount     int    `json:"task_count"`
	PendingTasks  int    `json:"pending_tasks"`
	ContextSize   int    `json:"context_size"`
	LastUpdated   int64  `json:"last_updated"`
	RootHash      string `json:"root_hash"`
}

// GetStats 获取统计
func (m *MemoryHub) GetStats(session *core.Session) *SessionStats {
	return &SessionStats{
		Wallet:       session.Wallet,
		MemoryCount:  len(session.Memories),
		TaskCount:    len(session.Tasks),
		PendingTasks: len(session.GetPendingTasks()),
		ContextSize:  len(session.Context),
		LastUpdated:  session.UpdatedAt,
		RootHash:     session.RootHash,
	}
}

// ==================== Legacy Agent API ====================
// Write 写入记忆 (legacy, 使用 agent ID 作为 wallet)
func (m *MemoryHub) Write(ctx context.Context, agentID string, content []byte, tags []string) (*core.Memory, error) {
	session := m.GetOrCreateSession(agentID)
	mem := session.AddMemory(content, tags)

	// 上传到 0G Storage
	result, err := m.storage.Upload(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to upload: %w", err)
	}

	mem.CID = result.CID
	mem.TxHash = result.TxHash

	return mem, nil
}

// Read 读取记忆 (legacy)
func (m *MemoryHub) Read(ctx context.Context, agentID string, memoryID string) (*core.Memory, error) {
	session := m.GetSession(agentID)
	if session == nil {
		return nil, fmt.Errorf("session not found for agent: %s", agentID)
	}

	for _, mem := range session.Memories {
		if mem.ID == memoryID {
			return mem, nil
		}
	}

	return nil, fmt.Errorf("memory not found: %s", memoryID)
}

// QueryOptions 查询选项
type QueryOptions struct {
	LastN int64
	Since int64
	Until int64
	Tag   string
}

// Query 查询记忆 (legacy)
func (m *MemoryHub) Query(agentID string, opts *QueryOptions) ([]*core.Memory, error) {
	session := m.GetSession(agentID)
	if session == nil {
		return []*core.Memory{}, nil
	}

	memories := make([]*core.Memory, 0)
	for _, mem := range session.Memories {
		// 过滤标签
		if opts.Tag != "" {
			hasTag := false
			for _, t := range mem.Tags {
				if t == opts.Tag {
					hasTag = true
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		// 过滤时间范围
		if opts.Since > 0 && mem.Timestamp < opts.Since {
			continue
		}
		if opts.Until > 0 && mem.Timestamp > opts.Until {
			continue
		}

		memories = append(memories, mem)
	}

	// 返回最后 N 条
	if opts.LastN > 0 && int64(len(memories)) > opts.LastN {
		start := len(memories) - int(opts.LastN)
		memories = memories[start:]
	}

	return memories, nil
}

// Import 导入记忆 (legacy)
func (m *MemoryHub) Import(agentID string, data []byte) error {
	_, err := m.ImportSession(agentID, data)
	return err
}

// SystemInfo 系统信息
type SystemInfo struct {
	IndexInfo IndexInfo `json:"index_info"`
}

// IndexInfo 索引信息
type IndexInfo struct {
	Dir         string           `json:"dir"`
	Agents      int              `json:"agents"`
	TotalMem    int              `json:"total_memories"`
	Chains      map[string]Chain  `json:"chains"`
}

// Chain 链信息
type Chain struct {
	HeadID string `json:"head_id"`
	TailID string `json:"tail_id"`
	Length int    `json:"length"`
}

// GetInfo 获取系统信息
func (m *MemoryHub) GetInfo() *SystemInfo {
	// 返回基本信息
	return &SystemInfo{
		IndexInfo: IndexInfo{
			Dir:      m.config.DataDir,
			Agents:   1,
			TotalMem: 0,
			Chains:   make(map[string]Chain),
		},
	}
}

// Verify 验证记忆完整性 (legacy)
func (m *MemoryHub) Verify(agentID string) (bool, error) {
	session := m.GetSession(agentID)
	if session == nil {
		return false, fmt.Errorf("session not found for agent: %s", agentID)
	}

	for i, mem := range session.Memories {
		if !mem.VerifyIntegrity() {
			return false, fmt.Errorf("memory %d integrity check failed", i)
		}
	}

	return true, nil
}

// Replay 回放记忆
func (m *MemoryHub) Replay(agentID string, opts *QueryOptions) ([]string, error) {
	memories, err := m.Query(agentID, opts)
	if err != nil {
		return nil, err
	}

	lines := make([]string, 0)
	for _, mem := range memories {
		if len(mem.Content) > 0 {
			lines = append(lines, string(mem.Content))
		}
	}

	return lines, nil
}
