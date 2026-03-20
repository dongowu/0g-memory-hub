package core

import (
	"context"
	"log"
	"sync"
	"time"
)

// BufferConfig 缓冲区配置
type BufferConfig struct {
	MaxSize    int           // 最大队列长度
	BatchSize  int           // 批量大小
	FlushInterval time.Duration // 刷新间隔
	Workers    int           // 并发工作数
}

// DefaultBufferConfig 默认配置
func DefaultBufferConfig() *BufferConfig {
	return &BufferConfig{
		MaxSize:      1000,
		BatchSize:    10,
		FlushInterval: 5 * time.Second,
		Workers:      4,
	}
}

// Buffer 并发写入缓冲区
type Buffer struct {
	mu       sync.Mutex
	config   *BufferConfig
	pending  []*Memory        // 待处理的记忆
	closed   bool
	wg       sync.WaitGroup

	// 回调函数
	uploader func([]*Memory) error // 上传函数
}

// NewBuffer 创建一个新的缓冲区
func NewBuffer(config *BufferConfig) *Buffer {
	if config == nil {
		config = DefaultBufferConfig()
	}
	return &Buffer{
		config:  config,
		pending: make([]*Memory, 0, config.MaxSize),
	}
}

// SetUploader 设置上传回调
func (b *Buffer) SetUploader(uploader func([]*Memory) error) {
	b.uploader = uploader
}

// Push 添加记忆到缓冲区
func (b *Buffer) Push(m *Memory) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return ErrBufferClosed
	}

	// 达到批量大小，立即处理
	if len(b.pending) >= b.config.BatchSize {
		if err := b.flushUnsafe(); err != nil {
			log.Printf("Buffer flush error: %v", err)
		}
	}

	b.pending = append(b.pending, m)
	return nil
}

// PushBatch 批量添加
func (b *Buffer) PushBatch(memories []*Memory) error {
	for _, m := range memories {
		if err := b.Push(m); err != nil {
			return err
		}
	}
	return nil
}

// flushUnsafe 刷新缓冲区（需要持有锁）
func (b *Buffer) flushUnsafe() error {
	if len(b.pending) == 0 {
		return nil
	}

	// 复制待处理的记忆
	toProcess := make([]*Memory, len(b.pending))
	copy(toProcess, b.pending)

	// 清空缓冲区
	b.pending = b.pending[:0]

	// 使用上传回调处理
	if b.uploader != nil {
		return b.uploader(toProcess)
	}

	return nil
}

// Flush 手动刷新
func (b *Buffer) Flush() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flushUnsafe()
}

// Start 启动后台刷新
func (b *Buffer) Start(ctx context.Context) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		ticker := time.NewTicker(b.config.FlushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				b.Flush()
				return
			case <-ticker.C:
				b.mu.Lock()
				if len(b.pending) > 0 {
					if err := b.flushUnsafe(); err != nil {
						log.Printf("Buffer flush error: %v", err)
					}
				}
				b.mu.Unlock()
			}
		}
	}()
}

// Close 关闭缓冲区
func (b *Buffer) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	b.wg.Wait()
	return b.flushUnsafe()
}

// PendingCount 返回待处理数量
func (b *Buffer) PendingCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pending)
}

// BufferPool 缓冲区池（用于高并发场景）
type BufferPool struct {
	mu      sync.RWMutex
	buffers map[string]*Buffer // AgentID -> Buffer
	config  *BufferConfig
	index   *MemoryIndex
}

// NewBufferPool 创建缓冲区池
func NewBufferPool(config *BufferConfig, index *MemoryIndex) *BufferPool {
	if config == nil {
		config = DefaultBufferConfig()
	}
	return &BufferPool{
		buffers: make(map[string]*Buffer),
		config:  config,
		index:   index,
	}
}

// Get 获取或创建指定 Agent 的缓冲区
func (p *BufferPool) Get(agentID string) *Buffer {
	p.mu.RLock()
	if buf, ok := p.buffers[agentID]; ok {
		p.mu.RUnlock()
		return buf
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// 双重检查
	if buf, ok := p.buffers[agentID]; ok {
		return buf
	}

	buf := NewBuffer(p.config)
	buf.SetUploader(func(memories []*Memory) error {
		// 将记忆追加到链
		for _, m := range memories {
			if err := p.index.Append(agentID, m); err != nil {
				return err
			}
		}
		return nil
	})

	p.buffers[agentID] = buf
	return buf
}

// Close 关闭所有缓冲区
func (p *BufferPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for _, buf := range p.buffers {
		if err := buf.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// BufferStats 缓冲区统计
type BufferStats struct {
	PoolSize     int            `json:"pool_size"`
	AgentBuffers map[string]int `json:"agent_buffers"` // AgentID -> pending count
}

// GetStats 获取统计信息
func (p *BufferPool) GetStats() *BufferStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := &BufferStats{
		PoolSize:     len(p.buffers),
		AgentBuffers: make(map[string]int),
	}

	for agentID, buf := range p.buffers {
		stats.AgentBuffers[agentID] = buf.PendingCount()
	}

	return stats
}

// ErrBufferClosed 缓冲区已关闭
var ErrBufferClosed = &BufferError{"buffer is closed"}

// BufferError 缓冲区错误
type BufferError struct {
	msg string
}

func (e *BufferError) Error() string {
	return e.msg
}
