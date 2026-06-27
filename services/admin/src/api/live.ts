import { http } from './client'

export type LiveRoomType = 'push' | 'iptv'

export interface LiveRoom {
  id: string
  title: string
  description?: string
  cover_url?: string
  room_type: LiveRoomType
  source_url?: string
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

export const liveApi = {
  list: (params?: { status?: string; page?: number; page_size?: number }) =>
    http.get<{ data: LiveRoom[]; total: number; page: number; size: number }>(
      '/api/v1/live/rooms',
      { params },
    ),

  get: (id: string) => http.get<{ data: LiveRoom }>(`/api/v1/live/rooms/${id}`),

  create: (data: {
    title: string
    description?: string
    cover_url?: string
    room_type?: LiveRoomType
    source_url?: string
  }) => http.post<{ data: LiveRoom }>('/api/v1/live/rooms', data),

  update: (id: string, data: {
    title?: string
    description?: string
    cover_url?: string
    source_url?: string
  }) => http.patch<{ data: LiveRoom }>(`/api/v1/live/rooms/${id}`, data),

  delete: (id: string) => http.delete<{ status: string }>(`/api/v1/live/rooms/${id}`),

  stop: (id: string) => http.post<{ data: LiveRoom }>(`/api/v1/live/rooms/${id}/stop`),
}
