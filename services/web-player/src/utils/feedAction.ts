import type { Router } from 'vue-router'

/** CMS 行跳转配置（row.config.action） */
export interface FeedRowAction {
  type: 'none' | 'live' | 'library' | 'search' | 'route' | 'url' | 'media' | 'play'
  target?: string
  label?: string
}

const PRESET_PATHS: Record<string, string> = {
  live: '/live',
  library: '/library',
}

/** 从 row.config 解析跳转动作（支持 action 对象或 link 协议字符串） */
export function parseFeedAction(config?: Record<string, unknown>): FeedRowAction | null {
  if (!config) return null

  const raw = config.action
  if (raw && typeof raw === 'object') {
    const a = raw as FeedRowAction
    if (!a.type || a.type === 'none') return null
    return a
  }

  const link = config.link
  if (typeof link !== 'string' || !link.trim()) return null
  return parseActionLink(link.trim())
}

/** 协议：route:/live | url:https://... | live | media:uuid | play:uuid | search:关键词 */
export function parseActionLink(link: string): FeedRowAction | null {
  if (link === 'live' || link === 'library') {
    return { type: link, label: '' }
  }
  const idx = link.indexOf(':')
  if (idx <= 0) return null
  const scheme = link.slice(0, idx)
  const rest = link.slice(idx + 1)
  switch (scheme) {
    case 'route':
      return rest ? { type: 'route', target: rest.startsWith('/') ? rest : `/${rest}` } : null
    case 'url':
      return rest ? { type: 'url', target: rest } : null
    case 'media':
      return rest ? { type: 'media', target: rest } : null
    case 'play':
      return rest ? { type: 'play', target: rest } : null
    case 'search':
      return { type: 'search', target: rest }
    case 'live':
    case 'library':
      return { type: scheme }
    default:
      return null
  }
}

export function actionLabel(action: FeedRowAction | null, fallback = '查看详情 →'): string {
  if (!action) return ''
  if (action.label?.trim()) return action.label.trim()
  switch (action.type) {
    case 'live':
      return '进入直播 →'
    case 'library':
      return '我的片库 →'
    case 'search':
      return '搜索 →'
    default:
      return fallback
  }
}

export function hasFeedAction(config?: Record<string, unknown>): boolean {
  return parseFeedAction(config) !== null
}

/** 执行 CMS 配置的跳转 */
export function runFeedAction(action: FeedRowAction, router: Router) {
  switch (action.type) {
    case 'live':
    case 'library':
      router.push(PRESET_PATHS[action.type])
      return
    case 'route':
      if (action.target) router.push(action.target)
      return
    case 'url':
      if (action.target) window.open(action.target, '_blank', 'noopener,noreferrer')
      return
    case 'media':
      if (action.target) router.push(`/media/${action.target}`)
      return
    case 'play':
      if (action.target) router.push(`/play/${action.target}`)
      return
    case 'search':
      router.push({
        path: '/search',
        query: action.target ? { q: action.target } : undefined,
      })
      return
    default:
      return
  }
}
