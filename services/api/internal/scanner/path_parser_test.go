package scanner

import "testing"

func TestParseFilePath_NumericEpisodeInAlbum(t *testing.T) {
	got := ParseFilePath("/media/冰河世纪4/053.MP4")
	if got.Type != "episode" {
		t.Fatalf("Type = %q, want episode", got.Type)
	}
	if got.Title != "冰河世纪4" {
		t.Fatalf("Title = %q, want 冰河世纪4", got.Title)
	}
	if got.Episode == nil || *got.Episode != 53 {
		t.Fatalf("Episode = %v, want 53", got.Episode)
	}
	if got.Season == nil || *got.Season != 1 {
		t.Fatalf("Season = %v, want 1", got.Season)
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
