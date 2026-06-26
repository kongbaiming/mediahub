import { http } from './client'

export interface Album {
  id: string
  title: string
  overview?: string
  album_type?: string
}

export interface Category {
  id: string
  name: string
  slug: string
  kind?: string
}

export interface MediaCredit {
  id: string
  role: string
  character_name?: string
  person?: {
    id: string
    name: string
    original_name?: string
    profile_path?: string
    profile_url?: string
    biography?: string
    place_of_birth?: string
    known_for_department?: string
    birthday?: string
  }
}

export interface ContentRating {
  country: string
  system: string
  rating: string
}

export const catalogApi = {
  albums: () => http.get<{ data: Album[] }>('/api/v1/albums'),

  categories: (kind = 'genre') =>
    http.get<{ data: Category[] }>('/api/v1/categories', { params: { kind } }),

  credits: (mediaId: string, role = 'actor') =>
    http.get<{ data: MediaCredit[] }>(`/api/v1/works/${mediaId}/credits`, {
      params: { role },
    }),

  ratings: (mediaId: string) =>
    http.get<{ data: ContentRating[] }>(`/api/v1/works/${mediaId}/ratings`),
}
