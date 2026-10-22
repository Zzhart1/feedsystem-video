import { postJson } from './client'
import { formatTime } from './normalize'

export async function listNotifications() {
  const res = await postJson('/notification/list', {}, { authRequired: true })
  const list = res.notifications || []
  return list.map((n) => ({
    ...n,
    time_text: formatTime(n.created_at),
  }))
}

export function markRead(id) {
  return postJson('/notification/markRead', { id }, { authRequired: true })
}

export async function unreadCount() {
  const res = await postJson('/notification/unreadCount', {}, { authRequired: true })
  return Number(res.count ?? 0)
}

// SSE 实时流：GET /notification/stream?token=...（EventSource 不支持自定义 Header，只能走 query）
export function openNotificationStream(token, onEvent, onError) {
  const es = new EventSource(`/api/notification/stream?token=${encodeURIComponent(token)}`)
  es.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data)
      onEvent(data)
    } catch {
      /* 忽略非 JSON 的心跳/注释行 */
    }
  }
  es.onerror = () => { if (onError) onError(es) }
  return es
}
