package scanner

import (
	"strings"
	"testing"
)

func TestParseFilePath_NumericEpisodeInAlbum(t *testing.T) {
	got := ParseFilePath("/media/冰河世纪4/053.MP4")
	if got.Type != "movie" {
		t.Fatalf("Type = %q, want movie", got.Type)
	}
	if got.Title != "冰河世纪4" {
		t.Fatalf("Title = %q, want 冰河世纪4", got.Title)
	}
	if got.Episode != nil {
		t.Fatalf("Episode = %v, want nil", got.Episode)
	}
}

func TestParseFilePath_SxxExxUsesFolderWhenTitleWeak(t *testing.T) {
	got := ParseFilePath("/media/tvshows/庆余年/S01E05.mkv")
	if got.Type != "episode" {
		t.Fatalf("Type = %q, want episode", got.Type)
	}
	if got.Title != "庆余年" {
		t.Fatalf("Title = %q, want 庆余年", got.Title)
	}
}

func TestParseFilePath_MovieInFolderUnchanged(t *testing.T) {
	got := ParseFilePath("/media/功夫熊猫四/Kung Fu Panda 4.mp4")
	if got.Type != "movie" {
		t.Fatalf("Type = %q, want movie", got.Type)
	}
	if got.Title != "Kung Fu Panda 4" {
		t.Fatalf("Title = %q", got.Title)
	}
}

func TestParseFilePath_SeasonSubfolder(t *testing.T) {
	got := ParseFilePath("/media/权力的游戏/Season 1/01.mkv")
	if got.Title != "权力的游戏" {
		t.Fatalf("Title = %q, want 权力的游戏", got.Title)
	}
	if got.Season == nil || *got.Season != 1 {
		t.Fatalf("Season = %v, want 1", got.Season)
	}
	if got.Episode == nil || *got.Episode != 1 {
		t.Fatalf("Episode = %v, want 1", got.Episode)
	}
}

func TestParseFilePath_NumericEpisodeUnderMovieCategoryDir(t *testing.T) {
	// NAS 挂载 /volume1/data -> /media 时，电影在 /media/电影/片名/053.mp4
	got := ParseFilePath("/media/电影/冰河世纪4/053.MP4")
	if got.Type != "movie" {
		t.Fatalf("Type = %q, want movie", got.Type)
	}
	if got.Title != "冰河世纪4" {
		t.Fatalf("Title = %q, want 冰河世纪4", got.Title)
	}
}

func TestParseFilePath_NumericEpisodeInTVAlbum(t *testing.T) {
	got := ParseFilePath("/media/tvshows/庆余年/01.mkv")
	if got.Type != "episode" {
		t.Fatalf("Type = %q, want episode", got.Type)
	}
	if got.Title != "庆余年" {
		t.Fatalf("Title = %q, want 庆余年", got.Title)
	}
}

func TestParseFilePath_DirectFileInCategoryDirIsMovie(t *testing.T) {
	got := ParseFilePath("/media/电影/Kung Fu Panda 4.mp4")
	if got.Type != "movie" {
		t.Fatalf("Type = %q, want movie", got.Type)
	}
}

func TestParseFilePath_S02E01WithQualitySuffix(t *testing.T) {
	got := ParseFilePath("/media/唐丨朝诡事录之西行(2024)/S02E01.4K.mkv")
	if got.Type != "episode" {
		t.Fatalf("Type = %q, want episode", got.Type)
	}
	if got.Title != "唐朝诡事录之西行" {
		t.Fatalf("Title = %q, want 唐朝诡事录之西行", got.Title)
	}
	if got.Season == nil || *got.Season != 2 {
		t.Fatalf("Season = %v, want 2", got.Season)
	}
	if got.Episode == nil || *got.Episode != 1 {
		t.Fatalf("Episode = %v, want 1", got.Episode)
	}
}

