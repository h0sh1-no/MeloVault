<template>
  <div class="downloads-page">
    <div class="toolbar">
      <el-input
        v-model="search"
        placeholder="搜索歌曲名或用户名..."
        clearable
        class="search-input"
        @keyup.enter="loadDownloads"
        @clear="loadDownloads"
      >
        <template #prefix><Search :size="16" /></template>
      </el-input>
      <el-button type="primary" :loading="loading" @click="loadDownloads">搜索</el-button>
    </div>

    <el-table :data="records" v-loading="loading" stripe class="downloads-table">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column label="歌曲" min-width="180">
        <template #default="{ row }">
          <div class="song-info">
            <div class="song-name">{{ row.song_name || '—' }}</div>
            <div class="song-artists">{{ row.artists || '—' }}</div>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="username" label="用户" width="130" />
      <el-table-column label="品质" width="100">
        <template #default="{ row }">
          <el-tag size="small" :type="qualityTagType(row.quality)">{{ qualityLabel(row.quality) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="格式" width="80">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ row.file_type || '—' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="大小" width="100">
        <template #default="{ row }">{{ formatSize(row.file_size) }}</template>
      </el-table-column>
      <el-table-column label="时间" width="160">
        <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrap">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        background
        @size-change="loadDownloads"
        @current-change="loadDownloads"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Search } from 'lucide-vue-next'
import { ElMessage } from 'element-plus'
import { useAdminStore } from '@/stores/admin'

const adminStore = useAdminStore()
const records = ref([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const search = ref('')

function qualityTagType(q) {
  if (!q) return 'info'
  const ql = q.toLowerCase()
  if (ql.includes('master') || ql.includes('hires') || ql === 'lossless') return 'danger'
  if (ql === 'exhigh' || ql === 'higher' || ql === 'sky' || ql === 'jyeffect') return 'warning'
  return 'success'
}

const qualityLabelMap = {
  standard: '标准',
  exhigh: '极高',
  lossless: '无损',
  hires: 'Hi-Res',
  sky: '沉浸环绕声',
  jyeffect: '高清环绕声',
  jymaster: '超清母带',
}

function qualityLabel(q) {
  if (!q) return '—'
  return qualityLabelMap[q.toLowerCase()] || q
}

function formatSize(bytes) {
  if (!bytes) return '—'
  const mb = bytes / 1024 / 1024
  return mb >= 1 ? `${mb.toFixed(1)} MB` : `${(bytes / 1024).toFixed(0)} KB`
}

function formatDate(str) {
  if (!str) return '—'
  return new Date(str).toLocaleString('zh-CN', { hour12: false })
}

async function loadDownloads() {
  loading.value = true
  try {
    const data = await adminStore.listDownloads(page.value, pageSize.value, search.value)
    records.value = data?.list ?? []
    total.value = data?.total ?? 0
  } catch {
    ElMessage.error('获取下载记录失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadDownloads)
</script>

<style scoped>
.downloads-page {
  --tbl-bg: rgba(255,255,255,0.03);
  --tbl-text: rgba(255,255,255,0.85);
  --tbl-header-bg: rgba(255,255,255,0.05);
  --tbl-header-text: rgba(255,255,255,0.5);
  --tbl-stripe: rgba(255,255,255,0.02);
  --tbl-hover: rgba(124,58,237,0.08);
}

[data-theme="day"] .downloads-page {
  --tbl-bg: #ffffff;
  --tbl-text: rgba(0,0,0,0.85);
  --tbl-header-bg: #fafafa;
  --tbl-header-text: rgba(0,0,0,0.5);
  --tbl-stripe: rgba(0,0,0,0.02);
  --tbl-hover: rgba(230,57,70,0.06);
}

.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}
.search-input { max-width: 320px; }

.downloads-table { border-radius: 12px; overflow: hidden; }

:deep(.el-table) { background: var(--tbl-bg) !important; color: var(--tbl-text); }
:deep(.el-table th) {
  background: var(--tbl-header-bg) !important;
  color: var(--tbl-header-text);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
:deep(.el-table tr) { background: transparent !important; }
:deep(.el-table--striped .el-table__body tr.el-table__row--striped td) {
  background: var(--tbl-stripe) !important;
}
:deep(.el-table__body tr:hover > td) { background: var(--tbl-hover) !important; }

.song-name { font-size: 14px; font-weight: 500; color: var(--text-primary); }
.song-artists { font-size: 12px; color: var(--text-faint); }

.pagination-wrap {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
