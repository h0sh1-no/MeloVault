<template>
  <div class="search-page">
    <div class="search-header">
      <h2>搜索: {{ query }}</h2>
      <span class="result-count">共 {{ total }} 首歌曲</span>
    </div>

    <div
      class="search-fly"
      :class="{ 'search-fly--expanded': searchExpanded }"
    >
      <el-input
        v-model="searchQueryStore.query"
        placeholder="搜索歌曲、歌手、专辑..."
        size="large"
        clearable
        @keyup.enter="handleSearch"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
    </div>

    <div class="song-list" v-loading="loading">
      <div
        v-for="song in songs"
        :key="song.id"
        class="song-item"
        @click="playSong(song)"
      >
        <el-avatar shape="square" :size="48" :src="song.picUrl">
          <el-icon><Headset /></el-icon>
        </el-avatar>
        <div class="song-info">
          <div class="song-name">{{ song.name }}</div>
          <div class="song-meta">
            <span class="artist">{{ song.artist_string || song.artists }}</span>
            <span class="divider">·</span>
            <span class="album">{{ song.album }}</span>
          </div>
        </div>
        <div class="song-actions">
          <el-button
            :icon="isFavorited(song.id) ? StarFilled : Star"
            circle
            :type="isFavorited(song.id) ? 'warning' : 'default'"
            size="small"
            @click.stop="toggleFavorite(song)"
          />
          <el-dropdown trigger="click" @click.stop>
            <el-button circle size="small" :icon="More" @click.stop />
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="openAddToPlaylist(song)">
                  <el-icon><FolderAdd /></el-icon> 添加到歌单
                </el-dropdown-item>
                <el-dropdown-item @click="downloadSong(song)">
                  <el-icon><Download /></el-icon> 下载
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>

      <el-empty v-if="!loading && songs.length === 0" description="暂无搜索结果" />
    </div>

    <div class="pagination" v-if="total > 0">
      <el-pagination
        v-model:current-page="page"
        :page-size="30"
        :total="total"
        layout="prev, pager, next"
        @current-change="handlePageChange"
      />
    </div>

    <AddToPlaylistDialog v-model="showPlaylistDialog" :song="playlistTargetSong" />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Search, Star, StarFilled, Download, Headset, FolderAdd, More } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api, { downloadBlob } from '@/api'
import { usePlayerStore } from '@/stores/player'
import { useFavoriteStore } from '@/stores/favorite'
import { useAuthStore } from '@/stores/auth'
import { useSearchQueryStore } from '@/stores/searchQuery'
import { useSearchTransitionStore } from '@/stores/searchTransition'
import AddToPlaylistDialog from '@/components/AddToPlaylistDialog.vue'

const route = useRoute()
const router = useRouter()
const playerStore = usePlayerStore()
const favoriteStore = useFavoriteStore()
const authStore = useAuthStore()
const searchQueryStore = useSearchQueryStore()
const searchTransitionStore = useSearchTransitionStore()

const query = computed(() => route.query.q || '')
const searchExpanded = ref(false)
const songs = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const showPlaylistDialog = ref(false)
const playlistTargetSong = ref(null)

function openAddToPlaylist(song) {
  if (!authStore.isLoggedIn) {
    ElMessage.warning('请先登录')
    return
  }
  playlistTargetSong.value = song
  showPlaylistDialog.value = true
}

watch(query, () => {
  searchQueryStore.query = query.value
  page.value = 1
  search()
}, { immediate: true })

watch(() => searchTransitionStore.leaving, (leaving) => {
  if (leaving) searchExpanded.value = false
})

onMounted(() => {
  searchQueryStore.query = query.value
  nextTick(() => {
    requestAnimationFrame(() => {
      searchExpanded.value = true
    })
  })
})

async function search() {
  if (!query.value) return
  loading.value = true
  try {
    const res = await api.get('/api/search', {
      params: { keyword: query.value, limit: 30 }
    })
    if (res.data.success) {
      songs.value = res.data.data || []
      total.value = res.data.data?.length || 0
      if (authStore.isLoggedIn && songs.value.length > 0) {
        favoriteStore.checkFavorites(songs.value.map(s => s.id))
      }
    }
  } catch (e) {
    ElMessage.error('搜索失败')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  const q = searchQueryStore.query.trim()
  if (q) {
    router.push({ name: 'Search', query: { q } })
  }
}

function handlePageChange(newPage) {
  page.value = newPage
  search()
}

function playSong(song) {
  playerStore.play(song, songs.value)
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
        artists: song.artist_string || song.artists,
        album: song.album,
        pic_url: song.picUrl
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
        artists: song.artist_string || song.artists,
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
</script>

<style lang="scss" scoped>
.search-page {
  max-width: 1000px;
  margin: 0 auto;
  padding: 24px;

  @media (max-width: 640px) {
    padding: 16px 12px;
  }
}

.search-header {
  display: flex;
  align-items: baseline;
  gap: 16px;
  margin-bottom: 24px;

  @media (max-width: 640px) {
    gap: 8px;
    margin-bottom: 16px;
  }

  h2 {
    color: var(--text-primary);
    font-size: 24px;
    font-weight: var(--title-weight);
    text-transform: var(--title-transform);
    letter-spacing: var(--title-letter-spacing);
    margin: 0;

    @media (max-width: 640px) {
      font-size: 20px;
    }
  }

  .result-count {
    color: var(--text-muted);
    font-size: 14px;
  }
}

/* 搜索框：淡入 / 淡出 */
.search-fly {
  margin-bottom: 24px;
  min-height: 46px;
  opacity: 0;
  transition: opacity 0.22s ease;

  &.search-fly--expanded {
    opacity: 1;
  }

  :deep(.el-input__wrapper) {
    background: var(--bg-input);
    border: var(--border-width) solid var(--border-color);
    border-radius: var(--radius);
    transition: border-color 0.2s, box-shadow 0.2s;

    &:hover, &.is-focus {
      border-color: var(--accent);
    }

    .el-input__inner {
      color: var(--text-primary);

      &::placeholder {
        color: var(--text-faint);
      }
    }
  }
}

[data-theme="day"] .search-fly :deep(.el-input__wrapper) {
  box-shadow: var(--shadow);

  &:hover, &.is-focus {
    box-shadow: var(--shadow-hover);
  }
}

.song-list {
  min-height: 400px;
}

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

      .divider {
        margin: 0 8px;
      }
    }
  }

  .song-actions {
    display: flex;
    gap: 8px;
    opacity: 0;
    transition: opacity 0.2s;

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

    :deep(.el-avatar) {
      --el-avatar-size: 40px !important;
    }
  }
}

[data-theme="day"] .song-item:hover {
  box-shadow: var(--shadow);
}

.pagination {
  display: flex;
  justify-content: center;
  margin-top: 32px;
}

:deep(.el-button.is-circle) {
  background: var(--btn-bg);
  border: var(--border-width) solid var(--btn-border);
  color: var(--text-secondary);
  box-shadow: var(--btn-shadow);

  &:hover {
    background: var(--btn-hover-bg);
    color: var(--accent);
    box-shadow: var(--btn-hover-shadow);
    transform: var(--btn-hover-transform);
  }
}
</style>
