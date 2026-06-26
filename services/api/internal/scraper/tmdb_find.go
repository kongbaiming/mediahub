package scraper

import (
	"context"
	"fmt"
	"strings"
)

// FindResult TMDB /find 外部 ID 查询结果
type FindResult struct {
	MovieResults []SearchEntry `json:"movie_results"`
	TVResults    []SearchEntry `json:"tv_results"`
}

// FindByIMDBID 通过 IMDb ID 查找 TMDB 条目
func (c *TMDBClient) FindByIMDBID(ctx context.Context, imdbID string) (*FindResult, error) {
	imdbID = strings.TrimSpace(strings.ToLower(imdbID))
	if imdbID == "" {
		return nil, fmt.Errorf("imdb id 为空")
	}
	q := c.langQuery()
	q.Set("external_source", "imdb_id")
	var r FindResult
	if err := c.get(ctx, fmt.Sprintf("/find/%s", imdbID), q, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
