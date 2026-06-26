// Package hlsstore 持久化 HLS 转码任务状态（Redis + 进程内缓存）
package hlsstore

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// TaskRecord 可序列化的转码任务快照
type TaskRecord struct {
	MediaID   string    `json:"media_id"`
	Input     string    `json:"input,omitempty"`
	OutputDir string    `json:"output_dir,omitempty"`
	Status    string    `json:"status"` // running | done | failed
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at"`
	CopyVideo bool      `json:"copy_video"`
	Height    int       `json:"height"`
}

// Store HLS 任务状态存储
type Store struct {
	rdb    *redis.Client
	prefix string
	mem    map[string]*TaskRecord
	mu     sync.RWMutex
}

// New 构造；rdb 为 nil 时仅使用进程内缓存
func New(rdb *redis.Client) *Store {
	return &Store{
		rdb:    rdb,
		prefix: "mediahub:hls:task:",
		mem:    make(map[string]*TaskRecord),
	}
}

func (s *Store) key(mediaID string) string {
	return s.prefix + mediaID
}

func (s *Store) ttl(status string) time.Duration {
	switch status {
	case "done":
		return 24 * time.Hour
	case "failed":
		return 6 * time.Hour
	default:
		return 6 * time.Hour
	}
}

// Set 写入任务状态
func (s *Store) Set(ctx context.Context, rec *TaskRecord) {
	if rec == nil {
		return
	}
	cp := *rec
	s.mu.Lock()
	s.mem[rec.MediaID] = &cp
	s.mu.Unlock()

	if s.rdb == nil {
		return
	}
	data, err := json.Marshal(&cp)
	if err != nil {
		return
	}
	_ = s.rdb.Set(ctx, s.key(rec.MediaID), data, s.ttl(rec.Status)).Err()
}

// Get 读取任务状态（内存优先，Redis 次之）
func (s *Store) Get(ctx context.Context, mediaID string) (*TaskRecord, bool) {
	s.mu.RLock()
	if t, ok := s.mem[mediaID]; ok {
		cp := *t
		s.mu.RUnlock()
		return &cp, true
	}
	s.mu.RUnlock()

	if s.rdb == nil {
		return nil, false
	}
	data, err := s.rdb.Get(ctx, s.key(mediaID)).Bytes()
	if err != nil {
		return nil, false
	}
	var rec TaskRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, false
	}
	s.mu.Lock()
	s.mem[mediaID] = &rec
	s.mu.Unlock()
	return &rec, true
}

// Delete 删除任务记录
func (s *Store) Delete(ctx context.Context, mediaID string) {
	s.mu.Lock()
	delete(s.mem, mediaID)
	s.mu.Unlock()
	if s.rdb != nil {
		_ = s.rdb.Del(ctx, s.key(mediaID)).Err()
	}
}
