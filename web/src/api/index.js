import axios from 'axios'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'

let _suppressAuthRedirect = false
export function suppressAuthRedirect(v) { _suppressAuthRedirect = v }

const api = axios.create({
  baseURL: '',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('access_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

api.interceptors.response.use(
  (response) => {
    return response
  },
  async (error) => {
    const originalRequest = error.config

    if (error.response?.status === 401 && !originalRequest._retry && !originalRequest._skipAuthRetry) {
      originalRequest._retry = true

      const refreshToken = localStorage.getItem('refresh_token')
      if (refreshToken) {
        try {
          const res = await api.post('/api/auth/refresh',
            { refresh_token: refreshToken },
            { _skipAuthRetry: true }
          )
          if (res.data.success) {
            const authStore = useAuthStore()
            authStore.setTokens(res.data.data)
            originalRequest.headers.Authorization = `Bearer ${res.data.data.access_token}`
            return api(originalRequest)
          }
        } catch (e) {
          // Refresh failed
        }
      }

      const authStore = useAuthStore()
      authStore.logout()
      if (!_suppressAuthRedirect) {
        router.push({ name: 'Login', query: { redirect: router.currentRoute.value.fullPath } })
      }
    }

    return Promise.reject(error)
  }
)

export async function downloadBlob(params, filename) {
  const res = await api.get('/download', {
    params,
    responseType: 'blob',
    timeout: 300000,
  })
  const url = window.URL.createObjectURL(new Blob([res.data]))
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.style.display = 'none'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  window.URL.revokeObjectURL(url)
}

export default api
