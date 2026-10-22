<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { listByAuthorId } from '../api/video'
import { listMyLikedVideos } from '../api/like'
import { getAllFollowers, getAllVloggers, follow, unfollow } from '../api/social'
import { findByID, findByUsername } from '../api/account'
import { useAuthStore } from '../stores/auth'
import { formatCount } from '../api/normalize'
import VideoCard from '../components/VideoCard.vue'
import Toaster from '../utils/toast'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const profile = ref(null)
const videos = ref([])
const followers = ref([])
const vloggers = ref([])
const likedVideos = ref([])
const tab = ref('videos') // videos | likes | followers | vloggers
const following = ref(false)
const loading = ref(true)

const profileId = computed(() => Number(route.params.id) || auth.userId || 0)
const isMe = computed(() => auth.isLoggedIn && profileId.value === auth.userId)

async function load() {
  loading.value = true
  try {
    // 找账号信息
    let acc = null
    if (profileId.value && auth.isLoggedIn && profileId.value === auth.userId) {
      acc = auth.account
    }
    if (!acc && profileId.value) {
      const res = await findByID(profileId.value)
      acc = res.account || res
    }
    if (!acc) {
      acc = { id: profileId.value, username: '用户' + profileId.value }
    }
    profile.value = acc

    videos.value = await listByAuthorId(profileId.value)
    if (auth.isLoggedIn) {
      if (isMe.value) {
        likedVideos.value = await listMyLikedVideos()
        followers.value = await getAllFollowers()
        vloggers.value = await getAllVloggers()
      } else if (profileId.value !== auth.userId) {
        // 别人主页：查对方关注列表判断是否已关注（简单方式：对比我的关注列表）
        const mine = await getAllVloggers()
        following.value = mine.some((v) => v.id === profileId.value)
      }
    }
  } catch (e) {
    Toaster.error(e.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function toggleFollow() {
  if (!auth.isLoggedIn) {
    Toaster.info('登录后才能关注')
    router.push('/login')
    return
  }
  try {
    if (following.value) {
      await unfollow(profileId.value)
      following.value = false
      followers.value = followers.value.filter((f) => f.id !== auth.userId)
    } else {
      await follow(profileId.value)
      following.value = true
    }
  } catch (e) {
    Toaster.error(e.message || '操作失败')
  }
}

onMounted(load)

// /profile 与 /profile/:id 共用同一个组件，互相切换时 Vue Router 会复用组件实例、
// 不触发 onMounted。监听路由参数变化主动重新加载，避免显示旧数据或空白。
watch(() => route.params.id, () => { load() })
</script>

<template>
  <div class="page">
    <div class="card head">
      <div class="avatar">{{ (profile?.username || '?').slice(0, 1).toUpperCase() }}</div>
      <div class="who">
        <h1>{{ profile?.username || '加载中…' }}</h1>
        <div class="stats">
          <span>视频 {{ videos.length }}</span>
          <span>获赞 {{ formatCount(videos.reduce((s, v) => s + v.likes_count, 0)) }}</span>
        </div>
      </div>
      <button v-if="!isMe && profileId" class="btn sm" :class="{ fwd: following }" @click="toggleFollow">
        {{ following ? '已关注' : '+ 关注' }}
      </button>
    </div>

    <div class="tabs">
      <button :class="{ on: tab === 'videos' }" @click="tab = 'videos'">视频</button>
      <button v-if="isMe" :class="{ on: tab === 'likes' }" @click="tab = 'likes'">我的赞</button>
      <button v-if="isMe" :class="{ on: tab === 'followers' }" @click="tab = 'followers'">粉丝 ({{ followers.length }})</button>
      <button v-if="isMe" :class="{ on: tab === 'vloggers' }" @click="tab = 'vloggers'">关注 ({{ vloggers.length }})</button>
    </div>

    <div v-if="loading" class="empty"><div class="spinner dark"></div></div>

    <template v-else>
      <!-- 视频/我的赞：卡片网格 -->
      <div v-if="tab === 'videos' || tab === 'likes'" class="grid">
        <VideoCard v-for="v in (tab === 'videos' ? videos : likedVideos)" :key="v.id" :video="v" />
      </div>

      <!-- 粉丝/关注：列表 -->
      <div v-else class="card plist">
        <router-link v-for="u in (tab === 'followers' ? followers : vloggers)" :key="u.id" :to="`/profile/${u.id}`" class="prow">
          <span class="pavatar">{{ (u.username || '?').slice(0, 1).toUpperCase() }}</span>
          <span class="pname">{{ u.username }}</span>
        </router-link>
        <div v-if="!((tab === 'followers' ? followers : vloggers)?.length)" class="empty" style="padding: 30px 0">暂无</div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.head { display: flex; align-items: center; gap: 18px; padding: 24px 28px; margin-bottom: 16px; }
.avatar {
  width: 68px; height: 68px; border-radius: 50%;
  background: linear-gradient(135deg, #0ea5e9, #22c55e);
  color: #fff; font-size: 30px; font-weight: 800;
  display: flex; align-items: center; justify-content: center;
}
.who { flex: 1; }
.who h1 { font-size: 20px; }
.stats { color: var(--text-2); font-size: 13px; display: flex; gap: 16px; margin-top: 4px; }
.btn.fwd { background: transparent; border: 1px solid var(--accent); color: var(--accent); }
.tabs { display: flex; gap: 6px; margin-bottom: 16px; flex-wrap: wrap; }
.tabs button {
  border: 1px solid var(--border); background: var(--surface);
  padding: 7px 18px; border-radius: 999px; font-size: 13.5px; font-weight: 600; color: var(--text-2);
}
.tabs button.on { background: linear-gradient(135deg, #0ea5e9, #3b82f6); color: #fff; border-color: transparent; }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(250px, 1fr)); gap: 18px; }
.plist { padding: 8px 16px; }
.prow { display: flex; align-items: center; gap: 12px; padding: 10px 8px; border-bottom: 1px solid var(--border); }
.prow:hover { background: var(--surface); }
.pavatar {
  width: 36px; height: 36px; border-radius: 50%;
  background: linear-gradient(135deg, #0ea5e9, #22c55e);
  color: #fff; display: flex; align-items: center; justify-content: center; font-weight: 700;
}
.pname { font-weight: 600; }
</style>
