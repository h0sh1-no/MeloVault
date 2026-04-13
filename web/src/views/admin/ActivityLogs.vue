<template>
  <div class="activity-logs-page">
    <!-- Filters -->
    <div class="toolbar">
      <el-input
        v-model="search"
        placeholder="搜索用户名或IP..."
        clearable
        class="search-input"
        @keyup.enter="loadLogs"
        @clear="loadLogs"
      >
        <template #prefix><Search :size="16" /></template>
      </el-input>
      <el-select v-model="actionFilter" placeholder="全部操作" clearable class="filter-select" @change="loadLogs">
        <el-option label="登录" value="login" />
        <el-option label="播放" value="play" />
        <el-option label="下载" value="download" />
        <el-option label="搜索" value="search" />
        <el-option label="收藏" value="favorite" />
        <el-option label="浏览" value="browse" />
      </el-select>
      <el-button type="primary" :loading="loading" @click="loadLogs">查询</el-button>
    </div>

    <!-- Table -->
    <el-table :data="logs" v-loading="loading" class="logs-table" stripe size="small">
      <el-table-column label="时间" width="160">
        <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="用户" min-width="120">
        <template #default="{ row }">
          <span class="user-name" v-if="row.username">{{ row.username }}</span>
          <span class="anon-name" v-else>匿名</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-tag size="small" :type="actionTagType(row.action)">{{ actionLabel(row.action) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="IP 地址" width="150">
        <template #default="{ row }"><code class="ip-code">{{ row.ip || '—' }}</code></template>
      </el-table-column>
      <el-table-column label="省份" width="100">
        <template #default="{ row }">{{ row.province || '—' }}</template>
      </el-table-column>
      <el-table-column label="城市" width="100">
        <template #default="{ row }">{{ row.city || '—' }}</template>
      </el-table-column>
      <el-table-column label="详情" min-width="200">
        <template #default="{ row }">
          <span class="meta-text">{{ formatMeta(row.metadata) }}</span>
        </template>
      </el-table-column>
    </el-table>

    <!-- Pagination -->
    <div class="pagination-wrap">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[20, 50, 100]"
        layout="total, sizes, prev, pager, next"
        background
        @size-change="loadLogs"
        @current-change="loadLogs"
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
const loading = ref(false)
const logs = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const search = ref('')
const actionFilter = ref('')

function actionLabel(a) {
  return { login: '登录', play: '播放', download: '下载', search: '搜索', favorite: '收藏', browse: '浏览' }[a] || a
}

function actionTagType(a) {
  return { login: 'primary', play: '', download: 'warning', search: 'success', favorite: 'danger', browse: 'info' }[a] || 'info'
}

function formatDate(str) {
  if (!str) return '—'
  return new Date(str).toLocaleString('zh-CN', { hour12: false })
}

function formatMeta(meta) {
  if (!meta || meta === '{}' || meta === 'null') return '—'
  try {
    const obj = typeof meta === 'string' ? JSON.parse(meta) : meta
    if (!obj || Object.keys(obj).length === 0) return '—'
    return Object.entries(obj).map(([k, v]) => `${k}: ${v}`).join(', ')
  } catch {
    return '—'
  }
}

async function loadLogs() {
  loading.value = true
  try {
    const data = await adminStore.getActivityLogs(page.value, pageSize.value, {
      action: actionFilter.value,
      search: search.value
    })
    logs.value = data?.list || []
    total.value = data?.total || 0
  } catch {
    ElMessage.error('获取活动日志失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadLogs)
</script>

<style scoped>
.activity-logs-page {
  --tbl-bg: rgba(255,255,255,0.03);
  --tbl-text: rgba(255,255,255,0.85);
  --tbl-header-bg: rgba(255,255,255,0.05);
  --tbl-header-text: rgba(255,255,255,0.5);
  --tbl-stripe: rgba(255,255,255,0.02);
  --tbl-hover: rgba(124,58,237,0.08);
}

[data-theme="day"] .activity-logs-page {
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
  flex-wrap: wrap;
}
.search-input { max-width: 280px; }
.filter-select { width: 140px; }

:deep(.logs-table .el-table) {
  background: var(--tbl-bg) !important;
  color: var(--tbl-text);
}
:deep(.logs-table .el-table th) {
  background: var(--tbl-header-bg) !important;
  color: var(--tbl-header-text);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
:deep(.logs-table .el-table tr) { background: transparent !important; }
:deep(.logs-table .el-table--striped .el-table__body tr.el-table__row--striped td) {
  background: var(--tbl-stripe) !important;
}
:deep(.logs-table .el-table__body tr:hover > td) {
  background: var(--tbl-hover) !important;
}

.user-name { font-weight: 500; color: var(--text-primary); }
.anon-name { color: var(--text-faint); font-style: italic; }

.ip-code {
  font-size: 12px;
  color: var(--text-muted);
  background: var(--bg-elevated);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
}

.meta-text {
  font-size: 12px;
  color: var(--text-faint);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
  max-width: 300px;
}

.pagination-wrap {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
