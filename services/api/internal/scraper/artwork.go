package scraper

import (
	"context"
	"strings"
)

// ResolveMissingArtwork 在 TMDB 主条目无海报/背景时，尝试季图与关联条目（如「X之Y」→ 主剧 X 的季海报）
func (c *TMDBClient) ResolveMissingArtwork(ctx context.Context, isTV bool, tmdbID int, title string, seasonNum *int, year *int) (posterURL, backdropURL string) {
	if tmdbID <= 0 {
		return "", ""
	}
	if isTV {
		return c.resolveTVArtwork(ctx, tmdbID, title, seasonNum, year)
	}
	movie, err := c.GetMovie(ctx, tmdbID)
	if err != nil {
		return "", ""
	}
	return c.PosterURL(movie.PosterPath, "w500"), c.BackdropURL(movie.BackdropPath, "w1280")
}

func (c *TMDBClient) resolveTVArtwork(ctx context.Context, tvID int, title string, seasonNum *int, year *int) (posterURL, backdropURL string) {
	tv, err := c.GetTVShow(ctx, tvID)
	if err != nil {
		return c.artworkFromTVSearch(ctx, title, seasonNum, year, tvID)
	}

	posterURL = c.PosterURL(tv.PosterPath, "w500")
	backdropURL = c.BackdropURL(tv.BackdropPath, "w1280")
	if posterURL != "" && backdropURL != "" {
		return posterURL, backdropURL
	}

	if posterURL == "" {
		posterURL = c.tvSeasonPoster(ctx, tvID, tv, seasonNum, spinoffSubtitle(title))
	}
	if posterURL == "" || backdropURL == "" {
		sp, sb := c.artworkFromTVSearch(ctx, title, seasonNum, year, tvID)
		if posterURL == "" {
			posterURL = sp
		}
		if backdropURL == "" {
			backdropURL = sb
		}
	}
	return posterURL, backdropURL
}

func (c *TMDBClient) tvSeasonPoster(ctx context.Context, tvID int, tv *TMDBTVShow, seasonNum *int, subtitle string) string {
	if sn := pickSeasonNumber(tv, seasonNum, subtitle); sn > 0 {
		if s, err := c.GetSeason(ctx, tvID, sn); err == nil {
			if p := c.PosterURL(s.PosterPath, "w500"); p != "" {
				return p
			}
		}
	}
	for _, s := range tv.Seasons {
		if s.SeasonNumber <= 0 {
			continue
		}
		if p := c.PosterURL(s.PosterPath, "w500"); p != "" {
			return p
		}
		if full, err := c.GetSeason(ctx, tvID, s.SeasonNumber); err == nil {
			if p := c.PosterURL(full.PosterPath, "w500"); p != "" {
				return p
			}
		}
	}
	return ""
}

func (c *TMDBClient) artworkFromTVSearch(ctx context.Context, title string, seasonNum *int, year *int, matchedID int) (posterURL, backdropURL string) {
	subtitle := spinoffSubtitle(title)
	for _, q := range SearchQueries(title) {
		res, err := c.SearchTV(ctx, q, year)
		if err != nil || len(res.Results) == 0 {
			if year != nil {
				res, err = c.SearchTV(ctx, q, nil)
			}
		}
		if err != nil || len(res.Results) == 0 {
			continue
		}
		for _, e := range res.Results {
			if e.ID == matchedID {
				if p := c.PosterURL(e.PosterPath, "w500"); p != "" {
					posterURL = p
				}
				if b := c.BackdropURL(e.BackdropPath, "w1280"); b != "" {
					backdropURL = b
				}
				if posterURL != "" && backdropURL != "" {
					return posterURL, backdropURL
				}
				continue
			}
			tv, err := c.GetTVShow(ctx, e.ID)
			if err != nil {
				continue
			}
			p := c.PosterURL(tv.PosterPath, "w500")
			b := c.BackdropURL(tv.BackdropPath, "w1280")
			if p == "" {
				p = c.tvSeasonPoster(ctx, e.ID, tv, seasonNum, subtitle)
			}
			if p == "" {
				p = c.PosterURL(e.PosterPath, "w500")
			}
			if b == "" {
				b = c.BackdropURL(e.BackdropPath, "w1280")
			}
			if p != "" || b != "" {
				return p, b
			}
		}
	}
	return posterURL, backdropURL
}

func spinoffSubtitle(title string) string {
	t := normalizeSearchTitle(title)
	idx := strings.LastIndex(t, "之")
	if idx <= 2 || idx >= len(t)-1 {
		return ""
	}
	return strings.TrimSpace(t[idx+len("之"):])
}

func pickSeasonNumber(tv *TMDBTVShow, seasonNum *int, subtitle string) int {
	if seasonNum != nil && *seasonNum > 0 {
		return *seasonNum
	}
	if subtitle == "" {
		return 0
	}
	for _, s := range tv.Seasons {
		if s.SeasonNumber <= 0 {
			continue
		}
		if strings.Contains(s.Name, subtitle) {
			return s.SeasonNumber
		}
	}
	return 0
}
