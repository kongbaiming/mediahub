import axios from 'axios'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 30000,
})

// 自动注入 X-Profile-ID
http.interceptors.request.use((config) => {
  const profileId = localStorage.getItem('mediahub_profile_id')
  if (profileId && config.headers) {
    config.headers['X-Profile-ID'] = profileId
  }
  return config
})

http.interceptors.response.use(
  (res) => res.data,
  (err) => Promise.reject(err)
)

export interface FeedItem {
  media_id: string
  title: string
  year?: number
  poster_url?: string
  backdrop_url?: string
  rating: number
  type: string
  duration?: number
  overview?: string
  genres?: string[]
  progress?: number
}

export interface FeedRow {
  id: string
  type: string
  title?: string
  subtitle?: string
  card_style?: string
  items: FeedItem[]
}

export interface Feed {
  version: number
  platform: string
  updated_at: string
  rows: FeedRow[]
}

export const feedApi = {
  get: (platform = 'web') => http.get<Feed>(`/api/v1/feed/${platform}`),
}

export interface MediaDetail {
  id: string
  title: string
  original_title?: string
  year?: number
  type: string
  rating: number
  poster_url?: string
  backdrop_url?: string
  overview?: string
  runtime?: number
  genres: string[]
  has_subtitle: boolean
  video_codec?: string
  audio_codec?: string
  resolution?: string
  storage_path: string
}

export interface MediaSummary {
  id: string
  title: string
  year?: number
  type: string
  rating: number
  poster_url?: string
  backdrop_url?: string
  genres: string[]
}

export const mediaApi = {
  get: (id: string) => http.get<{ data: MediaDetail }>(`/api/v1/media/${id}`),
  list: (params: { q?: string; type?: string; page?: number; page_size?: number; sort?: string } = {}) =>
    http.get<{ items: MediaSummary[]; total: number; page: number; size: number }>('/api/v1/media', { params }),
}

export const historyApi = {
  record: (data: {
    media_id: string
    episode_id?: string
    progress: number
    duration: number
  }) => http.post('/api/v1/history', { profile_id: localStorage.getItem('mediahub_profile_id') || '', ...data, device: 'web' }),

  getResume: (mediaId: string) =>
    http.get<{ data: { progress: number; duration: number; completed: boolean; updated_at: string } | null }>(`/api/v1/resume/${mediaId}`),

  toggleFavorite: (data: { media_id: string; type?: string }) =>
    http.post('/api/v1/favorites', { profile_id: localStorage.getItem('mediahub_profile_id') || '', ...data }),

  getContinueWatching: (limit = 12) =>
    http.get<{ data: any[]; total: number }>('/api/v1/continue-watching', { params: { limit } }),
}

export const recommendApi = {
  hot: (limit = 20) =>
    http.get<{ data: MediaSummary[] }>('/api/v1/recommend/hot', { params: { limit } }),

  similar: (mediaId: string, limit = 12) =>
    http.get<{ data: MediaSummary[] }>(`/api/v1/recommend/similar/${mediaId}`, { params: { limit } }),

  forMe: (limit = 20) =>
    http.get<{ data: MediaSummary[] }>('/api/v1/recommend/for-me', { params: { limit } }),
}

export interface Profile {
  id: string
  user_id: string
  name: string
  avatar_url?: string
  is_kid: boolean
}

// Profile API（Web Player 暂不需要登录，但保留接口）
export const profileApi = {
  list: () => http.get<{ data: Profile[] }>('/api/v1/profiles'),
}
