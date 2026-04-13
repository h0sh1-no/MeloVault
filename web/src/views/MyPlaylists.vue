<template>
  <div class="my-playlists-page">
    <div class="page-header">
      <h2>我的歌单</h2>
      <el-button class="create-btn" round @click="showCreateDialog = true">
        <Plus :size="16" />
        <span>新建歌单</span>
      </el-button>
    </div>

    <div class="playlist-grid" v-if="loading || playlists.length > 0" v-loading="loading">
      <div
        v-for="pl in playlists"
        :key="pl.id"
        class="playlist-card"
        @click="$router.push(`/my-playlists/${pl.id}`)"
      >
        <div class="card-cover">
          <el-avatar shape="square" :size="120" :src="pl.cover_url || undefined">
            <ListMusic :size="40" />
          </el-avatar>
          <div class="card-count">
            <Music :size="12" />
            <span>{{ pl.song_count }}</span>
          </div>
        </div>
        <div class="card-info">
          <div class="card-name">{{ pl.name }}</div>
          <div class="card-meta">
            <span v-if="pl.is_public" class="public-badge">
              <Globe :size="12" /> 公开
            </span>
            <span v-else class="private-badge">
              <Lock :size="12" /> 私密
            </span>
          </div>
        </div>
        <el-dropdown trigger="click" @command="cmd => handleCommand(cmd, pl)" @click.stop>
          <button class="card-menu" @click.stop>
            <MoreVertical :size="16" />
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="edit">编辑歌单</el-dropdown-item>
              <el-dropdown-item command="togglePublic">
                {{ pl.is_public ? '设为私密' : '设为公开' }}
              </el-dropdown-item>
              <el-dropdown-item v-if="pl.is_public" command="share">分享歌单</el-dropdown-item>
              <el-dropdown-item command="delete" divided>
                <span style="color: var(--el-color-danger)">删除歌单</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <div class="empty-state" v-if="!loading && playlists.length === 0">
      <div class="empty-icon">
        <ListMusic :size="48" />
      </div>
      <h3>还没有创建歌单</h3>
      <p>创建自己的歌单，收集喜欢的音乐</p>
      <el-button class="action-btn" round @click="showCreateDialog = true">
        <Plus :size="16" />
        <span>新建歌单</span>
      </el-button>
    </div>

    <!-- Create / Edit dialog -->
    <el-dialog
      v-model="showCreateDialog"
      :title="editingPlaylist ? '编辑歌单' : '新建歌单'"
      width="420px"
      :close-on-click-modal="false"
    >
      <el-form :model="form" label-position="top">
        <el-form-item label="歌单名称" required>
          <el-input v-model="form.name" placeholder="输入歌单名称" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="添加描述（可选）" maxlength="500" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="closeDialog">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">
          {{ editingPlaylist ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Plus, ListMusic, Music, Globe, Lock, MoreVertical } from 'lucide-vue-next'
import { ElMessage, ElMessageBox } from 'element-plus'
import { usePlaylistStore } from '@/stores/playlist'
import { useAuthStore } from '@/stores/auth'

const playlistStore = usePlaylistStore()
const authStore = useAuthStore()

const playlists = ref([])
const loading = ref(false)
const showCreateDialog = ref(false)
const editingPlaylist = ref(null)
const submitting = ref(false)

const form = ref({ name: '', description: '' })

onMounted(() => {
  fetchPlaylists()
})

async function fetchPlaylists() {
  loading.value = true
  try {
    await playlistStore.fetchMyPlaylists()
    playlists.value = playlistStore.myPlaylists
  } catch (e) {
    ElMessage.error('获取歌单列表失败')
  } finally {
    loading.value = false
  }
}

function closeDialog() {
  showCreateDialog.value = false
  editingPlaylist.value = null
  form.value = { name: '', description: '' }
}

async function submitForm() {
  if (!form.value.name.trim()) {
    ElMessage.warning('请输入歌单名称')
    return
  }
  submitting.value = true
  try {
    if (editingPlaylist.value) {
      await playlistStore.updatePlaylist(editingPlaylist.value.id, {
        name: form.value.name.trim(),
        description: form.value.description
      })
      ElMessage.success('歌单已更新')
    } else {
      await playlistStore.createPlaylist(form.value.name.trim(), form.value.description)
      ElMessage.success('歌单已创建')
    }
    playlists.value = playlistStore.myPlaylists
    closeDialog()
  } catch (e) {
    ElMessage.error(editingPlaylist.value ? '更新失败' : '创建失败')
  } finally {
    submitting.value = false
  }
}

async function handleCommand(cmd, pl) {
  if (cmd === 'edit') {
    editingPlaylist.value = pl
    form.value = { name: pl.name, description: pl.description || '' }
    showCreateDialog.value = true
  } else if (cmd === 'togglePublic') {
    try {
      await playlistStore.updatePlaylist(pl.id, { is_public: !pl.is_public })
      playlists.value = playlistStore.myPlaylists
      ElMessage.success(pl.is_public ? '已设为私密' : '已设为公开')
    } catch (e) {
      ElMessage.error('操作失败')
    }
  } else if (cmd === 'share') {
    const userId = authStore.user?.id
    const url = `${window.location.origin}/shared/playlist/${pl.id}?sharer=${userId}`
    try {
      await navigator.clipboard.writeText(url)
      ElMessage.success('分享链接已复制到剪贴板')
    } catch {
      ElMessage.info('分享链接: ' + url)
    }
  } else if (cmd === 'delete') {
    try {
      await ElMessageBox.confirm(`确定删除歌单「${pl.name}」吗？歌单内的歌曲将一并删除。`, '删除歌单', {
        type: 'warning',
        confirmButtonText: '删除',
        cancelButtonText: '取消'
      })
      await playlistStore.deletePlaylist(pl.id)
      playlists.value = playlistStore.myPlaylists
      ElMessage.success('歌单已删除')
    } catch (e) {
      if (e !== 'cancel') ElMessage.error('删除失败')
    }
  }
}
</script>

<style lang="scss" scoped>
.my-playlists-page {
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
  margin-bottom: 28px;

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

  .create-btn {
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
}

.playlist-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 24px;
  min-height: 300px;

  @media (max-width: 640px) {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }
}

.playlist-card {
  position: relative;
  background: var(--bg-card);
  border: var(--border-width) solid var(--card-border);
  border-radius: var(--radius);
  padding: 16px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    border-color: var(--accent);
    box-shadow: var(--shadow-hover);
    transform: translateY(-2px);
  }

  .card-cover {
    position: relative;
    margin-bottom: 12px;

    :deep(.el-avatar) {
      width: 100% !important;
      height: auto !important;
      aspect-ratio: 1;
      border-radius: var(--radius-sm);
      border: var(--border-width) solid var(--card-border);
      background: var(--bg-elevated);
    }

    .card-count {
      position: absolute;
      bottom: 8px;
      right: 8px;
      display: flex;
      align-items: center;
      gap: 4px;
      background: rgba(0,0,0,0.6);
      color: #fff;
      padding: 2px 8px;
      border-radius: 12px;
      font-size: 12px;
    }
  }

  .card-info {
    .card-name {
      color: var(--text-primary);
      font-size: 14px;
      font-weight: 500;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      margin-bottom: 4px;
    }

    .card-meta {
      font-size: 12px;
      color: var(--text-muted);

      .public-badge, .private-badge {
        display: inline-flex;
        align-items: center;
        gap: 3px;
      }
    }
  }

  :deep(.el-dropdown) {
    position: absolute;
    top: 8px;
    right: 8px;
    opacity: 0;
    transition: opacity 0.2s;

    @media (hover: none) {
      opacity: 1;
    }
  }

  &:hover :deep(.el-dropdown) {
    opacity: 1;
  }

  .card-menu {
    background: var(--bg-elevated);
    border: var(--border-width) solid var(--card-border);
    border-radius: 50%;
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    color: var(--text-secondary);
    padding: 0;

    &:hover {
      color: var(--text-primary);
    }
  }
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
    color: var(--empty-icon-color);
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
    display: flex;
    align-items: center;
    gap: 6px;

    &:hover {
      opacity: 0.9;
      box-shadow: var(--btn-hover-shadow);
      transform: var(--btn-hover-transform);
    }
  }
}
</style>
