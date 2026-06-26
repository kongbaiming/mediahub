package scraper

import "testing"

func TestSpinoffSubtitle(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"唐朝诡事录之西行", "西行"},
		{"唐朝诡事录之西行(2024)", "西行"},
		{"唐朝诡事录", ""},
		{"之短", ""},
	}
	for _, tt := range tests {
		if got := spinoffSubtitle(tt.title); got != tt.want {
			t.Errorf("spinoffSubtitle(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}

func TestPickSeasonNumber(t *testing.T) {
	tv := &TMDBTVShow{
		Seasons: []TMDBSeason{
			{SeasonNumber: 1, Name: "第一季"},
			{SeasonNumber: 2, Name: "西行"},
		},
	}
	sn := 2
	if got := pickSeasonNumber(tv, &sn, ""); got != 2 {
		t.Fatalf("explicit season = %d, want 2", got)
	}
	if got := pickSeasonNumber(tv, nil, "西行"); got != 2 {
		t.Fatalf("subtitle match = %d, want 2", got)
	}
	if got := pickSeasonNumber(tv, nil, ""); got != 0 {
		t.Fatalf("no hint = %d, want 0", got)
	}
}
