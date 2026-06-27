// 布局 API
import { http } from './client'
import type { Layout, LayoutConfig } from './types'

export type { Layout, LayoutConfig } from './types'

export interface DynamicRules {
  hour_of_day?: { start: number; end: number }
  day_of_week?: number[]
  seasons?: number[]
  tags?: string[]
}

export interface FeedItem {
  media_id: string
  title: string
  year?: number
  poster_url?: string
  backdrop_url?: string
  rating: number
  type: string
}

export interface FeedRow {
  id: string
  type: string
  title?: string
  subtitle?: string
  card_style?: string
  items: FeedItem[]
}

export interface LayoutRow {
  id: string
  type: string
  title?: string
  subtitle?: string
  card_style?: string
  source: { type: string; params?: Record<string, any> }
  visible?: boolean
  config?: Record<string, any>
  _inherited?: boolean
}

export interface Publication {
  id: string
  layout_id: string
  version: number
  target_platform: 'web' | 'android-tv' | 'tvos'
  traffic_split?: Record<string, number>
  dynamic_rules?: DynamicRules
  enabled: boolean
  active_from?: string
  active_to?: string
  created_at: string
}

export const layoutApi = {
  list: (params: { is_template?: boolean; status?: string } = {}) => {
    const q: Record<string, any> = {}
    if (params.is_template !== undefined) q.is_template = params.is_template
    if (params.status) q.status = params.status
    return http.get<{ data: Layout[]; total: number }>('/api/v1/layouts', { params: q })
  },

  get: (id: string, params?: { editor?: boolean }) =>
    http.get<{ data: Layout }>(`/api/v1/layouts/${id}`, {
      params: params?.editor ? { editor: '1' } : undefined,
    }),

  preview: (id: string, platform = 'web') =>
    http.get<{ data: { rows: FeedRow[] } }>(`/api/v1/layouts/${id}/preview`, {
      params: { platform },
    }),

  /** 已发布平台的公开 Feed（预览失败时的回退） */
  getPublishedFeed: (platform = 'web') =>
    http.get<{ data: { rows: FeedRow[] } }>(`/api/v1/feed/${platform}`),

  create: (data: {
    name: string
    description?: string
    is_template?: boolean
    parent_id?: string
    config?: LayoutConfig
  }) => http.post<{ data: Layout }>('/api/v1/layouts', data),

  update: (id: string, data: Partial<Layout>) =>
    http.patch<{ data: Layout }>(`/api/v1/layouts/${id}`, data),

  delete: (id: string) => http.delete(`/api/v1/layouts/${id}`),

  publish: (id: string, data: {
    target_platform: 'web' | 'android-tv' | 'tvos'
    traffic_split?: Record<string, number>
    active_from?: string
    active_to?: string
    dynamic_rules?: DynamicRules
  }) => http.post<{ data: Layout }>(`/api/v1/layouts/${id}/publish`, data),

  listPublications: (id: string) =>
    http.get<{ data: Publication[]; total: number }>(`/api/v1/layouts/${id}/publications`),

  disablePublication: (pubId: string) =>
    http.delete(`/api/v1/layouts/publications/${pubId}`),
}
