<script setup>
import { ref, onMounted } from 'vue'
import { changePassword, updateProfile, findByUsername } from '../api/account'
import { useAuthStore } from '../stores/auth'
import Toaster from '../utils/toast'

const auth = useAuthStore()

const newUsername = ref('')
const bio = ref('')
const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')

onMounted(async () => {
  newUsername.value = auth.username
  if (!auth.account) {
    try {
      const res = await findByUsername(auth.username)
      const acc = res.account || res
      auth.setAccount(acc)
      bio.value = acc.bio || ''
    } catch { /* 忽略 */ }
  } else {
    bio.value = auth.account.bio || ''
  }
})

async function saveProfile() {
  try {
    const payload = {}
    if (newUsername.value && newUsername.value !== auth.username) payload.new_username = newUsername.value
    if (bio.value) payload.bio = bio.value
    const res = await updateProfile(payload)
    const acc = res.account || res
    auth.setAccount({ ...auth.account, ...acc, username: acc.username || newUsername.value })
    auth.setTokens({ token: auth.token, refresh_token: auth.refresh_token, account: auth.account })
    Toaster.success('资料已更新')
  } catch (e) {
    Toaster.error(e.message || '更新失败')
  }
}

async function savePassword() {
  if (!oldPassword.value || !newPassword.value) {
    Toaster.error('请填写旧密码和新密码')
    return
  }
  if (newPassword.value.length < 6) {
    Toaster.error('新密码至少 6 位')
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    Toaster.error('两次新密码不一致')
    return
  }
  try {
    await changePassword(oldPassword.value, newPassword.value)
    Toaster.success('密码已修改，请重新登录')
    auth.clearTokens()
  } catch (e) {
    Toaster.error(e.message || '修改失败')
  }
}
</script>

<template>
  <div class="page">
    <div class="card box">
      <h1>设置</h1>

      <section>
        <h2>个人资料</h2>
        <label>用户名</label>
        <input v-model="newUsername" class="input" />
        <label>个人简介</label>
        <textarea v-model="bio" class="input ta" rows="3" placeholder="介绍一下自己…"></textarea>
        <button class="btn" @click="saveProfile">保存资料</button>
      </section>

      <hr />

      <section>
        <h2>修改密码</h2>
        <label>旧密码</label>
        <input v-model="oldPassword" type="password" class="input" />
        <label>新密码</label>
        <input v-model="newPassword" type="password" class="input" />
        <label>确认新密码</label>
        <input v-model="confirmPassword" type="password" class="input" />
        <button class="btn" @click="savePassword">修改密码</button>
      </section>
    </div>
  </div>
</template>

<style scoped>
.box { max-width: 520px; margin: 0 auto; padding: 28px 30px; }
h1 { font-size: 20px; margin-bottom: 10px; }
section h2 { font-size: 15px; margin: 18px 0 4px; }
label { display: block; font-size: 13px; font-weight: 600; color: var(--text-2); margin: 12px 0 6px; }
.ta { resize: vertical; }
.btn { margin-top: 16px; }
hr { border: none; border-top: 1px solid var(--border); margin: 24px 0; }
</style>
