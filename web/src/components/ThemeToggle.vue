<template>
  <button
    class="theme-toggle"
    :class="{ 'theme-toggle--day': themeStore.isDayMode }"
    @click="themeStore.toggleTheme"
    :title="themeStore.isDayMode ? '切换到深色主题' : '切换到浅色主题'"
  >
    <span class="theme-toggle__icon">
      <Sun v-if="themeStore.isNightMode" :size="18" />
      <Moon v-else :size="18" />
    </span>
  </button>
</template>

<script setup>
import { Sun, Moon } from 'lucide-vue-next'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()
</script>

<style lang="scss" scoped>
.theme-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: var(--border-width) solid var(--border-color);
  background: var(--btn-bg);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.35s ease;
  position: relative;
  overflow: hidden;

  &:hover {
    background: var(--btn-hover-bg);
    color: var(--accent);
    box-shadow: var(--shadow-sm);
    transform: var(--btn-hover-transform);
  }

  &:active {
    transform: scale(0.92);
  }

  &__icon {
    display: flex;
    align-items: center;
    justify-content: center;
    transition: transform 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  &:hover &__icon {
    transform: rotate(25deg);
  }
}

[data-theme="night"] .theme-toggle:hover {
  box-shadow: 0 0 18px rgba(45, 212, 191, 0.35);
  color: var(--accent-hover);
}

[data-theme="day"] .theme-toggle:hover {
  box-shadow: var(--shadow-hover);
  color: var(--accent);
}
</style>
