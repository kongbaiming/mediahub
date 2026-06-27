package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mediahub/api/internal/apperr"
	"github.com/mediahub/api/internal/domain/live"
	"github.com/mediahub/api/pkg/httpclient"

	"github.com/google/uuid"
)

const m3uFetchTimeout = 60 * time.Second

// PreviewM3URequest 预览 M3U 列表
type PreviewM3URequest struct {
	PlaylistURL     string `json:"playlist_url"`
	PlaylistContent string `json:"playlist_content"`
}

// PreviewM3UResult 预览结果
type PreviewM3UResult struct {
	PlaylistURL string         `json:"playlist_url"`
	Total       int            `json:"total"`
	Groups      []M3UGroupStat `json:"groups"`
}

// ImportM3URequest 导入 M3U 列表
type ImportM3URequest struct {
	PlaylistURL          string   `json:"playlist_url"`
	PlaylistContent      string   `json:"playlist_content"`
	Groups               []string `json:"groups"`
	Replace              bool     `json:"replace"`
	AutoSync             *bool    `json:"auto_sync"`
	AutoSyncIntervalMins int      `json:"auto_sync_interval_minutes"`
}

// ImportM3UResult 导入结果
type ImportM3UResult struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

// PreviewM3U 拉取并预览 M3U 频道列表
func (s *LiveService) PreviewM3U(ctx context.Context, req PreviewM3URequest) (*PreviewM3UResult, error) {
	if !s.config.Enabled {
		return nil, apperr.Validation("直播功能未启用")
	}
	content, playlistURL, err := s.loadM3UContent(ctx, req.PlaylistURL, req.PlaylistContent)
	if err != nil {
		return nil, err
	}
	entries, err := ParseM3U(content, playlistURL)
	if err != nil {
		return nil, err
	}
	return &PreviewM3UResult{
		PlaylistURL: playlistURL,
		Total:       len(entries),
		Groups:      SummarizeM3UGroups(entries),
	}, nil
}

// ImportM3U 从 M3U 列表批量导入 IPTV 频道
func (s *LiveService) ImportM3U(ctx context.Context, req ImportM3URequest, userID uuid.UUID) (*ImportM3UResult, error) {
	if !s.config.Enabled {
		return nil, apperr.Validation("直播功能未启用")
	}
	content, playlistURL, err := s.loadM3UContent(ctx, req.PlaylistURL, req.PlaylistContent)
	if err != nil {
		return nil, err
	}
	entries, err := ParseM3U(content, playlistURL)
	if err != nil {
		return nil, err
	}
	entries = FilterM3UByGroup(entries, req.Groups)
	if len(entries) == 0 {
		return nil, apperr.Validation("筛选后没有可导入的频道")
	}

	result := &ImportM3UResult{}
	if req.Replace {
		if _, err := s.repo.DeleteIPTVByPlaylistURL(ctx, playlistURL); err != nil {
			return nil, err
		}
	}

	existing, err := s.repo.ListIPTVSourceURLs(ctx)
	if err != nil {
		return nil, err
	}
	existsSet := make(map[string]struct{}, len(existing))
	for _, u := range existing {
		existsSet[u] = struct{}{}
	}

	now := time.Now()
	var createdBy *uuid.UUID
	if userID != uuid.Nil {
		createdBy = &userID
	}
	for _, entry := range entries {
		streamURL, err := validateIPTVSourceURL(entry.StreamURL)
		if err != nil {
			result.Failed++
			if len(result.Errors) < 10 {
				result.Errors = append(result.Errors, entry.Title+": "+err.Error())
			}
			continue
		}
		if _, ok := existsSet[streamURL]; ok {
			result.Skipped++
			continue
		}

		key, err := generateStreamKey()
		if err != nil {
			return nil, apperr.Wrap(err, apperr.CodeInternal, "生成推流密钥失败")
		}

		title := strings.TrimSpace(entry.Title)
		if len([]rune(title)) > 200 {
			title = string([]rune(title)[:200])
		}

		room := &live.Room{
			Title:       title,
			Description: buildIPTVDescription(entry.GroupTitle, playlistURL),
			CoverURL:    strings.TrimSpace(entry.Logo),
			RoomType:    live.RoomTypeIPTV,
			SourceURL:   streamURL,
			GroupTitle:  strings.TrimSpace(entry.GroupTitle),
			PlaylistURL: playlistURL,
			Status:      live.StatusLive,
			StreamKey:   key,
			StartedAt:   &now,
			CreatedBy:   createdBy,
		}
		if err := s.repo.Create(ctx, room); err != nil {
			result.Failed++
			if len(result.Errors) < 10 {
				result.Errors = append(result.Errors, entry.Title+": 创建失败")
			}
			continue
		}
		existsSet[streamURL] = struct{}{}
		result.Created++
	}
	if result.Created > 0 || req.Replace {
		s.applyImportSyncConfig(ctx, playlistURL, req)
	}
	return result, nil
}

