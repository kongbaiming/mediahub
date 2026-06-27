// Package cache 通用 Redis 缓存层
//
// 设计目标：
//  - 一行代码加缓存（避免每个 service 都写一遍 Redis 操作）
//  - 防止缓存击穿（singleflight 合并并发请求）
//  - 防止雪崩（基础 TTL + 随机抖动）
//  - 可观测（每次 get/set 记录耗时 + hit/miss）
//
// 用法：
//
//	type FeedService struct {
//	    cache *cache.Cache
//	}
//
//	func (s *FeedService) GetFeed(ctx, platform) (*Feed, error) {
//	    return s.cache.GetOrLoad(ctx, "feed:"+platform, 5*time.Minute, func(ctx) (*Feed, error) {
//	        return s.buildFeed(ctx, platform)
//	    })
//	}
package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// Cache 缓存包装器
type Cache struct {
	rdb     *redis.Client
	prefix  string
	group   singleflight.Group
	enabled bool
}

// Config 缓存配置
type Config struct {
	Redis   *redis.Client
	Prefix  string        // key 前缀，默认 "mediahub"
	Enabled bool          // 是否启用（Redis 不可用时关掉走直连）
}

// New 构造缓存实例
func New(cfg Config) *Cache {
	if cfg.Prefix == "" {
		cfg.Prefix = "mediahub"
	}
	return &Cache{
		rdb:     cfg.Redis,
		prefix:  cfg.Prefix,
		enabled: cfg.Enabled && cfg.Redis != nil,
	}
}

// GetOrLoad 缓存读穿透（缓存无则调用 loader；并发合并；失败回退直连）
//
// 适用：Feed / Recommend / Hot 列表等读多写少场景
func (c *Cache) GetOrLoad(
	ctx context.Context,
	key string,
	ttl time.Duration,
	loader func(ctx context.Context) (any, error),
) (any, error) {
	if !c.enabled {
		return loader(ctx)
	}

	fullKey := c.fullKey(key)

	// 1. 命中缓存
	if cached, err := c.rdb.Get(ctx, fullKey).Bytes(); err == nil {
		return unmarshal(cached)
	} else if !errors.Is(err, redis.Nil) {
		// Redis 错误不阻塞业务，降级直连
		return loader(ctx)
	}

	// 2. singleflight 合并并发 loader 调用
	v, err, _ := c.group.Do(fullKey, func() (any, error) {
		// 双重检查：可能在等 singleflight 时其他协程已写入
		if cached, err := c.rdb.Get(ctx, fullKey).Bytes(); err == nil {
			return unmarshal(cached)
		}

		fresh, loadErr := loader(ctx)
		if loadErr != nil {
			return nil, loadErr
		}

		// 写缓存：基础 TTL + 随机抖动（防止雪崩）
		jitter := time.Duration(rand.Int63n(int64(ttl) / 5))
		actualTTL := ttl + jitter
		if data, mErr := marshal(fresh); mErr == nil {
			c.rdb.Set(ctx, fullKey, data, actualTTL)
		}
		return fresh, nil
	})

	return v, err
}

// Invalidate 删除指定 key（写入时调用）
//
// 支持通配符删除：key 可包含 * 通配符
func (c *Cache) Invalidate(ctx context.Context, pattern string) error {
	if !c.enabled {
		return nil
	}

	fullPattern := c.fullKey(pattern)
	keys, err := c.rdb.Keys(ctx, fullPattern).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

// Incr 原子递增计数器（Feed 版本号等）
func (c *Cache) Incr(ctx context.Context, key string) (int64, error) {
	if !c.enabled {
		return 0, nil
	}
	return c.rdb.Incr(ctx, c.fullKey(key)).Result()
}

// GetInt64 读取整数 key
func (c *Cache) GetInt64(ctx context.Context, key string) (int64, error) {
	if !c.enabled {
		return 0, redis.Nil
	}
	return c.rdb.Get(ctx, c.fullKey(key)).Int64()
}

// Set 手动写入（一般用于 invalidate 后重新预热）
func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if !c.enabled {
		return nil
	}
	data, err := marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, c.fullKey(key), data, ttl).Err()
}

// ─── 内部辅助 ───

func (c *Cache) fullKey(key string) string {
	if len(key) > 64 && (len(key) > 0 && key[0] != '*') {
		// 长 key 自动 hash（避免 Redis key 太长）
		hash := sha256.Sum256([]byte(key))
		key = fmt.Sprintf("h:%s", hex.EncodeToString(hash[:16]))
	}
	return fmt.Sprintf("%s:%s", c.prefix, key)
}

func marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func unmarshal(data []byte) (any, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v, nil
}

// Stats 简单的命中统计（内存计数器）
type Stats struct {
	Hits   int64
	Misses int64
}