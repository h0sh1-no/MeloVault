import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useAdminStore } from '@/stores/admin'
import { useSiteSettingsStore } from '@/stores/siteSettings'
import { useSearchTransitionStore } from '@/stores/searchTransition'
import { suppressAuthRedirect } from '@/api'

const SEARCH_FADE_DURATION_MS = 240

const routes = [
  // ── Setup (first-time) ──────────────────────────────────────────
  {
    path: '/setup',
    name: 'Setup',
    component: () => import('@/views/Setup.vue'),
    meta: { title: '初始化系统' }
  },

  // ── Public ──────────────────────────────────────────────────────
  {
    path: '/',
    name: 'Home',
    component: () => import('@/views/Home.vue'),
    meta: { title: '首页' }
  },
  {
    path: '/search',
    name: 'Search',
    component: () => import('@/views/Search.vue'),
    meta: { title: '搜索', requiresAuth: true }
  },
  {
    path: '/playlist/:id',
    name: 'Playlist',
    component: () => import('@/views/Playlist.vue'),
    meta: { title: '歌单详情' }
  },
  {
    path: '/album/:id',
    name: 'Album',
    component: () => import('@/views/Album.vue'),
    meta: { title: '专辑详情' }
  },

  // ── Shared playlist (public, optional auth) ─────────────────────
  {
    path: '/shared/playlist/:id',
    name: 'SharedPlaylist',
    component: () => import('@/views/SharedPlaylist.vue'),
    meta: { title: '分享歌单' }
  },

  // ── Authenticated ────────────────────────────────────────────────
  {
    path: '/my-playlists',
    name: 'MyPlaylists',
    component: () => import('@/views/MyPlaylists.vue'),
    meta: { title: '我的歌单', requiresAuth: true }
  },
  {
    path: '/my-playlists/:id',
    name: 'PlaylistDetail',
    component: () => import('@/views/PlaylistDetail.vue'),
    meta: { title: '歌单详情', requiresAuth: true }
  },
  {
    path: '/favorites',
    name: 'Favorites',
    component: () => import('@/views/Favorites.vue'),
    meta: { title: '我的收藏', requiresAuth: true }
  },
  {
    path: '/downloads',
    name: 'Downloads',
    component: () => import('@/views/Downloads.vue'),
    meta: { title: '下载历史', requiresAuth: true }
  },
  {
    path: '/profile',
    name: 'Profile',
    component: () => import('@/views/Profile.vue'),
    meta: { title: '个人中心', requiresAuth: true }
  },

  // ── Auth ─────────────────────────────────────────────────────────
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { title: '登录' }
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/Register.vue'),
    meta: { title: '注册' }
  },
  {
    path: '/auth/callback',
    name: 'AuthCallback',
    component: () => import('@/views/AuthCallback.vue'),
    meta: { title: '登录中...' }
  },

  // ── Admin ────────────────────────────────────────────────────────
  {
    path: '/admin',
    component: () => import('@/views/admin/Layout.vue'),
    meta: { requiresAuth: true, requiresAdmin: true },
    children: [
      {
        path: '',
        redirect: '/admin/dashboard'
      },
      {
        path: 'dashboard',
        name: 'AdminDashboard',
        component: () => import('@/views/admin/Dashboard.vue'),
        meta: { title: '数据总览' }
      },
      {
        path: 'users',
        name: 'AdminUsers',
        component: () => import('@/views/admin/Users.vue'),
        meta: { title: '用户管理' }
      },
      {
        path: 'analytics',
        name: 'AdminAnalytics',
        component: () => import('@/views/admin/Analytics.vue'),
        meta: { title: '用户分析' }
      },
      {
        path: 'activity',
        name: 'AdminActivity',
        component: () => import('@/views/admin/ActivityLogs.vue'),
        meta: { title: '活动日志' }
      },
      {
        path: 'downloads',
        name: 'AdminDownloads',
        component: () => import('@/views/admin/Downloads.vue'),
        meta: { title: '下载记录' }
      },
      {
        path: 'netease',
        name: 'AdminNetease',
        component: () => import('@/views/admin/NeteaseConfig.vue'),
        meta: { title: '网易云配置' }
      },
      {
        path: 'legal',
        name: 'AdminLegal',
        component: () => import('@/views/admin/Legal.vue'),
        meta: { title: '条款管理' }
      },
      {
        path: 'site-settings',
        name: 'AdminSiteSettings',
        component: () => import('@/views/admin/SiteSettings.vue'),
        meta: { title: '功能设置' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

let setupChecked = false

router.beforeEach(async (to, from, next) => {
  document.title = to.meta.title ? `${to.meta.title} - MeloVault` : 'MeloVault'

  const authStore = useAuthStore()
  const adminStore = useAdminStore()

  // ── Restore auth session on hard refresh ───────────────────────────
  if (authStore.accessToken && !authStore.user) {
    suppressAuthRedirect(true)
    await authStore.fetchUser()
    suppressAuthRedirect(false)
  }

  // ── Setup check (once per session, retries on API failure) ───────
  if (!setupChecked && to.name !== 'AuthCallback') {
    try {
      await adminStore.checkSetupStatus()
      setupChecked = true // only mark done on success
    } catch {
      // Backend unreachable — allow navigation but retry on next route change
    }
  }

  // ── Enforce setup state ───────────────────────────────────────────
  // Not yet initialized → force /setup for every route except /setup itself
  if (adminStore.setupInitialized === false && to.name !== 'Setup') {
    return next({ name: 'Setup' })
  }
  // Already initialized → block /setup
  if (adminStore.setupInitialized === true && to.name === 'Setup') {
    return next({ name: 'Home' })
  }

  // ── Admin routes ──────────────────────────────────────────────────
  if (to.meta.requiresAdmin) {
    if (!authStore.isLoggedIn) {
      return next({ name: 'Login', query: { redirect: to.fullPath } })
    }
    const role = authStore.user?.role
    if (role !== 'admin' && role !== 'superadmin') {
      return next({ name: 'Home' })
    }
    return next()
  }

  // ── Pre-fetch site settings for gated pages ──────────────────────
  const gatedPages = ['Playlist', 'Album', 'Login', 'Register']
  if (gatedPages.includes(to.name)) {
    const siteStore = useSiteSettingsStore()
    if (!siteStore.loaded) {
      try { await siteStore.fetch() } catch { /* proceed anyway */ }
    }

    // Feature-gate: playlist / album parse
    if (to.name === 'Playlist' || to.name === 'Album') {
      const f = siteStore.features
      const role = authStore.user?.role
      const isAdmin = role === 'admin' || role === 'superadmin'

      if (to.name === 'Playlist') {
        if (!f.playlist_parse_enabled) return next({ name: 'Home' })
        if (f.playlist_parse_admin_only && !isAdmin) return next({ name: 'Home' })
      }
      if (to.name === 'Album') {
        if (!f.album_parse_enabled) return next({ name: 'Home' })
        if (f.album_parse_admin_only && !isAdmin) return next({ name: 'Home' })
      }
    }
  }

  // ── Regular auth guard ────────────────────────────────────────────
  if (to.meta.requiresAuth && !authStore.isLoggedIn) {
    return next({ name: 'Login', query: { redirect: to.fullPath } })
  }

  if ((to.name === 'Login' || to.name === 'Register') && authStore.isLoggedIn) {
    return next({ name: 'Home' })
  }

  // ── Leaving search: play collapse animation, then navigate ────────────
  const transitionStore = useSearchTransitionStore()
  if (transitionStore.delayedPush) {
    transitionStore.delayedPush = false
    transitionStore.leaving = false
    transitionStore.pendingTo = null
    return next()
  }
  if (from.name === 'Search' && to.name !== 'Search') {
    transitionStore.leaving = true
    transitionStore.pendingTo = to
    next(false)
    setTimeout(() => {
      transitionStore.delayedPush = true
      router.push(transitionStore.pendingTo)
    }, SEARCH_FADE_DURATION_MS)
    return
  }

  next()
})

export default router
