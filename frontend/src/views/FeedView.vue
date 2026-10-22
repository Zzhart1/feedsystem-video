<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { listLatest, listByPopularity, listByFollowing, listLikesCount, listByTag } from '../api/feed'
import { like, unlike } from '../api/like'
import { listComments, publishComment } from '../api/comment'
import { follow, unfollow, getAllVloggers } from '../api/social'
import Toaster from '../utils/toast'
import VideoCard from '../components/VideoCard.vue'

const router = useRouter()
const auth = useAuthStore()

const tab = ref('latest')
const videos = ref([])
const loading = ref(false)
const hasMore = ref(true)
const tagInput = ref('')
const scroller = ref(null)
const bottomEl = ref(null)
const muted = ref(false)
const playingMap = ref({})

// 评论抽屉
const showComments = ref(false)
const activeVideo = ref(null)
const commentList = ref([])
const commentText = ref('')
const commentLoading = ref(false)
const followedMap = ref({}) // authorId -> bool
const viewMode = ref('feed') // 'feed' 全屏流 | 'grid' 网格卡片

const cursors = {
  latest: { latest_time: 0 },
  hot: { as_of: 0, offset: 0 },
  following: { latest_time: 0 },
  likes: { likes_count_before: undefined, id_before: undefined },
}

const videoEls = ref([])
let observer = null

async function load(reset = false) {
  if (loading.value || (!hasMore.value && !reset)) return
  if (tab.value === 'following' && !auth.isLoggedIn) {
    Toaster.info('关注流需要登录')
    return
  }
  loading.value = true
  try {
    let res
    if (tab.value === 'latest') {
      res = await listLatest({ limit: 10, latest_time: reset ? 0 : cursors.latest.latest_time })
      cursors.latest.latest_time = res.next_time || 0
      hasMore.value = res.has_more
    } else if (tab.value === 'hot') {
      res = await listByPopularity({ limit: 10, as_of: reset ? 0 : cursors.hot.as_of, offset: reset ? 0 : cursors.hot.offset })
      cursors.hot.as_of = res.as_of || 0
      cursors.hot.offset = res.next_offset || 0
      hasMore.value = res.has_more
    } else if (tab.value === 'following') {
      res = await listByFollowing({ limit: 10, latest_time: reset ? 0 : cursors.following.latest_time })
      cursors.following.latest_time = res.next_time || 0
      hasMore.value = res.has_more
    } else if (tab.value === 'tag') {
      res = await listByTag(tagInput.value.trim(), 10)
      hasMore.value = false // 标签流暂不分页
    } else {
      const body = { limit: 10 }
      if (!reset && cursors.likes.likes_count_before !== undefined) {
        body.likes_count_before = cursors.likes.likes_count_before
        body.id_before = cursors.likes.id_before
      }
      res = await listLikesCount(body)
      cursors.likes.likes_count_before = res.next_likes_count_before
      cursors.likes.id_before = res.next_id_before
      hasMore.value = res.has_more
    }
    videos.value = reset ? res.video_list : [...videos.value, ...res.video_list]
    await nextTick()
    observeVideos()
  } catch (e) {
    Toaster.error(e.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function switchTab(t) {
  if (tab.value === t) return
  tab.value = t
  videos.value = []
  hasMore.value = true
  videoEls.value = []
  if (scroller.value) scroller.value.scrollTop = 0
  load(true)
}

function doTagSearch() {
  if (!tagInput.value.trim()) { Toaster.info('输入标签名'); return }
  tab.value = 'tag'
  videos.value = []
  hasMore.value = false
  if (scroller.value) scroller.value.scrollTop = 0
  load(true)
}

// IntersectionObserver：可见的视频自动播放，离开暂停
function observeVideos() {
  if (observer) observer.disconnect()
  observer = new IntersectionObserver((entries) => {
    entries.forEach((e) => {
      const v = e.target
      const idx = Number(v.dataset.idx)
      if (e.isIntersecting && e.intersectionRatio > 0.6) {
        v.play().then(() => { playingMap.value[idx] = true }).catch(() => {})
      } else {
        v.pause()
        playingMap.value[idx] = false
      }
    })
  }, { root: scroller.value, threshold: [0, 0.6, 1] })
  videoEls.value.forEach((el) => el && observer.observe(el))
}

function setVideoEl(el, idx) {
  if (el) {
    el.dataset.idx = idx
    el.muted = muted.value
    videoEls.value[idx] = el
  }
}

// 点视频：暂停/播放切换
function togglePlay(idx) {
  const v = videoEls.value[idx]
  if (!v) return
  if (v.paused) {
    v.play().then(() => { playingMap.value[idx] = true }).catch(() => {})
  } else {
    v.pause()
    playingMap.value[idx] = false
  }
}

// 全局静音切换（所有视频同步）
function toggleMute() {
  muted.value = !muted.value
  videoEls.value.forEach((el) => {
    if (!el) return
    el.muted = muted.value
    if (!muted.value && !el.paused) {
      el.play().catch(() => {})
    }
  })
}

// 切换视图
function toggleView() {
  const toFeed = viewMode.value === 'grid'
  viewMode.value = toFeed ? 'feed' : 'grid'
  playingMap.value = {}
  // 先停掉旧 observer 和所有视频
  if (observer) { observer.disconnect(); observer = null }
  videoEls.value.forEach((el) => { if (el) el.pause() })
  videoEls.value = []
  if (toFeed) {
    nextTick(() => {
      setTimeout(() => { observeVideos() }, 50)
    })
  }
}

// 点头像：跳作者主页
function goAuthor(v) {
  router.push(`/profile/${v.author.id}`)
}

// 点 + 号：关注/取消关注
async function toggleFollow(v) {
  if (!auth.isLoggedIn) {
    Toaster.info('登录后才能关注')
    router.push('/login')
    return
  }
  const aid = v.author.id
  if (followedMap.value[aid]) {
    await unfollow(aid)
    followedMap.value[aid] = false
  } else {
    await follow(aid)
    followedMap.value[aid] = true
    Toaster.success('关注成功')
  }
}

// 评论抽屉
async function openComments(v) {
  activeVideo.value = v
  showComments.value = true
  commentLoading.value = true
  commentList.value = []
  try {
    commentList.value = await listComments(v.id)
  } catch {
    commentList.value = []
  } finally {
    commentLoading.value = false
  }
}

function closeComments() {
  showComments.value = false
  activeVideo.value = null
  commentList.value = []
  commentText.value = ''
}

async function sendComment() {
  const text = commentText.value.trim()
  if (!text || !activeVideo.value) return
  if (!auth.isLoggedIn) {
    Toaster.info('登录后才能评论')
    router.push('/login')
    return
  }
  try {
    await publishComment(activeVideo.value.id, text)
    commentText.value = ''
    commentList.value.unshift({ id: Date.now(), username: auth.username, content: text, created_at: Date.now() / 1000 })
    Toaster.success('评论成功')
  } catch (e) {
    Toaster.error(e.message || '评论失败')
  }
}

async function onLike(v) {
  if (!auth.isLoggedIn) {
    Toaster.info('登录后才能点赞')
    router.push('/login')
    return
  }
  try {
    if (v.is_liked) {
      await unlike(v.id)
      v.is_liked = false
      v.likes_count -= 1
    } else {
      await like(v.id)
      v.is_liked = true
      v.likes_count += 1
    }
  } catch (e) {
    Toaster.error(e.message || '操作失败')
  }
}

function goDetail(v) {
  router.push(`/video/${v.id}`)
}

onMounted(() => {
  load(true)
  // 加载已关注列表，填充 followedMap
  if (auth.isLoggedIn) {
    getAllVloggers().then((list) => {
      ;(list || []).forEach((u) => { followedMap.value[u.id] = true })
    }).catch(() => {})
  }
  // 底部触底加载
  if (bottomEl.value && scroller.value) {
    const io = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting) load()
    }, { root: scroller.value, rootMargin: '600px' })
    io.observe(bottomEl.value)
  }
})

