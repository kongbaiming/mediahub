package ailayout

// LibraryStats 媒资库统计（用于构造 prompt 上下文）
type LibraryStats struct {
	TotalMedia      int64
	MovieCount      int64
	TVShowCount     int64
	AnimeCount      int64
	DocumentaryCount int64
	Tags            []string
	Categories      []string
	Albums          []string
}

// GenerateRequest AI 布局生成请求
type GenerateRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

// GenerateResponse AI 布局生成响应
type GenerateResponse struct {
	Config      LayoutConfigJSON `json:"config"`
	Explanation string           `json:"explanation,omitempty"`
}

// LayoutConfigJSON 布局配置 JSON（与 domain/layout.LayoutConfig 结构一致）
type LayoutConfigJSON struct {
	Theme  string         `json:"theme,omitempty"`
	Rows   []RowJSON      `json:"rows"`
	Global map[string]any `json:"global,omitempty"`
}

// RowJSON 行配置
type RowJSON struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Title     string         `json:"title,omitempty"`
	Subtitle  string         `json:"subtitle,omitempty"`
	CardStyle string         `json:"card_style,omitempty"`
	Source    DataSourceJSON `json:"source,omitempty"`
	Visible   *bool          `json:"visible,omitempty"`
	Config    map[string]any `json:"config,omitempty"`
}

// DataSourceJSON 数据源配置
type DataSourceJSON struct {
	Type   string         `json:"type"`
	Params map[string]any `json:"params,omitempty"`
}