func TestParseFilePath_EpisodeMarkerInName(t *testing.T) {
	got := ParseFilePath("/media/葫芦小金刚[4KHDR.CN]/葫芦小金刚.Calabash.Brothers.II.1991.E04.2160P.H265.AAC.mkv")
	if got.Type != "episode" {
		t.Fatalf("Type = %q, want episode", got.Type)
	}
	if got.Episode == nil || *got.Episode != 4 {
		t.Fatalf("Episode = %v, want 4", got.Episode)
	}
}

func TestParseFilePath_BracketDotEpisode(t *testing.T) {
	got := ParseFilePath("/media/武林外传全集/[武林外传].01.郭女侠怒砸同福店.佟掌柜妙点迷路人.rmvb")
	if got.Type != "episode" {
		t.Fatalf("Type = %q, want episode", got.Type)
	}
	if got.Episode == nil || *got.Episode != 1 {
		t.Fatalf("Episode = %v, want 1", got.Episode)
	}
	if got.Title != "武林外传全集" {
		t.Fatalf("Title = %q, want 武林外传全集", got.Title)
	}
	if !strings.Contains(got.OriginalName, "郭女侠") {
		t.Fatalf("OriginalName = %q, want episode subtitle", got.OriginalName)
	}
	if !IsEpisodeFile(got, "tvshow") {
		t.Fatal("expected tvshow episode file")
	}
}

func TestParseFilePath_DetectiveChinatownUsesFolderTitle(t *testing.T) {
	got := ParseFilePath("/media/唐人街探案3【合集】/唐人街探案2网剧/Detective.Chinatown.S02E01.2020.2160p.WEB-DL.mkv")
	if got.Type != "episode" {
		t.Fatalf("Type = %q, want episode", got.Type)
	}
	if got.Title != "唐人街探案2网剧" {
		t.Fatalf("Title = %q, want 唐人街探案2网剧", got.Title)
	}
}

func TestIsMediaFile_EmbySuffix(t *testing.T) {
	if !IsMediaFile("/media/陈佩斯朱时茂小品集/吃面.mp4 5678") {
		t.Fatal("expected emby-style mp4 basename to match")
	}
}

func TestParseFilePath_SketchCollection(t *testing.T) {
	got := ParseFilePath("/media/陈佩斯朱时茂小品集[4KHDR.CN]/吃面.mp4 5678")
	if got.Type != "episode" {
		t.Fatalf("Type = %q, want episode", got.Type)
	}
	if !strings.Contains(got.Title, "小品集") {
		t.Fatalf("Title = %q, want album name containing 小品集", got.Title)
	}
	if got.Episode != nil {
		t.Fatalf("Episode should be nil for collection items, got %v", *got.Episode)
	}
}

func TestCollectionEpisodeTitle(t *testing.T) {
	if got := collectionEpisodeTitle("/media/陈佩斯朱时茂小品集/吃面.mp4 5678"); got != "吃面" {
		t.Fatalf("title = %q, want 吃面", got)
	}
}

func TestSeriesFolderTitle(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"葫芦小金刚[4KHDR.CN]Calabash.Brothers.II.1991.WEB-DL.2160P.H265.AAC", "葫芦小金刚"},
		{"陈佩斯朱时茂小品集[4KHDR.CN]4K修复版[收藏版]", "陈佩斯朱时茂小品集"},
		{"T唐朝诡事录0509", "唐朝诡事录"},
		{"冰河世纪4", "冰河世纪4"},
		{"唐丨朝诡事录之西行(2024).", "唐朝诡事录之西行"},
	}
	for _, tt := range tests {
		if got := SeriesFolderTitle(tt.in); got != tt.want {
			t.Errorf("SeriesFolderTitle(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSeriesFolderYear(t *testing.T) {
	y := SeriesFolderYear("唐丨朝诡事录之西行(2024).")
	if y == nil || *y != 2024 {
		t.Fatalf("year = %v, want 2024", y)
	}
}
