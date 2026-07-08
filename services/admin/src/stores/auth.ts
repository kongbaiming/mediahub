// Pinia auth store
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/auth'
import { profileApi } from '@/api/client'
import type { User, Profile } from '@/api/types'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem('mediahub_token') || '')
  const user = ref<User | null>(JSON.parse(localStorage.getItem('mediahub_user') || 'null'))
  const profiles = ref<Profile[]>([])
  const activeProfileId = ref<string>(localStorage.getItem('mediahub_profile_id') || '')

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const activeProfile = computed(() =>
    profiles.value.find((p) => p.id === activeProfileId.value) || profiles.value[0]
  )

  async function login(username: string, password: string) {
    const res = await authApi.login(username, password)
    token.value = res.token
    user.value = res.user
    profiles.value = res.profiles || []
    if (res.profiles?.[0]) {
      activeProfileId.value = res.profiles[0].id
    }
    localStorage.setItem('mediahub_token', res.token)
    localStorage.setItem('mediahub_user', JSON.stringify(res.user))
    if (activeProfileId.value) {
      localStorage.setItem('mediahub_profile_id', activeProfileId.value)
    }
    return res
  }

  async function register(username: string, password: string, displayName?: string) {
    const res = await authApi.register({ username, password, display_name: displayName })
    token.value = res.token
    user.value = res.user
    profiles.value = res.profiles || []
    if (res.profiles?.[0]) {
      activeProfileId.value = res.profiles[0].id
    }
    localStorage.setItem('mediahub_token', res.token)
    localStorage.setItem('mediahub_user', JSON.stringify(res.user))
    if (activeProfileId.value) {
      localStorage.setItem('mediahub_profile_id', activeProfileId.value)
    }
    return res
  }

  function logout() {
    token.value = ''
    user.value = null
    profiles.value = []
    activeProfileId.value = ''
    localStorage.removeItem('mediahub_token')
    localStorage.removeItem('mediahub_user')
    localStorage.removeItem('mediahub_profile_id')
  }

  function switchProfile(profileId: string) {
    activeProfileId.value = profileId
    localStorage.setItem('mediahub_profile_id', profileId)
  }

  async function bootstrap() {
    if (!token.value) return
    try {
      const me = await authApi.me()
      if (me.data) {
        user.value = {
          id: me.user_id,
          username: me.username,
          role: me.role,
          display_name: me.data.display_name,
          avatar_url: me.data.avatar_url,
        }
        localStorage.setItem('mediahub_user', JSON.stringify(user.value))
      }
    } catch {
      logout()
      return
    }
    try {
      const res = await profileApi.list()
      profiles.value = res.data || []
      const stored = activeProfileId.value
      if (!stored || !profiles.value.some((p) => p.id === stored)) {
        if (profiles.value[0]) {
          activeProfileId.value = profiles.value[0].id
          localStorage.setItem('mediahub_profile_id', activeProfileId.value)
        }
      }
    } catch {
      profiles.value = []
    }
  }

  return {
    token,
    user,
    profiles,
    activeProfileId,
    isLoggedIn,
    isAdmin,
    activeProfile,
    login,
    register,
    logout,
    switchProfile,
    bootstrap,
  }
})
