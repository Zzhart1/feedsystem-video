import { defineStore } from 'pinia'

const LS_KEY = 'feedsystem_auth'

function load() {
  try {
    const raw = localStorage.getItem(LS_KEY)
    return raw ? JSON.parse(raw) : { token: '', refresh_token: '', account: null }
  } catch {
    return { token: '', refresh_token: '', account: null }
  }
}

export const useAuthStore = defineStore('auth', {
  state: () => load(),
  getters: {
    isLoggedIn: (s) => !!s.token,
    userId: (s) => s.account?.id ?? 0,
    username: (s) => s.account?.username ?? '',
  },
  actions: {
    persist() {
      localStorage.setItem(LS_KEY, JSON.stringify({
        token: this.token,
        refresh_token: this.refresh_token,
        account: this.account,
      }))
    },
    setTokens({ token, refresh_token, account }) {
      this.token = token || ''
      if (refresh_token) this.refresh_token = refresh_token
      if (account) this.account = account
      this.persist()
    },
    setAccount(account) {
      this.account = account
      this.persist()
    },
    clearTokens() {
      this.token = ''
      this.refresh_token = ''
      this.account = null
      this.persist()
    },
  },
})
