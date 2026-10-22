import { postJson } from './client'

export function register(username, password) {
  return postJson('/account/register', { username, password })
}

export function login(username, password) {
  return postJson('/account/login', { username, password })
}

export function refresh(refresh_token) {
  return postJson('/account/refresh', { refresh_token })
}

export function logout() {
  return postJson('/account/logout', {}, { authRequired: true })
}

export function changePassword(old_password, new_password) {
  return postJson('/account/changePassword', { old_password, new_password }, { authRequired: true })
}

export function updateProfile(payload) {
  return postJson('/account/updateProfile', payload, { authRequired: true })
}

export function rename(new_username) {
  return postJson('/account/rename', { new_username }, { authRequired: true })
}

export function findByID(id) {
  return postJson('/account/findByID', { id })
}

export function findByUsername(username) {
  return postJson('/account/findByUsername', { username })
}
