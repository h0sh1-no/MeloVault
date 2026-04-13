import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'

const STORAGE_KEY = 'melovault-theme'

export const useThemeStore = defineStore('theme', () => {
  const saved = localStorage.getItem(STORAGE_KEY)
  const theme = ref(saved || 'night')

  const isDayMode = computed(() => theme.value === 'day')
  const isNightMode = computed(() => theme.value === 'night')

  function setTheme(value) {
    theme.value = value
    applyTheme()
  }

  function toggleTheme() {
    theme.value = theme.value === 'night' ? 'day' : 'night'
    applyTheme()
  }

  function applyTheme() {
    const root = document.documentElement
    root.setAttribute('data-theme', theme.value)
    localStorage.setItem(STORAGE_KEY, theme.value)
  }

  // Apply on init
  applyTheme()

  return {
    theme,
    isDayMode,
    isNightMode,
    setTheme,
    toggleTheme
  }
})
