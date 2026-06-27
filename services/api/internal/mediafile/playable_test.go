package mediafile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsPlayable(t *testing.T) {
	dir := t.TempDir()

	empty := filepath.Join(dir, "empty.mkv")
	if err := os.WriteFile(empty, nil, 0644); err != nil {
		t.Fatal(err)
	}
	ok, reason := IsPlayable(empty)
	if ok || reason == "" {
		t.Fatalf("expected invalid empty file, got ok=%v reason=%q", ok, reason)
	}

	valid := filepath.Join(dir, "ok.mkv")
	header := make([]byte, minPlayableBytes)
	header[0], header[1], header[2], header[3] = 0x1A, 0x45, 0xDF, 0xA3
	if err := os.WriteFile(valid, header, 0644); err != nil {
		t.Fatal(err)
	}
	ok, reason = IsPlayable(valid)
	if !ok || reason != "" {
		t.Fatalf("expected valid mkv header, got ok=%v reason=%q", ok, reason)
	}

	rmvb := filepath.Join(dir, "ok.rmvb")
	rmHeader := make([]byte, minPlayableBytes)
	copy(rmHeader, []byte(".RMF"))
	if err := os.WriteFile(rmvb, rmHeader, 0644); err != nil {
		t.Fatal(err)
	}
	ok, reason = IsPlayable(rmvb)
	if !ok || reason != "" {
		t.Fatalf("expected valid rmvb header, got ok=%v reason=%q", ok, reason)
	}
}

func TestShouldSkipScan(t *testing.T) {
	if !ShouldSkipScan("/downloads/.hidden.mkv") {
		t.Fatal("hidden file should skip")
	}
	if !ShouldSkipScan("/downloads/movie/file.mkv.part") {
		t.Fatal(".part should skip")
	}
	if ShouldSkipScan("/downloads/movie/file.mkv") {
		t.Fatal("normal mkv should not skip")
	}
}
