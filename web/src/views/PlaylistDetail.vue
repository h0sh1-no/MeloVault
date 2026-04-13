<template>
  <div class="playlist-detail-page" v-loading="loading">
    <div class="playlist-header" v-if="playlistData">
      <el-avatar shape="square" :size="160" :src="playlistData.cover_url || undefined" class="header-cover">
        <ListMusic :size="48" />
      </el-avatar>
      <div class="playlist-info">
        <h1>{{ playlistData.name }}</h1>
        <p class="description" v-if="playlistData.description">{{ playlistData.description }}</p>
        <div class="meta">
          <span>
            <Music :size="14" class="inline-icon" /> {{ songs.length }} 首
          </span>
          <span v-if="playlistData.is_public" class="public-tag">
            <Globe :size="14" class="inline-icon" /> 公开
          </span>
          <span v-else class="private-tag">
            <Lock :size="14" class="inline-icon" /> 私密
          </span>
        </div>
        <div class="actions">
          <el-button class="play-all-btn" @click="playAll" :disabled="songs.length === 0">
            <Play :size="16" /> 播放全部
          </el-button>
          <el-button round @click="togglePublic">
            {{ playlistData.is_public ? '设为私密' : '设为公开' }}
          </el-button>
          <el-button v-if="playlistData.is_public" round @click="sharePlaylist">
            <Share2 :size="14" /> 分享
          </el-button>
        </div>
      </div>
    </div>

    <div class="song-list" v-if="songs.length > 0">
      <div
        v-for="(song, index) in songs"
        :key="song.id"
        class="song-item"
        @click="playSong(song, index)"
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
        <div class="song-actions">
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
                <el-dropdown-item @click="removeSong(song)" style="color: var(--el-color-danger)">
                  <el-icon><X :size="14" /></el-icon> 移除
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>

    <div class="empty-songs" v-if="!loading && playlistData && songs.length === 0">
      <ListMusic :size="48" />
      <p>歌单还没有歌曲，去搜索添加吧</p>
      <el-button class="action-btn" round @click="$router.push('/')">去发现音乐</el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ListMusic, Music, Globe, Lock, Play, Share2, Heart, HeartOff, Download, X, Headphones, MoreHorizontal } from 'lucide-vue-next'
import { ElMessage, ElMessageBox } from 'element-plus'
import { usePlaylistStore } from '@/stores/playlist'
import { usePlayerStore } from '@/stores/player'
import { useFavoriteStore } from '@/stores/favorite'
import { useAuthStore } from '@/stores/auth'
import api, { downloadBlob } from '@/api'

const route = useRoute()
const playlistStore = usePlaylistStore()
const playerStore = usePlayerStore()
const favoriteStore = useFavoriteStore()
const authStore = useAuthStore()

const playlistData = ref(null)
const songs = ref([])
const loading = ref(false)

watch(() => route.params.id, () => {
  fetchDetail()
}, { immediate: true })

async function fetchDetail() {
  const id = route.params.id
  if (!id) return

  loading.value = true
  try {
    const res = await playlistStore.getPlaylistDetail(id)
    if (res.success) {
      playlistData.value = res.data.playlist
      songs.value = res.data.songs || []
      if (songs.value.length > 0) {
        favoriteStore.checkFavorites(songs.value.map(s => s.song_id))
      }
    }
  } catch (e) {
    ElMessage.error('获取歌单详情失败')
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

async function removeSong(song) {
  try {
    await ElMessageBox.confirm(`确定从歌单中移除「${song.song_name}」吗？`, '移除歌曲', { type: 'warning' })
    await playlistStore.removeSongFromPlaylist(route.params.id, song.song_id)
    songs.value = songs.value.filter(s => s.song_id !== song.song_id)
    ElMessage.success('已移除')
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('移除失败')
  }
}

async function togglePublic() {
  try {
    await playlistStore.updatePlaylist(playlistData.value.id, { is_public: !playlistData.value.is_public })
    playlistData.value.is_public = !playlistData.value.is_public
    ElMessage.success(playlistData.value.is_public ? '已设为公开' : '已设为私密')
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

async function sharePlaylist() {
  const userId = authStore.user?.id
  const url = `${window.location.origin}/shared/playlist/${playlistData.value.id}?sharer=${userId}`
  try {
    await navigator.clipboard.writeText(url)
    ElMessage.success('分享链接已复制到剪贴板')
  } catch {
    ElMessage.info('分享链接: ' + url)
  }
}
</script>

<style lang="scss" scoped>
.playlist-detail-page {
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
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
    }

    .meta {
      display: flex;
      gap: 20px;
      color: var(--text-muted);
      font-size: 14px;
      margin-bottom: 24px;

      @media (max-width: 640px) {
        justify-content: center;
        gap: 16px;
        margin-bottom: 16px;
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
        flex-wrap: wrap;
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

[data-theme="day"] .song-list .song-item:hover {
  box-shadow: var(--shadow);
}

.empty-songs {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  color: var(--text-faint);
  text-align: center;

  p {
    margin: 16px 0 24px;
    color: var(--text-muted);
  }

  .action-btn {
    background: var(--accent-btn-bg);
    border: var(--border-width) solid var(--btn-border);
    color: var(--accent-btn-text);
    font-weight: var(--title-weight);
    box-shadow: var(--btn-shadow);
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
