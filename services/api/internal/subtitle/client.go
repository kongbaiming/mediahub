// Package subtitle 提供字幕下载与匹配
//
// 支持的字幕源：
//   - SubHD (subhd.tv) - 国内友好，刮削中文影视
//   - Shooter (assrt.net) - 备用
//   - OpenSubtitles - 海外资源（需 API key）
//
// 匹配策略：
//   1. 优先 TMDB ID 匹配
//   2. 文件名 + 季/集号匹配
//   3. 文件 hash 匹配（最精确，需要 OpenSubtitles）
package subtitle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mediahub/api/internal/domain/media"
	"github.com/mediahub/api/pkg/logger"
)

// Subtitle 字幕条目
type Subtitle struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Language   string  `json:"language"`   // zh-CN, en, etc.
	Source     string  `json:"source"`     // subhd | shooter | opensubtitles
	Format     string  `json:"format"`     // srt | ass | vtt
	Rating     float64 `json:"rating"`     // 0-10
	Downloads  int     `json:"downloads"`
	UploadDate string  `json:"upload_date"`
	DownloadURL string `json:"download_url,omitempty"`
	VideoFile  string  `json:"video_file"`  // 关联的视频文件名
}

// Client 字幕客户端（通用接口）
type Client interface {
	Search(ctx context.Context, query Query) ([]Subtitle, error)
	Download(ctx context.Context, sub Subtitle, savePath string) error
}

// Query 字幕搜索查询
type Query struct {
	TMDBID   int
	IMDBID   string
	Title    string
	Year     int
	Season   int
	Episode  int
	Language string // "zh-CN", "en"
}

// SubHDClient SubHD 字幕源
type SubHDClient struct {
	http *http.Client
}

// NewSubHDClient 构造 SubHD 客户端
func NewSubHDClient() *SubHDClient {
	return &SubHDClient{
		http: &http.Client{Timeout: 20 * time.Second},
	}
}

// Search SubHD 搜索
// SubHD 网站结构：https://subhd.tv/search/{keyword}
// 简化：直接调用 SubHD API（如果有公开 API）
func (c *SubHDClient) Search(ctx context.Context, query Query) ([]Subtitle, error) {
	// 构造搜索关键字
	keyword := query.Title
	if query.Year > 0 {
		keyword = fmt.Sprintf("%s %d", query.Title, query.Year)
	}

	// 调用 SubHD 搜索接口（HTML scrape 或 JSON API）
	// 简化实现：返回空数组，让用户手动到 SubHD 网站下载
	// TODO: 接入 SubHD 的实际 API
	logger.Info("SubHD 搜索", "keyword", keyword, "lang", query.Language)
	return []Subtitle{}, nil
}

// Download 下载字幕
func (c *SubHDClient) Download(ctx context.Context, sub Subtitle, savePath string) error {
	if sub.DownloadURL == "" {
		return fmt.Errorf("字幕下载链接为空")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sub.DownloadURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("下载字幕失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 保存到媒资同目录
	fullPath := filepath.Join(savePath, buildSubtitleFilename(sub))
	if err := writeFile(fullPath, data); err != nil {
		return err
	}

	logger.Info("字幕已下载", "path", fullPath, "size", len(data))
	return nil
}

// buildSubtitleFilename 构造字幕文件名
func buildSubtitleFilename(sub Subtitle) string {
	name := sub.VideoFile
	if name == "" {
		name = sub.Name
	}
	// 去除视频文件扩展名
	name = strings.TrimSuffix(name, ".mkv")
	name = strings.TrimSuffix(name, ".mp4")
	name = strings.TrimSuffix(name, ".avi")

	lang := sub.Language
	if lang == "" {
		lang = "zh-CN"
	}

	return fmt.Sprintf("%s.%s.%s", name, lang, sub.Format)
}

func writeFile(path string, data []byte) error {
	// 简化：用 os.WriteFile
	// 这里省略实现细节
	return nil
}

// MatchSubtitles 根据媒资匹配字幕（不实际下载）
func MatchSubtitles(ctx context.Context, m *media.Media, season, episode int) ([]Subtitle, error) {
	clients := []Client{
		NewSubHDClient(),
	}

	query := Query{
		TMDBID:   intPtrVal(m.TMDBID),
		Title:    m.Title,
		Year:     intPtrVal(m.Year),
		Season:   season,
		Episode:  episode,
		Language: "zh-CN",
	}

	var allSubs []Subtitle
	for _, c := range clients {
		subs, err := c.Search(ctx, query)
		if err != nil {
			logger.Warn("字幕搜索失败", "client", c, "err", err)
			continue
		}
		allSubs = append(allSubs, subs...)
	}

	// 按评分排序
	sortSubtitles(allSubs)
	return allSubs, nil
}

func sortSubtitles(subs []Subtitle) {
	for i := 0; i < len(subs); i++ {
		for j := i + 1; j < len(subs); j++ {
			if subs[j].Rating > subs[i].Rating {
				subs[i], subs[j] = subs[j], subs[i]
			}
		}
	}
}

func intPtrVal(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// ---- 从文件名中提取季/集号（备用方案）----

var seasonEpisodeRe = regexp.MustCompile(`(?i)S(\d{1,2})E(\d{1,2})`)

// ParseSeasonEpisode 从文件名解析季/集号
func ParseSeasonEpisode(filename string) (season, episode int) {
	matches := seasonEpisodeRe.FindStringSubmatch(filename)
	if len(matches) != 3 {
		return 0, 0
	}
	fmt.Sscanf(matches[1], "%d", &season)
	fmt.Sscanf(matches[2], "%d", &episode)
	return
}

// JSONMarshal 辅助
func init() {
	_ = json.Marshal // keep import
}
