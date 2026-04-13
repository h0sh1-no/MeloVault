<template>
  <div class="netease-config">
    <!-- Account Pool -->
    <div class="pool-card">
      <div class="card-header">
        <div class="card-header-left">
          <DatabaseZap :size="20" />
          <h3>网易云号池</h3>
        </div>
        <div class="pool-stats" v-if="poolTotal > 0">
          <span class="stat-badge active">{{ poolActive }} 个活跃</span>
          <span class="stat-badge total">{{ poolTotal }} 个总计</span>
        </div>
      </div>
      <p class="card-desc">管理多个网易云账号，系统自动轮询使用活跃账号的 Cookie 发起请求。</p>

      <div v-if="poolLoading" class="pool-loading">
        <el-icon class="is-loading" :size="24"><Loading /></el-icon>
        <span>加载中...</span>
      </div>

      <div v-else-if="accounts.length === 0" class="pool-empty">
        <Users :size="40" class="empty-icon" />
        <p>号池为空，通过下方扫码或手动添加账号</p>
      </div>

      <div v-else class="pool-table-wrap">
        <table class="pool-table">
          <thead>
            <tr>
              <th>昵称</th>
              <th>MUSIC_U</th>
              <th>状态</th>
              <th>最后使用</th>
              <th>添加时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="acc in accounts" :key="acc.id" :class="{ inactive: !acc.is_active }">
              <td class="col-nickname">
                <span
                  v-if="editingId !== acc.id"
                  class="nickname-text"
                  @dblclick="startEdit(acc)"
                >{{ acc.nickname || '未命名' }}</span>
                <input
                  v-else
                  v-model="editNickname"
                  class="nickname-input"
                  @blur="saveNickname(acc.id)"
                  @keydown.enter="saveNickname(acc.id)"
                  @keydown.escape="editingId = null"
                  ref="nicknameInput"
                />
              </td>
              <td class="col-musicU">
                <code>{{ truncateMusicU(acc.music_u) }}</code>
              </td>
              <td class="col-status">
                <el-switch
                  :model-value="acc.is_active"
                  size="small"
                  @change="(val) => toggleActive(acc.id, val)"
                />
              </td>
              <td class="col-time">{{ formatTime(acc.last_used_at) }}</td>
              <td class="col-time">{{ formatTime(acc.created_at) }}</td>
              <td class="col-actions">
                <button class="icon-btn danger" @click="removeAccount(acc.id, acc.nickname)" title="删除">
                  <Trash2 :size="15" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Add Account Section -->
    <div class="config-grid">
      <!-- QR Login card -->
      <div class="config-card">
        <div class="card-header">
          <QrCode :size="20" />
          <h3>扫码添加账号</h3>
        </div>
        <p class="card-desc">使用网易云音乐 App 扫码，自动获取 Cookie 并添加到号池。可多次扫码添加不同账号。</p>

        <div class="qr-area">
          <div v-if="!qrKey && !qrLoading" class="qr-placeholder">
            <QrCode :size="64" class="qr-empty-icon" />
            <p>点击「生成二维码」开始登录</p>
          </div>

          <div v-else-if="qrLoading && !qrKey" class="qr-placeholder">
            <el-icon class="is-loading" :size="32"><Loading /></el-icon>
            <p>正在生成二维码...</p>
          </div>

          <template v-else>
            <div class="qr-image-wrap" :class="{ expired: qrStatus === 800, authorized: qrStatus === 803 }">
              <img v-if="qrDataUrl" :src="qrDataUrl" alt="QR code" class="qr-image" />

              <div v-if="qrStatus === 800" class="qr-overlay expired">
                <RefreshCw :size="32" />
                <span>已过期，请重新生成</span>
              </div>

              <div v-if="qrStatus === 803" class="qr-overlay success">
                <CheckCircle2 :size="32" />
                <span>登录成功！</span>
              </div>

              <div v-if="qrStatus === 802" class="qr-overlay scan">
                <Smartphone :size="28" />
                <span>已扫码，请在 App 内确认</span>
              </div>
            </div>

            <div class="qr-status-text" :class="statusClass">
              <component :is="statusIcon" :size="15" />
              {{ statusText }}
            </div>
          </template>
        </div>

        <div class="card-actions">
          <el-button
            type="primary"
            :loading="qrLoading"
            :disabled="polling"
            @click="generateQR"
          >
            <template #icon><QrCode /></template>
            {{ qrKey ? '重新生成' : '生成二维码' }}
          </el-button>

          <el-button
            v-if="qrKey && qrStatus !== 803 && qrStatus !== 800"
            :loading="polling"
            @click="polling ? stopPolling() : startPolling()"
          >
            {{ polling ? '停止轮询' : '开始检测' }}
          </el-button>
        </div>
      </div>

      <!-- Manual cookie card -->
      <div class="config-card">
        <div class="card-header">
          <ClipboardPaste :size="20" />
          <h3>手动添加账号</h3>
        </div>
        <p class="card-desc">
          将网易云音乐 Web 端的 Cookie 字符串粘贴到下方，添加到号池中。
          格式示例：<code>MUSIC_U=xxx; __csrf=yyy</code>
        </p>

        <el-input
          v-model="manualNickname"
          placeholder="账号昵称（可选）"
          class="nickname-field"
        />

        <el-input
          v-model="manualCookie"
          type="textarea"
          :rows="5"
          placeholder="MUSIC_U=...; __csrf=...; ..."
          class="cookie-textarea"
          resize="vertical"
        />

        <div class="card-actions">
          <el-button
            type="primary"
            :loading="savingManual"
            :disabled="!manualCookie.trim()"
            @click="addManualAccount"
          >
            <template #icon><Plus /></template>
            添加到号池
          </el-button>
          <el-button @click="manualCookie = ''; manualNickname = ''">清除</el-button>
        </div>

        <div v-if="saveResult" class="save-result" :class="saveResult.type">
          <component :is="saveResult.type === 'success' ? CheckCircle2 : XCircle" :size="15" />
          {{ saveResult.message }}
        </div>
      </div>
    </div>

    <!-- Usage tips -->
    <div class="tips-card">
      <div class="card-header">
        <Info :size="18" />
        <h4>使用说明</h4>
      </div>
      <ul class="tips-list">
        <li>号池中的活跃账号会被系统轮询使用，自动分担请求压力。</li>
        <li>可多次扫码添加不同的网易云账号，相同 MUSIC_U 的账号会自动更新而非重复添加。</li>
        <li>双击昵称可以修改账号备注名称。</li>
        <li>禁用账号不会删除，只是暂时不参与轮询。号池为空时会回退到 cookie.txt 文件。</li>
        <li>Cookie 有效期通常为数周至数月，失效后需重新扫码获取。</li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import {
  QrCode, ClipboardPaste, RefreshCw, CheckCircle2, XCircle,
  Smartphone, Info, Clock, Trash2, DatabaseZap, Users, Plus
} from 'lucide-vue-next'
import { Loading } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import QRCode from 'qrcode'
import { useAdminStore } from '@/stores/admin'

