<script setup>
import { useAuthStore } from '../stores/auth'
import { deleteComment } from '../api/comment'
import { formatTime } from '../api/normalize'
import Toaster from '../utils/toast'

const props = defineProps({
  comment: { type: Object, required: true },
})
const emit = defineEmits(['deleted'])
const auth = useAuthStore()

async function onDelete() {
  try {
    await deleteComment(props.comment.id)
    Toaster.success('评论已删除')
    emit('deleted', props.comment.id)
  } catch (e) {
    Toaster.error(e.message || '删除失败')
  }
}
</script>

<template>
  <div class="cmt">
    <span class="uname">{{ comment.username || '用户' }}</span>
    <span class="txt">{{ comment.content }}</span>
    <span class="time">{{ formatTime(comment.created_at) }}</span>
    <button
      v-if="auth.isLoggedIn && (auth.userId === comment.author_id)"
      class="del"
      @click="onDelete"
    >删除</button>
  </div>
</template>

<style scoped>
.cmt { display: flex; align-items: baseline; gap: 8px; padding: 9px 0; }
.uname { font-weight: 700; color: #0369a1; white-space: nowrap; }
.txt { flex: 1; word-break: break-word; }
.time { color: var(--text-2); font-size: 12px; white-space: nowrap; }
.del { border: none; background: none; color: var(--danger); font-size: 12px; padding: 2px 6px; border-radius: 6px; }
.del:hover { background: rgba(239, 68, 68, 0.15); }
</style>
