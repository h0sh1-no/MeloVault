<template>
  <el-config-provider :locale="zhCn">
    <div class="app-container">
      <navbar v-if="!isAuthPage && !isAdminPage" />
      <main class="main-content" :class="{ 'no-navbar': isAuthPage || isAdminPage }">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
      <player v-if="playerStore.currentSong && !isAuthPage" />
    </div>
  </el-config-provider>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import Navbar from '@/components/Navbar.vue'
import Player from '@/components/Player.vue'
import { usePlayerStore } from '@/stores/player'
import { useThemeStore } from '@/stores/theme'

const route = useRoute()
const playerStore = usePlayerStore()
// Initialize theme store so it applies data-theme attribute on mount
useThemeStore()

const isAuthPage = computed(() => {
  return ['/login', '/register', '/auth/callback'].includes(route.path)
})

const isAdminPage = computed(() => route.path.startsWith('/admin'))
</script>

<style lang="scss">
html, body, #app {
  margin: 0;
  padding: 0;
  height: 100%;
  background: var(--bg-deep);
  transition: background 0.45s ease;
}

.app-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-gradient);
  transition: background 0.45s ease;
  -webkit-user-select: none;
  -moz-user-select: none;
  -ms-user-select: none;
  user-select: none;

  /* 可编辑区域允许长按选中 */
  input,
  textarea,
  [contenteditable="true"] {
    -webkit-user-select: text;
    -moz-user-select: text;
    -ms-user-select: text;
    user-select: text;
  }
}

.main-content {
  flex: 1;
  padding-top: 60px;
  padding-bottom: 80px;
  overflow-y: auto;

  &.no-navbar {
    padding-top: 0;
    padding-bottom: 0;
  }

  @media (max-width: 640px) {
    padding-top: 56px;
    padding-bottom: 120px;
  }
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
