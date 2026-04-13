<template>
  <div class="admin-layout" :class="{ 'sidebar-collapsed': sidebarCollapsed }">
    <!-- Mobile top bar -->
    <header class="admin-mobile-header mobile-only">
      <button class="mobile-sidebar-btn" @click="mobileDrawerOpen = true">
        <MenuIcon :size="20" />
      </button>
      <div class="mobile-brand">
        <Headphones :size="18" class="mobile-brand-icon" />
        <span class="mobile-title">{{ currentPageTitle }}</span>
      </div>
      <ThemeToggle />
    </header>

    <!-- Desktop Sidebar -->
    <aside class="admin-sidebar desktop-sidebar" :class="{ collapsed: sidebarCollapsed }">
      <div class="sidebar-header" :class="{ collapsed: sidebarCollapsed }">
        <div v-show="!sidebarCollapsed" class="brand-mark">
          <Headphones :size="22" />
        </div>
        <el-tooltip :content="sidebarCollapsed ? '展开侧边栏' : '收起侧边栏'" placement="right" :show-after="300">
          <button class="collapse-btn" @click="sidebarCollapsed = !sidebarCollapsed">
            <ChevronLeft :size="16" :class="{ 'rotate-180': sidebarCollapsed }" />
          </button>
        </el-tooltip>
      </div>

      <nav class="sidebar-nav">
        <div v-for="group in navGroups" :key="group.title" class="nav-group">
          <transition name="label-fade">
            <div v-if="!sidebarCollapsed" class="nav-group-title">{{ group.title }}</div>
          </transition>
          <div v-if="sidebarCollapsed" class="nav-group-divider" />
          <el-tooltip
            v-for="item in group.items"
            :key="item.to"
            :content="item.label"
            placement="right"
            :disabled="!sidebarCollapsed"
            :show-after="300"
          >
            <router-link
              :to="item.to"
              class="nav-item"
              active-class="active"
            >
              <span class="nav-icon">
                <component :is="item.icon" :size="18" />
              </span>
              <transition name="label-fade">
                <span v-if="!sidebarCollapsed" class="nav-label">{{ item.label }}</span>
              </transition>
            </router-link>
          </el-tooltip>
        </div>
      </nav>

      <div class="sidebar-footer">
        <el-tooltip content="返回前台" placement="right" :disabled="!sidebarCollapsed" :show-after="300">
          <router-link to="/" class="nav-item">
            <span class="nav-icon"><ExternalLink :size="18" /></span>
            <transition name="label-fade">
              <span v-if="!sidebarCollapsed" class="nav-label">返回前台</span>
            </transition>
          </router-link>
        </el-tooltip>
        <el-tooltip content="退出登录" placement="right" :disabled="!sidebarCollapsed" :show-after="300">
          <button class="nav-item logout-btn" @click="handleLogout">
            <span class="nav-icon"><LogOut :size="18" /></span>
            <transition name="label-fade">
              <span v-if="!sidebarCollapsed" class="nav-label">退出登录</span>
            </transition>
          </button>
        </el-tooltip>
      </div>
    </aside>

    <!-- Mobile sidebar drawer -->
    <teleport to="body">
      <transition name="admin-mask-fade">
        <div
          v-if="mobileDrawerOpen"
          class="admin-drawer-mask"
          @click="mobileDrawerOpen = false"
          @touchstart="onTouchStart"
          @touchmove="onTouchMove"
          @touchend="onTouchEnd"
        />
      </transition>
      <transition name="admin-drawer-slide">
        <aside
          v-if="mobileDrawerOpen"
          class="admin-sidebar mobile-sidebar"
          @touchstart="onTouchStart"
          @touchmove="onTouchMove"
          @touchend="onTouchEnd"
        >
          <div class="sidebar-header">
            <div class="brand-mark">
              <Headphones :size="22" />
            </div>
            <span class="sidebar-title">管理</span>
            <button class="collapse-btn" @click="mobileDrawerOpen = false">
              <X :size="18" />
            </button>
          </div>

          <nav class="sidebar-nav">
            <div v-for="group in navGroups" :key="group.title" class="nav-group">
              <div class="nav-group-title">{{ group.title }}</div>
              <router-link
                v-for="item in group.items"
                :key="item.to"
                :to="item.to"
                class="nav-item"
                active-class="active"
                @click="mobileDrawerOpen = false"
              >
                <span class="nav-icon">
                  <component :is="item.icon" :size="18" />
                </span>
                <span class="nav-label">{{ item.label }}</span>
              </router-link>
            </div>
          </nav>

          <div class="sidebar-footer">
            <router-link to="/" class="nav-item" @click="mobileDrawerOpen = false">
              <span class="nav-icon"><ExternalLink :size="18" /></span>
              <span class="nav-label">返回前台</span>
            </router-link>
            <button class="nav-item logout-btn" @click="handleLogout">
              <span class="nav-icon"><LogOut :size="18" /></span>
              <span class="nav-label">退出登录</span>
            </button>
          </div>
        </aside>
      </transition>
    </teleport>

    <!-- Main content -->
    <main class="admin-main">
      <header class="admin-header desktop-only">
        <div class="header-left">
          <h2 class="page-title">{{ currentPageTitle }}</h2>
          <span class="page-desc">{{ currentPageDesc }}</span>
        </div>
        <div class="header-right">
          <ThemeToggle />
        </div>
      </header>

      <div class="admin-content">
        <router-view v-slot="{ Component }">
          <transition name="page-fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Headphones, LayoutDashboard, Users, Download, Music2,
  ChevronLeft, ExternalLink, LogOut,
  BarChart3, Activity, ScrollText, SlidersHorizontal,
  Menu as MenuIcon, X
} from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { ElMessage, ElMessageBox } from 'element-plus'
import ThemeToggle from '@/components/ThemeToggle.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const sidebarCollapsed = ref(false)
const mobileDrawerOpen = ref(false)