onUnmounted(() => { if (observer) observer.disconnect() })
</script>

<template>
  <div class="feed-page">
    <!-- 顶部 tab（固定悬浮） -->
    <div class="tabs">
      <button :class="{ on: tab === 'latest' }" @click="switchTab('latest')">最新</button>
      <button :class="{ on: tab === 'hot' }" @click="switchTab('hot')">热门</button>
      <button :class="{ on: tab === 'following' }" @click="switchTab('following')">关注</button>
      <button v-if="auth.isLoggedIn" :class="{ on: tab === 'likes' }" @click="switchTab('likes')">赞最多</button>
      <div class="tag-search">
        <input v-model="tagInput" class="tag-input" placeholder="搜标签 #..." @keyup.enter="doTagSearch" />
        <button class="tag-btn" @click="doTagSearch">搜</button>
        <button class="view-toggle" @click="toggleView" :title="viewMode === 'feed' ? '切换网格视图' : '切换全屏视图'">
          {{ viewMode === 'feed' ? '▦' : '▶' }}
        </button>
      </div>
    </div>

    <!-- 滚动内容 -->
    <div ref="scroller" class="scroller" :class="{ snap: viewMode === 'feed' }">
      <div v-if="!videos.length && !loading" class="empty">
        <div class="big">🎬</div>
        <p v-if="tab === 'following' && !auth.isLoggedIn">登录后查看关注的人的视频</p>
        <p v-else>这里空空如也</p>
      </div>

      <!-- 网格卡片视图 -->
      <div v-if="viewMode === 'grid' && videos.length" class="tag-grid">
        <VideoCard v-for="v in videos" :key="v.id" :video="v" />
      </div>

      <!-- 全屏滑动流视图 -->
      <template v-else>
      <div v-for="(v, idx) in videos" :key="v.id" class="slide">
        <video
          :ref="(el) => setVideoEl(el, idx)"
          :src="v.play_url"
          class="vd"
          autoplay
          loop
          playsinline
          @click="togglePlay(idx)"
        ></video>

        <div v-if="playingMap[idx] === false" class="big-play" @click="togglePlay(idx)">▶</div>
        <button class="mute-btn" @click="toggleMute">{{ muted ? '🔇' : '🔊' }}</button>

        <div class="rail">
          <div class="avatar-wrap">
            <div class="avatar" @click.stop="goAuthor(v)">{{ v.author.username.slice(0,1).toUpperCase() }}</div>
            <button
              v-if="v.author.id !== auth.userId"
              class="follow-btn"
              :class="{ followed: followedMap[v.author.id] }"
              @click.stop="toggleFollow(v)"
            >{{ followedMap[v.author.id] ? '✓' : '+' }}</button>
          </div>
          <button class="rail-btn" :class="{ liked: v.is_liked }" @click.stop="onLike(v)">
            <span class="ic">♥</span>
            <span class="num">{{ v.likes_count }}</span>
          </button>
          <button class="rail-btn" @click.stop="openComments(v)">
            <span class="ic">💬</span>
            <span class="num">评论</span>
          </button>
        </div>

        <div class="info">
          <div class="author">@{{ v.author.username }}</div>
          <div class="title">{{ v.title }}</div>
          <div v-if="v.description" class="desc">{{ v.description }}</div>
        </div>
      </div>
      </template>

      <div ref="bottomEl" class="bottom">
        <div v-if="loading" class="spinner dark"></div>
        <span v-else-if="!hasMore && videos.length" class="end">— 到底了 —</span>
      </div>
    </div>

    <!-- 评论抽屉（右侧滑出） -->
    <transition name="slide-right">
      <div v-if="showComments" class="comment-drawer">
        <div class="cd-head">
          <span>评论 {{ commentList.length }}</span>
          <button class="cd-close" @click="closeComments">✕</button>
        </div>
        <div class="cd-list">
          <div v-if="commentLoading" class="cd-empty">加载中…</div>
          <div v-else-if="!commentList.length" class="cd-empty">还没有评论，抢沙发～</div>
          <div v-for="c in commentList" :key="c.id" class="cd-item">
            <div class="cd-avatar">{{ (c.username || '?').slice(0,1).toUpperCase() }}</div>
            <div class="cd-body">
              <div class="cd-user">{{ c.username }}</div>
              <div class="cd-text">{{ c.content }}</div>
            </div>
          </div>
        </div>
        <div class="cd-input">
          <input v-model="commentText" class="cd-input-field" placeholder="友善评论…" @keyup.enter="sendComment" />
          <button class="cd-send" @click="sendComment">发送</button>
        </div>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.feed-page { position: relative; height: calc(100vh - 60px); display: flex; flex-direction: column; background: #000; }

.tabs {
  display: flex; align-items: center; gap: 6px;
  padding: 10px 16px; z-index: 10;
  background: rgba(11, 13, 18, 0.85); backdrop-filter: blur(10px);
}
.tabs button {
  border: none; background: transparent;
  padding: 6px 14px; border-radius: 999px;
  font-size: 14px; font-weight: 600; color: var(--text-2);
}
.tabs button.on { background: var(--accent); color: #fff; }
.tag-search { margin-left: auto; display: flex; gap: 6px; }
.tag-input {
  width: 130px; padding: 6px 12px; border-radius: 999px;
  border: 1px solid var(--border); background: var(--surface); color: var(--text);
  font-size: 13px; outline: none;
}
.tag-btn {
  border: none; border-radius: 999px; padding: 6px 14px;
  background: var(--accent); color: #fff; font-size: 13px; font-weight: 600;
}

.scroller { flex: 1; overflow-y: auto; background: var(--bg); }
.scroller.snap { scroll-snap-type: y mandatory; background: #000; }
.view-toggle {
  border: 1px solid var(--border); background: var(--surface); color: var(--text);
  border-radius: 8px; padding: 6px 10px; cursor: pointer; font-size: 14px;
}

.slide {
  height: 100%; scroll-snap-align: start; scroll-snap-stop: always;
  position: relative; background: #000;
  display: flex; align-items: center; justify-content: center;
  overflow: hidden;
}
.vd {
  max-height: 100%;
  max-width: 100%;
  width: auto;
  object-fit: contain;
  background: #000;
}

.rail {
  position: absolute; right: 14px; bottom: 100px;
  display: flex; flex-direction: column; align-items: center; gap: 18px;
}
.avatar {
  width: 46px; height: 46px; border-radius: 50%;
  background: linear-gradient(135deg, #0ea5e9, #22c55e);
  color: #fff; font-weight: 800; font-size: 18px;
  display: flex; align-items: center; justify-content: center;
  border: 2px solid #fff; cursor: pointer;
}
.avatar-wrap { position: relative; }
.follow-btn {
  position: absolute; left: 50%; bottom: -8px; transform: translateX(-50%);
  width: 22px; height: 22px; border-radius: 50%;
  border: none; background: #fe2c55; color: #fff;
  font-size: 14px; font-weight: 800; line-height: 1;
  display: flex; align-items: center; justify-content: center;
  cursor: pointer;
}
.follow-btn.followed { background: #536471; }
.rail-btn {
  border: none; background: transparent; color: #fff;
  display: flex; flex-direction: column; align-items: center; gap: 2px;
  cursor: pointer;
}
.rail-btn .ic { font-size: 30px; filter: drop-shadow(0 2px 4px rgba(0,0,0,0.5)); }
.rail-btn.liked .ic { color: #ef4444; }
.rail-btn .num { font-size: 12px; font-weight: 600; }

.info {
  position: absolute; left: 16px; right: 90px; bottom: 30px;
  color: #fff; text-shadow: 0 1px 4px rgba(0,0,0,0.7);
}
.author { font-weight: 800; font-size: 15px; margin-bottom: 4px; }
.title { font-size: 16px; font-weight: 700; margin-bottom: 4px; }
.desc { font-size: 13px; opacity: 0.85; }

.big-play {
  position: absolute; left: 50%; top: 50%;
  transform: translate(-50%, -50%);
  width: 72px; height: 72px; border-radius: 50%;
  background: rgba(0,0,0,0.5); color: #fff;
  font-size: 32px; display: flex; align-items: center; justify-content: center;
  cursor: pointer; z-index: 5; padding-left: 6px;
}
.mute-btn {
  position: absolute; right: 16px; bottom: 16px;
  width: 40px; height: 40px; border-radius: 50%;
  border: none; background: rgba(0,0,0,0.5); color: #fff;
  font-size: 18px; cursor: pointer; z-index: 5;
}

.bottom { display: flex; justify-content: center; align-items: center; min-height: 80px; }
.end { color: var(--text-2); font-size: 13px; }
.empty { height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center; color: var(--text-2); }
.empty .big { font-size: 48px; margin-bottom: 12px; }
.spinner.dark { border-color: rgba(255,255,255,0.2); border-top-color: #fff; }

.tag-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 18px;
  padding: 20px;
}

/* 评论抽屉 */
.comment-drawer {
  position: absolute; right: 0; top: 0; bottom: 0;
  width: 360px; max-width: 90%;
  background: var(--surface); border-left: 1px solid var(--border);
  display: flex; flex-direction: column; z-index: 20;
}
.cd-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 16px; border-bottom: 1px solid var(--border);
  font-weight: 700; font-size: 14px;
}
.cd-close { border: none; background: none; color: var(--text-2); font-size: 16px; cursor: pointer; }
.cd-list { flex: 1; overflow-y: auto; padding: 12px 16px; }
.cd-empty { text-align: center; color: var(--text-2); padding: 30px 0; font-size: 13px; }
.cd-item { display: flex; gap: 10px; margin-bottom: 16px; }
.cd-avatar {
  width: 32px; height: 32px; border-radius: 50%; flex-shrink: 0;
  background: linear-gradient(135deg, #0ea5e9, #22c55e);
  color: #fff; font-size: 14px; font-weight: 700;
  display: flex; align-items: center; justify-content: center;
}
.cd-user { font-weight: 600; font-size: 13px; margin-bottom: 2px; }
.cd-text { font-size: 13px; color: var(--text); line-height: 1.4; }
.cd-input {
  display: flex; gap: 8px; padding: 12px 16px;
  border-top: 1px solid var(--border);
}
.cd-input-field {
  flex: 1; border: 1px solid var(--border); border-radius: 10px;
  padding: 8px 12px; background: var(--bg); color: var(--text); font-size: 13px; outline: none;
}
.cd-send {
  border: none; border-radius: 10px; padding: 8px 16px;
  background: var(--accent); color: #fff; font-size: 13px; font-weight: 600; cursor: pointer;
}
.slide-right-enter-active, .slide-right-leave-active { transition: transform 0.25s ease; }
.slide-right-enter-from, .slide-right-leave-to { transform: translateX(100%); }
</style>
