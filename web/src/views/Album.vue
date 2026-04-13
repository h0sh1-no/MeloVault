<template>
  <div class="album-page" v-loading="loading">
    <div class="album-header" v-if="album">
      <el-avatar shape="square" :size="180" :src="album.coverImgUrl" class="header-cover">
        <Disc3 :size="48" />
      </el-avatar>
      <div class="album-info">
        <h1>{{ album.name }}</h1>
        <p class="artist">{{ album.artist }}</p>
        <div class="meta">
          <span v-if="album.publishTime">
            <el-icon><Calendar /></el-icon>
            {{ formatDate(album.publishTime) }}
          </span>
          <span><Music :size="14" class="inline-icon" /> {{ songs.length }} 首</span>
        </div>
        <p class="description" v-if="album.description">{{ album.description }}</p>
        <div class="actions">
          <el-button class="play-all-btn" @click="playAll">
            <el-icon><Play /></el-icon> 播放全部
          </el-button>
        </div>
      </div>
    </div>

    <div class="song-list">
      <div
        v-for="(song, index) in songs"
        :key="song.id"
        class="song-item"
        @click="playSong(song, index)"
      >
        <span class="index">{{ index + 1 }}</span>
        <div class="song-info">
          <div class="song-name">{{ song.name }}</div>
          <div class="song-meta">
            <span class="artist">{{ song.artists }}</span>
            <span class="divider" v-if="song.dt">·</span>
            <span class="duration" v-if="song.dt">{{ formatDuration(song.dt) }}</span>
          </div>
        </div>
        <div class="song-actions">
          <el-button
            circle
            :type="isFavorited(song.id) ? 'warning' : 'default'"
            size="small"
            @click.stop="toggleFavorite(song)"
          >
            <Heart v-if="isFavorited(song.id)" :size="14" />
            <HeartOff v-else :size="14" />
          </el-button>
          
          <el-dropdown trigger="click" @click.stop>
            <el-button circle size="small" @click.stop>
              <MoreHorizontal :size="14" />
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="openAddToPlaylist(song)">
                  <el-icon><FolderPlus :size="14" /></el-icon> 添加到歌单
                </el-dropdown-item>
                <el-dropdown-item @click="downloadSong(song)">
                  <el-icon><Download :size="14" /></el-icon> 下载
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>

    <AddToPlaylistDialog v-model="showPlaylistDialog" :song="playlistTargetSong" />
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Calendar } from '@element-plus/icons-vue'
import { Disc3, Music, Play, Heart, HeartOff, Download, FolderPlus, MoreHorizontal } from 'lucide-vue-next'
import { ElMessage } from 'element-plus'
import api, { downloadBlob } from '@/api'
import { usePlayerStore } from '@/stores/player'
import { useFavoriteStore } from '@/stores/favorite'
import { useAuthStore } from '@/stores/auth'
import AddToPlaylistDialog from '@/components/AddToPlaylistDialog.vue'

const route = useRoute()
const playerStore = usePlayerStore()
const favoriteStore = useFavoriteStore()
const authStore = useAuthStore()

const album = ref(null)
const songs = ref([])
const loading = ref(false)
const showPlaylistDialog = ref(false)
const playlistTargetSong = ref(null)

function openAddToPlaylist(song) {
  if (!authStore.isLoggedIn) {
    ElMessage.warning('请先登录')
    return
  }
  playlistTargetSong.value = {
    id: song.id,
    name: song.name,
    artists: song.artists,
    album: song.album || album.value?.name,
    picUrl: song.picUrl || album.value?.coverImgUrl
  }
  showPlaylistDialog.value = true
}

watch(() => route.params.id, () => {
  fetchAlbum()
}, { immediate: true })

async function fetchAlbum() {
  const id = route.params.id
  if (!id) return

  loading.value = true
  try {
    const res = await api.get('/album', { params: { id } })
    if (res.data.success) {
      album.value = res.data.data.album
      songs.value = res.data.data.album.songs || []
      if (authStore.isLoggedIn && songs.value.length > 0) {
        favoriteStore.checkFavorites(songs.value.map(s => s.id))
      }
    }
  } catch (e) {
    ElMessage.error('获取专辑失败')
  } finally {
    loading.value = false
  }
}

