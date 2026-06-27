/** CMS 布局行跳转协议（与播放端 feedAction 保持一致） */
export interface LayoutRowAction {
  type: 'none' | 'live' | 'library' | 'search' | 'route' | 'url' | 'media' | 'play'
  target?: string
  label?: string
}

export const ROW_ACTION_PRESETS: { value: LayoutRowAction['type']; label: string; hint?: string }[] = [
  { value: 'none', label: '无跳转' },
  { value: 'live', label: '直播页', hint: 'route:/live' },
  { value: 'library', label: '片库', hint: 'route:/library' },
  { value: 'search', label: '搜索页', hint: 'search:关键词（可选）' },
  { value: 'route', label: '自定义站内路径', hint: '如 /live、/library' },
  { value: 'url', label: '外部链接', hint: 'https://...' },
  { value: 'media', label: '媒资详情', hint: '媒资 UUID' },
  { value: 'play', label: '直接播放', hint: '媒资 UUID' },
]

export function ensureRowActionConfig(config?: Record<string, unknown>): Record<string, unknown> {
  const c = config ? { ...config } : {}
  if (!c.action || typeof c.action !== 'object') {
    c.action = { type: 'none' } satisfies LayoutRowAction
  }
  return c
}

export function actionNeedsTarget(type: LayoutRowAction['type']): boolean {
  return ['route', 'url', 'media', 'play', 'search'].includes(type)
}

export function actionTargetPlaceholder(type: LayoutRowAction['type']): string {
  switch (type) {
    case 'route':
      return '/live'
    case 'url':
      return 'https://example.com'
    case 'media':
    case 'play':
      return '媒资 UUID'
    case 'search':
      return '可选，默认关键词'
    default:
      return ''
  }
}
