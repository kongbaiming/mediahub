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
    source: { type: 'trending', params: { limit: 20 } },
    ...partial,
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

export const LAYOUT_PRESETS = [IMMERSIVE_PRESET, RANKING_TOPIC_PRESET]
