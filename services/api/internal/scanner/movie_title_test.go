package scanner

import (
	"strings"
	"testing"
)

func TestMovieSearchCandidates_WeakFilenameUsesFolder(t *testing.T) {
	path := "/media/唐人街探案3【合集】/唐z人z街z探z案3 (2021) 4K 60FPS/D.C.3.2021.2160p.WEB-DL.60fps.H265.10bit.AAC.mp4"
	candidates := MovieSearchCandidates(path, "D C 3")
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
	candidates := MovieSearchCandidates(path, title)
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
