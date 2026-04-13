import { ref } from 'vue'
import { defineStore } from 'pinia'
import api from '@/api'

export const useSiteSettingsStore = defineStore('siteSettings', () => {
  const features = ref({
    playlist_parse_enabled: true,
    playlist_parse_admin_only: false,
    album_parse_enabled: true,
    album_parse_admin_only: false,
    allow_register: true,
    allow_email_register: true,
    allow_linuxdo_register: true,
    allow_email_login: true,
    allow_linuxdo_login: true,
    linuxdo_configured: false,
    smtp_configured: false,
    site_url: '',
  })
  const loaded = ref(false)

  async function fetch() {
    try {
      const res = await api.get('/api/site-settings')
      if (res.data.success && res.data.data) {
        features.value = res.data.data
      }
      loaded.value = true
    } catch {
      loaded.value = true
    }
  }

  async function update(partial) {
    const res = await api.put('/api/admin/site-settings', partial)
    if (res.data.success && res.data.data) {
      features.value = res.data.data
    }
    return res.data
  }

  return { features, loaded, fetch, update }
})
