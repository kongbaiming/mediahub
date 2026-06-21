package recommend

import "github.com/mediahub/api/internal/domain/media"

// Item 推荐结果（库内媒资或 TMDB 发现项）
type Item struct {
	Media     *media.Media
	TMDBID    int
	Title     string
	PosterURL string
	BackdropURL string
	MediaType string // movie | tvshow
	Rating    float64
	Year      *int
	Overview  string
	Genres    []string
	External  bool // 库外 TMDB 推荐
}
