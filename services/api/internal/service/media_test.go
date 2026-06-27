package service

import (
	"testing"

	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"

	"github.com/google/uuid"
)

func TestToDetail_SortsSeasonsAndEpisodes(t *testing.T) {
	m := &media.Media{
		BaseModel: common.BaseModel{ID: uuid.New()},
		Type:      common.MediaTypeTVShow,
		Kind:      media.MediaKindSeries,
		Title:     "测试剧",
		Seasons: []media.Season{
			{
				BaseModel:    common.BaseModel{ID: uuid.New()},
				SeasonNumber: 2,
				Episodes: []media.Episode{
					{BaseModel: common.BaseModel{ID: uuid.New()}, EpisodeNumber: 3, Title: "S2E3"},
					{BaseModel: common.BaseModel{ID: uuid.New()}, EpisodeNumber: 1, Title: "S2E1"},
				},
			},
			{
				BaseModel:    common.BaseModel{ID: uuid.New()},
				SeasonNumber: 1,
				Episodes: []media.Episode{
					{BaseModel: common.BaseModel{ID: uuid.New()}, EpisodeNumber: 2, Title: "S1E2"},
					{BaseModel: common.BaseModel{ID: uuid.New()}, EpisodeNumber: 1, Title: "S1E1"},
				},
			},
		},
	}

	d := toDetail(m)
	if len(d.Seasons) != 2 {
		t.Fatalf("seasons = %d, want 2", len(d.Seasons))
	}
	if d.Seasons[0].SeasonNumber != 1 || d.Seasons[1].SeasonNumber != 2 {
		t.Fatalf("season order = [%d,%d], want [1,2]", d.Seasons[0].SeasonNumber, d.Seasons[1].SeasonNumber)
	}
	if d.Seasons[0].Episodes[0].EpisodeNumber != 1 || d.Seasons[0].Episodes[1].EpisodeNumber != 2 {
		t.Fatalf("S1 episodes = [%d,%d], want [1,2]",
			d.Seasons[0].Episodes[0].EpisodeNumber, d.Seasons[0].Episodes[1].EpisodeNumber)
	}
	if d.Seasons[1].Episodes[0].EpisodeNumber != 1 || d.Seasons[1].Episodes[1].EpisodeNumber != 3 {
		t.Fatalf("S2 episodes = [%d,%d], want [1,3]",
			d.Seasons[1].Episodes[0].EpisodeNumber, d.Seasons[1].Episodes[1].EpisodeNumber)
	}
}
