package layout

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestLayoutConfig_JSON(t *testing.T) {
	cfg := LayoutConfig{
		Theme: "dark",
		Rows: []Row{
			{
				ID:        "row-1",
				Type:      "hero-banner",
				Title:     "编辑推荐",
				CardStyle: "banner",
				Source: DataSource{
					Type: "manual",
					Params: map[string]any{
						"ids": []string{"tt1", "tt2"},
					},
				},
				Visible: true,
			},
			{
				ID:        "row-2",
				Type:      "shelf",
				Title:     "继续观看",
				CardStyle: "poster",
				Source: DataSource{
					Type: "continue-watching",
				},
			},
		},
		Global: map[string]any{
			"max_rows": 20,
		},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded LayoutConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Theme != cfg.Theme {
		t.Errorf("Theme = %q, want %q", decoded.Theme, cfg.Theme)
	}
	if len(decoded.Rows) != len(cfg.Rows) {
		t.Fatalf("Rows len = %d, want %d", len(decoded.Rows), len(cfg.Rows))
	}
	if decoded.Rows[0].ID != "row-1" {
		t.Errorf("Rows[0].ID = %q, want %q", decoded.Rows[0].ID, "row-1")
	}
	if decoded.Rows[1].Source.Type != "continue-watching" {
		t.Errorf("Rows[1].Source.Type = %q, want %q", decoded.Rows[1].Source.Type, "continue-watching")
	}
}

func TestDynamicRules_Matches(t *testing.T) {
	tests := []struct {
		name  string
		rules *DynamicRules
		now   time.Time
		want  bool
	}{
		{
			name:  "nil rules always match",
			rules: nil,
			now:   time.Now(),
			want:  true,
		},
		{
			name: "hour range during day",
			rules: &DynamicRules{
				HourOfDay: &HourRange{Start: 9, End: 18},
			},
			now:  time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "hour range outside",
			rules: &DynamicRules{
				HourOfDay: &HourRange{Start: 9, End: 18},
			},
			now:  time.Date(2026, 1, 1, 20, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "cross-night range start",
			rules: &DynamicRules{
				HourOfDay: &HourRange{Start: 22, End: 6},
			},
			now:  time.Date(2026, 1, 1, 23, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "cross-night range end",
			rules: &DynamicRules{
				HourOfDay: &HourRange{Start: 22, End: 6},
			},
			now:  time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "cross-night range middle (no match)",
			rules: &DynamicRules{
				HourOfDay: &HourRange{Start: 22, End: 6},
			},
			now:  time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "day of week match",
			rules: &DynamicRules{
				DayOfWeek: []int{1, 2, 3, 4, 5},
			},
			now:  time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC), // Monday
			want: true,
		},
		{
			name: "day of week no match",
			rules: &DynamicRules{
				DayOfWeek: []int{1, 2, 3, 4, 5},
			},
			now:  time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC), // Saturday
			want: false,
		},
		{
			name: "combined hour + day match",
			rules: &DynamicRules{
				HourOfDay: &HourRange{Start: 18, End: 23},
				DayOfWeek: []int{5, 6}, // Fri, Sat
			},
			now:  time.Date(2026, 1, 2, 20, 0, 0, 0, time.UTC), // Friday 20:00
			want: true,
		},
		{
			name: "combined hour + day fail on hour",
			rules: &DynamicRules{
				HourOfDay: &HourRange{Start: 18, End: 23},
				DayOfWeek: []int{5, 6},
			},
			now:  time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC), // Friday 10:00
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rules.Matches(tt.now); got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFeedRow_JSON(t *testing.T) {
	row := FeedRow{
		ID:        "row-1",
		Type:      "shelf",
		Title:     "继续观看",
		CardStyle: "poster",
		Items: []FeedItem{
			{
				MediaID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				Title:     "Inception",
				Year:      yearPtr(2010),
				Rating:    8.8,
				PosterURL: "https://example.com/poster.jpg",
				Genres:    []string{"Action", "Sci-Fi"},
			},
		},
	}

	data, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded FeedRow
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.ID != row.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, row.ID)
	}
	if len(decoded.Items) != 1 {
		t.Fatalf("Items len = %d, want 1", len(decoded.Items))
	}
	if decoded.Items[0].Title != "Inception" {
		t.Errorf("Items[0].Title = %q, want %q", decoded.Items[0].Title, "Inception")
	}
}

func yearPtr(y int) *int { return &y }
