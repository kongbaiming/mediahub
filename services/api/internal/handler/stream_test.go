package handler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsHLSPlaylistComplete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "playlist.m3u8")

	partial := `#EXTM3U
#EXT-X-VERSION:3
#EXTINF:7.960000,
seg_000.ts
`
	if err := os.WriteFile(path, []byte(partial), 0644); err != nil {
		t.Fatal(err)
	}
	if isHLSPlaylistComplete(path) {
		t.Fatal("partial playlist should not be complete")
	}
	if !hlsPlaylistHasSegments(path) {
		t.Fatal("partial playlist should have segments")
	}

	complete := partial + "#EXT-X-ENDLIST\n"
	if err := os.WriteFile(path, []byte(complete), 0644); err != nil {
		t.Fatal(err)
	}
	if !isHLSPlaylistComplete(path) {
		t.Fatal("expected complete playlist")
	}
}
