<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { login } from '../api/account'
import { findByUsername } from '../api/account'
import { useAuthStore } from '../stores/auth'
import Toaster from '../utils/toast'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const username = ref('')
const password = ref('')
const loading = ref(false)

async function onSubmit() {
  if (!username.value || !password.value) {
    Toaster.error('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    const res = await login(username.value, password.value)
    // 登录返回 {token, refresh_token}；顺带拉一次账号信息存到 store
    let account = null
    try {
      const info = await findByUsername(username.value)
      account = info.account || null
    } catch { /* 拉不到账号信息不阻塞登录 */ }
    auth.setTokens({ token: res.token, refresh_token: res.refresh_token, account })
    Toaster.success(`欢迎回来，${account?.username || username.value}`)
    router.push(route.query.redirect || '/')
  } catch (e) {
    Toaster.error(e.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-wrap">
    <div class="auth-card card">
      <h1>登录 FeedFlow</h1>
      <p class="sub">短视频 Feed 社区 · Go + Redis + RabbitMQ 全栈练习项目</p>
      <form @submit.prevent="onSubmit">
        <label>用户名</label>
        <input v-model="username" class="input" placeholder="输入用户名" autocomplete="username" />
        <label>密码</label>
        <input v-model="password" type="password" class="input" placeholder="输入密码" autocomplete="current-password" />
        <button class="btn" type="submit" :disabled="loading">
          {{ loading ? '登录中…' : '登 录' }}
        </button>
      </form>
      <div class="foot">
        还没有账号？
        <router-link to="/register">去注册</router-link>
      </div>
    </div>
  </div>
</template>

<style scoped>
.auth-wrap { min-height: calc(100vh - 60px); display: flex; align-items: center; justify-content: center; padding: 24px; }
.auth-card { width: 380px; padding: 34px 34px 28px; }
h1 { font-size: 22px; margin-bottom: 6px; }
.sub { color: var(--text-2); font-size: 13px; margin-bottom: 24px; }
label { display: block; font-size: 13px; font-weight: 600; margin: 14px 0 6px; color: var(--text-2); }
.btn { width: 100%; margin-top: 22px; padding: 11px; }
.foot { margin-top: 18px; text-align: center; font-size: 13px; color: var(--text-2); }
.foot a { color: var(--accent); font-weight: 600; }
</style>
