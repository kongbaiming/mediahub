package downloader

import "testing"

func TestIsTorrentCompleted(t *testing.T) {
	tests := []struct {
		name string
		t    Torrent
		want bool
	}{
		{
			name: "progress 100 uploading",
			t:    Torrent{Progress: 1, State: "uploading"},
			want: true,
		},
		{
			name: "progress 100 still downloading",
			t:    Torrent{Progress: 1, State: "downloading"},
			want: false,
		},
		{
			name: "incomplete",
			t:    Torrent{Progress: 0.99, State: "downloading"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTorrentCompleted(tt.t); got != tt.want {
				t.Fatalf("isTorrentCompleted() = %v, want %v", got, tt.want)
			}
		})
	}
}
