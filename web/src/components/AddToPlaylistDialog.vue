<template>
  <el-dialog v-model="visible" title="添加到歌单" width="400px" :close-on-click-modal="false" @close="onClose">
    <div class="quick-create" v-if="!showNewForm">
      <el-button class="new-playlist-btn" text @click="showNewForm = true">
        <Plus :size="16" />
        <span>新建歌单</span>
      </el-button>
    </div>

    <div class="new-playlist-form" v-if="showNewForm">
      <el-input
        v-model="newName"
        placeholder="输入新歌单名称"
        maxlength="200"
        size="small"
        @keyup.enter="createAndAdd"
      >
        <template #append>
          <el-button :loading="creating" @click="createAndAdd">创建并添加</el-button>
        </template>
      </el-input>
    </div>

    <div class="playlist-list" v-loading="loadingList">
      <div
        v-for="pl in playlists"
        :key="pl.id"
        class="playlist-option"
        @click="addToPlaylist(pl)"
      >
        <el-avatar shape="square" :size="40" :src="pl.cover_url || undefined">
          <ListMusic :size="18" />
        </el-avatar>
        <div class="option-info">
          <div class="option-name">{{ pl.name }}</div>
          <div class="option-count">{{ pl.song_count }} 首</div>
        </div>
        <el-icon v-if="addedSet.has(pl.id)" class="added-icon"><Check /></el-icon>
      </div>

      <div class="empty-tip" v-if="!loadingList && playlists.length === 0">
        暂无歌单，请先创建
      </div>
    </div>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { Plus, ListMusic } from 'lucide-vue-next'
import { Check } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { usePlaylistStore } from '@/stores/playlist'

const props = defineProps({
  modelValue: Boolean,
  song: Object
})

const emit = defineEmits(['update:modelValue'])

const playlistStore = usePlaylistStore()
const playlists = ref([])
const loadingList = ref(false)
const showNewForm = ref(false)
const newName = ref('')
const creating = ref(false)
const addedSet = ref(new Set())

const visible = ref(false)

watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val) {
    addedSet.value = new Set()
    showNewForm.value = false
    newName.value = ''
    fetchPlaylists()
  }
})

watch(visible, (val) => {
  emit('update:modelValue', val)
})

async function fetchPlaylists() {
  loadingList.value = true
  try {
    await playlistStore.fetchMyPlaylists()
    playlists.value = playlistStore.myPlaylists
  } catch (e) {
    ElMessage.error('获取歌单失败')
  } finally {
    loadingList.value = false
  }
}

async function addToPlaylist(pl) {
  if (!props.song) return
  if (addedSet.value.has(pl.id)) return
  try {
    await playlistStore.addSongToPlaylist(pl.id, {
      song_id: props.song.id || props.song.song_id,
      song_name: props.song.name || props.song.song_name,
      artists: props.song.artists || props.song.artist_string,
      album: props.song.album || props.song.al?.name,
      pic_url: props.song.picUrl || props.song.pic_url || props.song.pic || props.song.al?.picUrl
    })
    addedSet.value.add(pl.id)
    ElMessage.success(`已添加到「${pl.name}」`)
  } catch (e) {
    ElMessage.error('添加失败')
  }
}

async function createAndAdd() {
  if (!newName.value.trim()) {
    ElMessage.warning('请输入歌单名称')
    return
  }
  creating.value = true
  try {
    const res = await playlistStore.createPlaylist(newName.value.trim())
    if (res.success) {
      playlists.value = playlistStore.myPlaylists
      await addToPlaylist(res.data)
      showNewForm.value = false
      newName.value = ''
    }
  } catch (e) {
    ElMessage.error('创建失败')
  } finally {
    creating.value = false
  }
}

function onClose() {
  emit('update:modelValue', false)
}
</script>

<style lang="scss" scoped>
.quick-create {
  margin-bottom: 12px;

  .new-playlist-btn {
    color: var(--accent);
    font-size: 14px;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 0;

    &:hover { opacity: 0.8; }
  }
}

.new-playlist-form {
  margin-bottom: 16px;
}

.playlist-list {
  max-height: 320px;
  overflow-y: auto;
  min-height: 100px;
}

.playlist-option {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 0.2s;

  &:hover {
    background: var(--bg-elevated-hover);
  }

  :deep(.el-avatar) {
    border: var(--border-width) solid var(--card-border);
    border-radius: var(--radius-sm);
    flex-shrink: 0;
  }

  .option-info {
    flex: 1;
    min-width: 0;

    .option-name {
      font-size: 14px;
      color: var(--text-primary);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .option-count {
      font-size: 12px;
      color: var(--text-muted);
      margin-top: 2px;
    }
  }

  .added-icon {
    color: var(--el-color-success);
    font-size: 18px;
    flex-shrink: 0;
  }
}

.empty-tip {
  text-align: center;
  color: var(--text-faint);
  font-size: 14px;
  padding: 32px 0;
}
</style>
