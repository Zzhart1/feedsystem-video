import { postJson } from './client'
import { normalizeFeedVideoList } from './normalize'

export function listLatest({ limit, latest_time }) {
  return postJson('/feed/listLatest', { limit, latest_time }).then((res) => ({
    ...res,
    video_list: normalizeFeedVideoList(res.video_list),
  }))
}

export function listLikesCount({ limit, likes_count_before, id_before }) {
  const body = { limit }
  if (likes_count_before !== undefined && likes_count_before !== null) {
    body.likes_count_before = likes_count_before ?? 0
    body.id_before = id_before ?? 0
  }
  return postJson('/feed/listLikesCount', body).then((res) => ({
    ...res,
    video_list: normalizeFeedVideoList(res.video_list),
  }))
}

export function listByPopularity({ limit, as_of = 0, offset = 0 }) {
  return postJson('/feed/listByPopularity', { limit, as_of, offset }).then((res) => ({
    ...res,
    video_list: normalizeFeedVideoList(res.video_list),
  }))
}

export function listByFollowing({ limit, latest_time = 0 }) {
  return postJson('/feed/listByFollowing', { limit, latest_time }, { authRequired: true }).then((res) => ({
    ...res,
    video_list: normalizeFeedVideoList(res.video_list),
  }))
}

export function listByTag(tag_name, limit = 10) {
  return postJson('/feed/listByTag', { tag_name, limit }).then((res) => ({
    video_list: normalizeFeedVideoList(res.video_list),
  }))
}
