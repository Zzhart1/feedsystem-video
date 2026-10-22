import { postForm, postJson } from './client'
import { normalizeVideo, normalizeVideoList } from './normalize'

export function publishVideo({ title, description, play_url, cover_url, tags }) {
  const body = { title, description, play_url, cover_url }
  if (tags && tags.length) body.tags = tags
  return postJson('/video/publish', body, { authRequired: true }).then(normalizeVideo)
}

export function uploadVideo(file) {
  const fd = new FormData()
  fd.append('file', file)
  return postForm('/video/uploadVideo', fd, { authRequired: true })
}

export function uploadCover(file) {
  const fd = new FormData()
  fd.append('file', file)
  return postForm('/video/uploadCover', fd, { authRequired: true })
}

export function deleteVideo(id) {
  return postJson('/video/delete', { id }, { authRequired: true })
}

export async function listByAuthorId(authorId) {
  const res = await postJson('/video/listByAuthorID', { author_id: authorId })
  return normalizeVideoList(res)
}

export async function getDetail(id) {
  const res = await postJson('/video/getDetail', { id })
  return normalizeVideo(res)
}

// ---- 分片上传 + 断点续传 ----
export function initChunkUpload({ file_name, file_size, chunk_size, total_chunks, file_hash }) {
  return postJson('/video/chunk/init', {
    file_name, file_size, chunk_size, total_chunks, file_hash,
  }, { authRequired: true })
}

export function uploadChunk(uploadId, chunkIndex, chunkHash, blob) {
  const fd = new FormData()
  fd.append('upload_id', uploadId)
  fd.append('chunk_index', String(chunkIndex))
  fd.append('chunk_hash', chunkHash)
  fd.append('file', blob)
  return postForm('/video/chunk/upload', fd, { authRequired: true })
}

export function chunkStatus(uploadId) {
  return postJson('/video/chunk/status', { upload_id: uploadId }, { authRequired: true })
}

export function completeChunkUpload(uploadId) {
  return postJson('/video/chunk/complete', { upload_id: uploadId }, { authRequired: true })
}
