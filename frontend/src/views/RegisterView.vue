<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { register } from '../api/account'
import Toaster from '../utils/toast'

const router = useRouter()
const username = ref('')
const password = ref('')
const confirm = ref('')
const loading = ref(false)

async function onSubmit() {
  if (!username.value || !password.value) {
    Toaster.error('请填写用户名和密码')
    return
  }
  if (password.value.length < 6) {
    Toaster.error('密码至少 6 位')
    return
  }
  if (password.value !== confirm.value) {
    Toaster.error('两次密码不一致')
    return
  }
  loading.value = true
  try {
    await register(username.value, password.value)
    Toaster.success('注册成功，请登录')
    router.push('/login')
  } catch (e) {
    Toaster.error(e.message || '注册失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-wrap">
    <div class="auth-card card">
      <h1>注册账号</h1>
      <p class="sub">创建你的 FeedFlow 账号</p>
      <form @submit.prevent="onSubmit">
        <label>用户名</label>
        <input v-model="username" class="input" placeholder="3-20 个字符" />
        <label>密码</label>
        <input v-model="password" type="password" class="input" placeholder="至少 6 位" />
        <label>确认密码</label>
        <input v-model="confirm" type="password" class="input" placeholder="再输一遍密码" />
        <button class="btn" type="submit" :disabled="loading">
          {{ loading ? '注册中…' : '注 册' }}
        </button>
      </form>
      <div class="foot">
        已有账号？
        <router-link to="/login">去登录</router-link>
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
