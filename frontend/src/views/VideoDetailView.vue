<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getDetail, deleteVideo } from '../api/video'
import { like, unlike, isLiked } from '../api/like'
import { listComments, publishComment } from '../api/comment'
import { follow, unfollow, getAllVloggers } from '../api/social'
import { useAuthStore } from '../stores/auth'
import { formatTime, formatCount } from '../api/normalize'
import UserAvatar from '../components/UserAvatar.vue'
import CommentItem from '../components/CommentItem.vue'
import Toaster from '../utils/toast'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const video = ref(null)
const liked = ref(false)
const likeCount = ref(0)
const comments = ref([])
const commentText = ref('')
const following = ref(false)
const loading = ref(true)
const commentLoading = ref(false)

const videoId = computed(() => Number(route.params.id))
const isMine = computed(() => video.value && auth.isLoggedIn && video.value.author_id === auth.userId)

async function load() {
  loading.value = true
  try {
    video.value = await getDetail(videoId.value)
    likeCount.value = video.value.likes_count
    if (auth.isLoggedIn) {
      isLiked(videoId.value).then((ok) => (liked.value = ok)).catch(() => {})
      // 查询是否已关注作者
      getAllVloggers().then((list) => {
        following.value = (list || []).some((u) => u.id === video.value.author_id)
      }).catch(() => {})
    }
    await loadComments()
  } catch (e) {
    Toaster.error(e.message || '视频加载失败')
    router.push('/')
    return
  } finally {
    loading.value = false
  }
}

async function loadComments() {
  try {
    comments.value = await listComments(videoId.value)
  } catch {
    comments.value = []
  }
}

async function toggleLike() {
  if (!auth.isLoggedIn) {
    Toaster.info('登录后才能点赞')
    router.push('/login')
    return
  }
  try {
    if (liked.value) {
      await unlike(videoId.value)
      liked.value = false
      likeCount.value -= 1
    } else {
      await like(videoId.value)
      liked.value = true
      likeCount.value += 1
    }
  } catch (e) {
    Toaster.error(e.message || '操作失败')
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
      await unfollow(video.value.author_id)
      following.value = false
    } else {
      await follow(video.value.author_id)
      following.value = true
    }
  } catch (e) {
    Toaster.error(e.message || '操作失败')
  }
}

async function submitComment() {
  if (!auth.isLoggedIn) {
    Toaster.info('登录后才能评论')
    router.push('/login')
    return
  }
  const text = commentText.value.trim()
  if (!text) return
  commentLoading.value = true
  try {
    await publishComment(videoId.value, text)
    commentText.value = ''
    Toaster.success('评论成功')
    await loadComments()
  } catch (e) {
    Toaster.error(e.message || '评论失败')
  } finally {
    commentLoading.value = false
  }
}

async function onDeleteVideo() {
  if (!confirm('确定删除这个视频吗？')) return
  try {
    await deleteVideo(videoId.value)
    Toaster.success('已删除')
    router.push('/profile')
  } catch (e) {
    Toaster.error(e.message || '删除失败')
  }
}

watch(videoId, () => { liked.value = false; following.value = false; load() })
onMounted(load)
</script>

<template>
  <div class="page">
    <div v-if="loading" class="empty"><div class="spinner dark"></div></div>

    <div v-else-if="video" class="detail">
      <div class="player card">
        <video v-if="video.play_url" :src="video.play_url" controls autoplay class="vd" preload="metadata"></video>
        <div v-else class="vd placeholder">视频资源为空</div>
      </div>

      <div class="head card">
        <h1 class="title">{{ video.title || '未命名视频' }}</h1>
        <p v-if="video.description" class="desc">{{ video.description }}</p>

        <div class="author-row">
          <div class="author">
            <UserAvatar :name="video.username" :size="40" />
            <div>
              <router-link :to="`/profile/${video.author_id}`" class="aname">{{ video.username }}</router-link>
              <div class="atime">{{ formatTime(video.created_at) }} · ♥ {{ formatCount(likeCount) }}</div>
            </div>
          </div>
          <div class="actions">
            <button v-if="!isMine" class="btn sm" :class="{ liked: liked }" @click="toggleLike">
              {{ liked ? '♥ 已赞' : '♡ 点赞' }}
            </button>
            <button v-if="isMine" class="btn danger sm" @click="onDeleteVideo">删除视频</button>
            <button v-if="!isMine && auth.isLoggedIn" class="btn ghost sm" :class="{ fwd: following }" @click="toggleFollow">
              {{ following ? '已关注' : '+ 关注' }}
            </button>
          </div>
        </div>
      </div>

      <div class="comments card">
        <h3>评论（{{ comments.length }}）</h3>
        <form class="cform" @submit.prevent="submitComment">
          <input v-model="commentText" class="input" placeholder="友善评论…" />
          <button class="btn sm" type="submit" :disabled="commentLoading">发送</button>
        </form>
        <div v-if="comments.length" class="clist">
          <CommentItem
            v-for="c in comments"
            :key="c.id"
            :comment="c"
            @deleted="comments = comments.filter((x) => x.id !== $event)"
          />
        </div>
        <div v-else class="empty" style="padding: 24px 0">还没有评论，抢沙发～</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.detail { display: flex; flex-direction: column; gap: 16px; }
.player { overflow: hidden; }
.vd { width: 100%; aspect-ratio: 16 / 9; background: #000; object-fit: contain; }
.placeholder { display: flex; align-items: center; justify-content: center; color: var(--text-2); }
.head { padding: 18px 22px; }
.title { font-size: 20px; font-weight: 700; }
.desc { color: var(--text-2); margin-top: 6px; }
.author-row { display: flex; align-items: center; justify-content: space-between; margin-top: 16px; gap: 12px; flex-wrap: wrap; }
.author { display: flex; align-items: center; gap: 12px; }
.aname { font-weight: 700; font-size: 15px; }
.atime { color: var(--text-2); font-size: 12.5px; }
.actions { display: flex; gap: 10px; }
.btn.liked { background: linear-gradient(135deg, #ef4444, #f97316); }
.btn.fwd { background: transparent; border: 1px solid var(--accent); color: var(--accent); }
.comments { padding: 18px 22px 22px; }
.comments h3 { font-size: 15px; margin-bottom: 14px; }
.cform { display: flex; gap: 10px; margin-bottom: 12px; }
.clist { border-top: 1px solid var(--border); }
</style>
