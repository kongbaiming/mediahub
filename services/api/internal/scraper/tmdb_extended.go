package scraper

import (
	"context"
	"fmt"
	"net/url"
)

// TMDBCastMember 演员
type TMDBCastMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Character   string `json:"character"`
	Order       int    `json:"order"`
	ProfilePath string `json:"profile_path"`
}

// TMDBCrewMember 剧组
type TMDBCrewMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Job         string `json:"job"`
	Department  string `json:"department"`
	ProfilePath string `json:"profile_path"`
}

// TMDBCredits 演职员
type TMDBCredits struct {
	Cast []TMDBCastMember `json:"cast"`
	Crew []TMDBCrewMember `json:"crew"`
}

// TMDBVideo 视频（预告等）
type TMDBVideo struct {
	ID       string `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	Site     string `json:"site"`
	Type     string `json:"type"`
	Size     int    `json:"size"`
	Official bool   `json:"official"`
}

// TMDBVideos 视频列表
type TMDBVideos struct {
	Results []TMDBVideo `json:"results"`
}

// TMDBReleaseDates 分级
type TMDBReleaseDates struct {
	Results []struct {
		ISO31661 string `json:"iso_3166_1"`
		ReleaseDates []struct {
			Certification string `json:"certification"`
		} `json:"release_dates"`
	} `json:"results"`
}

// TMDBContentRatings TV 分级
type TMDBContentRatings struct {
	Results []struct {
		ISO31661    string `json:"iso_3166_1"`
		Rating      string `json:"rating"`
	} `json:"results"`
}

func (c *TMDBClient) GetMovieCredits(ctx context.Context, id int) (*TMDBCredits, error) {
	var cr TMDBCredits
	if err := c.get(ctx, fmt.Sprintf("/movie/%d/credits", id), c.langQuery(), &cr); err != nil {
		return nil, err
	}
	return &cr, nil
}

func (c *TMDBClient) GetTVCredits(ctx context.Context, id int) (*TMDBCredits, error) {
	var cr TMDBCredits
	if err := c.get(ctx, fmt.Sprintf("/tv/%d/credits", id), c.langQuery(), &cr); err != nil {
		return nil, err
	}
	return &cr, nil
}

// TMDBPerson 影人详情（language 参数决定本地化姓名）
type TMDBPerson struct {
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	OriginalName       string  `json:"original_name"`
	ProfilePath        string  `json:"profile_path"`
	Biography          string  `json:"biography"`
	Birthday           string  `json:"birthday"`
	PlaceOfBirth       string  `json:"place_of_birth"`
	Gender             int     `json:"gender"`
	KnownForDepartment string  `json:"known_for_department"`
	Popularity         float64 `json:"popularity"`
}

func (c *TMDBClient) GetPerson(ctx context.Context, id int) (*TMDBPerson, error) {
	var p TMDBPerson
	if err := c.get(ctx, fmt.Sprintf("/person/%d", id), c.langQuery(), &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *TMDBClient) GetMovieVideos(ctx context.Context, id int) (*TMDBVideos, error) {
	var v TMDBVideos
	if err := c.get(ctx, fmt.Sprintf("/movie/%d/videos", id), c.langQuery(), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *TMDBClient) GetTVVideos(ctx context.Context, id int) (*TMDBVideos, error) {
	var v TMDBVideos
	if err := c.get(ctx, fmt.Sprintf("/tv/%d/videos", id), c.langQuery(), &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *TMDBClient) GetMovieReleaseDates(ctx context.Context, id int) (*TMDBReleaseDates, error) {
	var rd TMDBReleaseDates
	if err := c.get(ctx, fmt.Sprintf("/movie/%d/release_dates", id), url.Values{}, &rd); err != nil {
		return nil, err
	}
	return &rd, nil
}

func (c *TMDBClient) GetTVContentRatings(ctx context.Context, id int) (*TMDBContentRatings, error) {
	var cr TMDBContentRatings
	if err := c.get(ctx, fmt.Sprintf("/tv/%d/content_ratings", id), url.Values{}, &cr); err != nil {
		return nil, err
	}
	return &cr, nil
}

// SearchMulti 综合搜索（含 person）
func (c *TMDBClient) SearchMulti(ctx context.Context, query string) (*SearchResult, error) {
	q := c.langQuery()
	q.Set("query", query)
	q.Set("include_adult", "false")
	var r SearchResult
	if err := c.get(ctx, "/search/multi", q, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// YouTubeURL TMDB YouTube 预告链接
func YouTubeURL(key string) string {
	if key == "" {
		return ""
	}
	return "https://www.youtube.com/watch?v=" + key
}