func (s *LiveService) applyImportSyncConfig(ctx context.Context, playlistURL string, req ImportM3URequest) {
	if s.syncRepo == nil {
		return
	}
	enabled := true
	if req.AutoSync != nil {
		enabled = *req.AutoSync
	}
	interval := 1440
	if req.AutoSyncIntervalMins > 0 {
		interval = live.NormalizeSyncInterval(req.AutoSyncIntervalMins)
	}
	_ = s.syncRepo.Upsert(ctx, &live.M3USyncJob{
		PlaylistURL:     playlistURL,
		Enabled:         enabled,
		IntervalMinutes: interval,
	})
}

func fetchRemoteText(ctx context.Context, rawURL string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, m3uFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", apperr.Wrap(err, apperr.CodeInternal, "构建请求失败")
	}
	req.Header.Set("User-Agent", "MediaHub-M3U-Importer/1.0")

	resp, err := httpclient.External.Do(req)
	if err != nil {
		return "", apperr.Validation(fmt.Sprintf(
			"无法拉取 M3U 列表：%v。请确认 NAS 能访问该地址，或在 .env 中配置 HTTPS_PROXY 后重启 api 容器",
			err,
		))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", apperr.Validation("拉取 M3U 列表失败，HTTP "+resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return "", apperr.Wrap(err, apperr.CodeInternal, "读取 M3U 列表失败")
	}
	return string(body), nil
}

func (s *LiveService) loadM3UContent(ctx context.Context, rawURL, rawContent string) (content, playlistURL string, err error) {
	rawContent = strings.TrimSpace(rawContent)
	if rawContent != "" {
		if !IsM3UPlaylist(rawContent) {
			return "", "", apperr.Validation("粘贴的内容不是有效的 M3U 频道列表")
		}
		base := strings.TrimSpace(rawURL)
		if base == "" {
			base = "inline://m3u"
		} else if u, e := validateM3UPlaylistURL(base); e == nil {
			base = u
		}
		return rawContent, base, nil
	}
	playlistURL, err = validateM3UPlaylistURL(rawURL)
	if err != nil {
		return "", "", err
	}
	content, err = fetchRemoteText(ctx, playlistURL)
	return content, playlistURL, err
}

func buildIPTVDescription(groupTitle, playlistURL string) string {
	parts := make([]string, 0, 2)
	if groupTitle != "" {
		parts = append(parts, "分组: "+groupTitle)
	}
	if playlistURL != "" {
		parts = append(parts, "来源: "+playlistURL)
	}
	return strings.Join(parts, "\n")
}

func (s *LiveService) detectAndRejectM3UPlaylist(ctx context.Context, sourceURL string) error {
	content, err := fetchRemoteText(ctx, sourceURL)
	if err != nil {
		return nil
	}
	if !IsM3UPlaylist(content) {
		return nil
	}
	entries, err := ParseM3U(content, sourceURL)
	if err != nil || len(entries) <= 1 {
		return nil
	}
	return apperr.Validation("该链接是 M3U 频道列表，请使用「导入 M3U」功能批量导入")
}
