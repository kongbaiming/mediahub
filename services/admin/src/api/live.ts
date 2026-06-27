import { http } from './client'

export type LiveRoomType = 'push' | 'iptv'

export interface LiveRoom {
  id: string
  title: string
  description?: string
  cover_url?: string
  room_type: LiveRoomType
  source_url?: string
  group_title?: string
  playlist_url?: string
  status: 'idle' | 'live' | 'ended'
  stream_key: string
  viewer_count: number
  started_at?: string
  ended_at?: string
  created_at: string
  updated_at: string
  rtmp_url?: string
  hls_url?: string
  play_url?: string
  stream_path?: string
}

export interface M3UGroupStat {
  name: string
  count: number
}

export interface M3UPreviewResult {
  playlist_url: string
  total: number
  groups: M3UGroupStat[]
}

export interface M3UImportResult {
  created: number
  skipped: number
  failed: number
  errors?: string[]
}

export interface LivePlaylistStat {
  url: string
  count: number
  sync_enabled: boolean
  interval_minutes: number
  last_sync_at?: string
  last_sync_status?: string
  last_sync_message?: string
}

export const SYNC_INTERVAL_OPTIONS = [
  { value: 60, label: '每 1 小时' },
  { value: 360, label: '每 6 小时' },
  { value: 720, label: '每 12 小时' },
  { value: 1440, label: '每 24 小时' },
  { value: 10080, label: '每 7 天' },
] as const

export const liveApi = {
  list: (params?: {
    status?: string
    room_type?: string
    group_title?: string
    search?: string
    page?: number
    page_size?: number
  }) =>
    http.get<{ data: LiveRoom[]; total: number; page: number; size: number }>(
      '/api/v1/live/rooms',
      { params },
    ),

  groups: () => http.get<{ data: M3UGroupStat[] }>('/api/v1/live/groups'),

  playlists: () =>
    http.get<{ data: LivePlaylistStat[] }>('/api/v1/live/rooms/m3u/playlists'),

  get: (id: string) => http.get<{ data: LiveRoom }>(`/api/v1/live/rooms/${id}`),

  create: (data: {
    title: string
    description?: string
    cover_url?: string
    room_type?: LiveRoomType
    source_url?: string
  }) => http.post<{ data: LiveRoom }>('/api/v1/live/rooms', data),

  previewM3U: (data: { playlist_url?: string; playlist_content?: string }) =>
    http.post<{ data: M3UPreviewResult }>('/api/v1/live/rooms/m3u/preview', data),

  importM3U: (data: {
    playlist_url?: string
    playlist_content?: string
    groups?: string[]
    replace?: boolean
    auto_sync?: boolean
    auto_sync_interval_minutes?: number
  }) => http.post<{ data: M3UImportResult }>('/api/v1/live/rooms/m3u/import', data),

  syncM3U: (playlist_url: string) =>
    http.post<{ data: M3UImportResult }>('/api/v1/live/rooms/m3u/sync', { playlist_url }),

  updateSyncConfig: (data: {
    playlist_url: string
    enabled: boolean
    interval_minutes: number
  }) => http.put<{ data: LivePlaylistStat }>('/api/v1/live/rooms/m3u/sync-config', data),

  update: (id: string, data: {
    title?: string
    description?: string
    cover_url?: string
    source_url?: string
  }) => http.patch<{ data: LiveRoom }>(`/api/v1/live/rooms/${id}`, data),

  delete: (id: string) => http.delete<{ status: string }>(`/api/v1/live/rooms/${id}`),

  stop: (id: string) => http.post<{ data: LiveRoom }>(`/api/v1/live/rooms/${id}/stop`),
}
