<template>
  <div ref="homePageRef" class="home-page">
    <div class="hero">
      <h1>发现音乐</h1>
    </div>

    <!-- Song recommendation zones (logged-in only) -->
    <template v-if="authStore.isLoggedIn">
      <div v-for="zone in zones" :key="zone.key" class="rec-section">
        <div class="rec-section__header">
          <component :is="zone.icon" :size="20" class="rec-section__icon" />
          <h2 class="rec-section__title">{{ zone.label }}</h2>
        </div>
        <div class="rec-track-row" v-loading="zone.loading">
          <div
            v-for="song in zone.songs"
            :key="song.id"
            class="rec-card"
            @click="playSong(song)"
          >
            <div class="rec-card__cover-wrap">
              <img v-if="song.picUrl" :src="song.picUrl" class="rec-card__cover" />
              <div v-else class="rec-card__cover rec-card__cover--placeholder">
                <el-icon :size="28"><Headset /></el-icon>
              </div>
              <div class="rec-card__play-overlay">
                <Play :size="22" />
              </div>
            </div>
            <div class="rec-card__info">
              <div class="rec-card__name">{{ song.name }}</div>
              <div class="rec-card__artist">{{ song.artist_string || song.artists }}</div>
            </div>
          </div>
          <div v-if="!zone.loading && zone.songs.length === 0" class="rec-empty">
            暂无内容
          </div>
        </div>
      </div>
    </template>

    <!-- Not logged in hint -->
    <div v-else class="home-guest">
      <div class="home-guest__icon"><MusicIcon :size="48" /></div>
      <p class="home-guest__text">登录后探索个性化音乐推荐</p>
      <router-link to="/login">
        <el-button class="home-guest__btn" round>立即登录</el-button>
      </router-link>
    </div>

    <!-- Parser tools section (feature-gated) -->
    <div v-if="canUsePlaylist || canUseAlbum" class="tools-section">
      <p class="section-label">解析工具</p>
      <div class="quick-links">
        <div v-if="canUsePlaylist" class="quick-link" @click="showPlaylistDialog = true">
          <el-icon :size="32"><List /></el-icon>
          <span>歌单解析</span>
        </div>
        <div v-if="canUseAlbum" class="quick-link" @click="showAlbumDialog = true">
          <DiscIcon :size="32" />
          <span>专辑解析</span>
        </div>
      </div>
    </div>

    <!-- Playlist Dialog -->
    <el-dialog v-model="showPlaylistDialog" title="解析歌单" :width="dialogWidth">
      <el-input
        v-model="playlistId"
        placeholder="输入网易云歌单ID或链接"
        clearable
      />
      <template #footer>
        <el-button @click="showPlaylistDialog = false">取消</el-button>
        <el-button type="primary" @click="goPlaylist">解析</el-button>
      </template>
    </el-dialog>

    <!-- Album Dialog -->
    <el-dialog v-model="showAlbumDialog" title="解析专辑" :width="dialogWidth">
      <el-input
        v-model="albumId"
        placeholder="输入网易云专辑ID或链接"
        clearable
      />
      <template #footer>
        <el-button @click="showAlbumDialog = false">取消</el-button>
        <el-button type="primary" @click="goAlbum">解析</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { List, Headset } from '@element-plus/icons-vue'
import { Disc3 as DiscIcon, Play, Mic, Flower2, Globe, Radio, Music as MusicIcon } from 'lucide-vue-next'
import { useSiteSettingsStore } from '@/stores/siteSettings'
import { useAuthStore } from '@/stores/auth'
import { usePlayerStore } from '@/stores/player'
import api from '@/api'

const router = useRouter()
const siteStore = useSiteSettingsStore()
const authStore = useAuthStore()
const playerStore = usePlayerStore()

const homePageRef = ref(null)
const showPlaylistDialog = ref(false)
const showAlbumDialog = ref(false)
const playlistId = ref('')
const albumId = ref('')

const isMobile = ref(window.innerWidth <= 640)
const dialogWidth = computed(() => isMobile.value ? '92vw' : '400px')

function onResize() { isMobile.value = window.innerWidth <= 640 }
onMounted(() => window.addEventListener('resize', onResize))
onUnmounted(() => window.removeEventListener('resize', onResize))

const isAdmin = computed(() => {
  const role = authStore.user?.role
  return role === 'admin' || role === 'superadmin'
})

const canUsePlaylist = computed(() => {
  const f = siteStore.features
  if (!f.playlist_parse_enabled) return false
  if (f.playlist_parse_admin_only && !isAdmin.value) return false
  return true
})

