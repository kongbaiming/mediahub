package handler

import (
	"path/filepath"
	"testing"
)

func TestResolveStreamPath_hostToContainer(t *testing.T) {
	aliases := BuildPathAliases("/media", "/downloads", "/volume1/media", "/volume1/downloads")
	got := resolveStreamPath("/volume1/media/movies/Foo/Foo.mkv", aliases)
	want := filepath.Join("/media", "movies", "Foo", "Foo.mkv")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveStreamPath_alreadyContainer(t *testing.T) {
	aliases := BuildPathAliases("/media", "/downloads", "/volume1/media", "/volume1/downloads")
	got := resolveStreamPath("/media/movies/Foo.mkv", aliases)
	want := filepath.Join("/media", "movies", "Foo.mkv")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveStreamPath_downloads(t *testing.T) {
	aliases := BuildPathAliases("/media", "/downloads", "/volume1/media", "/volume1/downloads")
	got := resolveStreamPath("/volume1/downloads/movie/Bar/Bar.mkv", aliases)
	want := filepath.Join("/downloads", "movie", "Bar", "Bar.mkv")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
