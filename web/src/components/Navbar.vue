<template>
  <nav class="navbar">
    <div class="navbar-left">
      <router-link to="/" class="logo">
        <Headphones :size="24" />
        <span class="logo-text">MeloVault</span>
      </router-link>
    </div>

    <div v-show="route.name !== 'Search'" class="navbar-center">
      <el-input
        v-model="searchQueryStore.query"
        placeholder="搜索音乐..."
        :prefix-icon="Search"
        clearable
        autocomplete="off"
        name="melovault-search"
        type="search"
        @keyup.enter="handleSearch"
        class="search-input"
      />
    </div>

    <div class="navbar-right desktop-only">
      <ThemeToggle />

      <template v-if="authStore.isLoggedIn">
        <router-link to="/my-playlists" class="nav-link">
          <el-icon><Collection /></el-icon>
          <span>歌单</span>
        </router-link>
        <router-link to="/favorites" class="nav-link">
          <el-icon><Star /></el-icon>
          <span>收藏</span>
        </router-link>
        <router-link to="/downloads" class="nav-link">
          <el-icon><Download /></el-icon>
          <span>下载</span>
        </router-link>
        <el-dropdown trigger="click" @command="handleUserCommand">
          <div class="user-dropdown">
            <el-avatar :size="32" :src="authStore.user?.avatar">
              {{ authStore.user?.username?.charAt(0).toUpperCase() }}
            </el-avatar>
            <span class="username">{{ authStore.user?.username }}</span>
            <el-icon><ArrowDown /></el-icon>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">
                <el-icon><User /></el-icon>个人中心
              </el-dropdown-item>
              <el-dropdown-item command="settings">
                <el-icon><Setting /></el-icon>音质设置
              </el-dropdown-item>
              <el-dropdown-item
                v-if="authStore.user?.role === 'admin' || authStore.user?.role === 'superadmin'"
                command="admin"
              >
                <el-icon><Setting /></el-icon>管理后台
              </el-dropdown-item>
              <el-dropdown-item command="logout" divided>
                <el-icon><SwitchButton /></el-icon>退出登录
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </template>
      <template v-else>
        <router-link to="/login" class="nav-link login-btn-link">
          <el-button class="login-btn" round>登录</el-button>
        </router-link>
      </template>
    </div>

    <!-- Mobile hamburger -->
    <button class="mobile-menu-btn mobile-only" @click="mobileMenuOpen = true">
      <Menu :size="22" />
    </button>

    <!-- Mobile drawer -->
    <teleport to="body">
      <transition name="drawer-fade">
        <div v-if="mobileMenuOpen" class="mobile-drawer-mask" @click="mobileMenuOpen = false"></div>
      </transition>
      <transition name="drawer-slide">
        <aside v-if="mobileMenuOpen" class="mobile-drawer">
          <div class="drawer-header">
            <router-link to="/" class="logo" @click="mobileMenuOpen = false">
              <Headphones :size="22" />
              <span>MeloVault</span>
            </router-link>
            <button class="drawer-close" @click="mobileMenuOpen = false">
              <X :size="20" />
            </button>
          </div>

          <div class="drawer-body">
            <div class="drawer-body__scroll">
            <template v-if="authStore.isLoggedIn">
              <div class="drawer-user">
                <el-avatar :size="40" :src="authStore.user?.avatar">
                  {{ authStore.user?.username?.charAt(0).toUpperCase() }}
                </el-avatar>
                <div class="drawer-user-info">
                  <span class="drawer-username">{{ authStore.user?.username }}</span>
                  <span class="drawer-role">{{ authStore.user?.role === 'superadmin' ? '超级管理员' : authStore.user?.role === 'admin' ? '管理员' : '用户' }}</span>
                </div>
              </div>
              <div class="drawer-divider"></div>
              <router-link to="/my-playlists" class="drawer-item" @click="mobileMenuOpen = false">
                <el-icon><Collection /></el-icon>我的歌单
              </router-link>
              <router-link to="/favorites" class="drawer-item" @click="mobileMenuOpen = false">
                <el-icon><Star /></el-icon>我的收藏
              </router-link>
              <router-link to="/downloads" class="drawer-item" @click="mobileMenuOpen = false">
                <el-icon><Download /></el-icon>下载历史
              </router-link>
              <router-link to="/profile" class="drawer-item" @click="mobileMenuOpen = false">
                <el-icon><User /></el-icon>个人中心
              </router-link>
              <router-link
                v-if="authStore.user?.role === 'admin' || authStore.user?.role === 'superadmin'"
                to="/admin"
                class="drawer-item"
                @click="mobileMenuOpen = false"
              >
                <el-icon><Setting /></el-icon>管理后台
              </router-link>
              <div class="drawer-divider"></div>
              <button class="drawer-item drawer-logout" @click="handleLogout">
                <el-icon><SwitchButton /></el-icon>退出登录
              </button>
            </template>
            <template v-else>
              <router-link to="/login" class="drawer-item" @click="mobileMenuOpen = false">
                <el-icon><User /></el-icon>登录 / 注册
              </router-link>
            </template>
            </div>
            <div class="drawer-theme">
              <ThemeToggle />
            </div>
          </div>
        </aside>
      </transition>
    </teleport>
  </nav>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useSearchQueryStore } from '@/stores/searchQuery'
