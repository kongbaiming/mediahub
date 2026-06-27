package service

import "testing"

func TestValidateIPTVSourceURL_IPv6Literal(t *testing.T) {
	raw := "http://[2409:8087:8:21::18]:6610/otttv.bj.chinamobile.com/PLTV/88888888/224/3221226895/1.m3u8?"
	got, err := validateIPTVSourceURL(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != raw {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeM3UPlaylistURL_Zhi35(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://live.zhi35.com/", "https://live.zhi35.com/iptv.m3u"},
		{"https://live.zhi35.com", "https://live.zhi35.com/iptv.m3u"},
		{"https://live.zhi35.com/iptv.m3u", "https://live.zhi35.com/iptv.m3u"},
		{"https://example.com/list.m3u8", "https://example.com/list.m3u8"},
	}
	for _, c := range cases {
		got, err := validateM3UPlaylistURL(c.in)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("%q => %q, want %q", c.in, got, c.want)
		}
	}
}