function playSong(song, index) {
  const songList = songs.value.map(s => ({
    id: s.id,
    name: s.name,
    artists: s.artists,
    album: s.album || album.value?.name,
    pic_url: s.picUrl || album.value?.coverImgUrl
  }))
  playerStore.play(songList[index], songList)
}

function playAll() {
  if (songs.value.length > 0) {
    playSong(songs.value[0], 0)
  }
}

function isFavorited(songId) {
  return favoriteStore.isFavorited(songId)
}

async function toggleFavorite(song) {
  if (!authStore.isLoggedIn) {
    ElMessage.warning('请先登录')
    return
  }
  try {
    if (isFavorited(song.id)) {
      await favoriteStore.remove(song.id)
      ElMessage.success('已取消收藏')
    } else {
      await favoriteStore.add({
        song_id: song.id,
        song_name: song.name,
        artists: song.artists,
        album: song.album || album.value?.name,
        pic_url: song.picUrl || album.value?.coverImgUrl
      })
      ElMessage.success('已添加收藏')
    }
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

async function downloadSong(song) {
  if (!authStore.isLoggedIn) {
    ElMessage.warning('请先登录后再下载')
    return
  }
  const quality = playerStore.downloadQuality
  try {
    const res = await api.get('/download', {
      params: { id: song.id, quality, format: 'json' }
    })
    if (res.data.success) {
      await api.post('/api/downloads', {
        song_id: song.id,
        song_name: song.name,
        artists: song.artists,
        quality: res.data.data.quality || quality,
        file_type: res.data.data.file_type,
        file_size: res.data.data.file_size
      })
      await downloadBlob({ id: song.id, quality }, res.data.data.filename || `${song.name}.mp3`)
      ElMessage.success('下载完成')
    }
  } catch (e) {
    ElMessage.error('下载失败')
  }
}

function formatDate(timestamp) {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  return date.toLocaleDateString('zh-CN')
}

function formatDuration(ms) {
  if (!ms) return ''
  const seconds = Math.floor(ms / 1000)
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}:${secs.toString().padStart(2, '0')}`
}
</script>

<style lang="scss" scoped>
.album-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
  min-height: calc(100vh - 140px);

  @media (max-width: 640px) {
    padding: 16px 12px;
  }
}

.album-header {
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

    @media (max-width: 640px) {
      :deep(.el-avatar) {
        width: 140px !important;
        height: 140px !important;
      }
    }
  }

  .album-info {
    flex: 1;

    h1 {
      color: var(--text-primary);
      font-size: 28px;
      font-weight: var(--title-weight);
      text-transform: var(--title-transform);
      letter-spacing: var(--title-letter-spacing);
      margin: 0 0 8px;

      @media (max-width: 640px) {
        font-size: 22px;
      }
    }

    .artist {
      color: var(--accent);
      font-size: 16px;
      font-weight: var(--el-font-weight);
      margin: 0 0 16px;
    }

    .meta {
      display: flex;
      gap: 24px;
      color: var(--text-muted);
      font-size: 14px;
      margin-bottom: 16px;

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

    .description {
      color: var(--text-muted);
      font-size: 14px;
      margin: 0 0 24px;
      display: -webkit-box;
      -webkit-line-clamp: 3;
      -webkit-box-orient: vertical;
      overflow: hidden;
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
  transition: all 0.2s;

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

    &:hover {
      background: var(--bg-elevated-hover);
      border-color: var(--card-border);
      box-shadow: var(--shadow-sm);
    }

    .index {
      width: 32px;
      text-align: center;
      color: var(--text-faint);
      font-size: 14px;
      font-weight: var(--el-font-weight);
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

        .divider {
          margin: 0 8px;
        }
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

    &:hover .song-actions {
      opacity: 1;
    }

    @media (max-width: 640px) {
      gap: 10px;
      padding: 10px 8px;

      .index { width: 24px; font-size: 13px; }
    }
  }
}

[data-theme="day"] .song-list .song-item:hover {
  box-shadow: var(--shadow);
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
