package oshash

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeFile_tooSmall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.mp4")
	if err := os.WriteFile(path, make([]byte, 1000), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ComputeFile(path); err == nil {
		t.Fatal("expected error for small file")
	}
}

func TestComputeReader_knownSize(t *testing.T) {
	data := make([]byte, MinFileSize)
	for i := range data {
		data[i] = byte(i % 251)
	}
	r := sectionReaderAt{data: data}
	got, err := ComputeReader(r, int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if got.Hash == "" || len(got.Hash) != 16 {
		t.Fatalf("hash = %q", got.Hash)
	}
	if got.FileSize != int64(len(data)) {
		t.Fatalf("size = %d", got.FileSize)
	}
	// 确定性：同内容同 hash
	got2, err := ComputeReader(r, int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if got.Hash != got2.Hash {
		t.Fatalf("hash not stable: %s vs %s", got.Hash, got2.Hash)
	}
}

type sectionReaderAt struct {
	data []byte
}

func (s sectionReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(s.data)) {
		return 0, os.ErrInvalid
	}
	return copy(p, s.data[off:]), nil
}
