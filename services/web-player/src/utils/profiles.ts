import { profileApi, historyApi, type Profile } from '@/api'

export interface LocalProfile {
  id: string
  name: string
  is_kid: boolean
}

function toLocal(p: Profile): LocalProfile {
  return { id: p.id, name: p.name, is_kid: p.is_kid }
}

export function saveProfilesLocal(profiles: LocalProfile[]) {
  localStorage.setItem('mediahub_profiles', JSON.stringify(profiles))
}

export function loadProfilesLocal(): LocalProfile[] {
  const stored = localStorage.getItem('mediahub_profiles')
  if (!stored) return []
  try {
    return JSON.parse(stored) as LocalProfile[]
  } catch {
    return []
  }
}

export function setActiveProfileId(id: string) {
  localStorage.setItem('mediahub_profile_id', id)
}

export function getActiveProfileId(): string {
  return localStorage.getItem('mediahub_profile_id') || ''
}

/** 从后端同步 Profile 列表，失败时回退本地缓存 */
export async function syncProfiles(): Promise<LocalProfile[]> {
  await historyApi.ensureProfileId()
  try {
    const remote = await profileApi.listWeb()
    const locals = remote.map(toLocal)
    if (locals.length > 0) {
      saveProfilesLocal(locals)
      const active = getActiveProfileId()
      if (!locals.some((p) => p.id === active)) {
        setActiveProfileId(locals[0].id)
      }
      return locals
    }
  } catch {
    // 离线或 API 不可用时使用本地缓存
  }
  const cached = loadProfilesLocal()
  if (cached.length > 0) {
    return cached
  }
  const fallback: LocalProfile[] = [
    { id: '00000000-0000-0000-0000-000000000001', name: '默认', is_kid: false },
  ]
  saveProfilesLocal(fallback)
  setActiveProfileId(fallback[0].id)
  return fallback
}
