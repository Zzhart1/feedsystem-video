// 字段容错：后端不同接口字段名略有差异（play_url / video_url、username 嵌套位置等），统一转成前端用字段
export function normalizeFeedVideoItem(v) {
  if (!v) return null
  return {
    id: v.id,
    author: {
      id: v.author?.id ?? v.author_id ?? 0,
      username: v.author?.username ?? v.username ?? '未知用户',
    },
    title: v.title ?? '',
    description: v.description ?? '',
    play_url: v.play_url ?? v.video_url ?? '',
    cover_url: v.cover_url ?? '',
    create_time: v.create_time ?? 0,
    likes_count: Number(v.likes_count ?? 0),
    comments_count: Number(v.comments_count ?? 0),
    is_liked: !!v.is_liked,
    popularity: Number(v.popularity ?? 0),
  }
}

export function normalizeFeedVideoList(list) {
  return (list || []).map(normalizeFeedVideoItem).filter(Boolean)
}

export function normalizeVideo(v) {
  if (!v) return null
  return {
    id: v.id,
    author_id: v.author_id ?? v.account_id ?? 0,
    username: v.username ?? v.author?.username ?? '未知用户',
    title: v.title ?? '',
    description: v.description ?? '',
    play_url: v.play_url ?? v.video_url ?? '',
    cover_url: v.cover_url ?? '',
    visibility: v.visibility ?? 'public',
    likes_count: Number(v.likes_count ?? 0),
    comments_count: Number(v.comments_count ?? 0),
    popularity: Number(v.popularity ?? 0),
    created_at: v.created_at ?? v.create_time ?? 0,
    is_liked: !!v.is_liked,
  }
}

export function normalizeVideoList(list) {
  return (list || []).map(normalizeVideo).filter(Boolean)
}

// 展示用：时间戳可能是毫秒 int64 / ISO 字符串 / 秒，统一成可读文本
export function formatTime(t) {
  if (!t) return ''
  let ms = typeof t === 'string' ? Date.parse(t) : t
  if (ms > 0 && ms < 1e12) ms *= 1000 // 秒 → 毫秒
  if (Number.isNaN(ms) || ms <= 0) return ''
  const d = new Date(ms)
  const now = Date.now()
  const diff = now - ms
  if (diff < 60_000) return '刚刚'
  if (diff < 3600_000) return `${Math.floor(diff / 60_000)} 分钟前`
  if (diff < 86_400_000) return `${Math.floor(diff / 3600_000)} 小时前`
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return y === new Date().getFullYear() ? `${m}-${day}` : `${y}-${m}-${day}`
}

export function formatCount(n) {
  const num = Number(n || 0)
  if (num >= 1_0000) return `${(num / 1_0000).toFixed(1)}w`
  if (num >= 1000) return `${(num / 1000).toFixed(1)}k`
  return String(num)
}
