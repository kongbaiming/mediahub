package recommend

import "github.com/mediahub/api/internal/domain/media"

// Item 推荐结果（库内媒资或 TMDB 发现项）
type Item struct {
	Media       *media.Media `json:"media,omitempty"`
	TMDBID      int          `json:"tmdb_id,omitempty"`
	Title       string       `json:"title"`
	PosterURL   string       `json:"poster_url,omitempty"`
	BackdropURL string       `json:"backdrop_url,omitempty"`
	MediaType   string       `json:"media_type,omitempty"`
	Rating      float64      `json:"rating,omitempty"`
	Year        *int         `json:"year,omitempty"`
	Overview    string       `json:"overview,omitempty"`
	Genres      []string     `json:"genres,omitempty"`
	External    bool         `json:"external,omitempty"`
}
