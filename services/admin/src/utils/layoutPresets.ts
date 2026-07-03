import type { LayoutRow } from '@/api/layout'

export const LAYOUT_SCHEMAS = [
  { value: 'standard', label: '标准', hint: '通用顶栏 + 内容区' },
  { value: 'web-v2', label: 'Web 信息流', hint: 'Hero + 多区块 Feed' },
  { value: 'immersive', label: '沉浸式', hint: '全宽背景、大图专题、影院感' },
] as const

export type LayoutSchema = (typeof LAYOUT_SCHEMAS)[number]['value']

export interface LayoutPreset {
  name: string
  description: string
  theme: string
  global: Record<string, unknown>
  rows: Omit<LayoutRow, '_inherited'>[]
}

function row(
  partial: Omit<LayoutRow, '_inherited'> & { source?: LayoutRow['source'] },
): Omit<LayoutRow, '_inherited'> {
  return {
    visible: true,
    card_style: 'poster',
    ...partial,
    source: partial.source ?? { type: 'trending', params: { limit: 20 } },
  }
}

/** 沉浸式模版：全屏 Hero + 榜单 + 沉浸式专题 + 继续观看 */
export const IMMERSIVE_PRESET: LayoutPreset = {
  name: '沉浸式模版',
  description: '全宽 Hero、榜单排行、沉浸式专题头图，适合影院级首页',
  theme: 'dark',
  global: { layout_schema: 'immersive' },
  rows: [
    row({
      id: 'hero-immersive',
      type: 'hero-banner',
      title: '精选',
      card_style: 'banner',
      source: { type: 'recommend-algorithm', params: { algo: 'hot', limit: 6 } },
    }),
    row({
      id: 'ranking-hot',
      type: 'ranking',
      title: '热门榜单',
      subtitle: '按评分排序',
      card_style: 'landscape',
      source: { type: 'trending', params: { limit: 10, sort: 'rating' } },
      config: { show_rank: true },
    }),
    row({
      id: 'topic-featured',
      type: 'topic',
      title: '本周专题',
      subtitle: '绑定专题专辑后展示沉浸式头图',
      card_style: 'landscape',
      source: { type: 'album', params: { album_id: '', limit: 12 } },
      config: { display: 'immersive' },
    }),
    row({
      id: 'continue-immersive',
      type: 'shelf',
      title: '继续观看',
      card_style: 'landscape',
      source: { type: 'continue-watching', params: { limit: 12 } },
    }),
  ],
}

/** 榜单 + 专题模版：多榜单 + 专题行 */
export const RANKING_TOPIC_PRESET: LayoutPreset = {
  name: '榜单专题模版',
  description: '电影榜、剧集榜 + 专题专辑，适合运营活动页',
  theme: 'dark',
  global: { layout_schema: 'web-v2' },
  rows: [
    row({
      id: 'ranking-movies',
      type: 'ranking',
      title: '电影榜',
      subtitle: '高分电影 TOP',
      card_style: 'poster',
      source: { type: 'library', params: { type: 'movie', sort: 'rating', limit: 10 } },
      config: { show_rank: true },
    }),
    row({
      id: 'ranking-tv',
      type: 'ranking',
      title: '剧集榜',
      subtitle: '口碑剧集 TOP',
      card_style: 'poster',
      source: { type: 'library', params: { type: 'tvshow', sort: 'rating', limit: 10 } },
      config: { show_rank: true },
    }),
    row({
      id: 'topic-album',
      type: 'topic',
      title: '专题推荐',
      subtitle: '选择专题专辑作为数据源',
      card_style: 'landscape',
      source: { type: 'album', params: { album_id: '', limit: 16 } },
      config: { display: 'default' },
    }),
    row({
      id: 'recent-ranking',
      type: 'ranking',
      title: '新片榜',
      subtitle: '最近入库',
      card_style: 'landscape',
      source: { type: 'recently-added', params: { limit: 10 } },
      config: { show_rank: true },
    }),
  ],
}

/** 追剧模式：继续观看 + 猜你喜欢 + 高分剧集 + 最近添加 */
export const BINGE_PRESET: LayoutPreset = {
  name: '追剧模式',
  description: '继续观看、猜你喜欢、高分剧集、最近添加，适合追剧用户',
  theme: 'dark',
  global: { layout_schema: 'web-v2' },
  rows: [
    row({
      id: 'continue-binge',
      type: 'shelf',
      title: '继续观看',
      card_style: 'landscape',
      source: { type: 'continue-watching', params: { limit: 12 } },
    }),
    row({
      id: 'guess-binge',
      type: 'shelf',
      title: '猜你喜欢',
      card_style: 'poster',
      source: { type: 'guess-you-like', params: { limit: 20 } },
    }),
    row({
      id: 'ranking-tv-binge',
      type: 'ranking',
      title: '高分剧集',
      subtitle: '口碑剧集 TOP',
      card_style: 'poster',
      source: { type: 'library', params: { type: 'tvshow', sort: 'rating', min_rating: 7, limit: 10 } },
      config: { show_rank: true },
    }),
    row({
      id: 'recent-binge',
      type: 'shelf',
      title: '最近更新',
      card_style: 'landscape',
      source: { type: 'recently-added', params: { limit: 12 } },
    }),
  ],
}

