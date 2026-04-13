<template>
  <div class="favorites-page">
    <div class="page-header">
      <h2>我的收藏</h2>
      <span class="count" v-if="total > 0">共 {{ total }} 首</span>
    </div>

    <div class="song-list" v-if="loading || favorites.length > 0" v-loading="loading">
      <div
        v-for="song in favorites"
        :key="song.id"
        class="song-item"
        @click="playSong(song)"
      >
        <el-avatar shape="square" :size="48" :src="song.pic_url">
          <el-icon><Headset /></el-icon>
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
            :icon="StarFilled"
            type="warning"
            circle
            size="small"
            @click.stop="removeFavorite(song)"
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
    </div>

    <div class="empty-state" v-if="!loading && favorites.length === 0">
      <div class="empty-icon">
        <el-icon :size="64"><Star /></el-icon>
      </div>
      <h3>还没有收藏任何歌曲</h3>
      <p>搜索你喜欢的音乐，点击收藏按钮即可添加到这里</p>
      <el-button class="action-btn" round @click="$router.push('/')">
        <el-icon><Search /></el-icon>
        <span>去发现音乐</span>
      </el-button>
    </div>

    <div class="pagination" v-if="total > pageSize">
      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        @current-change="handlePageChange"
      />
    </div>

    <AddToPlaylistDialog v-model="showPlaylistDialog" :song="playlistTargetSong" />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Star, StarFilled, Download, Headset, Search, FolderAdd, More } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api, { downloadBlob } from '@/api'
import { usePlayerStore } from '@/stores/player'
import { useFavoriteStore } from '@/stores/favorite'
import AddToPlaylistDialog from '@/components/AddToPlaylistDialog.vue'

const playerStore = usePlayerStore()
const favoriteStore = useFavoriteStore()

const favorites = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const showPlaylistDialog = ref(false)
const playlistTargetSong = ref(null)

function openAddToPlaylist(song) {
  playlistTargetSong.value = {
    song_id: song.song_id,
    song_name: song.song_name,
    artists: song.artists,
    album: song.album,
    pic_url: song.pic_url
  }
  showPlaylistDialog.value = true
}

onMounted(() => {
  fetchFavorites()
})

async function fetchFavorites() {
  loading.value = true
  try {
    const res = await api.get('/api/favorites', {
      params: { page: page.value, page_size: pageSize.value }
    })
    if (res.data.success) {
      favorites.value = res.data.data.list
      total.value = res.data.data.total
    }
  } catch (e) {
    ElMessage.error('获取收藏列表失败')
  } finally {
    loading.value = false
  }
}

function handlePageChange(newPage) {
  page.value = newPage
  fetchFavorites()
}

function playSong(song) {
  playerStore.play({
    id: song.song_id,
    name: song.song_name,
    artists: song.artists,
    album: song.album,
    pic_url: song.pic_url
  }, favorites.value.map(s => ({
    id: s.song_id,
    name: s.song_name,
    artists: s.artists,
    album: s.album,
    pic_url: s.pic_url
  })))
}

async function removeFavorite(song) {
  try {
    await ElMessageBox.confirm('确定取消收藏吗?', '提示', {
      type: 'warning'
    })
    await favoriteStore.remove(song.song_id)
    await fetchFavorites()
    ElMessage.success('已取消收藏')
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('操作失败')
    }
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
</script>

<style lang="scss" scoped>
.favorites-page {
  max-width: 1000px;
  margin: 0 auto;
  padding: 24px;

  @media (max-width: 640px) {
    padding: 16px 12px;
  }
}

.page-header {
  display: flex;
  align-items: baseline;
  gap: 16px;
  margin-bottom: 24px;

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

  .count {
    color: var(--text-muted);
    font-size: 14px;
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

  @media (max-width: 640px) {
    gap: 10px;
    padding: 10px 8px;

    :deep(.el-avatar) {
      --el-avatar-size: 40px !important;
    }
  }

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
  }
}

[data-theme="day"] .song-item:hover {
  box-shadow: var(--shadow);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  text-align: center;

  .empty-icon {
    width: 120px;
    height: 120px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    background: var(--empty-icon-bg);
    border: var(--border-width) solid var(--card-border);
    margin-bottom: 24px;

    .el-icon {
      color: var(--empty-icon-color);
    }
  }

  h3 {
    color: var(--text-secondary);
    font-size: 18px;
    font-weight: var(--title-weight);
    margin: 0 0 8px;
  }

  p {
    color: var(--text-faint);
    font-size: 14px;
    margin: 0 0 28px;
  }

  .action-btn {
    background: var(--accent-btn-bg);
    border: var(--border-width) solid var(--btn-border);
    color: var(--accent-btn-text);
    font-weight: var(--title-weight);
    padding: 12px 28px;
    font-size: 14px;
    box-shadow: var(--btn-shadow);

    &:hover {
      opacity: 0.9;
      box-shadow: var(--btn-hover-shadow);
      transform: var(--btn-hover-transform);
    }
  }
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
    box-shadow: var(--btn-hover-shadow);
    transform: var(--btn-hover-transform);
  }
}
</style>
