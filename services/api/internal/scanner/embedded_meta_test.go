package scanner

import "testing"

func TestExtractEmbeddedMeta_MovieTitle(t *testing.T) {
	p := &ProbeResult{
		Format: ProbeFormat{
			Duration: "5820",
			Tags: map[string]string{
				"title":    "冰河世纪4",
				"imdb_id":  "tt1667888",
				"date":     "2012-07-13",
			},
		},
	}
	meta := ExtractEmbeddedMeta(p)
	if meta.Title != "冰河世纪4" {
		t.Fatalf("title = %q", meta.Title)
	}
	if meta.IMDBID != "tt1667888" {
		t.Fatalf("imdb = %q", meta.IMDBID)
	}
	if meta.Year == nil || *meta.Year != 2012 {
		t.Fatalf("year = %v", meta.Year)
	}
	if meta.DurationSec != 5820 {
		t.Fatalf("duration = %d", meta.DurationSec)
	}
}

func TestExtractEmbeddedMeta_TVShow(t *testing.T) {
	p := &ProbeResult{
		Format: ProbeFormat{
			Tags: map[string]string{
				"show":           "庆余年",
				"season_number":  "2",
				"episode_id":     "5",
				"title":          "第5集",
			},
		},
	}
	meta := ExtractEmbeddedMeta(p)
	if meta.Show != "庆余年" {
		t.Fatalf("show = %q", meta.Show)
	}
	if meta.Season == nil || *meta.Season != 2 {
		t.Fatalf("season = %v", meta.Season)
	}
	if meta.Episode == nil || *meta.Episode != 5 {
		t.Fatalf("episode = %v", meta.Episode)
	}
}

func TestPrependEmbeddedCandidates(t *testing.T) {
	emb := &EmbeddedMeta{Title: "Frozen II"}
	out := PrependEmbeddedCandidates([]string{"053", "weak"}, emb)
	if out[0] != "Frozen II" {
		t.Fatalf("first = %q, want Frozen II", out[0])
	}
}
