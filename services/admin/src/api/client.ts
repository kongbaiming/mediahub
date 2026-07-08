// API 客户端封装（axios）
import axios, { type AxiosInstance, type InternalAxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'

const baseURL = import.meta.env.VITE_API_BASE_URL || ''

/** 响应拦截器已解包 res.data，此处修正 TS 返回类型 */
type UnwrappedHttp = Omit<
  AxiosInstance,
  'get' | 'post' | 'put' | 'patch' | 'delete' | 'request'
> & {
  get<T = unknown>(url: string, config?: object): Promise<T>
  post<T = unknown>(url: string, data?: unknown, config?: object): Promise<T>
  put<T = unknown>(url: string, data?: unknown, config?: object): Promise<T>
  patch<T = unknown>(url: string, data?: unknown, config?: object): Promise<T>
  delete<T = unknown>(url: string, config?: object): Promise<T>
  request<T = unknown>(config: object): Promise<T>
}

export const http: UnwrappedHttp = axios.create({
  baseURL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器：注入 token
http.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = localStorage.getItem('mediahub_token')
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`
  }
  const profileId = localStorage.getItem('mediahub_profile_id')
  if (profileId && config.headers) {
    config.headers['X-Profile-ID'] = profileId
  }
  return config
})

// 响应拦截器：统一错误处理
http.interceptors.response.use(
  (response) => response.data,
  (error) => {
    const status = error.response?.status
    const msg = error.response?.data?.message || error.message

    if (status === 401) {
      localStorage.removeItem('mediahub_token')
      localStorage.removeItem('mediahub_user')
      if (!location.pathname.startsWith('/login')) {
        location.href = '/login'
      }
    } else if (status === 403) {
      ElMessage.error(`权限不足：${msg}`)
    } else if (status >= 500) {
      ElMessage.error(`服务异常：${msg}`)
    } else if (status >= 400) {
      ElMessage.warning(msg || '请求失败')
    } else if (error.code === 'ECONNABORTED' || msg?.includes('timeout')) {
      ElMessage.warning('请求超时，任务可能仍在后台执行，请稍后刷新页面查看结果')
    }

    return Promise.reject(error)
  }
)

export interface Download {
  hash: string
  name: string
  size: number
  progress: number
  dlspeed: number
  upspeed: number
  ratio: number
  state: string
  category: string
  save_path: string
  added_on: number
  completion_on: number
  eta: number
}

export interface AdminWantItem {
  id: string
  profile_id: string
  profile_name: string
  media_id?: string
  tmdb_id?: number
  media_type?: string
  title: string
  year?: number
  poster_url?: string
  in_library: boolean
  local_media_id?: string
  external: boolean
  created_at: string
}

export interface IndexerRelease {
  title: string
  link: string
  size: number
  seeders: number
  peers: number
  indexer: string
  publish_date?: string
}

export interface LibraryMissingItem {
  tmdb_id?: number
  title: string
  poster_url?: string
  backdrop_url?: string
  media_type?: string
  rating?: number
  year?: number
  overview?: string
  genres?: string[]
  external?: boolean
}

export const libraryMissingApi = {
  list: (params: { limit?: number; discover_limit?: number; profile_id?: string } = {}) =>
    http.get<{ data: LibraryMissingItem[]; total: number }>(
      '/api/v1/admin/recommendations/library-missing',
      { params },
    ),
}

export const adminWantApi = {
  list: (limit = 200) =>
    http.get<{ data: AdminWantItem[]; total: number }>('/api/v1/admin/want-to-watch', {
      params: { limit },
    }),
}

export const indexerApi = {
  search: (params: { q: string; type?: string; limit?: number }) =>
    http.get<{ data: IndexerRelease[]; status: string; message?: string; query?: string }>(
      '/api/v1/indexer/search',
      { params, timeout: 60000 },
    ),
}

export const downloaderApi = {
  list: (category?: string) =>
    http.get<{ data: Download[]; total: number; status?: string; message?: string }>(
      '/api/v1/downloader/list',
      { params: category ? { category } : {} },
    ).catch((err) => {
      if (err.response?.status === 404) {
        return { data: [], total: 0, status: 'disabled', message: '下载器未启用（DOWNLOADER_ENABLED=false）' }
      }
      throw err
    }),

  add: (data: { url: string; category?: string; save_path?: string }) =>
    http.post<{ status: string; hash: string }>('/api/v1/downloader/add', data),

  remove: (hash: string, deleteFiles = false) =>
    http.delete(`/api/v1/downloader/${hash}`, { params: { delete_files: deleteFiles } }),

  pause: (hash: string) => http.post(`/api/v1/downloader/${hash}/pause`),
  resume: (hash: string) => http.post(`/api/v1/downloader/${hash}/resume`),
  checkCompleted: () => http.post<{ status: string; imported: number; message?: string }>('/api/v1/downloader/check-completed'),
  health: () =>
    http.get<{ status: string; error?: string }>('/api/v1/downloader/health').catch((err) => {
      if (err.response?.status === 404) {
        return { status: 'disabled', error: '下载器未启用' }
      }
      throw err
    }),
}

export const profileApi = {
  list: () => http.get<{ data: import('./types').Profile[]; total: number }>('/api/v1/profiles'),

  create: (data: { name: string; is_kid?: boolean; pin?: string; avatar?: string }) =>
    http.post<{ data: import('./types').Profile }>('/api/v1/profiles', data),

  update: (id: string, data: { name?: string; is_kid?: boolean; pin?: string; avatar?: string }) =>
    http.patch<{ data: import('./types').Profile }>(`/api/v1/profiles/${id}`, data),

  remove: (id: string) => http.delete<{ status: string }>(`/api/v1/profiles/${id}`),
}

export const dashboardApi = {
  health: () =>
    http.get<{ data: Record<string, { status: string; message?: string }> }>('/api/v1/dashboard/health'),
}

export const SCAN_INTERVAL_OPTIONS = [
  { value: 15, label: '每 15 分钟' },
  { value: 30, label: '每 30 分钟' },
  { value: 60, label: '每 1 小时' },
  { value: 360, label: '每 6 小时' },
  { value: 720, label: '每 12 小时' },
  { value: 1440, label: '每 24 小时' },
  { value: 10080, label: '每 7 天' },
] as const

export interface MediaScanConfig {
  enabled: boolean
  interval_minutes: number
  roots: string[]
  last_scan_at?: string
  last_scan_status?: string
  last_scan_message?: string
  last_scan_added?: number
  last_scan_total?: number
}

export const scannerApi = {
  getConfig: () => http.get<{ data: MediaScanConfig }>('/api/v1/scanner/config'),

  updateConfig: (data: { enabled: boolean; interval_minutes: number }) =>
    http.put<{ data: MediaScanConfig }>('/api/v1/scanner/config', data),

  scan: (root?: string) =>
    http.post<{
      data: { total: number; added: number; skipped: number; failed: number }
    }>('/api/v1/scanner/scan', root ? { root } : {}, { timeout: 600000 }),
}

export interface Subtitle {
  id: string
  name: string
  language: string
  source: string
  format: string
  rating: number
  downloads: number
  upload_date: string
  download_url?: string
}

export const subtitleApi = {
  search: (data: { media_id: string; season?: number; episode?: number; lang?: string }) =>
    http.post<{ data: Subtitle[]; total: number }>('/api/v1/subtitle/search', data),
  download: (mediaId: string, subtitle: Subtitle) =>
    http.post<{ status: string }>(`/api/v1/subtitle/${mediaId}/download`, { subtitle }),
}