const navGroups = [
  {
    title: '数据',
    items: [
      { to: '/admin/dashboard', label: '数据总览', icon: LayoutDashboard, desc: '查看系统整体运行状态' },
      { to: '/admin/analytics', label: '用户分析', icon: BarChart3, desc: '深入分析用户行为数据' },
      { to: '/admin/activity', label: '活动日志', icon: Activity, desc: '跟踪系统活动与操作记录' },
      { to: '/admin/downloads', label: '下载记录', icon: Download, desc: '管理所有下载任务与历史' },
    ]
  },
  {
    title: '管理',
    items: [
      { to: '/admin/users', label: '用户管理', icon: Users, desc: '管理系统用户与权限' },
    ]
  },
  {
    title: '配置',
    items: [
      { to: '/admin/netease', label: '网易云配置', icon: Music2, desc: '设置网易云 API 连接参数' },
      { to: '/admin/legal', label: '条款管理', icon: ScrollText, desc: '编辑用户协议和隐私政策' },
      { to: '/admin/site-settings', label: '功能设置', icon: SlidersHorizontal, desc: '控制站点功能开关与参数' },
    ]
  }
]

const allNavItems = navGroups.flatMap(g => g.items)

const currentPageTitle = computed(() => {
  const found = allNavItems.find(item => route.path.startsWith(item.to))
  return found?.label ?? '管理后台'
})

const currentPageDesc = computed(() => {
  const found = allNavItems.find(item => route.path.startsWith(item.to))
  return found?.desc ?? ''
})

let touchStartX = 0
let touchCurrentX = 0

function onTouchStart(e) {
  touchStartX = e.touches[0].clientX
  touchCurrentX = touchStartX
}

function onTouchMove(e) {
  touchCurrentX = e.touches[0].clientX
}

function onTouchEnd() {
  const diff = touchStartX - touchCurrentX
  if (diff > 60) {
    mobileDrawerOpen.value = false
  }
}

async function handleLogout() {
  try {
    await ElMessageBox.confirm('确认退出登录？', '提示', {
      confirmButtonText: '退出',
      cancelButtonText: '取消',
      type: 'warning'
    })
  } catch {
    return
  }

  mobileDrawerOpen.value = false
  authStore.logout()
  ElMessage.success('已退出登录')
  router.push('/login')
}
</script>

<style scoped>
.admin-layout {
  --admin-bar-height: 66px;
  display: flex;
  min-height: 100vh;
  background: var(--bg-deep);
  color: var(--text-primary);
  transition: background 0.4s, color 0.35s;
}

@media (max-width: 768px) {
  .admin-layout {
    flex-direction: column;
  }
}

/* ── Visibility helpers ─────────────────────────────── */

.desktop-only { display: flex; }
.desktop-sidebar { display: flex; }
.mobile-only { display: none; }

@media (max-width: 768px) {
  .desktop-only { display: none !important; }
  .desktop-sidebar { display: none !important; }
  .mobile-only { display: flex !important; }
}

