<template>
  <div class="player">
    <div class="player-progress" @click="seekByClick">
      <div class="progress-bar" :style="{ width: playerStore.progress + '%' }"></div>
    </div>

    <div class="player-content">
      <!-- Song info -->
      <div class="song-info" v-if="playerStore.currentSong">
        <el-avatar
          shape="square"
          :size="56"
          :src="playerStore.currentSong.pic || playerStore.currentSong.pic_url"
          class="song-cover"
          @click="showLyrics = true"
          style="cursor:pointer"
        >
          <Headphones :size="22" />
        </el-avatar>
        <div class="song-details" @click="showLyrics = true" style="cursor:pointer">
          <div class="song-name">{{ playerStore.currentSong.name || playerStore.currentSong.song_name }}</div>
          <div class="song-artist">{{ playerStore.currentSong.artists || playerStore.currentSong.artist_string }}</div>
        </div>
        <button class="icon-btn hide-mobile" :class="{ 'icon-btn--active': isFavorited }" @click="toggleFavorite" title="收藏">
          <Heart :size="18" :fill="isFavorited ? '#f59e0b' : 'none'" :color="isFavorited ? '#f59e0b' : 'currentColor'" />
        </button>
      </div>

      <!-- Controls -->
      <div class="player-controls">
        <button class="icon-btn" @click="playerStore.prev" title="上一首">
          <SkipBack :size="20" />
        </button>
        <button class="icon-btn icon-btn--play" @click="playerStore.toggle" :title="playerStore.isPlaying ? '暂停' : '播放'">
          <component :is="playerStore.isPlaying ? Pause : Play" :size="26" />
        </button>
        <button class="icon-btn" @click="playerStore.next" title="下一首">
          <SkipForward :size="20" />
        </button>
        <button
          class="icon-btn"
          :class="{ 'icon-btn--active': playerStore.repeatMode !== 'none' }"
          @click="playerStore.toggleRepeat"
          :title="repeatTitle"
        >
          <component :is="repeatIcon" :size="18" />
        </button>
      </div>

      <!-- Time + quality + volume -->
      <div class="player-extra">
        <span class="time">
          {{ formatTime(playerStore.currentTime) }} / {{ formatTime(playerStore.duration) }}
        </span>
        <el-popover placement="top" :width="280" trigger="click" popper-class="quality-popover-themed">
          <template #reference>
            <button class="icon-btn quality-btn" title="音质设置">
              <Settings2 :size="16" />
              <span class="quality-badge" v-if="qualityBadgeText">{{ qualityBadgeText }}</span>
            </button>
          </template>
          <div class="quality-settings">
            <div class="quality-section">
              <div class="quality-section-title">播放音质</div>
              <el-select v-model="localStreamingQuality" size="small" style="width: 100%">
                <el-option v-for="q in qualityOptions" :key="q.value" :label="q.label" :value="q.value" />
              </el-select>
            </div>
            <div class="quality-section">
              <div class="quality-section-title">下载音质</div>
              <el-select v-model="localDownloadQuality" size="small" style="width: 100%">
                <el-option v-for="q in qualityOptions" :key="q.value" :label="q.label" :value="q.value" />
              </el-select>
            </div>
          </div>
        </el-popover>
        <div class="volume-control">
          <button class="icon-btn" @click="toggleMute" :title="volumeValue === 0 ? '取消静音' : '静音'">
            <component :is="volumeIcon" :size="18" />
          </button>
          <el-slider
            v-model="volumeValue"
            :show-tooltip="false"
            size="small"
            style="width: 80px"
          />
        </div>
      </div>
    </div>

    <audio
      ref="audioRef"
      :src="audioUrl"
      @timeupdate="onTimeUpdate"
      @durationchange="onDurationChange"
      @ended="playerStore.onEnded"
      @error="onError"
      @canplay="onCanPlay"
    />

    <LyricsPage :visible="showLyrics" @close="showLyrics = false" />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import {
  Play, Pause,
  SkipBack, SkipForward,
  Repeat, Repeat1,
  Volume2, Volume1, VolumeX,
  Heart,
  Headphones,
  Settings2
} from 'lucide-vue-next'
import LyricsPage from '@/components/LyricsPage.vue'
import { ElMessage } from 'element-plus'
import { usePlayerStore } from '@/stores/player'
import { useAuthStore } from '@/stores/auth'
import { useFavoriteStore } from '@/stores/favorite'
import { useSettingsStore } from '@/stores/settings'
import api from '@/api'