/** 儿童专区：动画 + 合家欢 + 教育 */
export const KIDS_PRESET: LayoutPreset = {
  name: '儿童专区',
  description: '动画榜单、合家欢分类、教育纪录片，适合儿童 Profile',
  theme: 'dark',
  global: { layout_schema: 'web-v2' },
  rows: [
    row({
      id: 'hero-kids',
      type: 'hero-banner',
      title: '动画精选',
      card_style: 'banner',
      source: { type: 'library', params: { type: 'anime', sort: 'rating', limit: 6 } },
    }),
    row({
      id: 'ranking-anime',
      type: 'ranking',
      title: '动画榜',
      subtitle: '最受欢迎的动画',
      card_style: 'poster',
      source: { type: 'library', params: { type: 'anime', sort: 'rating', limit: 10 } },
      config: { show_rank: true },
    }),
    row({
      id: 'shelf-family',
      type: 'shelf',
      title: '合家欢',
      card_style: 'poster',
      source: { type: 'tag', params: { slug: 'family', limit: 16 } },
    }),
    row({
      id: 'shelf-documentary',
      type: 'shelf',
      title: '教育纪录片',
      card_style: 'landscape',
      source: { type: 'library', params: { type: 'documentary', sort: 'rating', limit: 12 } },
    }),
  ],
}

/** 电影之夜：热门 + 恐怖/动作 + 新片 */
export const MOVIE_NIGHT_PRESET: LayoutPreset = {
  name: '电影之夜',
  description: '热门电影榜单、类型专区、新片推荐，适合周末观影',
  theme: 'dark',
  global: { layout_schema: 'web-v2' },
  rows: [
    row({
      id: 'hero-movie',
      type: 'hero-banner',
      title: '今晚看什么',
      card_style: 'banner',
      source: { type: 'recommend-algorithm', params: { algo: 'hot', limit: 6 } },
    }),
    row({
      id: 'ranking-movie-hot',
      type: 'ranking',
      title: '热门电影',
      subtitle: '大家都在看',
      card_style: 'landscape',
      source: { type: 'library', params: { type: 'movie', sort: 'rating', limit: 10 } },
      config: { show_rank: true },
    }),
    row({
      id: 'shelf-action',
      type: 'shelf',
      title: '动作大片',
      card_style: 'poster',
      source: { type: 'tag', params: { slug: 'action', limit: 16 } },
    }),
    row({
      id: 'shelf-new-movie',
      type: 'shelf',
      title: '最新上线',
      card_style: 'landscape',
      source: { type: 'recently-added', params: { limit: 12 } },
    }),
  ],
}

/** 发现探索：猜你喜欢 + 冷门佳片 + 分类浏览 */
export const DISCOVER_PRESET: LayoutPreset = {
  name: '发现探索',
  description: '猜你喜欢、冷门佳片、分类浏览，适合探索新内容',
  theme: 'dark',
  global: { layout_schema: 'web-v2' },
  rows: [
    row({
      id: 'guess-discover',
      type: 'shelf',
      title: '猜你喜欢',
      card_style: 'poster',
      source: { type: 'guess-you-like', params: { limit: 20 } },
    }),
    row({
      id: 'shelf-hidden-gem',
      type: 'shelf',
      title: '冷门佳片',
      subtitle: '评分高但看的人少',
      card_style: 'poster',
      source: { type: 'library', params: { sort: 'rating', min_rating: 8, limit: 16 } },
    }),
    row({
      id: 'category-discover',
      type: 'category-grid',
      title: '分类浏览',
      card_style: 'poster',
      source: { type: 'library', params: { sort: 'rating', limit: 20 } },
    }),
    row({
      id: 'shelf-recent-discover',
      type: 'shelf',
      title: '最近入库',
      card_style: 'landscape',
      source: { type: 'recently-added', params: { limit: 12 } },
    }),
  ],
}

export const LAYOUT_PRESETS = [
  IMMERSIVE_PRESET,
  RANKING_TOPIC_PRESET,
  BINGE_PRESET,
  KIDS_PRESET,
  MOVIE_NIGHT_PRESET,
  DISCOVER_PRESET,
]