const canUseAlbum = computed(() => {
  const f = siteStore.features
  if (!f.album_parse_enabled) return false
  if (f.album_parse_admin_only && !isAdmin.value) return false
  return true
})

const zones = ref([
  { key: 'chinese',  icon: Mic,      label: '华语流行', keyword: '华语流行',  songs: [], loading: true },
  { key: 'japanese', icon: Flower2, label: '日语推荐', keyword: '日语推荐',  songs: [], loading: true },
  { key: 'western',  icon: Globe,    label: '欧美热歌', keyword: 'pop',        songs: [], loading: true },
  { key: 'classic',  icon: Radio,    label: '经典怀旧', keyword: '经典老歌',  songs: [], loading: true },
])

async function fetchZone(zone) {
  try {
    const res = await api.get('/api/search', { params: { keyword: zone.keyword, limit: 10 } })
    if (res.data.success) {
      const songs = res.data.data || []
      const shuffled = songs.slice().sort(() => Math.random() - 0.5)
      zone.songs = shuffled
    }
  } catch {
    zone.songs = []
  } finally {
    zone.loading = false
  }
}

function playSong(song) {
  const allSongs = zones.value.flatMap(z => z.songs)
  playerStore.play(song, allSongs)
}

function onRecRowWheel(e) {
  const row = e.target.closest('.rec-track-row')
  if (!row) return
  e.preventDefault()
  row.scrollLeft += e.deltaY
}

onMounted(() => {
  if (!siteStore.loaded) siteStore.fetch()
  if (authStore.isLoggedIn) {
    zones.value.forEach(zone => fetchZone(zone))
  }
  const el = homePageRef.value
  if (el) {
    el.addEventListener('wheel', onRecRowWheel, { passive: false })
  }
})

onUnmounted(() => {
  const el = homePageRef.value
  if (el) {
    el.removeEventListener('wheel', onRecRowWheel)
  }
})

function goPlaylist() {
  const id = extractId(playlistId.value)
  if (id) {
    router.push({ name: 'Playlist', params: { id } })
    showPlaylistDialog.value = false
    playlistId.value = ''
  }
}

function goAlbum() {
  const id = extractId(albumId.value)
  if (id) {
    router.push({ name: 'Album', params: { id } })
    showAlbumDialog.value = false
    albumId.value = ''
  }
}

function extractId(input) {
  if (!input) return null
  const match = input.match(/\d+/)
  return match ? match[0] : null
}
</script>

<style lang="scss" scoped>
.home-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 32px 24px 48px;

  @media (max-width: 640px) {
    padding: 20px 16px 32px;
  }
}

/* ── Hero ─────────────────────────── */
.hero {
  text-align: center;
  margin-bottom: 40px;
  padding: 12px 0 0;

  h1 {
    font-size: 52px;
    font-weight: var(--title-weight);
    text-transform: var(--title-transform);
    letter-spacing: var(--title-letter-spacing);
    margin: 0;
    color: var(--text-primary);

    @media (max-width: 640px) {
      font-size: 34px;
    }
  }

  @media (max-width: 640px) {
    margin-bottom: 28px;
  }
}

[data-theme="night"] .hero h1 {
  background: var(--accent-gradient);
  background-size: 200% 200%;
  animation: accent-gradient-shift 6s ease infinite;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

[data-theme="day"] .hero h1 {
  color: var(--text-primary);
}

/* ── Recommendation sections ────── */
.rec-section {
  margin-bottom: 36px;

  @media (max-width: 640px) {
    margin-bottom: 28px;
  }

  &__header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 14px;
  }

  &__icon {
    flex-shrink: 0;
    color: var(--accent);
  }

  &__title {
    font-size: 17px;
    font-weight: 700;
    color: var(--text-primary);
    margin: 0;
    letter-spacing: 0.01em;
  }
}

.rec-track-row {
  display: flex;
  gap: 12px;
  overflow-x: auto;
  overflow-y: hidden;
  padding-bottom: 8px;
  scrollbar-width: none;
  min-height: 130px;
  -webkit-overflow-scrolling: touch;

  &::-webkit-scrollbar { display: none; }
}

