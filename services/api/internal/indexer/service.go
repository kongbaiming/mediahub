package indexer

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Service 索引搜索业务
type Service struct {
	client *Client
}

// NewService 构造
func NewService(cfg Config) *Service {
	return &Service{client: NewClient(cfg)}
}

// Enabled 索引器是否可用
func (s *Service) Enabled() bool {
	return s != nil && s.client != nil && s.client.Enabled()
}

// Search 搜索并过滤无效结果，按做种数降序
func (s *Service) Search(ctx context.Context, query, mediaType string, limit int) ([]Release, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("索引器未配置")
	}
	raw, err := s.client.Search(ctx, query, MediaSearchType(mediaType), limit*2)
	if err != nil {
		return nil, err
	}

	out := make([]Release, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, r := range raw {
		link := r.Link()
		if link == "" {
			continue
		}
		if _, ok := seen[link]; ok {
			continue
		}
		seen[link] = struct{}{}
		out = append(out, r)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Seeders != out[j].Seeders {
			return out[i].Seeders > out[j].Seeders
		}
		return out[i].Size > out[j].Size
	})

	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// BuildSearchQuery 根据标题和年份构造搜索词
func BuildSearchQuery(title string, year *int) string {
	q := strings.TrimSpace(title)
	if year != nil && *year > 0 {
		q = fmt.Sprintf("%s %d", q, *year)
	}
	return q
}

// MediaSearchType 映射媒资类型到 Prowlarr 搜索类型
func MediaSearchType(mediaType string) string {
	switch strings.ToLower(mediaType) {
	case "movie":
		return "movie"
	case "tvshow", "tv", "anime":
		return "tvSearch"
	default:
		return "search"
	}
}

// DownloadCategory 映射媒资类型到 qBit 分类目录
func DownloadCategory(mediaType string) string {
	switch strings.ToLower(mediaType) {
	case "anime":
		return "anime"
	case "tvshow", "tv":
		return "tvshow"
	default:
		return "movie"
	}
}
