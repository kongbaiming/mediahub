// 媒资 API
import { http } from './client'
import type { MediaListResponse, MediaDetail, Stats } from './types'

export interface MediaListParams {
  page?: number
  page_size?: number
  type?: string
  genre?: string
  year?: number
  min_rating?: number
  q?: string
  sort?: 'year' | 'rating' | 'title' | 'created_at'
  order?: 'asc' | 'desc'
  scrape_status?: string
}

export const mediaApi = {
  list: (params: MediaListParams = {}) => {
    const cleaned: Record<string, any> = {}
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined && v !== null && v !== '') cleaned[k] = v
    })
    return http.get<MediaListResponse>('/api/v1/media', { params: cleaned })
  },

  get: (id: string) => http.get<{ data: MediaDetail }>(`/api/v1/media/${id}`),

  create: (data: Partial<MediaDetail>) => http.post('/api/v1/media', data),

  update: (id: string, data: Partial<MediaDetail>) =>
    http.patch(`/api/v1/media/${id}`, data),

  uploadPoster: (id: string, file: File) => {
    const form = new FormData()
    form.append('poster', file)
    return http.post<{ data: { poster_url: string } }>(`/api/v1/media/${id}/poster`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 120000,
    })
  },

  delete: (id: string) => http.delete(`/api/v1/media/${id}`),

  rescan: (id: string) => http.post(`/api/v1/media/${id}/rescan`),

  batchRescan: (payload: { ids?: string[]; scrape_status?: string }) =>
    http.post<{ status: string; queued: number }>('/api/v1/media/batch-rescan', payload),

  stats: () => http.get<{ data: Stats }>('/api/v1/media/stats'),

  scrapeCandidates: (id: string) =>
    http.get<{ data: ScrapeCandidate[] }>(`/api/v1/media/${id}/scrape-candidates`),

  applyScrapeMatch: (id: string, payload: { tmdb_id: number; type: 'movie' | 'tv' }) =>
    http.post<{ status: string; media_id: string }>(`/api/v1/media/${id}/apply-scrape-match`, payload),
}

export interface ScrapeCandidate {
  tmdb_id: number
  type: 'movie' | 'tv'
  title: string
  original_title?: string
  year?: number
  overview?: string
  poster_url?: string
  runtime?: number
  rating?: number
  match_score?: number
}