const adminStore = useAdminStore()

// ── Account Pool ──────────────────────────────────────────────────────────────

const accounts = ref([])
const poolTotal = ref(0)
const poolActive = ref(0)
const poolLoading = ref(false)

const editingId = ref(null)
const editNickname = ref('')
const nicknameInput = ref(null)

async function loadAccounts() {
  poolLoading.value = true
  try {
    const data = await adminStore.listNeteaseAccounts()
    accounts.value = data.list || []
    poolTotal.value = data.total || 0
    poolActive.value = data.active || 0
  } catch (err) {
    ElMessage.error(err.response?.data?.message ?? '获取号池失败')
    accounts.value = []
    poolTotal.value = 0
    poolActive.value = 0
  } finally {
    poolLoading.value = false
  }
}

async function toggleActive(id, val) {
  try {
    await adminStore.updateNeteaseAccount(id, { is_active: val })
    await loadAccounts()
  } catch (err) {
    ElMessage.error(err.response?.data?.message ?? '操作失败')
  }
}

async function removeAccount(id, nickname) {
  try {
    await ElMessageBox.confirm(
      `确认删除账号「${nickname || '未命名'}」？此操作不可恢复。`,
      '删除确认',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch { return }

  try {
    await adminStore.deleteNeteaseAccount(id)
    ElMessage.success('账号已删除')
    await loadAccounts()
  } catch (err) {
    ElMessage.error(err.response?.data?.message ?? '删除失败')
  }
}

function startEdit(acc) {
  editingId.value = acc.id
  editNickname.value = acc.nickname || ''
  nextTick(() => {
    const inputs = nicknameInput.value
    if (inputs) {
      const el = Array.isArray(inputs) ? inputs[0] : inputs
      el?.focus?.()
    }
  })
}

async function saveNickname(id) {
  if (editingId.value !== id) return
  const name = editNickname.value.trim()
  editingId.value = null
  try {
    await adminStore.updateNeteaseAccount(id, { nickname: name })
    await loadAccounts()
  } catch (err) {
    ElMessage.error(err.response?.data?.message ?? '更新失败')
  }
}

function truncateMusicU(val) {
  if (!val) return '-'
  if (val.length <= 16) return val
  return val.slice(0, 8) + '...' + val.slice(-8)
}

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(t)
  if (isNaN(d.getTime())) return '-'
  const pad = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// ── QR Login ──────────────────────────────────────────────────────────────────

const qrKey = ref('')
const qrDataUrl = ref('')
const qrStatus = ref(null)
const qrLoading = ref(false)
const polling = ref(false)
let pollTimer = null

const statusText = computed(() => {
  const map = { 800: '二维码已过期', 801: '等待扫码', 802: '已扫码，请在 App 内确认', 803: '登录成功！' }
  return map[qrStatus.value] ?? '等待扫码'
})

const statusClass = computed(() => {
  if (qrStatus.value === 803) return 'success'
  if (qrStatus.value === 800) return 'error'
  if (qrStatus.value === 802) return 'warning'
  return 'info'
})

const statusIcon = computed(() => {
  if (qrStatus.value === 803) return CheckCircle2
  if (qrStatus.value === 800) return XCircle
  if (qrStatus.value === 802) return Smartphone
  return Clock
})

async function generateQR() {
  stopPolling()
  qrLoading.value = true
  qrStatus.value = 801
  qrKey.value = ''
  qrDataUrl.value = ''

  try {
    const data = await adminStore.getNeteaseQRKey()
    qrKey.value = data.key

    qrDataUrl.value = await QRCode.toDataURL(data.login_url, {
      width: 200,
      margin: 2,
      color: { dark: '#000000', light: '#ffffff' }
    })

    startPolling()
  } catch (err) {
    ElMessage.error(err.response?.data?.message ?? '生成二维码失败')
    qrStatus.value = null
  } finally {
    qrLoading.value = false
  }
}

function startPolling() {
  if (polling.value) return
  polling.value = true
  poll()
}

async function poll() {
  if (!polling.value || !qrKey.value) return
  try {
    const data = await adminStore.checkNeteaseQRStatus(qrKey.value)
    qrStatus.value = data.code
    if (data.code === 803) {
      stopPolling()
      await loadAccounts()
      if (data.saved_to_pool) {
        ElMessage.success('登录成功，账号已添加到号池！')
      } else if (data.saved_to_cookie_file) {
        ElMessage.warning(data.warning || '登录成功，但未写入号池，已回退写入 Cookie 文件。')
      } else if (data.saved_to_memory) {
        ElMessage.warning(data.warning || '登录成功，但未持久化，已临时写入内存 Cookie（重启后失效）。')
      } else {
        ElMessage.error(data.warning || '登录已授权，但未获取到可用 Cookie，请重新生成二维码重试。')
      }
      return
    }
    if (data.code === 800) {
      stopPolling()
      return
    }
  } catch {
    // ignore transient errors
  }
  if (polling.value) {
    pollTimer = setTimeout(poll, 2000)
  }
}

function stopPolling() {
  polling.value = false
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

// ── Manual Cookie ─────────────────────────────────────────────────────────────

const manualNickname = ref('')
const manualCookie = ref('')
const savingManual = ref(false)
const saveResult = ref(null)

async function addManualAccount() {
  savingManual.value = true
  saveResult.value = null
  try {
    await adminStore.addNeteaseAccount(manualNickname.value.trim(), manualCookie.value.trim())
    saveResult.value = { type: 'success', message: '账号已添加到号池' }
    ElMessage.success('账号添加成功')
    manualCookie.value = ''
    manualNickname.value = ''
    await loadAccounts()
  } catch (err) {
    const msg = err.response?.data?.message ?? '添加失败'
    saveResult.value = { type: 'error', message: msg }
    ElMessage.error(msg)
  } finally {
    savingManual.value = false
  }
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

onMounted(loadAccounts)
onUnmounted(stopPolling)
</script>

<style scoped>
.netease-config { max-width: 960px; }

/* Pool card */
.pool-card {
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
  margin-bottom: 10px;
  color: rgba(255,255,255,0.8);
}

.card-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.card-header h3, .card-header h4 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: #fff;
}

.pool-stats {
  margin-left: auto;
  display: flex;
  gap: 8px;
}

.stat-badge {
  font-size: 12px;
  padding: 2px 10px;
  border-radius: 20px;
  font-weight: 500;
}

.stat-badge.active { background: rgba(34,197,94,0.15); color: #4ade80; }
.stat-badge.total  { background: rgba(255,255,255,0.08); color: rgba(255,255,255,0.6); }

.card-desc {
  font-size: 13px;
  color: rgba(255,255,255,0.45);
  margin: 0 0 20px;
  line-height: 1.6;
}

.card-desc code {
  background: rgba(255,255,255,0.08);
  padding: 1px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  color: var(--accent);
}

/* Pool table */
.pool-table-wrap {
  overflow-x: auto;
  border-radius: 10px;
  border: 1px solid rgba(255,255,255,0.06);
}

.pool-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.pool-table th {
  text-align: left;
  padding: 10px 14px;
  font-weight: 500;
  color: rgba(255,255,255,0.5);
  background: rgba(255,255,255,0.03);
  border-bottom: 1px solid rgba(255,255,255,0.06);
  white-space: nowrap;
}

.pool-table td {
  padding: 10px 14px;
  border-bottom: 1px solid rgba(255,255,255,0.04);
  color: rgba(255,255,255,0.8);
}

.pool-table tr.inactive td { opacity: 0.45; }
.pool-table tr.inactive td.col-status { opacity: 1; }

.col-nickname { min-width: 100px; }
.col-musicU code {
  font-size: 11px;
  background: rgba(255,255,255,0.06);
  padding: 2px 6px;
  border-radius: 4px;
  color: var(--accent);
}
.col-time { white-space: nowrap; font-size: 12px; color: rgba(255,255,255,0.4); }

.nickname-text {
  cursor: pointer;
  border-bottom: 1px dashed rgba(255,255,255,0.2);
}

.nickname-input {
  background: rgba(255,255,255,0.08);
  border: 1px solid rgba(255,255,255,0.15);
  border-radius: 4px;
  color: #fff;
  padding: 2px 6px;
  font-size: 13px;
  width: 120px;
  outline: none;
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  border: none;
  background: transparent;
  cursor: pointer;
  color: rgba(255,255,255,0.4);
  transition: all 0.2s;
}

.icon-btn:hover { background: rgba(255,255,255,0.08); color: rgba(255,255,255,0.8); }
.icon-btn.danger:hover { background: rgba(239,68,68,0.15); color: #f87171; }

.pool-loading, .pool-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 0;
  gap: 12px;
  color: rgba(255,255,255,0.35);
}

.pool-empty .empty-icon { opacity: 0.25; }
.pool-empty p { font-size: 13px; margin: 0; }

/* Add account grid */
.config-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 20px;
}

@media (max-width: 700px) { .config-grid { grid-template-columns: 1fr; } }

.config-card, .tips-card {
  background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.07);
  border-radius: 16px;
  padding: 24px;
}

.nickname-field { margin-bottom: 12px; }

:deep(.nickname-field .el-input__inner) {
  background: var(--bg-input) !important;
  border-color: var(--border-color) !important;
  color: var(--text-primary);
}

/* QR area */
.qr-area { min-height: 200px; display: flex; flex-direction: column; align-items: center; margin-bottom: 20px; }

.qr-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 180px;
  color: rgba(255,255,255,0.3);
  text-align: center;
  gap: 12px;
}

.qr-empty-icon { opacity: 0.3; }

.qr-placeholder p { font-size: 13px; margin: 0; }

.qr-image-wrap {
  position: relative;
  border-radius: 12px;
  overflow: hidden;
  border: 2px solid rgba(255,255,255,0.1);
  transition: border-color 0.3s;
}

.qr-image-wrap.expired { border-color: rgba(239,68,68,0.4); }
.qr-image-wrap.authorized { border-color: rgba(34,197,94,0.5); }

.qr-image {
  display: block;
  width: 200px;
  height: 200px;
}

.qr-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  text-align: center;
}

