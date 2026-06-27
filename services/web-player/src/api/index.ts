import axios, { type AxiosInstance, type InternalAxiosRequestConfig } from 'axios'

/**
 * 后端响应格式（统一约定）：
 *  - 详情 / Feed / Profile：{ data: T }
 *  - 列表：{ items: T[]; total: number; page: number; size: number }
 *  - 续播：{ data: T | null }
 *  - 简单操作：{ status: string } / { status: string; ...payload }
 *
 * axios 拦截器解出 res.data（去掉 axios 外壳）。
 * api 方法用类型断言把响应转为业务类型 T。
 */

const http: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 30000,
})

http.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const profileId = localStorage.getItem('mediahub_profile_id')
  if (profileId && config.headers) {
    config.headers['X-Profile-ID'] = profileId
  }
  return config
})

// 拦截器解出 res.data — 运行时拿到的是后端的整个 response body
http.interceptors.response.use(
  (res) => res.data,
  (err) => Promise.reject(err),
)

// ─── 类型定义 ───

export interface FeedItem {
  media_id?: string
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
  tmdb_id?: number
  external?: boolean
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

export interface EpisodeDetail {
  id: string
  episode_number: number
  title?: string
  file_path?: string
  duration?: number
}

export interface SeasonDetail {
  id: string
  season_number: number
  title?: string
  episodes?: EpisodeDetail[]
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
  has_subtitle?: boolean
  video_codec?: string
  audio_codec?: string
  resolution?: string
  storage_path?: string
  seasons?: SeasonDetail[]
  external?: boolean
  tmdb_id?: number
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

export interface ResumeInfo {
  progress: number
  duration: number
  completed: boolean
  updated_at: string
  episode_id?: string
}

export interface Profile {
  id: string
  user_id: string
  name: string
  avatar_url?: string
  is_kid: boolean
}

export interface MediaCredit {
  id: string
  role: string
  character_name?: string
  billing_order?: number
  person?: PersonBrief
}

export interface PersonBrief {
  id: string
  name: string
  profile_path?: string
  profile_url?: string
  biography?: string
  place_of_birth?: string
  known_for_department?: string
  birthday?: string
  tmdb_person_id?: number
}

export interface Person extends PersonBrief {
  original_name?: string
}

export interface PersonWork extends MediaSummary {
  external?: boolean
  tmdb_id?: number
}

export interface TMDBMediaDetail {
  external: boolean
  tmdb_id: number
  local_media_id?: string
  type: string
  title: string
  original_title?: string
  year?: number
  overview?: string
  poster_url?: string
  backdrop_url?: string
  rating: number
  runtime?: number
  genres: string[]
  credits?: MediaCredit[]
}

export interface ContentRating {
  country: string
  system: string
  rating: string
}

export interface EpisodeNext {
  id: string
  episode_number: number
  title?: string
  file_path?: string
  season_id?: string
}

export interface MediaExtra {
  id: string
  extra_type: string
  title?: string
  external_url?: string
  source?: string
}

// ─── API 方法 ───
// 每个方法显式标注返回类型，调用方直接拿到业务类型，不需要 .data 包装

export const feedApi = {
  async get(platform = 'web'): Promise<Feed> {
    const body = (await http.get<unknown>(`/api/v1/feed/${platform}`)) as
      | { data: Feed }
      | Feed
    if (body && typeof body === 'object' && 'data' in body && body.data) {
      return body.data
    }
    return body as Feed
  },
}

export const mediaApi = {
  async get(id: string): Promise<MediaDetail> {
    const body = (await http.get<unknown>(`/api/v1/media/${id}`)) as { data: MediaDetail }
    return body.data
  },

  async getTmdb(type: string, tmdbId: number): Promise<TMDBMediaDetail> {
    const body = (await http.get<unknown>(`/api/v1/media/tmdb/${type}/${tmdbId}`)) as {
      data: TMDBMediaDetail
    }
    return body.data
  },

  async tmdbSimilar(type: string, tmdbId: number, limit = 12): Promise<PersonWork[]> {
    const body = (await http.get<unknown>(`/api/v1/media/tmdb/${type}/${tmdbId}/similar`, {
      params: { limit },
    })) as { data: PersonWork[] }
    return body.data || []
  },

  async list(
    params: {
      q?: string
      type?: string
      page?: number
      page_size?: number
      sort?: string
    } = {},
  ): Promise<{ items: MediaSummary[]; total: number; page: number; size: number }> {
    return (await http.get<unknown>('/api/v1/media', { params })) as unknown as {
      items: MediaSummary[]
      total: number
      page: number
      size: number
    }
  },
}

export const historyApi = {
  async getDefaultProfile(): Promise<Profile> {
    const body = (await http.get<unknown>('/api/v1/playback/default-profile')) as {
      data: Profile
    }
    return body.data
  },

  async ensureProfileId(): Promise<string> {
    const stored = localStorage.getItem('mediahub_profile_id')
    if (stored) return stored
    try {
      const profile = await historyApi.getDefaultProfile()
      localStorage.setItem('mediahub_profile_id', profile.id)
      return profile.id
    } catch {
      const fallback = '00000000-0000-0000-0000-000000000001'
      localStorage.setItem('mediahub_profile_id', fallback)
      return fallback
    }
  },

  async record(data: {
    media_id: string
    episode_id?: string
    progress: number
    duration: number
  }): Promise<{ status: string }> {
    return http.post('/api/v1/history', {
      profile_id: localStorage.getItem('mediahub_profile_id') || '',
      ...data,
      device: 'web',
    }) as Promise<{ status: string }>
  },

  async getResume(mediaId: string): Promise<ResumeInfo | null> {
    const body = (await http.get<unknown>(`/api/v1/resume/${mediaId}`)) as {
      data: ResumeInfo | null
    }
    return body.data
  },

  async toggleFavorite(data: { media_id: string; type?: string }): Promise<{ status: string }> {
    return http.post('/api/v1/favorites', {
      profile_id: localStorage.getItem('mediahub_profile_id') || '',
      ...data,
    }) as Promise<{ status: string }>
  },

  async getContinueWatching(limit = 12): Promise<{ data: LibraryItem[]; total: number }> {
    return (await http.get<unknown>('/api/v1/library/continue-watching', {
      params: { limit },
    })) as unknown as { data: LibraryItem[]; total: number }
  },
}

export const recommendApi = {
  async hot(limit = 20): Promise<MediaSummary[]> {
    const body = (await http.get<unknown>('/api/v1/recommend/hot', {
      params: { limit },
    })) as { data: MediaSummary[] }
    return body.data || []
  },

  async similar(mediaId: string, limit = 12): Promise<MediaSummary[]> {
    const body = (await http.get<unknown>(`/api/v1/recommend/similar/${mediaId}`, {
      params: { limit },
    })) as { data: MediaSummary[] }
    return body.data || []
  },

  async forMe(limit = 20): Promise<MediaSummary[]> {
    const body = (await http.get<unknown>('/api/v1/recommend/for-me', {
      params: { limit },
    })) as { data: MediaSummary[] }
    return body.data || []
  },
}

export interface LibraryItem {
  media_id?: string
  tmdb_id?: number
  media_type?: string
  title?: string
  year?: number
  poster_url?: string
  external?: boolean
  media?: MediaSummary
  progress?: number
  duration?: number
  updated_at?: string
  episode_id?: string
}

// Profile API（Web 播放端，无需登录）
export const profileApi = {
  async listWeb(): Promise<Profile[]> {
    const body = (await http.get<unknown>('/api/v1/playback/profiles')) as {
      data: Profile[]
    }
    return body.data || []
  },

  async create(data: {
    name: string
    is_kid?: boolean
    pin?: string
  }): Promise<Profile> {
    const body = (await http.post<unknown>('/api/v1/playback/profiles', data)) as {
      data: Profile
    }
    return body.data
  },

  async verifyPin(profileId: string, pin: string): Promise<void> {
    await http.post(`/api/v1/playback/profiles/${profileId}/verify-pin`, { pin })
  },

  async list(): Promise<Profile[]> {
    const body = (await http.get<unknown>('/api/v1/profiles')) as { data: Profile[] }
    return body.data || []
  },
}

export const catalogApi = {
  async credits(mediaId: string, role = ''): Promise<MediaCredit[]> {
    const body = (await http.get<unknown>(`/api/v1/works/${mediaId}/credits`, {
      params: role ? { role } : {},
    })) as { data: MediaCredit[] }
    return body.data || []
  },

  async ratings(mediaId: string): Promise<ContentRating[]> {
    const body = (await http.get<unknown>(`/api/v1/works/${mediaId}/ratings`)) as {
      data: ContentRating[]
    }
    return body.data || []
  },

  async nextEpisode(mediaId: string, afterEpisodeId: string): Promise<EpisodeNext | null> {
    const body = (await http.get<unknown>(`/api/v1/works/${mediaId}/next-episode`, {
      params: { after_episode_id: afterEpisodeId },
    })) as { data: EpisodeNext | null }
    return body.data ?? null
  },

  async extras(mediaId: string, type = 'trailer'): Promise<MediaExtra[]> {
    const body = (await http.get<unknown>(`/api/v1/works/${mediaId}/extras`, {
      params: { type },
    })) as { data: MediaExtra[] }
    return body.data || []
  },

  async person(personId: string): Promise<Person> {
    const body = (await http.get<unknown>(`/api/v1/persons/${personId}`)) as { data: Person }
    return body.data
  },

  async personByTmdb(tmdbId: number): Promise<Person> {
    const body = (await http.get<unknown>(`/api/v1/persons/by-tmdb/${tmdbId}`)) as { data: Person }
    return body.data
  },

  async personWorks(personId: string, opts?: { limit?: number; excludeMediaId?: string }): Promise<PersonWork[]> {
    const body = (await http.get<unknown>(`/api/v1/persons/${personId}/works`, {
      params: {
        limit: opts?.limit ?? 24,
        exclude_media_id: opts?.excludeMediaId || undefined,
      },
    })) as { data: PersonWork[] }
    return body.data || []
  },
}

export const libraryApi = {
  async wantList(): Promise<LibraryItem[]> {
    const body = (await http.get<unknown>('/api/v1/library/want-to-watch')) as {
      data: Array<Record<string, unknown>>
    }
    return (body.data || []).map(normalizeWantItem)
  },

  async favoritesList(): Promise<{ media_id: string }[]> {
    const body = (await http.get<unknown>('/api/v1/library/favorites')) as {
      data: Array<{ media_id: string }>
    }
    return body.data || []
  },

  async addWant(mediaId: string): Promise<void> {
    await http.post(`/api/v1/library/want-to-watch/${mediaId}`)
  },

  async addWantTmdb(data: {
    tmdb_id: number
    type: string
    title: string
    year?: number
    poster_url?: string
  }): Promise<void> {
    await http.post('/api/v1/library/want-to-watch/tmdb', data)
  },

  async removeWantTmdb(type: string, tmdbId: number): Promise<void> {
    await http.delete(`/api/v1/library/want-to-watch/tmdb/${type}/${tmdbId}`)
  },

  async removeWant(mediaId: string): Promise<void> {
    await http.delete(`/api/v1/library/want-to-watch/${mediaId}`)
  },

  async toggleFavorite(mediaId: string): Promise<{ added: boolean }> {
    const body = (await http.post<unknown>(`/api/v1/library/favorites/${mediaId}`)) as {
      added: boolean
    }
    return { added: !!body.added }
  },

  async continueWatching(limit = 24): Promise<LibraryItem[]> {
    const body = (await http.get<unknown>('/api/v1/library/continue-watching', {
      params: { limit },
    })) as { data: LibraryItem[] }
    return body.data || []
  },
}

// ─── 直播 ───

export interface LiveRoom {
  id: string
  title: string
  description?: string
  cover_url?: string
  status: 'idle' | 'live' | 'ended'
  stream_key: string
  viewer_count: number
  started_at?: string
  ended_at?: string
  created_at: string
  updated_at: string
  play_url?: string
}

export const liveApi = {
  async list(params?: { status?: string }): Promise<LiveRoom[]> {
    const body = (await http.get<unknown>('/api/v1/live/rooms', { params })) as {
      data: LiveRoom[]
    }
    return body.data || []
  },

  async get(id: string): Promise<LiveRoom> {
    const body = (await http.get<unknown>(`/api/v1/live/rooms/${id}`)) as { data: LiveRoom }
    return body.data
  },

  playlistUrl(id: string): string {
    return `/api/v1/live/rooms/${id}/playlist.m3u8`
  },
}

function normalizeWantItem(raw: Record<string, unknown>): LibraryItem {
  const media = raw.media as MediaSummary | undefined
  const mediaId = (raw.media_id as string) || media?.id
  const tmdbId = raw.tmdb_id as number | undefined
  const external = !mediaId && !!tmdbId
  return {
    media_id: mediaId,
    tmdb_id: tmdbId,
    media_type: (raw.media_type as string) || media?.type,
    title: (raw.title as string) || media?.title,
    year: (raw.year as number) || media?.year,
    poster_url: (raw.poster_url as string) || media?.poster_url,
    external,
    media,
  }
}