const playerStore = usePlayerStore()
const authStore = useAuthStore()
const favoriteStore = useFavoriteStore()
const settingsStore = useSettingsStore()

const showLyrics = ref(false)
const audioRef = ref(null)
const audioUrl = ref('')
const volumeValue = ref(playerStore.volume * 100)
const prevVolume = ref(80)
const playbackQualityName = ref('')
const isFetchingUrl = ref(false)
let fetchAbortController = null

const qualityOptions = [
  { value: 'jymaster', label: '超清母带' },
  { value: 'jyeffect', label: '高清环绕声' },
  { value: 'sky', label: '沉浸环绕声' },
  { value: 'hires', label: 'Hi-Res' },
  { value: 'lossless', label: '无损' },
  { value: 'exhigh', label: '极高' },
  { value: 'standard', label: '标准' },
]

const qualityLabelMap = {
  standard: '标准', exhigh: '极高', lossless: '无损',
  hires: 'Hi-Res', sky: '环绕声', jyeffect: '高清环绕', jymaster: '超清母带',
}

const localStreamingQuality = computed({
  get: () => playerStore.streamingQuality,
  set: (v) => {
    playerStore.setStreamingQuality(v)
    if (authStore.isLoggedIn) settingsStore.update({ streaming_quality: v })
  }
})

const localDownloadQuality = computed({
  get: () => playerStore.downloadQuality,
  set: (v) => {
    playerStore.setDownloadQuality(v)
    if (authStore.isLoggedIn) settingsStore.update({ download_quality: v })
  }
})

const qualityBadgeText = computed(() => {
  return playbackQualityName.value || qualityLabelMap[playerStore.streamingQuality] || ''
})

async function fetchSongUrl(song) {
  if (fetchAbortController) {
    fetchAbortController.abort()
    fetchAbortController = null
  }
  if (!song) {
    audioUrl.value = ''
    playbackQualityName.value = ''
    isFetchingUrl.value = false
    return
  }
  isFetchingUrl.value = true
  const controller = new AbortController()
  fetchAbortController = controller
  try {
    const res = await api.get('/song', {
      params: { id: song.id, level: playerStore.streamingQuality, type: 'url' },
      signal: controller.signal
    })
    if (controller.signal.aborted) return
    if (res.data.success && res.data.data?.url) {
      audioUrl.value = res.data.data.url
      playbackQualityName.value = res.data.data.quality_name || qualityLabelMap[res.data.data.level] || ''
    } else {
      audioUrl.value = ''
      playbackQualityName.value = ''
      ElMessage.error('无法获取播放链接，可能是版权限制')
      playerStore.pause()
    }
  } catch {
    if (controller.signal.aborted) return
    audioUrl.value = ''
    ElMessage.error('获取播放链接失败')
    playerStore.pause()
  } finally {
    if (fetchAbortController === controller) {
      fetchAbortController = null
      isFetchingUrl.value = false
    }
  }
}

watch(() => playerStore.currentSong, (song) => fetchSongUrl(song), { immediate: true })

watch(() => playerStore.streamingQuality, () => {
  if (playerStore.currentSong) {
    fetchSongUrl(playerStore.currentSong)
  }
})

watch(() => playerStore.isPlaying, (playing) => {
  if (!audioRef.value) return
  if (playing) {
    if (isFetchingUrl.value) return
    if (audioUrl.value && audioRef.value.readyState >= 3) {
      audioRef.value.play().catch(() => playerStore.pause())
    }
  } else {
    audioRef.value.pause()
  }
})

watch(audioUrl, async (url) => {
  if (!url || !audioRef.value) return
  await nextTick()
  audioRef.value.load()
})

watch(volumeValue, (val) => {
  playerStore.setVolume(val / 100)
})

const isFavorited = computed(() => {
  if (!playerStore.currentSong) return false
  return favoriteStore.isFavorited(playerStore.currentSong.id)
})

const repeatIcon = computed(() => {
  return playerStore.repeatMode === 'one' ? Repeat1 : Repeat
})
const repeatTitle = computed(() => {
  const map = { none: '不循环', one: '单曲循环', all: '列表循环' }
  return map[playerStore.repeatMode] || '循环'
})

