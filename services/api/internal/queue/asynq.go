// Package queue 封装 Asynq 异步任务队列（基于 Redis）
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Task 类型常量
const (
	TypeScrapeMedia      = "scrape:media"        // 刮削媒资元数据
	TypeGenerateThumb    = "thumb:generate"      // 生成缩略图
	TypeScanDirectory    = "scan:directory"      // 扫描目录
	TypeDownloadSubtitle = "subtitle:download"   // 下载字幕
	TypeTranscodeHLS     = "transcode:hls"       // 转码 HLS
)

// Queue 队列封装
type Queue struct {
	client    *asynq.Client
	inspector *asynq.Inspector
}

// Config 队列配置
type Config struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	Concurrency   int
}

// New 创建队列（client + server 分离）
func New(cfg Config) (*Queue, *asynq.Server, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	client := asynq.NewClient(redisOpt)
	inspector := asynq.NewInspector(redisOpt)

	concurrency := cfg.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: concurrency,
		Queues: map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		},
		RetryDelayFunc: func(n int, err error, t *asynq.Task) time.Duration {
			// 指数退避：1s, 2s, 4s, 8s ... 上限 60s
			d := time.Duration(1<<n) * time.Second
			if d > 60*time.Second {
				d = 60 * time.Second
			}
			return d
		},
	})

	return &Queue{client: client, inspector: inspector}, srv, nil
}

// Client 暴露 client（用于入队）
func (q *Queue) Client() *asynq.Client { return q.client }

// Inspector 暴露 inspector
func (q *Queue) Inspector() *asynq.Inspector { return q.inspector }

// Enqueue 入队一个任务
func (q *Queue) Enqueue(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	info, err := q.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return nil, fmt.Errorf("入队失败: %w", err)
	}
	return info, nil
}

// EnqueueScrape 入队刮削任务（默认队列）
// TaskID 固定为 scrape:{mediaID}，避免同一媒资并发刮削互相覆盖元数据。
func (q *Queue) EnqueueScrape(ctx context.Context, mediaID string) error {
	payload, err := json.Marshal(map[string]string{"media_id": mediaID})
	if err != nil {
		return fmt.Errorf("序列化刮削 payload: %w", err)
	}
	task := asynq.NewTask(TypeScrapeMedia, payload)
	_, err = q.Enqueue(ctx, task,
		asynq.Queue("default"),
		asynq.MaxRetry(3),
		asynq.Timeout(5*time.Minute),
		asynq.TaskID("scrape:"+mediaID),
	)
	if err != nil && err != asynq.ErrTaskIDConflict {
		return err
	}
	return nil
}

// EnqueueThumb 入队缩略图任务
func (q *Queue) EnqueueThumb(ctx context.Context, mediaID string) error {
	payload, err := json.Marshal(map[string]string{"media_id": mediaID})
	if err != nil {
		return fmt.Errorf("序列化缩略图 payload: %w", err)
	}
	task := asynq.NewTask(TypeGenerateThumb, payload)
	_, err = q.Enqueue(ctx, task,
		asynq.Queue("low"),
		asynq.MaxRetry(2),
		asynq.Timeout(2*time.Minute),
	)
	return err
}

// EnqueueScan 入队扫描任务
func (q *Queue) EnqueueScan(ctx context.Context, path string) error {
	payload, err := json.Marshal(map[string]string{"path": path})
	if err != nil {
		return fmt.Errorf("序列化扫描 payload: %w", err)
	}
	task := asynq.NewTask(TypeScanDirectory, payload)
	_, err = q.Enqueue(ctx, task,
		asynq.Queue("critical"),
		asynq.MaxRetry(1),
		asynq.Timeout(10*time.Minute),
		asynq.TaskID(fmt.Sprintf("scan:%s", path)),
	)
	return err
}

// Close 关闭 client
func (q *Queue) Close() error {
	return q.client.Close()
}
