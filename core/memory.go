package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Memory 是记忆的基本单元
type Memory struct {
	ID        string    `json:"id"`        // 唯一ID
	PrevID    string    `json:"prev_id"`  // 前一个记忆ID（链式指针）
	CID       string    `json:"cid"`       // 0G Storage 的 CID
	TxHash    string    `json:"tx_hash"`   // 链上交易哈希
	AgentID   string    `json:"agent_id"` // Agent 标识
	Content   []byte    `json:"content"`  // 记忆内容
	Hash      string    `json:"hash"`     // 内容哈希（完整性验证）
	Timestamp int64     `json:"timestamp"` // 时间戳
	Tags      []string  `json:"tags"`     // 标签
	Size      int       `json:"size"`     // 内容大小
	Status    string    `json:"status"`   // pending, uploaded, anchored, failed
}

// MemoryEntry 用于 JSON 序列化（不包含 Content）
type MemoryEntry struct {
	ID        string   `json:"id"`
	PrevID    string   `json:"prev_id"`
	CID       string   `json:"cid"`
	TxHash    string   `json:"tx_hash"`
	AgentID   string   `json:"agent_id"`
	Hash      string   `json:"hash"`
	Timestamp int64    `json:"timestamp"`
	Tags      []string `json:"tags"`
	Size      int      `json:"size"`
	Status    string   `json:"status"`
}

// NewMemory 创建一个新的记忆
func NewMemory(agentID string, content []byte, tags []string) *Memory {
	// 计算内容哈希
	hash := sha256.Sum256(content)

	return &Memory{
		ID:        uuid.New().String(),
		AgentID:   agentID,
		Content:   content,
		Hash:      hex.EncodeToString(hash[:]),
		Timestamp: time.Now().Unix(),
		Tags:      tags,
		Size:      len(content),
		Status:    "pending",
	}
}

// NewMemoryFromEntry 从 MemoryEntry 恢复 Memory（不含 Content）
func NewMemoryFromEntry(entry *MemoryEntry) *Memory {
	return &Memory{
		ID:        entry.ID,
		PrevID:    entry.PrevID,
		CID:       entry.CID,
		TxHash:    entry.TxHash,
		AgentID:   entry.AgentID,
		Hash:      entry.Hash,
		Timestamp: entry.Timestamp,
		Tags:      entry.Tags,
		Size:      entry.Size,
		Status:    entry.Status,
	}
}

// ToEntry 转换为 MemoryEntry（不含 Content，用于序列化）
func (m *Memory) ToEntry() *MemoryEntry {
	return &MemoryEntry{
		ID:        m.ID,
		PrevID:    m.PrevID,
		CID:       m.CID,
		TxHash:    m.TxHash,
		AgentID:   m.AgentID,
		Hash:      m.Hash,
		Timestamp: m.Timestamp,
		Tags:      m.Tags,
		Size:      m.Size,
		Status:    m.Status,
	}
}

// VerifyIntegrity 验证记忆完整性
func (m *Memory) VerifyIntegrity() bool {
	if len(m.Content) == 0 {
		return m.CID != "" // 没有内容时只检查 CID
	}
	hash := sha256.Sum256(m.Content)
	return m.Hash == hex.EncodeToString(hash[:])
}

// SetCID 设置 0G Storage CID
func (m *Memory) SetCID(cid string) {
	m.CID = cid
}

// SetTxHash 设置链上交易哈希
func (m *Memory) SetTxHash(txHash string) {
	m.TxHash = txHash
}

// SetStatus 设置状态
func (m *Memory) SetStatus(status string) {
	m.Status = status
}

// SetPrevID 设置前一个记忆ID
func (m *Memory) SetPrevID(prevID string) {
	m.PrevID = prevID
}

// MemoryJSON 用于 JSON 序列化
type MemoryJSON struct {
	ID        string   `json:"id"`
	PrevID    string   `json:"prev_id"`
	CID       string   `json:"cid"`
	TxHash    string   `json:"tx_hash"`
	AgentID   string   `json:"agent_id"`
	Content   string   `json:"content,omitempty"` // Base64 编码
	Hash      string   `json:"hash"`
	Timestamp int64    `json:"timestamp"`
	Tags      []string `json:"tags"`
	Size      int      `json:"size"`
	Status    string   `json:"status"`
}

// ToJSON 转换为 JSON 格式
func (m *Memory) ToJSON() ([]byte, error) {
	jsonData := MemoryJSON{
		ID:        m.ID,
		PrevID:    m.PrevID,
		CID:       m.CID,
		TxHash:    m.TxHash,
		AgentID:   m.AgentID,
		Hash:      m.Hash,
		Timestamp: m.Timestamp,
		Tags:      m.Tags,
		Size:      m.Size,
		Status:    m.Status,
	}
	if len(m.Content) > 0 {
		jsonData.Content = hex.EncodeToString(m.Content)
	}
	return json.Marshal(jsonData)
}

// String 返回记忆的字符串表示
func (m *Memory) String() string {
	return fmt.Sprintf("Memory[id=%s, agent=%s, timestamp=%d, tags=%v, size=%d, status=%s]",
		m.ID[:8], m.AgentID, m.Timestamp, m.Tags, m.Size, m.Status)
}
