<template>
  <div class="downloads-page">
    <div class="page-header">
      <h2>下载历史</h2>
      <el-button type="danger" plain @click="clearHistory" v-if="downloads.length > 0">
        清空历史
      </el-button>
    </div>

    <div class="download-list" v-loading="loading">
      <div v-for="item in downloads" :key="item.id" class="download-item">
        <el-avatar shape="square" :size="48">
          <Music :size="20" />
        </el-avatar>
        <div class="download-info">
          <div class="song-name">{{ item.song_name }}</div>
          <div class="song-meta">
            <span class="artist">{{ item.artists }}</span>
            <span class="divider">·</span>
            <span class="quality">{{ formatQuality(item.quality) }}</span>
            <span class="divider">·</span>
            <span class="size">{{ formatSize(item.file_size) }}</span>
          </div>
        </div>
        <div class="download-time">
          {{ formatDate(item.created_at) }}
        </div>
        <div class="download-actions">
          <el-button circle size="small" @click="downloadAgain(item)">
            <Download :size="14" />
          </el-button>
          <el-button
            :icon="Delete"
            type="danger"
            circle
            size="small"
            @click="deleteItem(item)"
          />
        </div>
      </div>

      <el-empty v-if="!loading && downloads.length === 0" description="暂无下载历史" />
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
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Delete } from '@element-plus/icons-vue'
import { Music, Download } from 'lucide-vue-next'
import { ElMessage, ElMessageBox } from 'element-plus'
import api, { downloadBlob } from '@/api'
import { usePlayerStore } from '@/stores/player'

const playerStore = usePlayerStore()

const downloads = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)

onMounted(() => {
  fetchDownloads()
})

async function fetchDownloads() {
  loading.value = true
  try {
    const res = await api.get('/api/downloads', {
      params: { page: page.value, page_size: pageSize.value }
    })
    if (res.data.success) {
      downloads.value = res.data.data.list
      total.value = res.data.data.total
    }
  } catch (e) {
    ElMessage.error('获取下载历史失败')
  } finally {
    loading.value = false
  }
}

function handlePageChange(newPage) {
  page.value = newPage
  fetchDownloads()
}

async function clearHistory() {
  try {
    await ElMessageBox.confirm('确定清空所有下载历史吗?', '提示', {
      type: 'warning'
    })
    await api.delete('/api/downloads')
    downloads.value = []
    total.value = 0
    ElMessage.success('已清空')
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('操作失败')
    }
  }
}

async function deleteItem(item) {
  try {
    await api.delete(`/api/downloads/${item.id}`)
    await fetchDownloads()
    ElMessage.success('已删除')
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

async function downloadAgain(item) {
  const quality = playerStore.downloadQuality
  try {
    const res = await api.get('/download', {
      params: { id: item.song_id, quality, format: 'json' }
    })
    if (res.data.success) {
      await downloadBlob({ id: item.song_id, quality }, res.data.data.filename || `${item.song_name}.mp3`)
    }
  } catch (e) {
    ElMessage.error('下载失败')
  }
}

function formatQuality(quality) {
  const map = {
    standard: '标准',
    exhigh: '极高',
    lossless: '无损',
    hires: 'Hi-Res',
    sky: '沉浸环绕声',
    jyeffect: '高清环绕声',
    jymaster: '超清母带'
  }
  return map[quality] || quality
}

function formatSize(bytes) {
  if (!bytes) return ''
  const mb = bytes / (1024 * 1024)
  return mb.toFixed(2) + ' MB'
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN')
}
</script>

<style lang="scss" scoped>
.downloads-page {
  max-width: 1000px;
  margin: 0 auto;
  padding: 24px;

  @media (max-width: 640px) {
    padding: 16px 12px;
  }
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
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
}

.download-list {
  min-height: 400px;
}

.download-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  border-radius: var(--radius-sm);
  transition: all 0.2s;
  border: var(--border-width) solid transparent;

  @media (max-width: 640px) {
    gap: 10px;
    padding: 10px 8px;
    flex-wrap: wrap;
  }

  &:hover {
    background: var(--bg-elevated-hover);
    border-color: var(--card-border);
    box-shadow: var(--shadow-sm);
  }

  :deep(.el-avatar) {
    background: var(--avatar-bg);
    color: var(--text-muted);
    border: var(--border-width) solid var(--card-border);
    border-radius: var(--radius-sm);
  }

  .download-info {
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

  .download-time {
    color: var(--text-faint);
    font-size: 13px;
    min-width: 100px;
    text-align: right;

    @media (max-width: 640px) {
      display: none;
    }
  }

  .download-actions {
    display: flex;
    gap: 8px;
    opacity: 0;

    @media (hover: none) {
      opacity: 1;
    }
  }

  &:hover .download-actions {
    opacity: 1;
  }
}

[data-theme="day"] .download-item:hover {
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
    box-shadow: var(--btn-hover-shadow);
    transform: var(--btn-hover-transform);
  }
}
</style>
