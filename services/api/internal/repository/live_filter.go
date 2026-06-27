package repository

// LiveListFilter 直播间列表筛选
type LiveListFilter struct {
	Status      string
	RoomType    string
	GroupTitle  string
	PlaylistURL string
	Search      string
	Limit       int
	Offset      int
}

// LiveGroupStat 分组统计
type LiveGroupStat struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// LivePlaylistStat M3U 来源统计
type LivePlaylistStat struct {
	URL   string `json:"url"`
	Count int64  `json:"count"`
}
