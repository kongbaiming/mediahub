package scanner

import "testing"

func TestParseFileName_Movie(t *testing.T) {
	tests := []struct {
		input   string
		wantType string
		wantTitle string
		wantYear *int
		wantRes  string
		wantVC   string
	}{
		{
			input: "Inception.2010.1080p.BluRay.x264-GROUP.mkv",
			wantType: "movie",
			wantTitle: "Inception",
			wantYear: ptr(2010),
			wantRes: "1080p",
			wantVC: "X264",
		},
		{
			input: "The Matrix (1999) 2160p UHD BluRay HEVC-GROUP.mkv",
			wantType: "movie",
			wantTitle: "The Matrix",
			wantYear: ptr(1999),
			wantRes: "2160p",
			wantVC: "HEVC",
		},
		{
			input: "Avatar.2009.BluRay.mkv",
			wantType: "movie",
			wantTitle: "Avatar",
			wantYear: ptr(2009),
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseFileName(tt.input)
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
			if !equalPtr(got.Year, tt.wantYear) {
				t.Errorf("Year = %v, want %v", ptrVal(got.Year), ptrVal(tt.wantYear))
			}
			if tt.wantRes != "" && got.Resolution != tt.wantRes {
				t.Errorf("Resolution = %q, want %q", got.Resolution, tt.wantRes)
			}
			if tt.wantVC != "" && got.VideoCodec != tt.wantVC {
				t.Errorf("VideoCodec = %q, want %q", got.VideoCodec, tt.wantVC)
			}
		})
	}
}

func TestParseFileName_Episode(t *testing.T) {
	tests := []struct {
		input       string
		wantType    string
		wantTitle   string
		wantSeason  *int
		wantEpisode *int
	}{
		{
			input: "Breaking.Bad.S05E14.1080p.BluRay.x264.mkv",
			wantType: "episode",
			wantTitle: "Breaking Bad",
			wantSeason: ptr(5),
			wantEpisode: ptr(14),
		},
		{
			input: "Game.of.Thrones - S08E06 - The Iron Throne.mkv",
			wantType: "episode",
			wantTitle: "Game of Thrones",
			wantSeason: ptr(8),
			wantEpisode: ptr(6),
		},
		{
			input: "Show.S01E01.WEB-DL.mkv",
			wantType: "episode",
			wantTitle: "Show",
			wantSeason: ptr(1),
			wantEpisode: ptr(1),
		},
		{
			input: "大明王朝1566.2007.EP46.HD1080P.X264.AAC.Mandarin.CHS.BDE4.mp4",
			wantType: "episode",
			wantTitle: "大明王朝1566 2007",
			wantSeason: ptr(1),
			wantEpisode: ptr(46),
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseFileName(tt.input)
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
			if !equalPtr(got.Season, tt.wantSeason) {
				t.Errorf("Season = %v, want %v", ptrVal(got.Season), ptrVal(tt.wantSeason))
			}
			if !equalPtr(got.Episode, tt.wantEpisode) {
				t.Errorf("Episode = %v, want %v", ptrVal(got.Episode), ptrVal(tt.wantEpisode))
			}
		})
	}
}

func TestParseFilePath_EP46(t *testing.T) {
	path := `/media/[哔嘀影视-bde4.com]大明王朝1566.2007.EP01-46.HD1080P.X264.AAC.Mandarin.CHS/大明王朝1566.2007.EP46.HD1080P.X264.AAC.Mandarin.CHS.BDE4.mp4`
	got := ParseFilePath(path)
	if got.Type != "episode" {
		t.Fatalf("Type = %q, want episode", got.Type)
	}
	if !equalPtr(got.Season, ptr(1)) {
		t.Errorf("Season = %v, want 1", ptrVal(got.Season))
	}
	if !equalPtr(got.Episode, ptr(46)) {
		t.Errorf("Episode = %v, want 46", ptrVal(got.Episode))
	}
	if !IsEpisodeFile(got, "tvshow") {
		t.Error("IsEpisodeFile should be true for tvshow")
	}
}

func TestIsMediaFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/volume1/media/Inception.mkv", true},
		{"/volume1/media/Show.mp4", true},
		{"/volume1/media/old.avi", true},
		{"/volume1/media/legacy.rmvb", true},
		{"/volume1/media/legacy.rm", true},
		{"/volume1/media/high.ts", true},
		{"/volume1/media/4k.m2ts", true},
		{"/volume1/media/notes.txt", false},
		{"/volume1/media/subtitle.srt", false},
		{"/volume1/media/.DS_Store", false},
		{"movie.mp4", true},
		{"movie.xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsMediaFile(tt.path); got != tt.want {
				t.Errorf("IsMediaFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestInferMediaType(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pType   string
		want    string
	}{
		{
			name:  "movie in movies dir",
			path:  "/volume1/media/movies/Inception.mkv",
			pType: "movie",
			want:  "movie",
		},
		{
			name:  "episode in tvshows dir → tvshow",
			path:  "/volume1/media/tvshows/Breaking Bad/S05E14.mkv",
			pType: "episode",
			want:  "tvshow",
		},
		{
			name:  "movie in anime dir → anime",
			path:  "/volume1/media/anime/Spirited Away.mkv",
			pType: "movie",
			want:  "anime",
		},
		{
			name:  "movie in documentaries dir → documentary",
			path:  "/volume1/media/documentaries/Planet Earth.mkv",
			pType: "movie",
			want:  "documentary",
		},
		{
			name:  "chinese anime dir",
			path:  "/volume1/media/动画/Spirited Away.mkv",
			pType: "movie",
			want:  "anime",
		},
		{
			name:  "unknown dir fallback to movie",
			path:  "/volume1/media/未知/movie.mkv",
			pType: "movie",
			want:  "movie",
		},
		{
			name:  "episode in unknown dir → tvshow",
			path:  "/volume1/media/未知/S01E01.mkv",
			pType: "episode",
			want:  "tvshow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ParsedFile{Type: tt.pType}
			got := InferMediaType(p, tt.path)
			if got != tt.want {
				t.Errorf("InferMediaType(%q, %q) = %q, want %q", tt.path, tt.pType, got, tt.want)
			}
		})
	}
}

// ---- helpers ----

func ptr[T any](v T) *T { return &v }

func ptrVal(p *int) int {
	if p == nil {
		return -1
	}
	return *p
}

func equalPtr(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
