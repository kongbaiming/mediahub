package scanner

import "testing"

func TestPlaybackHintMKVHEVC4K(t *testing.T) {
	p := &ProbeResult{
		Format: ProbeFormat{FormatName: "matroska,webm"},
		Streams: []ProbeStream{
			{CodecType: "video", CodecName: "hevc", Width: 3840, Height: 2160},
			{CodecType: "audio", CodecName: "aac"},
		},
	}
	h := p.PlaybackHint("/media/show/S01E01.mkv")
	if h.DirectPlayable {
		t.Fatal("mkv hevc should not direct play in browser")
	}
	if !h.HLSCopyable {
		t.Fatal("hevc should be hls copyable")
	}
	if h.Recommended != PlaybackHLSCopy {
		t.Fatalf("expected hls_copy, got %s", h.Recommended)
	}
	if h.Resolution != "4K" {
		t.Fatalf("expected 4K resolution label, got %s", h.Resolution)
	}
}

func TestPlaybackHintMP4H264(t *testing.T) {
	p := &ProbeResult{
		Format: ProbeFormat{FormatName: "mov,mp4,m4a,3gp,3g2,mj2"},
		Streams: []ProbeStream{
			{CodecType: "video", CodecName: "h264", Width: 1920, Height: 1080},
		},
	}
	h := p.PlaybackHint("/media/movie.mp4")
	if !h.DirectPlayable || h.Recommended != PlaybackDirect {
		t.Fatalf("mp4 h264 should direct play, got %+v", h)
	}
}