.qr-overlay.expired  { background: rgba(0,0,0,0.75); color: #f87171; }
.qr-overlay.success  { background: rgba(0,0,0,0.75); color: #4ade80; }
.qr-overlay.scan     { background: rgba(0,0,0,0.55); color: #fbbf24; }

.qr-status-text {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  font-size: 13px;
  font-weight: 500;
}

.qr-status-text.success { color: #4ade80; }
.qr-status-text.error   { color: #f87171; }
.qr-status-text.warning { color: #fbbf24; }
.qr-status-text.info    { color: rgba(255,255,255,0.5); }

.card-actions { display: flex; gap: 10px; flex-wrap: wrap; }

.cookie-textarea { margin-bottom: 16px; }

:deep(.el-textarea__inner) {
  background: var(--bg-input) !important;
  border-color: var(--border-color) !important;
  color: var(--text-primary);
  font-family: monospace;
  font-size: 12px;
  resize: vertical;
}

.save-result {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 13px;
}

.save-result.success { background: rgba(34,197,94,0.1); color: #4ade80; }
.save-result.error   { background: rgba(239,68,68,0.1); color: #f87171; }

.tips-card { margin-top: 0; }

.tips-list {
  margin: 12px 0 0;
  padding-left: 20px;
  color: rgba(255,255,255,0.45);
  font-size: 13px;
  line-height: 2;
}

/* Day theme overrides */
[data-theme="day"] .pool-card,
[data-theme="day"] .config-card,
[data-theme="day"] .tips-card {
  background: #fff; border-color: rgba(0,0,0,0.08);
  box-shadow: 0 1px 3px rgba(0,0,0,0.06);
}

[data-theme="day"] .card-header { color: rgba(0,0,0,0.6); }
[data-theme="day"] .card-header h3,
[data-theme="day"] .card-header h4 { color: rgba(0,0,0,0.85); }

[data-theme="day"] .card-desc { color: rgba(0,0,0,0.5); }
[data-theme="day"] .card-desc code { background: rgba(0,0,0,0.05); color: var(--accent); }

[data-theme="day"] .stat-badge.active { background: rgba(34,197,94,0.1); color: #16a34a; }
[data-theme="day"] .stat-badge.total  { background: rgba(0,0,0,0.05); color: rgba(0,0,0,0.5); }

[data-theme="day"] .pool-table th { color: rgba(0,0,0,0.5); background: rgba(0,0,0,0.02); border-color: rgba(0,0,0,0.06); }
[data-theme="day"] .pool-table td { color: rgba(0,0,0,0.8); border-color: rgba(0,0,0,0.05); }
[data-theme="day"] .pool-table-wrap { border-color: rgba(0,0,0,0.08); }
[data-theme="day"] .col-musicU code { background: rgba(0,0,0,0.04); color: var(--accent); }
[data-theme="day"] .col-time { color: rgba(0,0,0,0.4); }
[data-theme="day"] .nickname-text { border-color: rgba(0,0,0,0.15); }
[data-theme="day"] .nickname-input { background: #fff; border-color: rgba(0,0,0,0.15); color: rgba(0,0,0,0.85); }
[data-theme="day"] .icon-btn { color: rgba(0,0,0,0.35); }
[data-theme="day"] .icon-btn:hover { background: rgba(0,0,0,0.05); color: rgba(0,0,0,0.7); }
[data-theme="day"] .icon-btn.danger:hover { background: rgba(239,68,68,0.1); color: #dc2626; }
[data-theme="day"] .pool-loading,
[data-theme="day"] .pool-empty { color: rgba(0,0,0,0.3); }

[data-theme="day"] .qr-placeholder { color: rgba(0,0,0,0.3); }
[data-theme="day"] .qr-image-wrap { border-color: rgba(0,0,0,0.1); }
[data-theme="day"] .qr-status-text.info { color: rgba(0,0,0,0.5); }

[data-theme="day"] .tips-list { color: rgba(0,0,0,0.5); }
</style>
