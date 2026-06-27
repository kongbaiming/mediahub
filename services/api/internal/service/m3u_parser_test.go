package service

import "testing"

func TestParseM3U(t *testing.T) {
	content := `#EXTM3U
#EXTINF:-1 tvg-id="CCTV-8" tvg-logo="https://example.com/cctv8.png" group-title="央视台",CCTV-8 电视剧
http://183.196.25.171:808/hls/77/index.m3u8
#EXTINF:-1 group-title="卫视台" tvg-logo="https://example.com/zj.png",浙江卫视
http://zwebl02.cztv.com/live/channel.m3u8
`
	entries, err := ParseM3U(content, "https://example.com/list.m3u8")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Title != "CCTV-8 电视剧" || entries[0].GroupTitle != "央视台" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
	if entries[0].Logo != "https://example.com/cctv8.png" {
		t.Fatalf("unexpected logo: %s", entries[0].Logo)
	}
}

func TestIsM3UPlaylist(t *testing.T) {
	if !IsM3UPlaylist("#EXTM3U\n#EXTINF:-1,Test\nhttp://a/b.m3u8\n") {
		t.Fatal("expected playlist")
	}
	if IsM3UPlaylist("#EXTM3U\nhttp://a/b.m3u8\n") {
		t.Fatal("expected not playlist without EXTINF")
	}
}