import { Search, Star, Download, User, SwitchButton, ArrowDown, Setting, Collection } from '@element-plus/icons-vue'
import { Headphones, Menu, X } from 'lucide-vue-next'
import { ElMessage } from 'element-plus'
import ThemeToggle from '@/components/ThemeToggle.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const searchQueryStore = useSearchQueryStore()

const mobileMenuOpen = ref(false)

function handleSearch() {
  const q = searchQueryStore.query.trim()
  if (!q) return
  if (!authStore.isLoggedIn) {
    ElMessage.warning('请先登录后再搜索')
    router.push({ name: 'Login', query: { redirect: `/search?q=${encodeURIComponent(q)}` } })
    return
  }
  router.push({ name: 'Search', query: { q } })
}

function handleUserCommand(command) {
  if (command === 'profile') {
    router.push({ name: 'Profile' })
  } else if (command === 'settings') {
    router.push({ name: 'Profile', query: { tab: 'quality' } })
  } else if (command === 'admin') {
    router.push('/admin')
  } else if (command === 'logout') {
    authStore.logout()
    router.push({ name: 'Home' })
  }
}

function handleLogout() {
  mobileMenuOpen.value = false
  authStore.logout()
  router.push({ name: 'Home' })
}
</script>

<style lang="scss" scoped>
.navbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 60px;
  background: var(--bg-navbar);
  backdrop-filter: blur(16px);
  border-bottom: var(--border-width) solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  z-index: 1000;
  transition: background 0.4s, border-color 0.4s, box-shadow 0.4s;

  @media (max-width: 640px) {
    height: 56px;
    padding: 0 12px;
    gap: 8px;
  }
}

[data-theme="night"] .navbar {
  box-shadow: 0 1px 24px rgba(0, 0, 0, 0.35);
}

[data-theme="day"] .navbar {
  backdrop-filter: none;
  box-shadow: var(--shadow-sm);
}

.navbar-left {
  .logo {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--text-primary);
    text-decoration: none;
    font-size: 20px;
    font-weight: var(--title-weight);
    letter-spacing: var(--title-letter-spacing);
    transition: color 0.3s;
    white-space: nowrap;

    svg {
      color: var(--accent);
    }

    @media (max-width: 640px) {
      font-size: 17px;
      gap: 6px;
    }
  }
}

.navbar-center {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  width: 500px;
  pointer-events: auto;

  @media (max-width: 768px) {
    position: static;
    transform: none;
    flex: 1;
    width: auto;
    margin: 0 8px;
  }

  .search-input {
    :deep(.el-input__wrapper) {
      background: var(--bg-input);
      border: var(--border-width) solid var(--border-color);
      border-radius: var(--radius);
      box-shadow: none;
      transition: all 0.3s;

      &:hover, &.is-focus {
        border-color: var(--accent);
        background: var(--bg-elevated);
      }

      .el-input__inner {
        color: var(--text-primary);

        &::placeholder {
          color: var(--text-faint);
        }
      }

      .el-input__prefix {
        color: var(--text-faint);
      }
    }
  }
}

[data-theme="day"] .navbar-center .search-input {
  :deep(.el-input__wrapper) {
    box-shadow: var(--shadow-sm);

    &:hover, &.is-focus {
      box-shadow: var(--shadow-hover);
    }
  }
}

