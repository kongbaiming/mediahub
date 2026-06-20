package recommend

import (
	"testing"

	"github.com/mediahub/api/internal/domain/common"
	"github.com/mediahub/api/internal/domain/media"
)

func makeMedia(id, title string, year int, genres []string, rating float64) *media.Media {
	y := year
	return &media.Media{
		Type:   common.MediaTypeMovie,
		Title:  title,
		Year:   &y,
		Rating: rating,
		Genres: genres,
	}
}

func TestContentSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    *media.Media
		b    *media.Media
		want float64 // 最小值
	}{
		{
			name: "identical",
			a:    makeMedia("1", "A", 2020, []string{"Action"}, 8.0),
			b:    makeMedia("1", "A", 2020, []string{"Action"}, 8.0),
			want: 1.0,
		},
		{
			name: "same genre and year",
			a:    makeMedia("1", "A", 2020, []string{"Action", "Sci-Fi"}, 8.0),
			b:    makeMedia("2", "B", 2020, []string{"Action", "Sci-Fi"}, 8.0),
			want: 0.7, // 至少 type(0.3) + tag(0.4) + rating(0.2) = 0.9
		},
		{
			name: "different type",
			a:    makeMedia("1", "A", 2020, []string{"Action"}, 8.0),
			b:    func() *media.Media {
				m := makeMedia("2", "B", 2020, []string{"Action"}, 8.0)
				m.Type = common.MediaTypeTVShow
				return m
			}(),
			want: 0.5, // 至少 tag(0.4) + rating(0.2)
		},
		{
			name: "completely different",
			a:    makeMedia("1", "A", 1990, []string{"Horror"}, 5.0),
			b:    makeMedia("2", "B", 2024, []string{"Comedy"}, 9.5),
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contentSimilarity(tt.a, tt.b)
			if tt.name == "identical" {
				if got != 1.0 {
					t.Errorf("identical similarity = %v, want 1.0", got)
				}
				return
			}
			if got < tt.want {
				t.Errorf("similarity = %v, want >= %v", got, tt.want)
			}
			if got > 1.0 {
				t.Errorf("similarity = %v > 1.0", got)
			}
		})
	}
}

func TestJaccard(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want float64
	}{
		{
			name: "both empty",
			a:    []string{},
			b:    []string{},
			want: 0,
		},
		{
			name: "identical",
			a:    []string{"a", "b", "c"},
			b:    []string{"a", "b", "c"},
			want: 1.0,
		},
		{
			name: "no overlap",
			a:    []string{"a", "b"},
			b:    []string{"c", "d"},
			want: 0,
		},
		{
			name: "partial overlap",
			a:    []string{"a", "b", "c"},
			b:    []string{"b", "c", "d"},
			want: 0.5, // intersection=2, union=4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jaccard(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("jaccard = %v, want %v", got, tt.want)
			}
		})
	}
}
