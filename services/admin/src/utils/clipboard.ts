/**
 * 复制文本到剪贴板。
 * navigator.clipboard 仅在 HTTPS / localhost 可用；
 * 内网 HTTP（如 NAS :8080）需降级到 execCommand。
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  const value = text?.trim()
  if (!value) return false

  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value)
      return true
    } catch {
      // 非安全上下文等情况，继续走降级方案
    }
  }

  try {
    const ta = document.createElement('textarea')
    ta.value = value
    ta.setAttribute('readonly', '')
    ta.style.position = 'fixed'
    ta.style.left = '-9999px'
    ta.style.top = '0'
    document.body.appendChild(ta)
    ta.select()
    ta.setSelectionRange(0, value.length)
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  } catch {
    return false
  }
}
