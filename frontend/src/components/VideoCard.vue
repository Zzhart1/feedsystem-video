<script setup>
import { formatTime, formatCount } from '../api/normalize'

defineProps({
  video: { type: Object, required: true },
})
</script>

<template>
  <router-link :to="`/video/${video.id}`" class="vcard">
    <div class="cover-wrap">
      <img v-if="video.cover_url" :src="video.cover_url" class="cover" loading="lazy" alt="" />
      <div v-else class="cover placeholder">▶</div>
      <div class="play-ic">▶</div>
      <div v-if="video.is_liked" class="liked-tag">已赞</div>
    </div>
    <div class="info">
      <div class="title" :title="video.title">{{ video.title || '未命名视频' }}</div>
      <div class="meta">
        <span class="author">{{ video.author?.username || video.username || '未知用户' }}</span>
        <span class="dot">·</span>
        <span class="likes">♥ {{ formatCount(video.likes_count) }}</span>
        <span class="time">{{ formatTime(video.create_time || video.created_at) }}</span>
      </div>
    </div>
  </router-link>
</template>

<style scoped>
.vcard {
  display: block; background: var(--surface);
  border: 1px solid var(--border); border-radius: var(--radius);
  overflow: hidden; box-shadow: var(--shadow);
  transition: transform 0.14s ease, box-shadow 0.14s ease;
}
.vcard:hover { transform: translateY(-3px); box-shadow: 0 8px 28px rgba(16, 24, 40, 0.12); }
.cover-wrap { position: relative; aspect-ratio: 16 / 9; background: #0f172a; }
.cover { width: 100%; height: 100%; object-fit: cover; }
.placeholder {
  width: 100%; height: 100%;
  display: flex; align-items: center; justify-content: center;
  color: rgba(255,255,255,0.5); font-size: 40px;
}
.play-ic {
  position: absolute; left: 50%; top: 50%; transform: translate(-50%, -50%);
  width: 44px; height: 44px; border-radius: 50%;
  background: rgba(15, 23, 42, 0.55); color: #fff;
  display: flex; align-items: center; justify-content: center;
  font-size: 16px; opacity: 0; transition: opacity 0.15s ease;
}
.vcard:hover .play-ic { opacity: 1; }
.liked-tag {
  position: absolute; top: 8px; right: 8px;
  background: rgba(239, 68, 68, 0.9); color: #fff;
  font-size: 11px; padding: 2px 8px; border-radius: 999px;
}
.info { padding: 10px 12px 12px; }
.title {
  font-weight: 600; font-size: 14px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.meta { display: flex; align-items: center; gap: 4px; margin-top: 5px; font-size: 12px; color: var(--text-2); }
.author { font-weight: 600; color: var(--text); }
.time { margin-left: auto; }
</style>
