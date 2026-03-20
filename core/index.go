package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// MemoryIndex 索引系统
type MemoryIndex struct {
	mu     sync.RWMutex
	Dir    string                   // 存储目录
	Chains map[string]*MemoryChain  // AgentID -> MemoryChain
}

// NewMemoryIndex 创建一个新的索引
func NewMemoryIndex(dir string) *MemoryIndex {
	// 确保目录存在
	os.MkdirAll(dir, 0755)

	return &MemoryIndex{
		Dir:    dir,
		Chains: make(map[string]*MemoryChain),
	}
}

// GetOrCreateChain 获取或创建指定 Agent 的链
func (idx *MemoryIndex) GetOrCreateChain(agentID string) *MemoryChain {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if chain, ok := idx.Chains[agentID]; ok {
		return chain
	}

	chain := NewMemoryChain(agentID)
	idx.Chains[agentID] = chain

	// 尝试从磁盘加载
	idx.loadChain(agentID, chain)

	return chain
}

// loadChain 从磁盘加载链
func (idx *MemoryIndex) loadChain(agentID string, chain *MemoryChain) {
	chainFile := filepath.Join(idx.Dir, agentID+".json")
	data, err := os.ReadFile(chainFile)
	if err != nil {
		return // 文件不存在，正常
	}

	loadedChain, err := ImportChain(data)
	if err != nil {
		return
	}

	idx.Chains[agentID] = loadedChain
}

// SaveChain 保存链到磁盘
func (idx *MemoryIndex) SaveChain(agentID string) error {
	idx.mu.RLock()
	chain, ok := idx.Chains[agentID]
	idx.mu.RUnlock()

	if !ok {
		return nil
	}

	data, err := chain.Export()
	if err != nil {
		return err
	}

	chainFile := filepath.Join(idx.Dir, agentID+".json")
	return os.WriteFile(chainFile, data, 0644)
}

// Append 添加记忆到指定 Agent 的链
func (idx *MemoryIndex) Append(agentID string, m *Memory) error {
	chain := idx.GetOrCreateChain(agentID)
	if err := chain.Append(m); err != nil {
		return err
	}
	return idx.SaveChain(agentID)
}

// GetChain 获取指定 Agent 的链
func (idx *MemoryIndex) GetChain(agentID string) *MemoryChain {
	return idx.GetOrCreateChain(agentID)
}

// ListAgents 列出所有 Agent
func (idx *MemoryIndex) ListAgents() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	agents := make([]string, 0, len(idx.Chains))
	for agentID := range idx.Chains {
		agents = append(agents, agentID)
	}
	return agents
}

// IndexInfo 索引信息
type IndexInfo struct {
	Dir      string            `json:"dir"`
	Agents   int               `json:"agents"`
	Chains   map[string]*ChainInfo `json:"chains"`
	TotalMem int               `json:"total_memories"`
}

// GetInfo 获取索引信息
func (idx *MemoryIndex) GetInfo() *IndexInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	info := &IndexInfo{
		Dir:    idx.Dir,
		Agents: len(idx.Chains),
		Chains: make(map[string]*ChainInfo),
	}

	for agentID, chain := range idx.Chains {
		info.Chains[agentID] = chain.GetInfo()
		info.TotalMem += chain.Size()
	}

	return info
}

// ExportAll 导出所有数据
func (idx *MemoryIndex) ExportAll() ([]byte, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	data := make(map[string]*MemoryChain)
	for agentID, chain := range idx.Chains {
		data[agentID] = chain
	}

	return json.MarshalIndent(data, "", "  ")
}

// ImportAll 从数据导入
func (idx *MemoryIndex) ImportAll(data []byte) error {
	var chains map[string]*MemoryChain
	if err := json.Unmarshal(data, &chains); err != nil {
		return err
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	for agentID, chain := range chains {
		idx.Chains[agentID] = chain
		if err := idx.SaveChain(agentID); err != nil {
			return err
		}
	}

	return nil
}

// Stats 统计数据
type Stats struct {
	TotalAgents  int            `json:"total_agents"`
	TotalMemories int           `json:"total_memories"`
	Chains       map[string]int `json:"memories_per_agent"`
}

// GetStats 获取统计数据
func (idx *MemoryIndex) GetStats() *Stats {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	stats := &Stats{
		Chains: make(map[string]int),
	}

	for agentID, chain := range idx.Chains {
		stats.TotalAgents++
		stats.TotalMemories += chain.Size()
		stats.Chains[agentID] = chain.Size()
	}

	return stats
}

// Clean 删除过期的链数据（保留最近 N 条）
func (idx *MemoryIndex) Clean(agentID string, keepLast int) error {
	chain := idx.GetOrCreateChain(agentID)

	idx.mu.Lock()
	defer idx.mu.Unlock()

	// 保留最近的
	recent := chain.GetLastN(keepLast)

	// 重新构建链
	newChain := NewMemoryChain(agentID)
	for _, m := range recent {
		newChain.Append(m)
	}

	idx.Chains[agentID] = newChain
	return idx.SaveChain(agentID)
}
