<template>
  <div class="legal-manage">
    <!-- Tab selector -->
    <div class="type-tabs">
      <button
        v-for="tab in tabs"
        :key="tab.value"
        class="type-tab"
        :class="{ active: activeType === tab.value }"
        @click="switchType(tab.value)"
      >
        <component :is="tab.icon" :size="18" />
        {{ tab.label }}
      </button>
    </div>

    <!-- Editor card -->
    <div class="config-card">
      <div class="card-header">
        <FileText :size="20" />
        <h3>{{ activeLabel }}编辑</h3>
        <span v-if="currentDoc" class="last-update">
          上次更新: {{ formatDate(currentDoc.updated_at) }}
        </span>
      </div>

      <div class="editor-form">
        <div class="form-row">
          <label class="form-label">标题</label>
          <el-input
            v-model="form.title"
            placeholder="请输入标题"
            size="large"
            maxlength="200"
            show-word-limit
          />
        </div>

        <div class="form-row">
          <label class="form-label">内容（支持 HTML）</label>
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="14"
            placeholder="请输入内容，支持 HTML 格式..."
            resize="vertical"
            class="content-textarea"
          />
        </div>

        <div class="card-actions">
          <el-button
            type="primary"
            :loading="saving"
            :disabled="!form.title.trim() || !form.content.trim()"
            @click="handleSave"
          >
            <template #icon><Save /></template>
            保存发布
          </el-button>
          <el-button @click="resetForm">
            <template #icon><RotateCcw /></template>
            重置
          </el-button>
        </div>
      </div>
    </div>

    <!-- Preview card -->
    <div v-if="form.content.trim()" class="config-card preview-card">
      <div class="card-header">
        <Eye :size="20" />
        <h3>内容预览</h3>
      </div>
      <div class="preview-title">{{ form.title || '未命名' }}</div>
      <div class="preview-body" v-html="form.content"></div>
    </div>

    <!-- History card -->
    <div v-if="history.length > 0" class="config-card">
      <div class="card-header">
        <History :size="20" />
        <h3>历史版本</h3>
      </div>
      <div class="history-list">
        <div
          v-for="doc in history"
          :key="doc.id"
          class="history-item"
          :class="{ active: doc.is_active }"
        >
          <div class="history-info">
            <span class="history-title">{{ doc.title }}</span>
            <span v-if="doc.is_active" class="active-badge">当前生效</span>
            <span class="history-date">{{ formatDate(doc.created_at) }}</span>
          </div>
          <el-button size="small" text @click="loadVersion(doc)">
            加载此版本
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { FileText, Save, RotateCcw, Eye, History, ScrollText, ShieldAlert } from 'lucide-vue-next'
import { ElMessage } from 'element-plus'
import { useAdminStore } from '@/stores/admin'

const adminStore = useAdminStore()

const tabs = [
  { value: 'terms', label: '服务条款', icon: ScrollText },
  { value: 'disclaimer', label: '免责声明', icon: ShieldAlert }
]

const activeType = ref('terms')
const activeLabel = computed(() => tabs.find(t => t.value === activeType.value)?.label ?? '')

const currentDoc = ref(null)
const history = ref([])
const saving = ref(false)
const loading = ref(false)

const form = reactive({
  title: '',
  content: ''
})

function switchType(type) {
  activeType.value = type
  fetchDocuments()
}

async function fetchDocuments() {
  loading.value = true
  try {
    const docs = await adminStore.getLegalDocuments(activeType.value)
    history.value = docs || []
    const active = history.value.find(d => d.is_active)
    currentDoc.value = active || null
    if (active) {
      form.title = active.title
      form.content = active.content
    } else {
      form.title = ''
      form.content = ''
    }
  } catch {
    history.value = []
    currentDoc.value = null
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!form.title.trim() || !form.content.trim()) return
  saving.value = true
  try {
    await adminStore.saveLegalDocument(activeType.value, form.title.trim(), form.content)
    ElMessage.success(`${activeLabel.value}保存成功`)
    await fetchDocuments()
  } catch (err) {
    ElMessage.error(err.response?.data?.message ?? '保存失败')
  } finally {
    saving.value = false
  }
}

function resetForm() {
  if (currentDoc.value) {
    form.title = currentDoc.value.title
    form.content = currentDoc.value.content
  } else {
    form.title = ''
    form.content = ''
  }
}

function loadVersion(doc) {
  form.title = doc.title
  form.content = doc.content
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit'
  })
}

onMounted(fetchDocuments)
</script>

<style scoped>
.legal-manage {
  max-width: 900px;
}

.type-tabs {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}

