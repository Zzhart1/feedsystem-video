import { postJson } from './client'
import { normalizeVideoList } from './normalize'

export function like(video_id) {
  return postJson('/like/like', { video_id }, { authRequired: true })
}

export function unlike(video_id) {
  return postJson('/like/unlike', { video_id }, { authRequired: true })
}

export function isLiked(video_id) {
  return postJson('/like/isLiked', { video_id }, { authRequired: true }).then((res) => !!res.is_liked)
}

export async function listMyLikedVideos() {
  const res = await postJson('/like/listMyLikedVideos', {}, { authRequired: true })
  return normalizeVideoList(res)
}
