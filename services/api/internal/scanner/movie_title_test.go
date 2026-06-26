package scanner

import (
	"strings"
	"testing"
)

func TestMovieSearchCandidates_WeakFilenameUsesFolder(t *testing.T) {
	path := "/media/唐人街探案3【合集】/唐z人z街z探z案3 (2021) 4K 60FPS/D.C.3.2021.2160p.WEB-DL.60fps.H265.10bit.AAC.mp4"
	candidates := MovieSearchCandidates(path, "D C 3", nil)
	if len(candidates) < 2 {
		t.Fatalf("expected multiple candidates, got %v", candidates)
	}
	found := false
	for _, c := range candidates {
		if c == "唐人街探案3" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected folder-based title in %v", candidates)
	}
}

func TestMovieSearchCandidates_FrozenEnglish(t *testing.T) {
	path := "/media/冰雪奇缘2.3D出屏/Frozen.II.2019.3D.1080p.BluRay.x264.mkv"
	title := "3D冰雪奇缘2 3D出屏国配字幕 国粤台英4语 Frozen II"
	candidates := MovieSearchCandidates(path, title, nil)
	foundFrozen := false
	foundHan := false
	for _, c := range candidates {
		if strings.Contains(c, "Frozen") {
			foundFrozen = true
		}
		if c == "冰雪奇缘2" || c == "冰雪奇缘" {
			foundHan = true
		}
	}
	if !foundFrozen {
		t.Fatalf("expected Frozen II in %v", candidates)
	}
	if !foundHan {
		t.Fatalf("expected 冰雪奇缘2 in %v", candidates)
	}
}

func TestRefineMovieTitleFromPath_TitanicFolder(t *testing.T) {
	path := "/media/泰t尼克号 白星加长版/4K 120帧 泰坦尼克号白星加长版.mkv"
	p := ParseFilePath(path)
	if p.Title == "4K 120帧 泰坦尼克号白星加长版" {
		t.Fatalf("title should be refined from folder, got %q", p.Title)
	}
	if !containsHan(p.Title) {
		t.Fatalf("expected Chinese title, got %q", p.Title)
	}
}

func TestMovieSearchCandidates_IceAgeAlias(t *testing.T) {
	candidates := MovieSearchCandidates("/media/冰川时代4/053.MP4", "冰川时代4", nil)
	found := false
	for _, c := range candidates {
		if c == "冰河世纪4" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 冰河世纪4 alias in %v", candidates)
	}
}

func TestIsWeakMovieTitle(t *testing.T) {
	if !isWeakMovieTitle("D C 3") {
		t.Fatal("D C 3 should be weak")
	}
	if !isWeakMovieTitle("4K 60帧国语") {
		t.Fatal("4K 60帧国语 should be weak")
	}
	if isWeakMovieTitle("Kung Fu Panda 4") {
		t.Fatal("Kung Fu Panda 4 should not be weak")
	}
}

func TestTVSearchCandidates_SeriesAlbumPath(t *testing.T) {
	path := "/media/唐丨朝诡事录之西行(2024)"
	emb := &EmbeddedMeta{Title: "DAAI Mandarin", Show: "Mandarin News"}
	out := TVSearchCandidates(path, "唐朝诡事录之西行", emb, &SearchCandidateOpts{
		PreferFolderOverEmbedded: true,
		ReferenceTitle:           "唐朝诡事录之西行",
	})
	if len(out) == 0 {
		t.Fatal("expected candidates")
	}
	if out[0] != "唐朝诡事录之西行" {
		t.Fatalf("first = %q, want folder title first", out[0])
	}
	for _, c := range out {
		if c == "DAAI Mandarin" || c == "Mandarin News" {
			t.Fatalf("unreliable embedded should be filtered, got %q in %v", c, out)
		}
	}
}

func TestTVSearchCandidates_EpisodeFilePath(t *testing.T) {
	path := "/media/唐丨朝诡事录之西行(2024)/S02E01.4K.mkv"
	out := TVSearchCandidates(path, "唐朝诡事录之西行", nil, nil)
	if len(out) == 0 || out[0] != "唐朝诡事录之西行" {
		t.Fatalf("candidates = %v", out)
	}
}

func TestAlbumFolderName(t *testing.T) {
	if got := AlbumFolderName("/media/唐丨朝诡事录之西行(2024)"); got != "唐丨朝诡事录之西行(2024)" {
		t.Fatalf("album = %q", got)
	}
	if got := AlbumFolderName("/media/show/S02E01.mkv"); got != "show" {
		t.Fatalf("episode parent = %q", got)
	}
}
