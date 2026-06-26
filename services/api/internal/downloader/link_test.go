package downloader

import (
	"strings"
	"testing"
)

func TestNormalizeDownloadURL_thunder(t *testing.T) {
	raw := "thunder://QUFtYWduZXQ6P3h0PXVybjpidGloOkE4NUQxQkEzQTYxRTQwODY0NTk0NTlBMzBFRjhDMUVBOTE2QzUyMUUmZG49bGzGvc7e1b3Kwi5BbGwuUXVpZXQuaW4uUGVraW5nLjIwMTQueDI2NS5IRDRLLrn60+/W0NfWLrjfwutaWg=="
	got, err := NormalizeDownloadURL(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, "magnet:?") {
		t.Fatalf("expected magnet link, got %q", got)
	}
	if !strings.Contains(got, "urn:btih:A85D1BA3A61E4086459459A30EF8C1EA916C521E") {
		t.Fatalf("missing info hash in %q", got)
	}
}

func TestNormalizeDownloadURL_magnetPassthrough(t *testing.T) {
	raw := "magnet:?xt=urn:btih:abc"
	got, err := NormalizeDownloadURL(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != raw {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeDownloadURL_httpPassthrough(t *testing.T) {
	raw := "https://example.com/a.torrent"
	got, err := NormalizeDownloadURL(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != raw {
		t.Fatalf("got %q", got)
	}
}
