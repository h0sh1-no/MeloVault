import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const accessToken = ref(localStorage.getItem('access_token') || '')
  const refreshToken = ref(localStorage.getItem('refresh_token') || '')

  const isLoggedIn = computed(() => !!accessToken.value && !!user.value)

  async function loadSettings() {
    try {
      const { useSettingsStore } = await import('@/stores/settings')
      const settingsStore = useSettingsStore()
      await settingsStore.fetch()
    } catch {
      // settings sync is best-effort
    }
  }

  async function login(email, password) {
    const res = await api.post('/api/auth/login', { email, password })
    if (res.data.success) {
      setTokens(res.data.data.tokens)
      user.value = res.data.data.user
      loadSettings()
    }
    return res.data
  }

  async function register(username, email, password) {
    const res = await api.post('/api/auth/register', { username, email, password })
    if (res.data.success) {
      setTokens(res.data.data.tokens)
      user.value = res.data.data.user
      loadSettings()
    }
    return res.data
  }

  async function fetchUser() {
    if (!accessToken.value) return null
    try {
      const res = await api.get('/api/auth/me')
      if (res.data.success) {
        user.value = res.data.data
        loadSettings()
        return user.value
      }
    } catch (e) {
      logout()
    }
    return null
  }

  async function refresh() {
    if (!refreshToken.value) {
      logout()
      return false
    }
    try {
      const res = await api.post('/api/auth/refresh', { refresh_token: refreshToken.value })
      if (res.data.success) {
        setTokens(res.data.data)
        return true
      }
    } catch (e) {
      logout()
    }
    return false
  }

  function setTokens(tokens) {
    accessToken.value = tokens.access_token
    refreshToken.value = tokens.refresh_token
    localStorage.setItem('access_token', tokens.access_token)
    localStorage.setItem('refresh_token', tokens.refresh_token)
  }

  function logout() {
    user.value = null
    accessToken.value = ''
    refreshToken.value = ''
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    import('@/stores/settings').then(({ useSettingsStore }) => {
      useSettingsStore().reset()
    }).catch(() => {})
  }

  function init() {
    if (accessToken.value) {
      fetchUser()
    }
  }

  return {
    user,
    accessToken,
    refreshToken,
    isLoggedIn,
    login,
    register,
    fetchUser,
    refresh,
    setTokens,
    logout,
    init
  }
})
