<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { unreadCount, openNotificationStream } from '../api/notification'
import Toaster from '../utils/toast'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const unread = ref(0)
let es = null

function refreshUnread() {
  if (!auth.isLoggedIn) return
  unreadCount().then((n) => (unread.value = n)).catch(() => {})
}

function logout() {
  auth.clearTokens()
  if (es) { es.close(); es = null }
  router.push('/login')
}

onMounted(() => {
  refreshUnread()
  // 登录后挂 SSE 实时流：收到通知 → 未读数 +1 并轻提示
  watch(() => auth.isLoggedIn, (logged) => {
    if (es) { es.close(); es = null }
    if (logged && auth.token) {
      es = openNotificationStream(
        auth.token,
        (n) => {
          unread.value += 1
          const typeMap = { like: '赞了你的视频', comment: '评论了你的视频', follow: '关注了你' }
          Toaster.success(`收到新通知：${typeMap[n.type] || n.type || '新消息'}`)
        },
        () => { /* 断线自动重连由 EventSource 处理 */ },
      )
    }
  }, { immediate: true })
})

onUnmounted(() => { if (es) es.close() })
</script>

<template>
  <div class="shell">
    <header class="nav">
      <div class="nav-inner">
        <router-link to="/" class="logo">
          <span class="logo-mark">▶</span>
          <span class="logo-text">FeedFlow</span>
        </router-link>

        <nav class="links">
          <router-link to="/" exact-active-class="active">首页</router-link>
          <router-link to="/publish" class="publish-link">发布</router-link>
        </nav>

        <div class="right">
          <template v-if="auth.isLoggedIn">
            <router-link to="/messages" class="bell">
              🔔<span v-if="unread > 0" class="badge">{{ unread > 99 ? '99+' : unread }}</span>
            </router-link>
            <router-link to="/profile" class="me">
              <span class="avatar-mini">{{ auth.username.slice(0, 1).toUpperCase() }}</span>
              <span class="uname">{{ auth.username }}</span>
            </router-link>
            <button class="btn ghost sm" @click="logout">退出</button>
          </template>
          <template v-else>
            <router-link to="/login" class="btn ghost sm">登录</router-link>
            <router-link to="/register" class="btn sm">注册</router-link>
          </template>
        </div>
      </div>
    </header>

    <main class="main">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.shell { min-height: 100vh; display: flex; flex-direction: column; }
.nav {
  position: sticky; top: 0; z-index: 50;
  background: rgba(11, 13, 18, 0.92);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--border);
}
.nav-inner {
  max-width: 1080px; margin: 0 auto;
  padding: 0 20px; height: 60px;
  display: flex; align-items: center; gap: 28px;
}
.logo { display: flex; align-items: center; gap: 8px; font-weight: 800; font-size: 18px; }
.logo-mark {
  width: 30px; height: 30px; border-radius: 9px;
  background: linear-gradient(135deg, #0ea5e9, #3b82f6);
  color: #fff; display: inline-flex; align-items: center; justify-content: center;
  font-size: 13px;
}
.links { display: flex; gap: 6px; flex: 1; }
.links a { padding: 7px 14px; border-radius: 9px; color: var(--text-2); font-weight: 600; }
.links a:hover { background: var(--surface); color: var(--text); }
.links a.active { color: var(--accent); background: rgba(14, 165, 233, 0.08); }
.links .publish-link { color: var(--accent); }
.right { display: flex; align-items: center; gap: 14px; }
.bell { position: relative; font-size: 19px; padding: 6px; }
.badge {
  position: absolute; top: -2px; right: -6px;
  background: var(--danger); color: #fff;
  font-size: 11px; font-weight: 700;
  min-width: 17px; height: 17px; border-radius: 999px;
  display: inline-flex; align-items: center; justify-content: center;
  padding: 0 4px;
}
.me { display: flex; align-items: center; gap: 8px; }
.avatar-mini {
  width: 30px; height: 30px; border-radius: 50%;
  background: linear-gradient(135deg, #0ea5e9, #22c55e);
  color: #fff; display: inline-flex; align-items: center; justify-content: center;
  font-weight: 700; font-size: 14px;
}
.uname { font-weight: 600; }
.main { flex: 1; }
</style>
