package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// MemoryChain 维护链式结构
type MemoryChain struct {
	mu       sync.RWMutex
	AgentID  string    `json:"agent_id"`
	HeadID   string    `json:"head_id"`   // 最新记忆ID
	TailID   string    `json:"tail_id"`   // 最早的记忆ID
	Length   int       `json:"length"`    // 记忆数量
	memories map[string]*Memory `json:"-"` // 内存存储
}

// NewMemoryChain 创建一个新的记忆链
func NewMemoryChain(agentID string) *MemoryChain {
	return &MemoryChain{
		AgentID:  agentID,
		memories: make(map[string]*Memory),
	}
}

// Append 添加记忆到链尾
func (c *MemoryChain) Append(m *Memory) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 设置前一个指针
	if c.HeadID != "" {
		if prev, ok := c.memories[c.HeadID]; ok {
			m.SetPrevID(prev.ID)
		}
	}

	// 添加到链
	c.memories[m.ID] = m
	c.HeadID = m.ID

	if c.TailID == "" {
		c.TailID = m.ID
	}
	c.Length++

	return nil
}

// Get 获取指定记忆
func (c *MemoryChain) Get(id string) (*Memory, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	m, ok := c.memories[id]
	return m, ok
}

// GetHead 获取最新记忆
func (c *MemoryChain) GetHead() (*Memory, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.HeadID == "" {
		return nil, false
	}
	return c.memories[c.HeadID], true
}

// GetTail 获取最早记忆
func (c *MemoryChain) GetTail() (*Memory, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.TailID == "" {
		return nil, false
	}
	return c.memories[c.TailID], true
}

// GetAll 按时间顺序获取所有记忆（从新到旧）
func (c *MemoryChain) GetAll() []*Memory {
	c.mu.RLock()
	defer c.mu.RUnlock()

	memories := make([]*Memory, 0, len(c.memories))
	for _, m := range c.memories {
		memories = append(memories, m)
	}

	// 按时间排序（从新到旧）
	sort.Slice(memories, func(i, j int) bool {
		return memories[i].Timestamp > memories[j].Timestamp
	})

	return memories
}

// GetRange 获取指定范围的记忆
func (c *MemoryChain) GetRange(start, end int64) []*Memory {
	c.mu.RLock()
	defer c.mu.RUnlock()

	memories := make([]*Memory, 0)
	for _, m := range c.memories {
		if m.Timestamp >= start && m.Timestamp <= end {
			memories = append(memories, m)
		}
	}

	sort.Slice(memories, func(i, j int) bool {
		return memories[i].Timestamp > memories[j].Timestamp
	})

	return memories
}

// GetByTag 获取指定标签的记忆
func (c *MemoryChain) GetByTag(tag string) []*Memory {
	c.mu.RLock()
	defer c.mu.RUnlock()

	memories := make([]*Memory, 0)
	for _, m := range c.memories {
		for _, t := range m.Tags {
			if t == tag {
				memories = append(memories, m)
				break
			}
		}
	}

	sort.Slice(memories, func(i, j int) bool {
		return memories[i].Timestamp > memories[j].Timestamp
	})

	return memories
}

// GetLastN 获取最近 N 条记忆
func (c *MemoryChain) GetLastN(n int) []*Memory {
	c.mu.RLock()
	defer c.mu.RUnlock()

	memories := make([]*Memory, 0, n)
	for _, m := range c.memories {
		memories = append(memories, m)
	}

	sort.Slice(memories, func(i, j int) bool {
		return memories[i].Timestamp > memories[j].Timestamp
	})

	if len(memories) > n {
		return memories[:n]
	}
	return memories
}

// Size 返回记忆数量
func (c *MemoryChain) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.memories)
}

// VerifyChain 验证链的完整性
func (c *MemoryChain) VerifyChain() (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Length == 0 {
		return true, nil
	}

	// 从头开始遍历
	currentID := c.TailID
	for currentID != "" {
		m, ok := c.memories[currentID]
		if !ok {
			return false, fmt.Errorf("missing memory: %s", currentID)
		}

		// 验证内容完整性
		if !m.VerifyIntegrity() {
			return false, fmt.Errorf("integrity check failed for: %s", currentID)
		}

		currentID = m.PrevID
	}

	return true, nil
}

// ChainInfo 链的基本信息
type ChainInfo struct {
	AgentID  string `json:"agent_id"`
	HeadID   string `json:"head_id"`
	TailID   string `json:"tail_id"`
	Length   int    `json:"length"`
	HeadTime int64  `json:"head_time"`
	TailTime int64  `json:"tail_time"`
}

// GetInfo 获取链信息
func (c *MemoryChain) GetInfo() *ChainInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	info := &ChainInfo{
		AgentID: c.AgentID,
		HeadID:  c.HeadID,
		TailID:  c.TailID,
		Length:  c.Length,
	}

	if head, ok := c.memories[c.HeadID]; ok {
		info.HeadTime = head.Timestamp
	}
	if tail, ok := c.memories[c.TailID]; ok {
		info.TailTime = tail.Timestamp
	}

	return info
}

// Export 导出链数据
func (c *MemoryChain) Export() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data := struct {
		Chain   *ChainInfo    `json:"chain"`
		Memories []*MemoryEntry `json:"memories"`
	}{
		Chain: c.GetInfo(),
	}

	for _, m := range c.memories {
		data.Memories = append(data.Memories, m.ToEntry())
	}

	return json.MarshalIndent(data, "", "  ")
}

// ImportChain 从数据导入链
func ImportChain(data []byte) (*MemoryChain, error) {
	var importData struct {
		Chain   *ChainInfo    `json:"chain"`
		Memories []*MemoryEntry `json:"memories"`
	}

	if err := json.Unmarshal(data, &importData); err != nil {
		return nil, err
	}

	chain := &MemoryChain{
		AgentID:  importData.Chain.AgentID,
		HeadID:   importData.Chain.HeadID,
		TailID:   importData.Chain.TailID,
		Length:   importData.Chain.Length,
		memories: make(map[string]*Memory),
	}

	for _, entry := range importData.Memories {
		chain.memories[entry.ID] = NewMemoryFromEntry(entry)
	}

	return chain, nil
}

// ComputeChainHash 计算链的哈希（用于验证）
func (c *MemoryChain) ComputeChainHash() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hasher := sha256.New()
	for _, m := range c.GetAll() {
		hasher.Write([]byte(m.Hash))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}
