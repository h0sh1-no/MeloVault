<template>
  <div class="users-page">
    <!-- Toolbar -->
    <div class="toolbar">
      <el-input
        v-model="search"
        placeholder="搜索用户名或邮箱..."
        clearable
        class="search-input"
        @keyup.enter="loadUsers"
        @clear="loadUsers"
      >
        <template #prefix><Search :size="16" /></template>
      </el-input>
      <el-button type="primary" :loading="loading" @click="loadUsers">搜索</el-button>
      <el-button type="success" @click="openCreateDialog">
        <Plus :size="16" style="margin-right: 4px" /> 新建用户
      </el-button>
    </div>

    <!-- Table -->
    <el-table
      :data="users"
      v-loading="loading"
      class="users-table"
      row-key="id"
      stripe
    >
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column label="用户" min-width="160">
        <template #default="{ row }">
          <div class="user-cell">
            <el-avatar :src="row.avatar" :size="32" class="user-avatar">
              {{ row.username?.[0]?.toUpperCase() }}
            </el-avatar>
            <div>
              <div class="username">{{ row.username }}</div>
              <div class="email">{{ row.email || '—' }}</div>
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="来源" width="100">
        <template #default="{ row }">
          <el-tag size="small" :type="row.provider === 'linuxdo' ? 'warning' : 'info'">
            {{ row.provider }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="角色" width="120">
        <template #default="{ row }">
          <el-tag size="small" :type="roleTagType(row.role)">{{ roleLabel(row.role) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="fav_count" label="收藏" width="80" align="center" />
      <el-table-column prop="down_count" label="下载" width="80" align="center" />
      <el-table-column label="注册时间" width="160">
        <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <div class="action-cell">
            <el-button size="small" text @click="openUserDetail(row)">详情</el-button>
            <el-dropdown trigger="click" @command="(cmd) => handleAction(cmd, row)">
              <button class="more-btn" title="更多操作">
                <MoreHorizontal :size="16" />
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">
                    <Pencil :size="14" class="dropdown-icon" /> 编辑
                  </el-dropdown-item>
                  <el-dropdown-item command="resetPwd">
                    <KeyRound :size="14" class="dropdown-icon" /> 重置密码
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" :disabled="row.role === 'superadmin'" divided>
                    <Trash2 :size="14" class="dropdown-icon danger" /> <span class="danger-text">删除</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <!-- Pagination -->
    <div class="pagination-wrap">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        background
        @size-change="loadUsers"
        @current-change="loadUsers"
      />
    </div>

    <!-- Create User Dialog -->
    <el-dialog v-model="createDialogVisible" title="新建用户" width="460px" class="admin-dialog">
      <el-form :model="createForm" label-position="top">
        <el-form-item label="用户名" required>
          <el-input v-model="createForm.username" placeholder="2-50 个字符" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="createForm.email" placeholder="选填" />
        </el-form-item>
        <el-form-item label="密码" required>
          <el-input v-model="createForm.password" type="password" show-password placeholder="至少 6 个字符" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="createForm.role" style="width: 100%">
            <el-option label="普通用户" value="user" />
            <el-option label="管理员" value="admin" />
            <el-option
              label="超级管理员" value="superadmin"
              :disabled="currentUserRole !== 'superadmin'"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>

    <!-- Edit User Dialog -->
    <el-dialog v-model="editDialogVisible" title="编辑用户" width="420px" class="admin-dialog">
      <el-form :model="editForm" label-position="top">
        <el-form-item label="用户名">
          <el-input v-model="editForm.username" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="editForm.role" style="width: 100%">
            <el-option label="普通用户" value="user" />
            <el-option label="管理员" value="admin" />
            <el-option
              label="超级管理员" value="superadmin"
              :disabled="currentUserRole !== 'superadmin'"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSaveEdit">保存</el-button>
      </template>
    </el-dialog>

    <!-- Reset Password Dialog -->
    <el-dialog v-model="resetPwdDialogVisible" title="重置密码" width="420px" class="admin-dialog">
      <p class="reset-hint">为用户 <strong>{{ resetPwdForm.username }}</strong> 设置新密码</p>
      <el-form :model="resetPwdForm" label-position="top">
        <el-form-item label="新密码" required>
          <el-input v-model="resetPwdForm.password" type="password" show-password placeholder="至少 6 个字符" />
        </el-form-item>
        <el-form-item label="确认密码" required>
          <el-input v-model="resetPwdForm.confirmPassword" type="password" show-password placeholder="再次输入新密码" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetPwdDialogVisible = false">取消</el-button>
        <el-button type="warning" :loading="saving" @click="handleResetPassword">重置密码</el-button>
      </template>
    </el-dialog>

    <!-- User Detail Drawer -->
    <el-drawer
      v-model="detailVisible"
      :title="`用户详情 - ${detailUser?.username || ''}`"
      size="720px"
      class="detail-drawer"
      @close="onDrawerClose"
    >
      <template v-if="detailUser">
        <div class="detail-header">
          <el-avatar :src="detailUser.avatar" :size="56" class="detail-avatar">
            {{ detailUser.username?.[0]?.toUpperCase() }}
          </el-avatar>
          <div class="detail-info">
            <div class="detail-name">{{ detailUser.username }}</div>
            <div class="detail-meta">
              <el-tag size="small" :type="roleTagType(detailUser.role)">{{ roleLabel(detailUser.role) }}</el-tag>
              <span>{{ detailUser.email || '无邮箱' }}</span>
              <span>{{ detailUser.provider }}</span>
            </div>
            <div class="detail-stats">
              <span>收藏 <strong>{{ detailUser.fav_count }}</strong></span>
              <span>下载 <strong>{{ detailUser.down_count }}</strong></span>
              <span>注册于 {{ formatDate(detailUser.created_at) }}</span>
            </div>
          </div>
        </div>

        <!-- Activity Tabs -->
        <el-tabs v-model="detailTab" class="detail-tabs" @tab-change="onTabChange">
          <!-- All Activity -->
          <el-tab-pane label="活动记录" name="activity">
            <el-table :data="activityData.list" v-loading="activityData.loading" class="detail-table" size="small" stripe>
              <el-table-column label="时间" width="150">
                <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="90">
                <template #default="{ row }">
                  <el-tag size="small" :type="actionTagType(row.action)">{{ actionLabel(row.action) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="IP" width="130">
                <template #default="{ row }"><code class="ip-code">{{ row.ip || '—' }}</code></template>
              </el-table-column>
              <el-table-column label="省份" width="80">
                <template #default="{ row }">{{ row.province || '—' }}</template>
              </el-table-column>
              <el-table-column label="城市" width="80">
                <template #default="{ row }">{{ row.city || '—' }}</template>
              </el-table-column>
            </el-table>
            <div class="detail-pagination" v-if="activityData.total > tabPageSize">
              <el-pagination
                v-model:current-page="activityData.page"
                :page-size="tabPageSize"
                :total="activityData.total"
                layout="prev, pager, next"
                small background
                @current-change="() => loadTabData('activity')"
              />
            </div>
          </el-tab-pane>

          <!-- Plays -->
          <el-tab-pane label="听歌记录" name="plays">
            <el-table :data="playsData.list" v-loading="playsData.loading" class="detail-table" size="small" stripe>
              <el-table-column label="时间" width="150">
                <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
              </el-table-column>
              <el-table-column label="IP" width="130">
                <template #default="{ row }"><code class="ip-code">{{ row.ip || '—' }}</code></template>
              </el-table-column>
              <el-table-column label="省份" width="80">
                <template #default="{ row }">{{ row.province || '—' }}</template>
              </el-table-column>
              <el-table-column label="城市" width="80">
                <template #default="{ row }">{{ row.city || '—' }}</template>
              </el-table-column>
            </el-table>
            <div class="empty-state" v-if="!playsData.loading && !playsData.list.length">暂无听歌记录</div>
            <div class="detail-pagination" v-if="playsData.total > tabPageSize">
              <el-pagination
                v-model:current-page="playsData.page"
                :page-size="tabPageSize"
                :total="playsData.total"
                layout="prev, pager, next"
                small background
                @current-change="() => loadTabData('plays')"
              />
            </div>
          </el-tab-pane>

          <!-- Downloads -->
          <el-tab-pane label="下载记录" name="downloads">
            <el-table :data="downloadsData.list" v-loading="downloadsData.loading" class="detail-table" size="small" stripe>
              <el-table-column label="时间" width="150">
                <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
              </el-table-column>
              <el-table-column label="歌曲" min-width="140">
                <template #default="{ row }">
                  <div class="song-cell">
                    <span class="song-name">{{ row.song_name || '—' }}</span>
                    <span class="song-artist">{{ row.artists || '' }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="品质" width="80">
                <template #default="{ row }">
                  <el-tag size="small">{{ row.quality || '—' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="大小" width="90">
                <template #default="{ row }">{{ formatFileSize(row.file_size) }}</template>
              </el-table-column>
            </el-table>
            <div class="empty-state" v-if="!downloadsData.loading && !downloadsData.list.length">暂无下载记录</div>
            <div class="detail-pagination" v-if="downloadsData.total > tabPageSize">
              <el-pagination
                v-model:current-page="downloadsData.page"
                :page-size="tabPageSize"
                :total="downloadsData.total"
                layout="prev, pager, next"
                small background
                @current-change="() => loadTabData('downloads')"
              />
            </div>
          </el-tab-pane>

          <!-- Logins -->
          <el-tab-pane label="登录记录" name="logins">
            <el-table :data="loginsData.list" v-loading="loginsData.loading" class="detail-table" size="small" stripe>
              <el-table-column label="时间" width="150">
                <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
              </el-table-column>
              <el-table-column label="IP" width="130">
                <template #default="{ row }"><code class="ip-code">{{ row.ip || '—' }}</code></template>
              </el-table-column>
              <el-table-column label="省份" width="80">
                <template #default="{ row }">{{ row.province || '—' }}</template>
              </el-table-column>
              <el-table-column label="城市" width="80">
                <template #default="{ row }">{{ row.city || '—' }}</template>
              </el-table-column>
            </el-table>
            <div class="empty-state" v-if="!loginsData.loading && !loginsData.list.length">暂无登录记录</div>
            <div class="detail-pagination" v-if="loginsData.total > tabPageSize">
              <el-pagination
                v-model:current-page="loginsData.page"
                :page-size="tabPageSize"
                :total="loginsData.total"
                layout="prev, pager, next"
                small background
                @current-change="() => loadTabData('logins')"
              />
            </div>
          </el-tab-pane>
        </el-tabs>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { Search, Plus, MoreHorizontal, Pencil, KeyRound, Trash2 } from 'lucide-vue-next'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAdminStore } from '@/stores/admin'
import { useAuthStore } from '@/stores/auth'

const adminStore = useAdminStore()
const authStore = useAuthStore()
const currentUserRole = computed(() => authStore.user?.role)

const users = ref([])
const loading = ref(false)
const saving = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const search = ref('')

const createDialogVisible = ref(false)
const createForm = reactive({ username: '', email: '', password: '', role: 'user' })

const editDialogVisible = ref(false)
const editForm = reactive({ id: 0, username: '', role: '' })

const resetPwdDialogVisible = ref(false)
const resetPwdForm = reactive({ id: 0, username: '', password: '', confirmPassword: '' })

const detailVisible = ref(false)
const detailUser = ref(null)
const detailTab = ref('activity')
const tabPageSize = 15

function makeTabState() {
  return { list: [], total: 0, page: 1, loading: false }
}
const activityData = reactive(makeTabState())
const playsData = reactive(makeTabState())
const downloadsData = reactive(makeTabState())
const loginsData = reactive(makeTabState())

function roleTagType(role) {
  return { superadmin: 'danger', admin: 'warning', user: 'info' }[role] ?? 'info'
}
function roleLabel(role) {
  return { superadmin: '超级管理员', admin: '管理员', user: '普通用户' }[role] ?? role
}
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
function formatFileSize(bytes) {
  if (!bytes || bytes <= 0) return '—'
  const units = ['B', 'KB', 'MB', 'GB']
  let size = bytes, unit = 0
  while (size >= 1024 && unit < units.length - 1) { size /= 1024; unit++ }
  return `${size.toFixed(1)} ${units[unit]}`
}

// ── User List ──
async function loadUsers() {
  loading.value = true
  try {
    const data = await adminStore.listUsers(page.value, pageSize.value, search.value)
    users.value = data?.list ?? []
    total.value = data?.total ?? 0
  } catch {
    ElMessage.error('获取用户列表失败')
  } finally {
    loading.value = false
  }
}

// ── Create User ──
function openCreateDialog() {
  createForm.username = ''
  createForm.email = ''
  createForm.password = ''
  createForm.role = 'user'
  createDialogVisible.value = true
}

async function handleCreate() {
  if (!createForm.username.trim()) return ElMessage.warning('用户名不能为空')
  if (createForm.username.trim().length < 2) return ElMessage.warning('用户名至少 2 个字符')
  if (!createForm.password) return ElMessage.warning('密码不能为空')
  if (createForm.password.length < 6) return ElMessage.warning('密码至少 6 个字符')

  saving.value = true
  try {
    await adminStore.createUser(createForm.username.trim(), createForm.email.trim(), createForm.password, createForm.role)
    ElMessage.success('用户创建成功')
    createDialogVisible.value = false
    loadUsers()
  } catch (err) {
    ElMessage.error(err.response?.data?.message ?? '创建失败')
  } finally {
    saving.value = false
  }
}

// ── Edit User ──
function openEditDialog(row) {
  editForm.id = row.id
  editForm.username = row.username
  editForm.role = row.role
  editDialogVisible.value = true
}

async function handleSaveEdit() {
  if (!editForm.username.trim()) return ElMessage.warning('用户名不能为空')
  saving.value = true
  try {
    await adminStore.updateUser(editForm.id, editForm.username.trim(), editForm.role)
    ElMessage.success('更新成功')
    editDialogVisible.value = false
    loadUsers()
  } catch (err) {
    ElMessage.error(err.response?.data?.message ?? '更新失败')
  } finally {
    saving.value = false
  }
}

// ── Reset Password ──
function openResetPwdDialog(row) {
  resetPwdForm.id = row.id
  resetPwdForm.username = row.username
  resetPwdForm.password = ''
  resetPwdForm.confirmPassword = ''
  resetPwdDialogVisible.value = true
}

async function handleResetPassword() {
  if (!resetPwdForm.password) return ElMessage.warning('请输入新密码')
  if (resetPwdForm.password.length < 6) return ElMessage.warning('密码至少 6 个字符')
  if (resetPwdForm.password !== resetPwdForm.confirmPassword) return ElMessage.warning('两次输入的密码不一致')

  saving.value = true
  try {
    await adminStore.resetPassword(resetPwdForm.id, resetPwdForm.password)
    ElMessage.success('密码重置成功')
    resetPwdDialogVisible.value = false
  } catch (err) {
    ElMessage.error(err.response?.data?.message ?? '重置失败')
  } finally {
    saving.value = false
  }
}

// ── Delete User ──
async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`确认删除用户 "${row.username}"？此操作不可恢复。`, '警告', {
      confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning',
      confirmButtonClass: 'el-button--danger'
    })
  } catch { return }

  try {
    await adminStore.deleteUser(row.id)
    ElMessage.success('用户已删除')
    loadUsers()
  } catch (err) {
    ElMessage.error(err.response?.data?.message ?? '删除失败')
  }
}

// ── Action Dispatcher ──
function handleAction(cmd, row) {
  switch (cmd) {
    case 'edit': openEditDialog(row); break
    case 'resetPwd': openResetPwdDialog(row); break
    case 'delete': handleDelete(row); break
  }
}

// ── Detail Drawer ──
async function openUserDetail(row) {
  detailUser.value = row
  detailTab.value = 'activity'
  resetAllTabs()
  detailVisible.value = true
  await loadTabData('activity')
}

function onDrawerClose() {
  detailUser.value = null
  resetAllTabs()
}

function resetAllTabs() {
  for (const s of [activityData, playsData, downloadsData, loginsData]) {
    s.list = []; s.total = 0; s.page = 1; s.loading = false
  }
}

function onTabChange(tab) {
  const state = getTabState(tab)
  if (!state.list.length && !state.loading) {
    loadTabData(tab)
  }
}

function getTabState(tab) {
  return { activity: activityData, plays: playsData, downloads: downloadsData, logins: loginsData }[tab]
}

async function loadTabData(tab) {
  if (!detailUser.value) return
  const uid = detailUser.value.id
  const state = getTabState(tab)
  state.loading = true

  try {
    if (tab === 'downloads') {
      const data = await adminStore.getUserDownloads(uid, state.page, tabPageSize)
      state.list = data?.list || []
      state.total = data?.total || 0
    } else {
      const actionMap = { activity: '', plays: 'play', logins: 'login' }
      const data = await adminStore.getUserActivity(uid, state.page, tabPageSize, actionMap[tab])
      state.list = data?.list || []
      state.total = data?.total || 0
    }
  } catch {
    ElMessage.error('获取数据失败')
  } finally {
    state.loading = false
  }
}

onMounted(loadUsers)
</script>

<style scoped>
.users-page {
  --tbl-bg: rgba(255,255,255,0.03);
  --tbl-text: rgba(255,255,255,0.85);
  --tbl-header-bg: rgba(255,255,255,0.05);
  --tbl-header-text: rgba(255,255,255,0.5);
  --tbl-stripe: rgba(255,255,255,0.02);
  --tbl-hover: rgba(124,58,237,0.08);
  --dialog-surface: #1e1e32;
  --dialog-border: rgba(255,255,255,0.1);
  --drawer-bg: #13132b;
  --drawer-border: rgba(255,255,255,0.06);
}

[data-theme="day"] .users-page {
  --tbl-bg: #ffffff;
  --tbl-text: rgba(0,0,0,0.85);
  --tbl-header-bg: #fafafa;
  --tbl-header-text: rgba(0,0,0,0.5);
  --tbl-stripe: rgba(0,0,0,0.02);
  --tbl-hover: rgba(230,57,70,0.06);
  --dialog-surface: #ffffff;
  --dialog-border: rgba(0,0,0,0.1);
  --drawer-bg: #ffffff;
  --drawer-border: rgba(0,0,0,0.08);
}

.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}
.search-input { max-width: 320px; }

.users-table { background: transparent; border-radius: 12px; overflow: hidden; }

:deep(.el-table) { background: var(--tbl-bg) !important; color: var(--tbl-text); }
:deep(.el-table th) { background: var(--tbl-header-bg) !important; color: var(--tbl-header-text); font-size: 12px; text-transform: uppercase; letter-spacing: 0.05em; }
:deep(.el-table tr) { background: transparent !important; }
:deep(.el-table--striped .el-table__body tr.el-table__row--striped td) { background: var(--tbl-stripe) !important; }
:deep(.el-table__body tr:hover > td) { background: var(--tbl-hover) !important; }

.user-cell { display: flex; align-items: center; gap: 10px; }
.user-avatar { flex-shrink: 0; }
.username { font-size: 14px; font-weight: 500; color: var(--text-primary); }
.email { font-size: 12px; color: var(--text-faint); }

.action-cell { display: flex; align-items: center; gap: 4px; }

.more-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  border: none;
  background: transparent;
  cursor: pointer;
  color: var(--text-muted);
  transition: all 0.2s;
}
.more-btn:hover { background: var(--tbl-hover); color: var(--text-primary); }

.dropdown-icon { margin-right: 6px; vertical-align: -2px; }
.dropdown-icon.danger { color: var(--el-color-danger); }
.danger-text { color: var(--el-color-danger); }

.pagination-wrap { margin-top: 20px; display: flex; justify-content: flex-end; }

:deep(.admin-dialog .el-dialog) { background: var(--dialog-surface); border: 1px solid var(--dialog-border); }
:deep(.admin-dialog .el-dialog__title) { color: var(--text-primary); }

.reset-hint { color: var(--text-muted); font-size: 14px; margin-bottom: 16px; }
.reset-hint strong { color: var(--text-primary); }

/* Detail Drawer */
:deep(.detail-drawer .el-drawer) { background: var(--drawer-bg) !important; }
:deep(.detail-drawer .el-drawer__header) { color: var(--text-primary); border-bottom: 1px solid var(--drawer-border); margin-bottom: 0; padding-bottom: 16px; }

.detail-header {
  display: flex; gap: 16px; align-items: flex-start;
  padding-bottom: 20px; border-bottom: 1px solid var(--drawer-border); margin-bottom: 20px;
}
.detail-avatar { flex-shrink: 0; }
.detail-name { font-size: 18px; font-weight: 600; color: var(--text-primary); margin-bottom: 6px; }
.detail-meta { display: flex; gap: 8px; align-items: center; font-size: 13px; color: var(--text-muted); flex-wrap: wrap; margin-bottom: 8px; }
.detail-stats { display: flex; gap: 16px; font-size: 13px; color: var(--text-muted); }
.detail-stats strong { color: var(--text-secondary); font-weight: 600; }

:deep(.detail-tabs .el-tabs__header) { margin-bottom: 16px; }
:deep(.detail-tabs .el-tabs__item) { color: var(--text-muted); }
:deep(.detail-tabs .el-tabs__item.is-active) { color: var(--accent); }
:deep(.detail-tabs .el-tabs__active-bar) { background: var(--accent); }

:deep(.detail-table .el-table) { background: transparent !important; color: var(--tbl-text); }
:deep(.detail-table .el-table th) { background: var(--tbl-stripe) !important; color: var(--tbl-header-text); font-size: 12px; }
:deep(.detail-table .el-table tr) { background: transparent !important; }
:deep(.detail-table .el-table--striped .el-table__body tr.el-table__row--striped td) { background: var(--tbl-stripe) !important; }

.ip-code { font-size: 12px; color: var(--text-muted); background: var(--bg-elevated); padding: 2px 6px; border-radius: 4px; font-family: 'Courier New', monospace; }

.song-cell { display: flex; flex-direction: column; gap: 2px; }
.song-name { font-size: 13px; color: var(--text-primary); }
.song-artist { font-size: 12px; color: var(--text-faint); }

.detail-pagination { margin-top: 12px; display: flex; justify-content: flex-end; }
.empty-state { text-align: center; padding: 24px; color: var(--text-faint); font-size: 14px; }
</style>
