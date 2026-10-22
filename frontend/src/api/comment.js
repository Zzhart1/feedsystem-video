import { postJson } from './client'

export function listComments(video_id) {
  return postJson('/comment/listAll', { video_id }).then((res) => {
    // 兼容两种返回：数组直接返回 / {comments:[...]}
    if (Array.isArray(res)) return res
    return res.comments || res.comment_list || []
  })
}

export function publishComment(video_id, content) {
  return postJson('/comment/publish', { video_id, content }, { authRequired: true })
}

export function deleteComment(comment_id) {
  return postJson('/comment/delete', { comment_id }, { authRequired: true })
}
