package ailayout

import "fmt"

// SystemPrompt 系统提示词模板
const SystemPrompt = `你是 MediaHub 布局生成器。根据用户的描述，生成首页布局配置。

## 可用行类型
- hero-banner: 全宽轮播，适合首页顶部展示，card_style 用 banner
- ranking: 榜单，带序号，适合 TOP N 排行，card_style 用 landscape 或 poster
- topic: 专题，可绑定专辑，支持沉浸式展示，card_style 用 landscape
- shelf: 通用内容架，横向滚动卡片，card_style 可用 poster/landscape/square
- category-grid: 分类网格，适合浏览发现
- text-banner: 文字公告栏（只需 title 和 subtitle，不需要数据源）
- divider: 分隔线（只需 title，不需要数据源）

## 可用数据源
- manual: 手动指定媒资 ID（params: {ids: [uuid,...]}）
- library: 按类型/标签/年份/评分筛选（params: {type, genre, year, min_rating, sort, limit}）
  - type: movie | tvshow | anime | documentary
  - sort: rating | year | title | created_at
- trending: 热门（params: {limit, sort: rating}）
- continue-watching: 续播（params: {limit}）
- recently-added: 最近入库（params: {limit}）
- recommend-algorithm: 推荐算法（params: {algo: hot|content|cf|hybrid, limit}）
- guess-you-like: 猜你喜欢（params: {limit}）
- album: 专辑（params: {album_id}）
- category: 分类（params: {slug}）
- tag: 标签（params: {slug}）

## card_style 说明
- poster: 竖版海报（2:3）
- landscape: 横版（16:9）
- square: 正方形（1:1）
- banner: 超宽横幅（21:9），仅 hero-banner 使用

## 当前媒资库
%s

## 输出格式
返回纯 JSON，不要包含任何 markdown 标记或解释文字。结构如下：
{
  "theme": "dark",
  "global": { "layout_schema": "web-v2" },
  "rows": [
    {
      "id": "row-1",
      "type": "hero-banner",
      "title": "热门推荐",
      "card_style": "banner",
      "source": { "type": "recommend-algorithm", "params": { "algo": "hot", "limit": 6 } },
      "config": {}
    }
  ]
}

## 规则
1. 每个 row 的 id 必须唯一，格式为 "row-1", "row-2" 等
2. text-banner 和 divider 不需要 source
3. hero-banner 通常放在第一行，card_style 必须是 banner
4. ranking 行建议 config 中设置 show_rank: true
5. topic 行如果绑定专辑，需要在 config 中设置 album_id
6. 根据媒资库的实际内容生成，不要引用不存在的标签或专辑
7. 如果媒资库中没有某个分类的内容，不要创建该分类的数据源
8. 行数量建议 4-8 行，不要太多`

// VisionPrompt 图片识别提示词
const VisionPrompt = `分析这个 UI 原型图/截图，识别其中的布局结构，生成对应的 MediaHub 首页布局配置。

识别规则：
- 顶部大面积轮播/横幅 → hero-banner (card_style: banner)
- 带序号的横向列表 → ranking (card_style: landscape 或 poster)
- 横向滚动卡片行 → shelf (card_style: poster 或 landscape)
- 网格状卡片 → category-grid 或 shelf
- 大图 + 叠加文字 → topic (immersive display)
- 纯文字区域 → text-banner
- 分隔线 → divider

每个识别出的区域需要映射到合适的数据源。如果无法确定数据源，使用 recommend-algorithm (hot) 作为默认。

## 当前媒资库
%s

## 输出格式
返回纯 JSON，结构如下：
{
  "theme": "dark",
  "global": { "layout_schema": "web-v2" },
  "rows": [...]
}`

// BuildLibraryContext 构造媒资库上下文
func BuildLibraryContext(stats *LibraryStats) string {
	ctx := fmt.Sprintf("- 总数: %d 部\n", stats.TotalMedia)
	ctx += fmt.Sprintf("- 电影: %d 部\n", stats.MovieCount)
	ctx += fmt.Sprintf("- 剧集: %d 部\n", stats.TVShowCount)
	ctx += fmt.Sprintf("- 动画: %d 部\n", stats.AnimeCount)
	ctx += fmt.Sprintf("- 纪录片: %d 部\n", stats.DocumentaryCount)

	if len(stats.Tags) > 0 {
		ctx += "- 可用标签: "
		for i, t := range stats.Tags {
			if i > 0 {
				ctx += ", "
			}
			ctx += t
		}
		ctx += "\n"
	}

	if len(stats.Categories) > 0 {
		ctx += "- 可用分类: "
		for i, c := range stats.Categories {
			if i > 0 {
				ctx += ", "
			}
			ctx += c
		}
		ctx += "\n"
	}

	if len(stats.Albums) > 0 {
		ctx += "- 可用专辑: "
		for i, a := range stats.Albums {
			if i > 0 {
				ctx += ", "
			}
			ctx += a
		}
		ctx += "\n"
	}

	return ctx
}
