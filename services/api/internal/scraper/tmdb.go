// Package scraper 提供外部元数据刮削能力
package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// TMDBClient TMDB API 客户端
type TMDBClient struct {
	apiKey    string
	baseURL   string
	language  string
	imageBase string
	http      *http.Client
}

// NewTMDBClient 构造
func NewTMDBClient(apiKey, baseURL, language string) *TMDBClient {
	return &TMDBClient{
		apiKey:    apiKey,
		baseURL:   strings.TrimRight(baseURL, "/"),
		language:  language,
		imageBase: "https://image.tmdb.org/t/p",
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ---- 类型定义（简化版，覆盖核心字段）----

// TMDBMovie 电影详情
type TMDBMovie struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Overview      string  `json:"overview"`
	ReleaseDate   string  `json:"release_date"`
	Runtime       int     `json:"runtime"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	Genres        []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"genres"`
	IMDBID string `json:"imdb_id"`
}

// TMDBTVShow 剧集详情
type TMDBTVShow struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	OriginalName  string  `json:"original_name"`
	Overview      string  `json:"overview"`
	FirstAirDate  string  `json:"first_air_date"`
	LastAirDate   string  `json:"last_air_date"`
	EpisodeRunTime []int  `json:"episode_run_time"`
	NumberOfSeasons int   `json:"number_of_seasons"`
	NumberOfEpisodes int  `json:"number_of_episodes"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	Genres        []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"genres"`
	Seasons []TMDBSeason `json:"seasons"`
}

// TMDBSeason 季
type TMDBSeason struct {
	ID           int    `json:"id"`
	SeasonNumber int    `json:"season_number"`
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	AirDate      string `json:"air_date"`
	EpisodeCount int    `json:"episode_count"`
	PosterPath   string `json:"poster_path"`
}

// TMDBEpisode 集
type TMDBEpisode struct {
	ID            int    `json:"id"`
	EpisodeNumber int    `json:"episode_number"`
	Name          string `json:"name"`
	Overview      string `json:"overview"`
	AirDate       string `json:"air_date"`
	Runtime       int    `json:"runtime"`
	StillPath     string `json:"still_path"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Page         int            `json:"page"`
	Results      []SearchEntry  `json:"results"`
	TotalResults int            `json:"total_results"`
	TotalPages   int            `json:"total_pages"`
}

// SearchEntry 搜索条目
type SearchEntry struct {
	ID            int     `json:"id"`
	MediaType     string  `json:"media_type"` // movie | tv | person
	Title         string  `json:"title,omitempty"`
	Name          string  `json:"name,omitempty"`
	OriginalTitle string  `json:"original_title,omitempty"`
	OriginalName  string  `json:"original_name,omitempty"`
	Overview      string  `json:"overview"`
	ReleaseDate   string  `json:"release_date,omitempty"`
	FirstAirDate  string  `json:"first_air_date,omitempty"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
}

// ---- API 调用 ----

// SearchMovie 搜索电影
func (c *TMDBClient) SearchMovie(ctx context.Context, query string, year *int) (*SearchResult, error) {
	q := url.Values{}
	q.Set("query", query)
	q.Set("language", c.language)
	q.Set("include_adult", "false")
	if year != nil {
		q.Set("year", strconv.Itoa(*year))
	}
	return c.search(ctx, "/search/movie", q)
}

// SearchTV 搜索剧集
func (c *TMDBClient) SearchTV(ctx context.Context, query string, year *int) (*SearchResult, error) {
	q := url.Values{}
	q.Set("query", query)
	q.Set("language", c.language)
	q.Set("include_adult", "false")
	if year != nil {
		q.Set("first_air_date_year", strconv.Itoa(*year))
	}
	return c.search(ctx, "/search/tv", q)
}

// GetMovie 获取电影详情
func (c *TMDBClient) GetMovie(ctx context.Context, id int) (*TMDBMovie, error) {
	q := url.Values{}
	q.Set("language", c.language)
	var m TMDBMovie
	if err := c.get(ctx, fmt.Sprintf("/movie/%d", id), q, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// GetTVShow 获取剧集详情
func (c *TMDBClient) GetTVShow(ctx context.Context, id int) (*TMDBTVShow, error) {
	q := url.Values{}
	q.Set("language", c.language)
	var t TMDBTVShow
	if err := c.get(ctx, fmt.Sprintf("/tv/%d", id), q, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// GetSeason 获取季详情
func (c *TMDBClient) GetSeason(ctx context.Context, tvID, seasonNumber int) (*TMDBSeason, error) {
	q := url.Values{}
	q.Set("language", c.language)
	var s TMDBSeason
	if err := c.get(ctx, fmt.Sprintf("/tv/%d/season/%d", tvID, seasonNumber), q, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// GetTrending 获取热门
func (c *TMDBClient) GetTrending(ctx context.Context, mediaType, period string) (*SearchResult, error) {
	q := url.Values{}
	q.Set("language", c.language)
	periodStr := period
	if periodStr == "" {
		periodStr = "week"
	}
	return c.search(ctx, fmt.Sprintf("/trending/%s/%s", mediaType, periodStr), q)
}

// PosterURL 海报 URL
func (c *TMDBClient) PosterURL(path string, size string) string {
	if path == "" {
		return ""
	}
	if size == "" {
		size = "w500"
	}
	return fmt.Sprintf("%s/%s%s", c.imageBase, size, path)
}

// BackdropURL 背景图 URL
func (c *TMDBClient) BackdropURL(path string, size string) string {
	if path == "" {
		return ""
	}
	if size == "" {
		size = "w1280"
	}
	return fmt.Sprintf("%s/%s%s", c.imageBase, size, path)
}

// ---- 内部方法 ----

func (c *TMDBClient) search(ctx context.Context, path string, q url.Values) (*SearchResult, error) {
	var r SearchResult
	if err := c.get(ctx, path, q, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *TMDBClient) get(ctx context.Context, path string, q url.Values, out any) error {
	u := c.baseURL + path
	if q != nil {
		u += "?" + q.Encode() + "&api_key=" + c.apiKey
	} else {
		u += "?api_key=" + c.apiKey
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "MediaHub/0.1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("TMDB 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("TMDB API key 无效（401）")
	}
	if resp.StatusCode == 429 {
		return fmt.Errorf("TMDB 限速（429）")
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("TMDB 错误 %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}