/* ── Mobile top bar ─────────────────────────────────── */

.admin-mobile-header {
  align-items: center;
  justify-content: space-between;
  padding: 0 14px;
  height: 54px;
  background: var(--bg-navbar);
  border-bottom: var(--border-width) solid var(--border-color);
  position: sticky;
  top: 0;
  z-index: 10;
  gap: 12px;
}

.mobile-sidebar-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: var(--border-width) solid var(--border-color);
  background: var(--btn-bg);
  color: var(--text-secondary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.2s;
  flex-shrink: 0;
}

.mobile-sidebar-btn:active {
  background: var(--btn-hover-bg);
}

.mobile-brand {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.mobile-brand-icon {
  color: var(--accent);
  flex-shrink: 0;
}

.mobile-title {
  font-size: 15px;
  font-weight: var(--title-weight);
  color: var(--text-primary);
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Sidebar (fixed, does not scroll with content) ───── */

.admin-sidebar.desktop-sidebar {
  position: fixed;
  left: 0;
  top: 0;
  width: 232px;
  height: 100vh;
  height: 100dvh;
  min-height: 100vh;
  background: var(--bg-navbar);
  border-right: var(--border-width) solid var(--border-color);
  flex-direction: column;
  transition: width 0.28s cubic-bezier(0.4, 0, 0.2, 1), background 0.4s, border-color 0.4s;
  overflow: hidden;
  z-index: 20;
}

.admin-sidebar.desktop-sidebar.collapsed {
  width: 62px;
}

.admin-layout.sidebar-collapsed .admin-main {
  margin-left: 62px;
}

@media (max-width: 768px) {
  .admin-main {
    margin-left: 0 !important;
  }
}

.mobile-sidebar {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  z-index: 2001;
  display: flex;
  min-height: auto;
  height: 100vh;
  width: 260px;
  box-shadow: var(--shadow-lg);
}

.admin-drawer-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
  z-index: 2000;
}

/* ── Sidebar header ──────────────────────────────────── */

.sidebar-header {
  display: flex;
  align-items: center;
  gap: 10px;
  height: var(--admin-bar-height);
  min-height: var(--admin-bar-height);
  padding: 0 14px;
  border-bottom: var(--border-width) solid var(--border-color);
  box-sizing: border-box;
}

.sidebar-header.collapsed {
  justify-content: center;
  padding: 0 8px;
}

.brand-mark {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  background: var(--tag-bg);
  color: var(--accent);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: background 0.3s;
}

.sidebar-title {
  font-size: 15px;
  font-weight: var(--title-weight);
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  letter-spacing: 0.01em;
}

.collapse-btn {
  background: none;
  border: none;
  color: var(--text-faint);
  cursor: pointer;
  padding: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
  margin-left: auto;
  transition: color 0.2s, background 0.2s;
}

.sidebar-header.collapsed .collapse-btn {
  margin-left: 0;
}

.collapse-btn:hover {
  color: var(--text-primary);
  background: var(--btn-hover-bg);
}

.collapse-btn .rotate-180 {
  transform: rotate(180deg);
  transition: transform 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}

.collapse-btn svg {
  transition: transform 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}

/* ── Nav groups ──────────────────────────────────────── */

.sidebar-nav {
  flex: 1;
  padding: 8px 8px;
  overflow-y: auto;
  overflow-x: hidden;
  scrollbar-width: thin;
}

.sidebar-nav::-webkit-scrollbar {
  width: 3px;
}

.nav-group {
  margin-bottom: 4px;
}

.nav-group:last-child {
  margin-bottom: 0;
}

.nav-group-title {
  padding: 10px 12px 6px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-faint);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  white-space: nowrap;
  overflow: hidden;
}

.nav-group-divider {
  height: 1px;
  background: var(--border-color);
  margin: 6px 12px 8px;
}

/* ── Nav items ───────────────────────────────────────── */

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  text-decoration: none;
  font-size: 13.5px;
  font-weight: 500;
  transition: all 0.18s ease;
  cursor: pointer;
  border: none;
  background: none;
  width: 100%;
  text-align: left;
  white-space: nowrap;
  overflow: hidden;
  position: relative;
}

.nav-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  transition: color 0.18s;
}

.nav-item:hover {
  color: var(--text-primary);
  background: var(--bg-elevated-hover);
}

.nav-item.active {
  color: var(--accent);
  background: var(--bg-elevated);
}

