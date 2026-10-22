<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listNotifications, markRead, unreadCount } from '../api/notification'
import { useAuthStore } from '../stores/auth'
import Toaster from '../utils/toast'

const router = useRouter()
const auth = useAuthStore()
const items = ref([])
const loading = ref(true)

const typeText = {
  like: '赞了你的视频',
  comment: '评论了你的视频',
  follow: '关注了你',
}

async function load() {
  loading.value = true
  try {
    items.value = await listNotifications()
  } catch (e) {
    Toaster.error(e.message || '通知加载失败')
  } finally {
    loading.value = false
  }
}

async function onMark(item) {
  if (item.is_read) return
  try {
    await markRead(item.id)
    item.is_read = true
    const n = await unreadCount()
    // 更新导航角标（简单方式：刷新页面角标由 AppShell 的 SSE 维护，这里只置本地已读）
    auth.lastUnread = n
    Toaster.info('已标记为已读')
  } catch {
    /* 忽略 */
  }
}

function goTarget(item) {
  if (item.type === 'follow') router.push(`/profile/${item.sender_id}`)
  else if (item.target_id) router.push(`/video/${item.target_id}`)
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="card box">
      <h1>消息通知</h1>
      <p class="sub">实时推送由 SSE 长连接实现：别人点赞/评论/关注你时，顶部会立刻弹通知</p>

      <div v-if="loading" class="empty"><div class="spinner dark"></div></div>
      <div v-else-if="!items.length" class="empty">
        <div class="big">🔕</div>
        <p>暂时没有通知，去刷 Feed 吧</p>
      </div>

      <ul v-else class="list">
        <li v-for="item in items" :key="item.id" class="row" :class="{ unread: !item.is_read }" @click="goTarget(item)">
          <div class="left">
            <span class="dot" :class="{ read: item.is_read }"></span>
          </div>
          <div class="body">
            <div class="txt">
              <b>{{ typeText[item.type] || item.type }}</b>
              <span v-if="item.content" class="cnt">「{{ item.content }}」</span>
            </div>
            <div class="time">{{ item.time_text }}</div>
          </div>
          <button v-if="!item.is_read" class="btn ghost sm" @click.stop="onMark(item)">标已读</button>
        </li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.box { max-width: 640px; margin: 0 auto; padding: 24px 28px; }
h1 { font-size: 20px; }
.sub { color: var(--text-2); font-size: 13px; margin: 4px 0 16px; }
.list { list-style: none; }
.row {
  display: flex; align-items: center; gap: 12px;
  padding: 13px 4px; border-bottom: 1px solid var(--border);
  cursor: pointer;
}
.row:hover { background: var(--surface); }
.row.unread { background: rgba(14, 165, 233, 0.04); }
.dot { width: 8px; height: 8px; border-radius: 50%; background: var(--accent); display: inline-block; }
.dot.read { background: #cbd5e1; }
.body { flex: 1; }
.txt { font-size: 14px; }
.cnt { color: var(--text-2); }
.time { font-size: 12px; color: var(--text-2); }
</style>
