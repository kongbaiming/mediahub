// 认证 API
import { http } from './client'
import type { LoginResponse } from './types'

export const authApi = {
  login: (username: string, password: string) =>
    http.post<LoginResponse>('/api/v1/auth/login', { username, password }),

  register: (data: { username: string; password: string; display_name?: string }) =>
    http.post<LoginResponse>('/api/v1/auth/register', data),

  me: () => http.get<{ data: any; user_id: string; username: string; role: string }>('/api/v1/auth/me'),
}
