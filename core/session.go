package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Session 会话
type Session struct {
	ID        string     `json:"id"`
	Wallet    string     `json:"wallet"`    // 钱包地址
	RootHash  string     `json:"root_hash"` // 链上锚定的根哈希
	Name      string     `json:"name"`      // 会话名称
	CreatedAt int64     `json:"created_at"`
	UpdatedAt int64     `json:"updated_at"`
	Memories  []*Memory `json:"memories"`  // 记忆列表
	Tasks     []*Task   `json:"tasks"`     // 任务列表
	Context   []*Message `json:"context"`   // 当前上下文
	mu        sync.RWMutex
}

// Message 对话消息
type Message struct {
	ID        string `json:"id"`
	Role      string `json:"role"`       // user / assistant / system
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
	Hash      string `json:"hash"`       // 消息哈希
}

// Task 任务
type Task struct {
	ID          string        `json:"id"`
	Description string        `json:"description"`
	Status      string        `json:"status"` // pending / done / cancelled
	CreatedAt   int64         `json:"created_at"`
	UpdatedAt   int64         `json:"updated_at"`
	DueDate    int64         `json:"due_date"`   // 0 = 无截止
	Frequency   string        `json:"frequency"` // daily / weekly / once
	Updates    []*TaskUpdate `json:"updates"`    // 任务更新记录
}

// TaskUpdate 任务更新
type TaskUpdate struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// SessionManager 会话管理器
type SessionManager struct {
	mu      sync.RWMutex
	sessions map[string]*Session // wallet -> session
	dataDir string
}

func NewSessionManager(dataDir string) *SessionManager {
	os.MkdirAll(dataDir, 0755)
	return &SessionManager{
		sessions: make(map[string]*Session),
		dataDir: dataDir,
	}
}

// GetOrCreate 获取或创建会话
func (sm *SessionManager) GetOrCreate(wallet string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, ok := sm.sessions[wallet]; ok {
		return session
	}

	session := &Session{
		ID:        fmt.Sprintf("sess_%d", time.Now().UnixNano()),
		Wallet:    wallet,
		Name:      "New Session",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
		Memories:  make([]*Memory, 0),
		Tasks:     make([]*Task, 0),
		Context:   make([]*Message, 0),
	}

	// 尝试从磁盘加载
	if data, err := os.ReadFile(sm.sessionFile(wallet)); err == nil {
		if err := json.Unmarshal(data, session); err == nil {
			sm.sessions[wallet] = session
			return session
		}
	}

	sm.sessions[wallet] = session
	return session
}

// Get 获取会话
func (sm *SessionManager) Get(wallet string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[wallet]
}

// Save 保存会话
func (sm *SessionManager) Save(session *Session) error {
	session.UpdatedAt = time.Now().Unix()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(sm.sessionFile(session.Wallet), data, 0644)
}

// AddMessage 添加消息
func (s *Session) AddMessage(role, content string) *Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := &Message{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}

	// 计算消息哈希
	hash := sha256.Sum256([]byte(content))
	msg.Hash = hex.EncodeToString(hash[:])

	s.Context = append(s.Context, msg)
	s.UpdatedAt = time.Now().Unix()
	return msg
}

// AddMemory 添加记忆
func (s *Session) AddMemory(content []byte, tags []string) *Memory {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 设置前一个指针
	var prevID string
	if len(s.Memories) > 0 {
		prevID = s.Memories[len(s.Memories)-1].ID
	}

	mem := NewMemory(s.Wallet, content, tags)
	mem.SetPrevID(prevID)
	s.Memories = append(s.Memories, mem)
	s.UpdatedAt = time.Now().Unix()
	return mem
}

// AddTask 添加任务
func (s *Session) AddTask(description string, frequency string) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := &Task{
		ID:          fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Description: description,
		Status:      "pending",
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		Frequency:   frequency,
		Updates:     make([]*TaskUpdate, 0),
	}

	s.Tasks = append(s.Tasks, task)
	s.UpdatedAt = time.Now().Unix()
	return task
}

// UpdateTask 更新任务
func (s *Session) UpdateTask(taskID, content string) (*TaskUpdate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var task *Task
	for _, t := range s.Tasks {
		if t.ID == taskID {
			task = t
			break
		}
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	update := &TaskUpdate{
		ID:        fmt.Sprintf("update_%d", time.Now().UnixNano()),
		TaskID:    taskID,
		Content:   content,
		Timestamp: time.Now().Unix(),
	}

	task.Updates = append(task.Updates, update)
	task.UpdatedAt = time.Now().Unix()
	s.UpdatedAt = time.Now().Unix()
	return update, nil
}

// SetTaskStatus 设置任务状态
func (s *Session) SetTaskStatus(taskID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.Tasks {
		if t.ID == taskID {
			t.Status = status
			t.UpdatedAt = time.Now().Unix()
			s.UpdatedAt = time.Now().Unix()
			return nil
		}
	}
	return fmt.Errorf("task not found: %s", taskID)
}

// GetPendingTasks 获取待处理任务
func (s *Session) GetPendingTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0)
	for _, t := range s.Tasks {
		if t.Status == "pending" {
			tasks = append(tasks, t)
		}
	}
	return tasks
}

// GetContext 返回上下文摘要
func (s *Session) GetContext() []*Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Context
}

// ClearContext 清除上下文（节省 token）
func (s *Session) ClearContext() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Context = make([]*Message, 0)
}

// SetRootHash 设置链上根哈希
func (s *Session) SetRootHash(hash string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RootHash = hash
}

// Export 导出会话
func (s *Session) Export() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.MarshalIndent(s, "", "  ")
}

// Import 导入会话
func (sm *SessionManager) Import(wallet string, data []byte) (*Session, error) {
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	// 确保钱包匹配
	session.Wallet = wallet
	session.UpdatedAt = time.Now().Unix()

	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[wallet] = &session

	if err := sm.Save(&session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (sm *SessionManager) sessionFile(wallet string) string {
	return filepath.Join(sm.dataDir, wallet+".json")
}
