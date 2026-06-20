// API 类型定义
export interface User {
  id: string
  username: string
  display_name?: string
  role: string
  avatar_url?: string
}

export interface Profile {
  id: string
  user_id: string
  name: string
  avatar_url?: string
  is_kid: boolean
}

export interface LoginResponse {
  token: string
  user: User
  profiles: Profile[]
}

export interface MediaSummary {
  id: string
  title: string
  original_title?: string
  year?: number
  type: 'movie' | 'tvshow' | 'anime' | 'documentary'
  rating: number
  poster_url?: string
  backdrop_url?: string
  genres: string[]
  has_subtitle: boolean
}

export interface MediaDetail extends MediaSummary {
  overview?: string
  runtime?: number
  vote_count?: number
  tmdb_id?: number
  storage_path: string
  file_size: number
  video_codec?: string
  audio_codec?: string
  resolution?: string
  container?: string
  scrape_status: 'pending' | 'scraping' | 'done' | 'failed'
  scrape_error?: string
  seasons?: SeasonDetail[]
}

export interface SeasonDetail {
  id: string
  season_number: number
  title?: string
  overview?: string
  episode_count: number
  episodes?: EpisodeDetail[]
}

export interface EpisodeDetail {
  id: string
  episode_number: number
  title?: string
  duration: number
  still_url?: string
}

export interface MediaListResponse {
  items: MediaSummary[]
  total: number
  page: number
  size: number
}

export interface LayoutRow {
  id: string
  type: 'hero-banner' | 'shelf' | 'category-grid' | 'topic' | 'text-banner' | 'divider'
  title?: string
  subtitle?: string
  card_style?: 'poster' | 'landscape' | 'square' | 'banner'
  source: {
    type: string
    params?: Record<string, any>
  }
  visible?: boolean
  config?: Record<string, any>
  _inherited?: boolean
}

export interface LayoutConfig {
  theme?: 'dark' | 'light'
  rows: LayoutRow[]
  global?: Record<string, any>
}

export interface Layout {
  id: string
  name: string
  description?: string
  is_template: boolean
  parent_id?: string
  version: number
  status: 'draft' | 'published' | 'archived'
  config: LayoutConfig
  created_at: string
  updated_at: string
  last_published_at?: string
}

export interface FeedRow {
  id: string
  type: string
  title?: string
  subtitle?: string
  card_style?: string
  items: FeedItem[]
  config?: Record<string, any>
}

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

export interface Feed {
  version: number
  platform: string
  updated_at: string
  rows: FeedRow[]
}

export interface Stats {
  total: number
  by_type: Record<string, number>
  by_scrape: Record<string, number>
}