.type-tab {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border-radius: 12px;
  border: 1px solid rgba(255,255,255,0.07);
  background: rgba(255,255,255,0.04);
  color: rgba(255,255,255,0.5);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.type-tab:hover {
  color: rgba(255,255,255,0.8);
  background: rgba(255,255,255,0.08);
}

.type-tab.active {
  color: var(--accent, #a78bfa);
  background: rgba(167,139,250,0.12);
  border-color: rgba(167,139,250,0.3);
}

.config-card {
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.07);
  border-radius: 16px;
  padding: 24px;
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  color: rgba(255,255,255,0.8);
}

.card-header h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: #fff;
}

.last-update {
  margin-left: auto;
  font-size: 12px;
  color: rgba(255,255,255,0.35);
}

.editor-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.form-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-label {
  font-size: 13px;
  color: rgba(255,255,255,0.6);
  font-weight: 500;
}

.card-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.content-textarea :deep(.el-textarea__inner) {
  background: rgba(255,255,255,0.06) !important;
  border-color: rgba(255,255,255,0.1) !important;
  color: rgba(255,255,255,0.85);
  font-family: monospace;
  font-size: 13px;
  line-height: 1.7;
  resize: vertical;
}

.preview-card {
  border-color: rgba(167,139,250,0.15);
}

.preview-title {
  font-size: 18px;
  font-weight: 600;
  color: #fff;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(255,255,255,0.07);
}

.preview-body {
  color: rgba(255,255,255,0.7);
  font-size: 14px;
  line-height: 1.8;
  word-break: break-word;
}

.preview-body :deep(h1),
.preview-body :deep(h2),
.preview-body :deep(h3) {
  color: rgba(255,255,255,0.9);
  margin: 16px 0 8px;
}

.preview-body :deep(p) {
  margin: 8px 0;
}

.preview-body :deep(ul),
.preview-body :deep(ol) {
  padding-left: 20px;
  margin: 8px 0;
}

.preview-body :deep(a) {
  color: var(--accent, #a78bfa);
}

.history-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.history-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  border-radius: 10px;
  background: rgba(255,255,255,0.02);
  border: 1px solid rgba(255,255,255,0.05);
  transition: background 0.2s;
}

.history-item:hover {
  background: rgba(255,255,255,0.06);
}

.history-item.active {
  border-color: rgba(34,197,94,0.2);
  background: rgba(34,197,94,0.05);
}

.history-info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.history-title {
  font-size: 14px;
  color: rgba(255,255,255,0.8);
  font-weight: 500;
}

.active-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 20px;
  background: rgba(34,197,94,0.15);
  color: #4ade80;
  font-weight: 600;
}

.history-date {
  font-size: 12px;
  color: rgba(255,255,255,0.3);
}

/* Day theme overrides */
[data-theme="day"] .type-tab {
  background: rgba(0,0,0,0.03);
  border-color: rgba(0,0,0,0.08);
  color: rgba(0,0,0,0.5);
}

[data-theme="day"] .type-tab:hover {
  color: rgba(0,0,0,0.8);
  background: rgba(0,0,0,0.06);
}

[data-theme="day"] .type-tab.active {
  background: rgba(167,139,250,0.1);
  border-color: rgba(167,139,250,0.3);
  color: #7c3aed;
}

[data-theme="day"] .config-card {
  background: #fff;
  border-color: rgba(0,0,0,0.08);
  box-shadow: 0 1px 3px rgba(0,0,0,0.06);
}

[data-theme="day"] .card-header { color: rgba(0,0,0,0.6); }
[data-theme="day"] .card-header h3 { color: rgba(0,0,0,0.85); }
[data-theme="day"] .last-update { color: rgba(0,0,0,0.4); }
[data-theme="day"] .form-label { color: rgba(0,0,0,0.6); }
[data-theme="day"] .preview-title { color: rgba(0,0,0,0.85); border-color: rgba(0,0,0,0.08); }
[data-theme="day"] .preview-body { color: rgba(0,0,0,0.7); }
[data-theme="day"] .preview-body :deep(h1),
[data-theme="day"] .preview-body :deep(h2),
[data-theme="day"] .preview-body :deep(h3) { color: rgba(0,0,0,0.85); }
[data-theme="day"] .history-item { background: rgba(0,0,0,0.02); border-color: rgba(0,0,0,0.06); }
[data-theme="day"] .history-item:hover { background: rgba(0,0,0,0.04); }
[data-theme="day"] .history-title { color: rgba(0,0,0,0.8); }
[data-theme="day"] .history-date { color: rgba(0,0,0,0.4); }

[data-theme="day"] .content-textarea :deep(.el-textarea__inner) {
  background: #fafafa !important;
  border-color: rgba(0,0,0,0.1) !important;
  color: rgba(0,0,0,0.85);
}
</style>
