<template>
  <div class="shared-playlist-page" v-loading="loading">
    <div class="playlist-header" v-if="playlistData">
      <el-avatar shape="square" :size="180" :src="playlistData.cover_url || undefined" class="header-cover">
        <ListMusic :size="48" />
      </el-avatar>
      <div class="playlist-info">
        <h1>{{ playlistData.name }}</h1>
        <p class="description" v-if="playlistData.description">{{ playlistData.description }}</p>
        <div class="meta">
          <span><User :size="14" /> {{ playlistData.creator }}</span>
          <span><Music :size="14" /> {{ playlistData.song_count }} 首</span>
        </div>
        <div class="actions">
          <el-button class="play-all-btn" @click="playAll" :disabled="!authStore.isLoggedIn || songs.length === 0">
            <Play :size="16" /> 播放全部
          </el-button>
        </div>
      </div>
    </div>

    <!-- Visible songs -->
    <div class="song-list" v-if="songs.length > 0">
      <div
        v-for="(song, index) in songs"
        :key="song.id"
        class="song-item"
        :class="{ disabled: !authStore.isLoggedIn }"
        @click="authStore.isLoggedIn && playSong(song, index)"
      >
        <span class="index">{{ index + 1 }}</span>
        <el-avatar shape="square" :size="48" :src="song.pic_url">
          <Headphones :size="20" />
        </el-avatar>
        <div class="song-info">
          <div class="song-name">{{ song.song_name }}</div>
          <div class="song-meta">
            <span class="artist">{{ song.artists }}</span>
            <span class="divider">·</span>
            <span class="album">{{ song.album }}</span>
          </div>
        </div>
        <div class="song-actions" v-if="authStore.isLoggedIn">
          <el-button
            circle
            :type="isFavorited(song.song_id) ? 'warning' : 'default'"
            size="small"
            @click.stop="toggleFavorite(song)"
          >
            <Heart v-if="isFavorited(song.song_id)" :size="14" />
            <HeartOff v-else :size="14" />
          </el-button>
          
          <el-dropdown trigger="click" @click.stop>
            <el-button circle size="small" @click.stop>
              <MoreHorizontal :size="14" />
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="downloadSong(song)">
                  <el-icon><Download :size="14" /></el-icon> 下载
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>

    <!-- Truncated / blur overlay for unauthenticated users -->
    <div class="truncated-overlay" v-if="truncated">
      <div class="blurred-rows">
        <div class="blurred-row" v-for="i in blurRowCount" :key="i">
          <span class="blur-index"></span>
          <span class="blur-cover"></span>
          <div class="blur-text">
            <span class="blur-line blur-line--long"></span>
            <span class="blur-line blur-line--short"></span>
          </div>
        </div>
      </div>
      <div class="login-cta">
        <div class="lock-icon">
          <Lock :size="32" />
        </div>
        <h3>登录后查看完整歌单</h3>
        <p>还有 {{ (playlistData?.song_count || 0) - songs.length }} 首歌曲等你发现</p>
        <el-button class="login-btn" type="primary" round @click="goLogin">
          立即登录
        </el-button>
      </div>
    </div>

    <div class="empty-state" v-if="!loading && !playlistData">
      <ListMusic :size="48" />
      <h3>歌单不存在或未公开</h3>
      <p>该歌单可能已被删除或设为私密</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ListMusic, Music, User, Play, Lock, Heart, HeartOff, Download, Headphones, MoreHorizontal } from 'lucide-vue-next'
import { ElMessage } from 'element-plus'
import { usePlaylistStore } from '@/stores/playlist'
import { usePlayerStore } from '@/stores/player'
import { useFavoriteStore } from '@/stores/favorite'
import { useAuthStore } from '@/stores/auth'
import api, { downloadBlob } from '@/api'

const route = useRoute()
const router = useRouter()
const playlistStore = usePlaylistStore()
const playerStore = usePlayerStore()
const favoriteStore = useFavoriteStore()
const authStore = useAuthStore()

const playlistData = ref(null)
const songs = ref([])
const truncated = ref(false)
const loading = ref(false)

const blurRowCount = computed(() => {
  const remaining = (playlistData.value?.song_count || 0) - songs.value.length
  return Math.min(remaining, 5)
})

onMounted(() => {
  fetchSharedPlaylist()
})