.navbar-right {
  display: flex;
  align-items: center;
  gap: 16px;

  .nav-link {
    display: flex;
    align-items: center;
    gap: 4px;
    color: var(--text-secondary);
    text-decoration: none;
    font-size: 14px;
    font-weight: var(--el-font-weight);
    transition: color 0.2s;

    &:hover {
      color: var(--accent);
    }
  }

  .login-btn {
    background: var(--accent-btn-bg);
    color: var(--accent-btn-text);
    border: var(--border-width) solid var(--btn-border);
    font-weight: var(--title-weight);
    box-shadow: var(--btn-shadow);
    transition: all 0.2s;

    &:hover {
      box-shadow: var(--btn-hover-shadow);
      transform: var(--btn-hover-transform);
      opacity: 0.95;
    }
  }

  .user-dropdown {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    color: var(--text-primary);

    .username {
      font-size: 14px;
      font-weight: var(--el-font-weight);
    }
  }
}

// Desktop/Mobile visibility
.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}

.mobile-only {
  display: none !important;
  @media (max-width: 768px) {
    display: flex !important;
  }
}

.mobile-menu-btn {
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: none;
  background: var(--btn-bg);
  color: var(--text-secondary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.2s;

  &:active {
    background: var(--btn-hover-bg);
  }
}

// Mobile drawer
.mobile-drawer-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 2000;
}

.mobile-drawer {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  width: min(300px, 80vw);
  height: 100vh;
  height: 100dvh;
  background: var(--bg-navbar);
  border-left: var(--border-width) solid var(--border-color);
  z-index: 2001;
  display: flex;
  flex-direction: column;
  overflow: hidden;

  .drawer-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px;
    border-bottom: 1px solid var(--border-color);

    .logo {
      display: flex;
      align-items: center;
      gap: 8px;
      color: var(--text-primary);
      text-decoration: none;
      font-size: 18px;
      font-weight: var(--title-weight);

      svg { color: var(--accent); }
    }

    .drawer-close {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 36px;
      height: 36px;
      border: none;
      background: var(--btn-bg);
      color: var(--text-secondary);
      border-radius: 50%;
      cursor: pointer;

      &:active { background: var(--btn-hover-bg); }
    }
  }

  .drawer-body {
    flex: 1;
    min-height: 0;
    position: relative;
    padding: 0;
  }

  .drawer-body__scroll {
    height: 100%;
    overflow-y: auto;
    padding: 12px 0 72px; /* leave bottom space for fixed theme toggle */
  }

  .drawer-theme {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    display: flex;
    justify-content: flex-end;
    align-items: center;
    padding: 12px 20px 16px;
    border-top: 1px solid var(--border-color);
    background: var(--bg-navbar);
  }

  .drawer-user {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;

    .drawer-user-info {
      display: flex;
      flex-direction: column;
    }

    .drawer-username {
      font-size: 15px;
      font-weight: 600;
      color: var(--text-primary);
    }

    .drawer-role {
      font-size: 12px;
      color: var(--text-muted);
      margin-top: 2px;
    }
  }

  .drawer-divider {
    height: 1px;
    background: var(--border-color);
    margin: 8px 16px;
  }

  .drawer-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 20px;
    color: var(--text-secondary);
    text-decoration: none;
    font-size: 15px;
    transition: all 0.2s;
    border: none;
    background: none;
    width: 100%;
    text-align: left;
    cursor: pointer;

    .el-icon { font-size: 18px; }

    &:active {
      background: var(--bg-elevated-hover);
      color: var(--text-primary);
    }

    &.router-link-active {
      color: var(--accent);
    }
  }

  .drawer-logout {
    color: rgba(248, 113, 113, 0.8);
  }
}

// Drawer transitions
.drawer-fade-enter-active,
.drawer-fade-leave-active {
  transition: opacity 0.25s ease;
}
.drawer-fade-enter-from,
.drawer-fade-leave-to {
  opacity: 0;
}

.drawer-slide-enter-active,
.drawer-slide-leave-active {
  transition: transform 0.25s ease;
}
.drawer-slide-enter-from,
.drawer-slide-leave-to {
  transform: translateX(100%);
}
</style>