.rec-card {
  flex-shrink: 0;
  width: 120px;
  cursor: pointer;
  transition: transform 0.2s, opacity 0.2s;

  @media (max-width: 640px) {
    width: 100px;
  }

  &:hover {
    transform: translateY(-3px);

    .rec-card__play-overlay {
      opacity: 1;
    }
  }

  &__cover-wrap {
    position: relative;
    width: 120px;
    height: 120px;
    border-radius: var(--radius);
    overflow: hidden;
    margin-bottom: 8px;

    @media (max-width: 640px) {
      width: 100px;
      height: 100px;
    }
  }

  &__cover {
    width: 100%;
    height: 100%;
    object-fit: cover;

    &--placeholder {
      display: flex;
      align-items: center;
      justify-content: center;
      background: var(--card-bg);
      border: var(--border-width) solid var(--card-border);
      color: var(--text-faint);
    }
  }

  &__play-overlay {
    position: absolute;
    inset: 0;
    background: rgba(0, 0, 0, 0.45);
    display: flex;
    align-items: center;
    justify-content: center;
    opacity: 0;
    transition: opacity 0.2s;
    color: #fff;
    border-radius: var(--radius);
  }

  &__info {
    padding: 0 2px;
  }

  &__name {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    line-height: 1.4;
  }

  &__artist {
    font-size: 12px;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    margin-top: 2px;
  }
}

[data-theme="night"] .rec-card__cover-wrap {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
}

[data-theme="day"] .rec-card__cover-wrap {
  border: 1px solid var(--card-border);
  box-shadow: var(--shadow-sm);
}

[data-theme="day"] .rec-card:hover .rec-card__cover-wrap {
  box-shadow: var(--shadow-hover);
  transform: translateY(-2px);
}

.rec-empty {
  display: flex;
  align-items: center;
  color: var(--text-faint);
  font-size: 14px;
  padding: 20px 0;
}

/* ── Guest hint ─────────────────── */
.home-guest {
  text-align: center;
  padding: 60px 0 48px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;

  &__icon {
    color: var(--text-muted);
  }

  &__text {
    font-size: 16px;
    color: var(--text-muted);
    margin: 0;
  }

  &__btn {
    background: var(--accent-btn-bg);
    color: var(--accent-btn-text);
    border: var(--border-width) solid var(--btn-border);
    font-weight: var(--title-weight);
    box-shadow: var(--btn-shadow);

    &:hover {
      box-shadow: var(--btn-hover-shadow);
      transform: var(--btn-hover-transform);
    }
  }
}

/* ── Parser tools ───────────────── */
.tools-section {
  margin-top: 8px;
  margin-bottom: 40px;

  .section-label {
    text-align: center;
    font-size: 13px;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-faint);
    margin: 0 0 20px;
  }

  @media (max-width: 640px) {
    margin-bottom: 24px;
  }
}

.quick-links {
  display: flex;
  justify-content: center;
  gap: 20px;
  flex-wrap: wrap;

  @media (max-width: 640px) {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .quick-link {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    padding: 24px 32px;
    min-width: 140px;

    @media (max-width: 640px) {
      padding: 20px 16px;
      min-width: 0;
      gap: 10px;
    }
    background: var(--card-bg);
    backdrop-filter: var(--card-backdrop);
    border: var(--border-width) solid var(--card-border);
    border-radius: var(--radius);
    color: var(--text-primary);
    text-decoration: none;
    transition: all 0.3s;
    cursor: pointer;
    box-shadow: var(--shadow-sm);

    &:hover {
      background: var(--card-hover-bg);
      box-shadow: var(--shadow-hover);
      transform: translateY(-4px);

      .el-icon, svg {
        color: var(--accent);
      }
    }

    .el-icon, svg {
      color: var(--text-secondary);
      transition: color 0.3s;
    }

    span {
      font-size: 14px;
      font-weight: var(--el-font-weight);
      color: var(--text-secondary);
    }
  }
}

[data-theme="day"] .quick-links .quick-link {
  &:hover {
    transform: translateY(-3px);
    box-shadow: var(--shadow-hover);
  }
}

[data-theme="night"] .quick-links .quick-link:hover {
  animation: accent-border-cycle 3s linear infinite;
}

:deep(.el-dialog) {
  background: var(--dialog-bg);
  border: var(--border-width) solid var(--border-color);
  border-radius: var(--radius);
  box-shadow: var(--shadow-lg);

  .el-dialog__title {
    color: var(--text-primary);
    font-weight: var(--title-weight);
  }

  .el-input__wrapper {
    background: var(--bg-input);
    border: var(--border-width) solid var(--border-color);

    .el-input__inner {
      color: var(--text-primary);
    }
  }

  .el-button--primary {
    background: var(--accent-btn-bg);
    border: var(--border-width) solid var(--btn-border);
    color: var(--accent-btn-text);
    font-weight: var(--title-weight);
    box-shadow: var(--btn-shadow);

    &:hover {
      box-shadow: var(--btn-hover-shadow);
      transform: var(--btn-hover-transform);
    }
  }

  .el-button:not(.el-button--primary) {
    background: var(--btn-bg);
    border: var(--border-width) solid var(--border-color);
    color: var(--text-secondary);

    &:hover {
      color: var(--text-primary);
    }
  }
}
</style>
