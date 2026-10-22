import { postJson } from './client'
import { normalizeVideoList } from './normalize'

export function follow(target_account_id) {
  return postJson('/social/follow', { vlogger_id: target_account_id }, { authRequired: true })
}

export function unfollow(target_account_id) {
  return postJson('/social/unfollow', { vlogger_id: target_account_id }, { authRequired: true })
}

export async function getAllFollowers() {
  const res = await postJson('/social/getAllFollowers', {}, { authRequired: true })
  return res.followers || []
}

export async function getAllVloggers() {
  const res = await postJson('/social/getAllVloggers', {}, { authRequired: true })
  return res.vloggers || []
}

export async function getCounts() {
  const res = await postJson('/social/getCounts', {}, { authRequired: true })
  return {
    follower_count: Number(res.follower_count ?? res.followers_count ?? 0),
    following_count: Number(res.following_count ?? res.vloggers_count ?? 0),
    like_count: Number(res.like_count ?? res.likes_count ?? 0),
  }
}
