package cache

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// mockRedisClient — 简易内存版 Redis mock（用于单元测试）
type mockRedisClient struct {
	data map[string]mockEntry
}

type mockEntry struct {
	val     []byte
	expires time.Time
}

func newMockRedis() *mockRedisClient {
	return &mockRedisClient{data: make(map[string]mockEntry)}
}

// 测试 GetOrLoad 的命中/失效逻辑
type fakeLoader struct {
	calls atomic.Int64
	val   any
}

func (f *fakeLoader) load(ctx context.Context) (any, error) {
	f.calls.Add(1)
	return f.val, nil
}

func TestCacheGetOrLoad(t *testing.T) {
	t.Run("disabled cache calls loader every time", func(t *testing.T) {
		c := New(Config{Enabled: false})
		loader := &fakeLoader{val: "hello"}
		for i := 0; i < 5; i++ {
			v, err := c.GetOrLoad(context.Background(), "k", time.Minute, loader.load)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if v != "hello" {
				t.Fatalf("expected hello, got %v", v)
			}
		}
		if loader.calls.Load() != 5 {
			t.Fatalf("expected 5 calls, got %d", loader.calls.Load())
		}
	})

	t.Run("key prefix", func(t *testing.T) {
		c := New(Config{Prefix: "myapp"})
		_ = c.fullKey("foo")
	})
}

func TestCacheFullKey(t *testing.T) {
	c := New(Config{Prefix: "mediahub"})
	tests := []struct {
		key      string
		expected string
	}{
		{"feed:android-tv", "mediahub:feed:android-tv"},
		{"user:123", "mediahub:user:123"},
	}
	for _, tt := range tests {
		got := c.fullKey(tt.key)
		if got != tt.expected {
			t.Errorf("fullKey(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}

	// 长 key 应该 hash
	longKey := ""
	for i := 0; i < 100; i++ {
		longKey += "abcdefghij"
	}
	got := c.fullKey(longKey)
	if len(got) > 64+len("mediahub:") {
		t.Errorf("long key should be hashed, got len=%d", len(got))
	}
}