const volumeIcon = computed(() => {
  if (volumeValue.value === 0) return VolumeX
  if (volumeValue.value < 50) return Volume1
  return Volume2
})

function onTimeUpdate() {
  if (audioRef.value) playerStore.updateTime(audioRef.value.currentTime)
}
function onDurationChange() {
  if (audioRef.value) playerStore.updateDuration(audioRef.value.duration)
}
function onCanPlay() {
  onCanPlaySuccess()
  if (isFetchingUrl.value) return
  if (playerStore.isPlaying && audioRef.value) {
    audioRef.value.play().catch(() => playerStore.pause())
  }
}
let errorCount = 0
function onError() {
  if (isFetchingUrl.value || !audioUrl.value) return
  errorCount++
  if (errorCount > 3) {
    ElMessage.error('连续播放失败，已暂停')
    playerStore.pause()
    errorCount = 0
    return
  }
  ElMessage.error('播放失败，自动跳过')
  playerStore.next()
}
function onCanPlaySuccess() {
  errorCount = 0
}
function seekByClick(e) {
  const rect = e.currentTarget.getBoundingClientRect()
  const percent = (e.clientX - rect.left) / rect.width
  playerStore.seek(percent * playerStore.duration)
}
function toggleMute() {
  if (volumeValue.value > 0) {
    prevVolume.value = volumeValue.value
    volumeValue.value = 0
  } else {
    volumeValue.value = prevVolume.value || 80
  }
}
async function toggleFavorite() {
  if (!authStore.isLoggedIn) {
    ElMessage.warning('请先登录')
    return
  }
  const song = playerStore.currentSong
  try {
    if (isFavorited.value) {
      await favoriteStore.remove(song.id)
      ElMessage.success('已取消收藏')
    } else {
      await favoriteStore.add({
        song_id: song.id,
        song_name: song.name || song.song_name,
        artists: song.artists || song.artist_string,
        album: song.album || song.al_name,
        pic_url: song.pic || song.pic_url
      })
      ElMessage.success('已添加收藏')
    }
  } catch {
    ElMessage.error('操作失败')
  }
}
function formatTime(seconds) {
  if (!seconds || isNaN(seconds)) return '0:00'
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${mins}:${secs.toString().padStart(2, '0')}`
}

onMounted(() => {
  if (audioRef.value) playerStore.setAudioElement(audioRef.value)
})
</script>

<style lang="scss" scoped>
.player {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: 80px;
  background: var(--bg-player);
  backdrop-filter: blur(16px);
  border-top: var(--border-width) solid var(--border-color);
  z-index: 1000;
  transition: background 0.4s, border-color 0.4s, box-shadow 0.4s;

  @media (max-width: 640px) {
    height: auto;
    padding-bottom: env(safe-area-inset-bottom, 0px);
  }

  .player-progress {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 4px;
    background: var(--progress-bg);
    cursor: pointer;
    transition: height 0.15s;

    .progress-bar {
      height: 100%;
      background: var(--progress-fill);
      transition: width 0.1s linear;
    }

    &:hover {
      height: 6px;
    }
  }

  .player-content {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    align-items: center;
    height: 100%;
    padding: 0 24px;

    @media (max-width: 640px) {
      display: flex;
      flex-wrap: wrap;
      padding: 8px 12px 6px;
      gap: 4px;
    }
  }

  .song-info {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;

    @media (max-width: 640px) {
      gap: 10px;
      order: 1;
      flex: 1 1 60%;
    }

    .song-cover {
      flex-shrink: 0;
      background: var(--avatar-bg);
      color: var(--text-muted);
      border: var(--border-width) solid var(--card-border);
      border-radius: var(--radius-sm);

      @media (max-width: 640px) {
        :deep(.el-avatar) {
          width: 44px !important;
          height: 44px !important;
        }
      }
    }

    .song-details {
      min-width: 0;
      flex: 1;

      .song-name {
        color: var(--text-primary);
        font-size: 14px;
        font-weight: 500;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 180px;

        @media (max-width: 640px) {
          max-width: none;
          font-size: 13px;
        }
      }

      .song-artist {
        color: var(--text-muted);
        font-size: 12px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: 180px;
        margin-top: 2px;

        @media (max-width: 640px) {
          max-width: none;
          font-size: 11px;
        }
      }
    }
  }

  .player-controls {
    display: flex;
    align-items: center;
    gap: 8px;
    justify-content: center;

    @media (max-width: 640px) {
      order: 3;
      flex: 1 0 100%;
      gap: 12px;
      padding: 0;
    }
  }

  .player-extra {
    display: flex;
    align-items: center;
    gap: 16px;
    justify-content: flex-end;
    min-width: 0;

    @media (max-width: 640px) {
      order: 2;
      flex: 0 0 auto;
      gap: 6px;
    }

    .time {
      color: var(--text-muted);
      font-size: 12px;
      font-variant-numeric: tabular-nums;
      white-space: nowrap;

      @media (max-width: 640px) {
        display: none;
      }
    }

    .volume-control {
      display: flex;
      align-items: center;
      gap: 8px;

      @media (max-width: 640px) {
        display: none;
      }
    }

    .quality-btn {
      width: auto;
      padding: 0 10px;
      border-radius: var(--radius);
      gap: 4px;
      font-size: 12px;

      .quality-badge {
        color: var(--accent);
        font-weight: 500;
        white-space: nowrap;
      }
    }
  }
}

[data-theme="night"] .player {
  box-shadow: 0 -4px 24px rgba(0, 0, 0, 0.35);
}

[data-theme="day"] .player {
  backdrop-filter: none;
  box-shadow: 0 -2px 16px rgba(15, 23, 42, 0.06);
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: none;
  background: var(--btn-bg);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;
  flex-shrink: 0;

  &:hover {
    background: var(--btn-hover-bg);
    color: var(--text-primary);
  }

  &:active {
    transform: scale(0.92);
  }

  &--play {
    width: 44px;
    height: 44px;
    background: var(--accent-btn-bg);
    color: var(--accent-btn-text);
    border: var(--border-width) solid var(--btn-border);
    box-shadow: var(--btn-shadow);

    &:hover {
      opacity: 0.9;
      box-shadow: var(--btn-hover-shadow);
      transform: var(--btn-hover-transform);
    }
  }

  &--active {
    color: var(--accent);
    background: rgba(var(--accent-rgb), 0.15);
  }
}

[data-theme="night"] .icon-btn--active {
  background: rgba(var(--accent-rgb), 0.15);

  &:hover {
    background: rgba(var(--accent-rgb), 0.25);
  }
}

[data-theme="day"] .icon-btn {
  border: 1px solid var(--border-color);
  box-shadow: var(--shadow-sm);

  &:hover {
    box-shadow: var(--shadow-hover);
    transform: translateY(-1px);
  }

  &--play {
    border: 1px solid var(--border-color);
    box-shadow: var(--shadow);

    &:hover {
      box-shadow: var(--shadow-hover);
      transform: translateY(-1px);
    }
  }
}

[data-theme="day"] .icon-btn--active {
  background: rgba(var(--accent-rgb), 0.12);
  color: var(--accent);

  &:hover {
    background: rgba(var(--accent-rgb), 0.18);
  }
}

:deep(.el-slider__runway) {
  background: var(--progress-bg);
}
:deep(.el-slider__bar) {
  background: var(--accent);
}
:deep(.el-slider__button) {
  border-color: var(--accent);
  width: 12px;
  height: 12px;
}

.hide-mobile {
  @media (max-width: 640px) {
    display: none !important;
  }
}

@media (max-width: 640px) {
  .icon-btn {
    width: 40px;
    height: 40px;
  }
  .icon-btn--play {
    width: 46px;
    height: 46px;
  }
}
</style>

<style lang="scss">
.quality-popover-themed {
  background: var(--dialog-bg) !important;
  border: var(--border-width) solid var(--border-color) !important;
  box-shadow: var(--shadow) !important;

  .el-popper__arrow::before {
    background: var(--dialog-bg) !important;
    border-color: var(--border-color) !important;
  }

  .quality-settings {
    display: flex;
    flex-direction: column;
    gap: 16px;

    .quality-section-title {
      color: var(--text-secondary);
      font-size: 13px;
      font-weight: 500;
      margin-bottom: 8px;
    }

    .el-select {
      .el-input__wrapper {
        background: var(--bg-input);
        border: var(--border-width) solid var(--border-color);
        box-shadow: none;

        .el-input__inner {
          color: var(--text-primary);
        }

        .el-input__suffix {
          color: var(--text-muted);
        }
      }
    }
  }
}
</style>
