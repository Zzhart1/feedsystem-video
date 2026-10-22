// 统一 API 客户端：JSON / FormData 请求 + 401 自动刷新 token（单飞，防止并发重复刷新）
import { useAuthStore } from '../stores/auth'

export class ApiError extends Error {
  constructor(message, status, payload) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.payload = payload
  }
}

const API_BASE = '/api'

let isRefreshing = false
let refreshPromise = null

async function tryRefresh() {
  const auth = useAuthStore()
  if (!auth.refreshToken) return null
  if (isRefreshing) return refreshPromise
  isRefreshing = true
  refreshPromise = (async () => {
    try {
      const res = await fetch(`${API_BASE}/account/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: auth.refreshToken }),
      })
      if (!res.ok) { auth.clearTokens(); return null }
      const data = await res.json()
      auth.setTokens({ token: data.token, refresh_token: auth.refreshToken })
      return data.token
    } catch {
      auth.clearTokens()
      return null
    } finally {
      isRefreshing = false
    }
  })()
  return refreshPromise
}

function buildHeaders(auth, hasBody) {
  const headers = {}
  if (hasBody) headers['Content-Type'] = 'application/json'
  if (auth.token) headers.Authorization = `Bearer ${auth.token}`
  return headers
}

async function request(path, options, authRequired) {
  const auth = useAuthStore()
  if (authRequired && !auth.token) {
    throw new ApiError('需要先登录', 401)
  }
  const res = await fetch(`${API_BASE}${path}`, {
    method: options.method || 'POST',
    headers: buildHeaders(auth, options.body !== undefined && !(options.body instanceof FormData)),
    body: options.body,
  })
  if (res.status === 401 && path !== '/account/refresh') {
    const newToken = await tryRefresh()
    if (newToken) {
      const retry = await fetch(`${API_BASE}${path}`, {
        method: options.method || 'POST',
        headers: { ...buildHeaders(auth, true), Authorization: `Bearer ${newToken}` },
        body: options.body,
      })
      return handleResponse(retry)
    }
  }
  return handleResponse(res)
}

export function postJson(path, body = {}, opts = {}) {
  return request(path, { body: JSON.stringify(body ?? {}) }, opts.authRequired)
}

export function postForm(path, formData, opts = {}) {
  return request(path, { body: formData }, opts.authRequired)
}

async function handleResponse(res) {
  const auth = useAuthStore()
  const text = await res.text()
  let data = null
  if (text) {
    try { data = JSON.parse(text) } catch { data = text }
  }
  if (!res.ok) {
    if (res.status === 401) auth.clearTokens()
    const msg = data && typeof data === 'object' && data.error
      ? String(data.error)
      : `请求失败 (${res.status})`
    throw new ApiError(msg, res.status, data)
  }
  return data
}
