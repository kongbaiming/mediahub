// API 客户端封装（axios）
import axios, { type AxiosInstance, type InternalAxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'

const baseURL = import.meta.env.VITE_API_BASE_URL || ''

export const http: AxiosInstance = axios.create({
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

export const downloaderApi = {
  list: (category?: string) =>
    http.get<{ data: Download[]; total: number }>('/api/v1/downloader/list', {
      params: category ? { category } : {},
    }),

  add: (data: { url: string; category?: string; save_path?: string }) =>
    http.post<{ status: string; hash: string }>('/api/v1/downloader/add', data),

  remove: (hash: string, deleteFiles = false) =>
    http.delete(`/api/v1/downloader/${hash}`, { params: { delete_files: deleteFiles } }),

  pause: (hash: string) => http.post(`/api/v1/downloader/${hash}/pause`),
  resume: (hash: string) => http.post(`/api/v1/downloader/${hash}/resume`),
  checkCompleted: () => http.post<{ status: string; imported: number }>('/api/v1/downloader/check-completed'),
  health: () => http.get<{ status: string }>('/api/v1/downloader/health'),
}

export const scannerApi = {
  scan: (root?: string) =>
    http.post<{
      data: { total: number; added: number; skipped: number; failed: number }
    }>('/api/v1/scanner/scan', root ? { root } : {}),
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