async function fetchSharedPlaylist() {
  const id = route.params.id
  const sharerId = route.query.sharer
  if (!id) return

  loading.value = true
  try {
    const res = await playlistStore.getSharedPlaylist(id, sharerId)
    if (res.success) {
      playlistData.value = res.data.playlist
      songs.value = res.data.songs || []
      truncated.value = res.data.truncated || false
      if (authStore.isLoggedIn && songs.value.length > 0) {
        favoriteStore.checkFavorites(songs.value.map(s => s.song_id))
      }
    }
  } catch (e) {
    // playlist not found or not public
  } finally {
    loading.value = false
  }
}

function playSong(song, index) {
  const list = songs.value.map(s => ({
    id: s.song_id,
    name: s.song_name,
    artists: s.artists,
    album: s.album,
    pic_url: s.pic_url
  }))
  playerStore.play(list[index], list)
}

function playAll() {
  if (songs.value.length > 0) playSong(songs.value[0], 0)
}

function isFavorited(songId) {
  return favoriteStore.isFavorited(songId)
}

async function toggleFavorite(song) {
  try {
    if (isFavorited(song.song_id)) {
      await favoriteStore.remove(song.song_id)
      ElMessage.success('已取消收藏')
    } else {
      await favoriteStore.add({
        song_id: song.song_id,
        song_name: song.song_name,
        artists: song.artists,
        album: song.album,
        pic_url: song.pic_url
      })
      ElMessage.success('已添加收藏')
    }
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

async function downloadSong(song) {
  const quality = playerStore.downloadQuality
  try {
    const res = await api.get('/download', {
      params: { id: song.song_id, quality, format: 'json' }
    })
    if (res.data.success) {
      await api.post('/api/downloads', {
        song_id: song.song_id,
        song_name: song.song_name,
        artists: song.artists,
        quality: res.data.data.quality || quality,
        file_type: res.data.data.file_type,
        file_size: res.data.data.file_size
      })
      await downloadBlob({ id: song.song_id, quality }, res.data.data.filename || `${song.song_name}.mp3`)
      ElMessage.success('下载完成')
    }
  } catch (e) {
    ElMessage.error('下载失败')
  }
}

function goLogin() {
  const currentPath = route.fullPath
  router.push({ name: 'Login', query: { redirect: currentPath } })
}
</script>

<style lang="scss" scoped>
.shared-playlist-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
  min-height: calc(100vh - 140px);

  @media (max-width: 640px) {
    padding: 16px 12px;
  }
}

.playlist-header {
  display: flex;
  gap: 32px;
  margin-bottom: 40px;

  @media (max-width: 640px) {
    flex-direction: column;
    align-items: center;
    gap: 16px;
    margin-bottom: 24px;
    text-align: center;
  }

  .header-cover {
    border: var(--border-width) solid var(--card-border);
    border-radius: var(--radius);
    box-shadow: var(--shadow);
    flex-shrink: 0;

    @media (max-width: 640px) {
      :deep(.el-avatar) {
        width: 120px !important;
        height: 120px !important;
      }
    }
  }

  .playlist-info {
    flex: 1;

    h1 {
      color: var(--text-primary);
      font-size: 28px;
      font-weight: var(--title-weight);
      text-transform: var(--title-transform);
      letter-spacing: var(--title-letter-spacing);
      margin: 0 0 12px;

      @media (max-width: 640px) {
        font-size: 22px;
      }
    }

    .description {
      color: var(--text-muted);
      font-size: 14px;
      margin: 0 0 16px;
      display: -webkit-box;
      -webkit-line-clamp: 3;
      -webkit-box-orient: vertical;
      overflow: hidden;
    }

    .meta {
      display: flex;
      gap: 24px;
      color: var(--text-muted);
      font-size: 14px;
      margin-bottom: 24px;

      @media (max-width: 640px) {
        justify-content: center;
        gap: 16px;
      }

      span {
        display: flex;
        align-items: center;
        gap: 4px;
      }
    }

    .actions {
      display: flex;
      gap: 12px;

      @media (max-width: 640px) {
        justify-content: center;
      }
    }
  }
}

.play-all-btn {
  background: var(--accent-btn-bg);
  border: var(--border-width) solid var(--btn-border);
  color: var(--accent-btn-text);
  font-weight: var(--title-weight);
  box-shadow: var(--btn-shadow);
  display: flex;
  align-items: center;
  gap: 6px;

  &:hover {
    opacity: 0.9;
    box-shadow: var(--btn-hover-shadow);
    transform: var(--btn-hover-transform);
  }
}

.song-list {
  .song-item {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 12px 16px;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all 0.2s;
    border: var(--border-width) solid transparent;

    &.disabled {
      cursor: default;
    }

    &:hover:not(.disabled) {
      background: var(--bg-elevated-hover);
      border-color: var(--card-border);
      box-shadow: var(--shadow-sm);
    }

    .index {
      width: 32px;
      text-align: center;
      color: var(--text-faint);
      font-size: 14px;
    }

    :deep(.el-avatar) {
      border: var(--border-width) solid var(--card-border);
      border-radius: var(--radius-sm);
    }

    .song-info {
      flex: 1;
      min-width: 0;

      .song-name {
        color: var(--text-primary);
        font-size: 15px;
        font-weight: 500;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }

      .song-meta {
        color: var(--text-muted);
        font-size: 13px;
        margin-top: 4px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;

        .divider { margin: 0 8px; }
      }
    }

    .song-actions {
      display: flex;
      gap: 8px;
      opacity: 0;

      @media (hover: none) {
        opacity: 1;
      }
    }

    &:hover .song-actions { opacity: 1; }

    @media (max-width: 640px) {
      gap: 10px;
      padding: 10px 8px;

      .index { width: 24px; font-size: 13px; }
    }
  }
}

[data-theme="day"] .song-list .song-item:hover:not(.disabled) {
  box-shadow: var(--shadow);
}

/* Truncated / blur overlay */
.truncated-overlay {
  position: relative;
  margin-top: 4px;
  overflow: hidden;
  border-radius: var(--radius);
}

.blurred-rows {
  filter: blur(6px);
  opacity: 0.5;
  pointer-events: none;
  user-select: none;

  .blurred-row {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 14px 16px;

    .blur-index {
      width: 32px;
      height: 16px;
      background: var(--text-faint);
      border-radius: 4px;
      opacity: 0.3;
    }

    .blur-cover {
      width: 48px;
      height: 48px;
      background: var(--text-faint);
      border-radius: var(--radius-sm);
      opacity: 0.2;
      flex-shrink: 0;
    }

    .blur-text {
      flex: 1;
      display: flex;
      flex-direction: column;
      gap: 6px;

      .blur-line {
        height: 14px;
        background: var(--text-faint);
        border-radius: 4px;
        opacity: 0.25;
      }

      .blur-line--long { width: 60%; }
      .blur-line--short { width: 40%; }
    }
  }
}

.login-cta {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: var(--bg-overlay, rgba(0, 0, 0, 0.15));
  backdrop-filter: blur(2px);
  border-radius: var(--radius);
  text-align: center;

  .lock-icon {
    width: 64px;
    height: 64px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    background: var(--bg-elevated);
    border: var(--border-width) solid var(--card-border);
    color: var(--accent);
    margin-bottom: 16px;
  }

  h3 {
    color: var(--text-primary);
    font-size: 18px;
    font-weight: var(--title-weight);
    margin: 0 0 8px;
  }

  p {
    color: var(--text-muted);
    font-size: 14px;
    margin: 0 0 20px;
  }

  .login-btn {
    background: var(--accent-btn-bg);
    border: var(--border-width) solid var(--btn-border);
    color: var(--accent-btn-text);
    font-weight: var(--title-weight);
    padding: 10px 32px;
    font-size: 15px;
    box-shadow: var(--btn-shadow);

    &:hover {
      opacity: 0.9;
      box-shadow: var(--btn-hover-shadow);
      transform: var(--btn-hover-transform);
    }
  }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  text-align: center;
  color: var(--text-faint);

  h3 {
    color: var(--text-secondary);
    font-size: 18px;
    font-weight: var(--title-weight);
    margin: 16px 0 8px;
  }

  p {
    color: var(--text-faint);
    font-size: 14px;
    margin: 0;
  }
}

:deep(.el-button.is-circle) {
  background: var(--btn-bg);
  border: var(--border-width) solid var(--btn-border);
  color: var(--text-secondary);
  box-shadow: var(--btn-shadow);

  &:hover {
    background: var(--btn-hover-bg);
    box-shadow: var(--btn-hover-shadow);
    transform: var(--btn-hover-transform);
  }
}
</style>
