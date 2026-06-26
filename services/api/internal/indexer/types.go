package indexer

// Release 索引搜索结果
type Release struct {
	Title       string `json:"title"`
	MagnetURL   string `json:"magnetUrl"`
	DownloadURL string `json:"downloadUrl"`
	Size        int64  `json:"size"`
	Seeders     int    `json:"seeders"`
	Peers       int    `json:"peers"`
	Indexer     string `json:"indexer"`
	PublishDate string `json:"publishDate"`
	GUID        string `json:"guid"`
}

// Link 返回可用的下载链接（优先 magnet）
func (r Release) Link() string {
	if r.MagnetURL != "" {
		return r.MagnetURL
	}
	return r.DownloadURL
}
