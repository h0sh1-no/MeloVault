import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'
import { usePlayerStore } from '@/stores/player'

export const useSettingsStore = defineStore('settings', () => {
  const settings = ref(null)
  const loading = ref(false)
  const loaded = ref(false)

  const streamingQuality = computed(() => settings.value?.streaming_quality ?? 'jymaster')
  const downloadQuality = computed(() => settings.value?.download_quality ?? 'jymaster')
  const volume = computed(() => settings.value?.volume ?? 0.8)
  const repeatMode = computed(() => settings.value?.repeat_mode ?? 'none')

  async function fetch() {
    const authStore = useAuthStore()
    if (!authStore.isLoggedIn) return

    loading.value = true
    try {
      const res = await api.get('/api/user/settings')
      if (res.data.success) {
        settings.value = res.data.data
        applyToPlayer()
        loaded.value = true
      }
    } catch {
      // Use localStorage fallback
    } finally {
      loading.value = false
    }
  }

  async function update(partial) {
    const authStore = useAuthStore()
    if (!authStore.isLoggedIn) {
      applyLocalOnly(partial)
      return
    }

    try {
      const res = await api.put('/api/user/settings', partial)
      if (res.data.success) {
        settings.value = res.data.data
        applyToPlayer()
      }
    } catch {
      applyLocalOnly(partial)
    }
  }

  function applyToPlayer() {
    if (!settings.value) return
    const playerStore = usePlayerStore()
    if (settings.value.streaming_quality) {
      playerStore.setStreamingQuality(settings.value.streaming_quality)
    }
    if (settings.value.download_quality) {
      playerStore.setDownloadQuality(settings.value.download_quality)
    }
    if (typeof settings.value.volume === 'number') {
      playerStore.setVolume(settings.value.volume)
    }
    if (settings.value.repeat_mode) {
      playerStore.repeatMode = settings.value.repeat_mode
    }
  }

  function applyLocalOnly(partial) {
    const playerStore = usePlayerStore()
    if (partial.streaming_quality) playerStore.setStreamingQuality(partial.streaming_quality)
    if (partial.download_quality) playerStore.setDownloadQuality(partial.download_quality)
    if (typeof partial.volume === 'number') playerStore.setVolume(partial.volume)
    if (partial.repeat_mode) playerStore.repeatMode = partial.repeat_mode
  }

  function reset() {
    settings.value = null
    loaded.value = false
  }

  return {
    settings,
    loading,
    loaded,
    streamingQuality,
    downloadQuality,
    volume,
    repeatMode,
    fetch,
    update,
    reset,
    applyToPlayer
  }
})