.nav-item.active .nav-icon {
  color: var(--accent);
}

.nav-item.active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 18px;
  border-radius: 0 3px 3px 0;
  background: var(--accent);
  transition: height 0.2s;
}

.nav-label {
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Sidebar footer ──────────────────────────────────── */

.sidebar-footer {
  padding: 8px;
  border-top: var(--border-width) solid var(--border-color);
}

.logout-btn {
  color: rgba(248, 113, 113, 0.65) !important;
}

.logout-btn:hover {
  color: #f87171 !important;
  background: rgba(248, 113, 113, 0.08) !important;
}

.logout-btn.active::before {
  display: none;
}

/* ── Main area (offset for fixed sidebar) ────────────── */

.admin-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
  margin-left: 232px;
  transition: margin-left 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}

.admin-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--admin-bar-height);
  padding: 0 28px;
  border-bottom: var(--border-width) solid var(--border-color);
  background: var(--bg-navbar);
  position: sticky;
  top: 0;
  z-index: 10;
  transition: background 0.4s, border-color 0.4s;
  gap: 16px;
  box-sizing: border-box;
}

.header-left {
  display: flex;
  align-items: baseline;
  gap: 12px;
  min-width: 0;
}

.page-title {
  font-size: 16px;
  font-weight: var(--title-weight);
  color: var(--text-primary);
  margin: 0;
  white-space: nowrap;
}

.page-desc {
  font-size: 12.5px;
  color: var(--text-faint);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

/* ── Admin header theme toggle (redesign) ────────────── */

.admin-header :deep(.theme-toggle) {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  border: var(--border-width) solid var(--border-color);
  background: var(--btn-bg);
  color: var(--text-secondary);
  transition: transform 0.25s cubic-bezier(0.34, 1.2, 0.64, 1),
    background 0.2s ease,
    color 0.2s ease,
    box-shadow 0.2s ease;
}

.admin-header :deep(.theme-toggle:hover) {
  background: var(--btn-hover-bg);
  color: var(--accent);
  transform: scale(1.06);
  box-shadow: var(--shadow-sm);
}

.admin-header :deep(.theme-toggle:active) {
  transform: scale(0.96);
  transition-duration: 0.1s;
}

.admin-header :deep(.theme-toggle .theme-toggle__icon) {
  transition: transform 0.5s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.admin-header :deep(.theme-toggle:hover .theme-toggle__icon) {
  transform: rotate(60deg);
}

[data-theme="night"] .admin-header :deep(.theme-toggle:hover) {
  box-shadow: 0 0 16px rgba(45, 212, 191, 0.3);
  color: var(--accent-hover);
}

[data-theme="day"] .admin-header :deep(.theme-toggle:hover) {
  box-shadow: var(--shadow-hover);
  color: var(--accent);
}

/* ── Content area ────────────────────────────────────── */

.admin-content {
  flex: 1;
  padding: 24px 28px;
  overflow-y: auto;
  scroll-behavior: smooth;
}

@media (max-width: 768px) {
  .admin-content {
    padding: 16px 14px;
  }
}

/* ── Page transition ─────────────────────────────────── */

.page-fade-enter-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.page-fade-leave-active {
  transition: opacity 0.12s ease;
}

.page-fade-enter-from {
  opacity: 0;
  transform: translateY(6px);
}

.page-fade-leave-to {
  opacity: 0;
}

/* ── Label fade transition ───────────────────────────── */

.label-fade-enter-active {
  transition: opacity 0.2s ease 0.06s, transform 0.2s ease 0.06s;
}

.label-fade-leave-active {
  transition: opacity 0.12s ease, transform 0.12s ease;
}

.label-fade-enter-from {
  opacity: 0;
  transform: translateX(-4px);
}

.label-fade-leave-to {
  opacity: 0;
  transform: translateX(-4px);
}

/* ── Drawer transitions ──────────────────────────────── */

.admin-mask-fade-enter-active,
.admin-mask-fade-leave-active {
  transition: opacity 0.3s ease;
}

.admin-mask-fade-enter-from,
.admin-mask-fade-leave-to {
  opacity: 0;
}

.admin-drawer-slide-enter-active {
  transition: transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.admin-drawer-slide-leave-active {
  transition: transform 0.22s cubic-bezier(0.4, 0, 1, 1);
}

.admin-drawer-slide-enter-from,
.admin-drawer-slide-leave-to {
  transform: translateX(-100%);
}
</style>
