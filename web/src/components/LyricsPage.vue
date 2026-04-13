<template>
  <Teleport to="body">
    <Transition name="lyrics-slide">
      <div v-if="visible" class="lyrics-page" @click.self="$emit('close')">
        <div class="lyrics-page__inner">
        <!-- Top bar -->
        <div class="lyrics-top">
          <button class="lyrics-close" @click="$emit('close')">
            <ChevronDown :size="28" />
          </button>
          <div class="lyrics-top__title">
            <div class="lyrics-top__name">{{ songName }}</div>
            <div class="lyrics-top__artist">{{ songArtist }}</div>
          </div>
          <div class="lyrics-top__spacer" />
        </div>

        <div class="lyrics-body">
          <!-- Mobile song cover (replaces cassette on small screens) -->
          <div class="lyrics-mobile-cover" :class="{ 'cassette--playing': playerStore.isPlaying }">
            <img v-if="songCover" :src="songCover" alt="Cover" class="lyrics-mobile-cover__img" />
          </div>

          <!-- Mobile song title -->
          <div class="lyrics-mobile-header">
            <div class="lyrics-mobile-header__name">{{ songName }}</div>
            <div class="lyrics-mobile-header__artist">{{ songArtist }}</div>
          </div>

          <!-- Desktop song info (hidden on mobile) -->
          <div class="lyrics-song-info">
            <div class="lyrics-song-info__name">{{ songName }}</div>
            <div class="lyrics-song-info__meta">
              <span v-if="songAlbum" class="lyrics-song-info__album">{{ songAlbum }}</span>
              <span class="lyrics-song-info__artist">{{ songArtist }}</span>
            </div>
          </div>

          <!-- Cassette visual -->
          <div class="lyrics-visual">
            <div class="cassette" :class="{ 'cassette--playing': playerStore.isPlaying }">
              <div class="cassette__body">
                <div class="cassette__label">
                  <div class="cassette__label-text">{{ songName }}</div>
                  <div class="cassette__label-sub">{{ songArtist }}</div>
                </div>
                <div class="cassette__window">
                  <div class="cassette__reel cassette__reel--left" :style="leftReelStyle">
                    <div class="cassette__reel-hub">
                      <div class="cassette__spoke" />
                      <div class="cassette__spoke" />
                      <div class="cassette__spoke" />
                    </div>
                    <div class="cassette__tape-wrap" :style="{ borderWidth: leftTapeWidth + 'px' }" />
                  </div>
                  <div class="cassette__tape-path">
                    <div class="cassette__tape-line" />
                  </div>
                  <div class="cassette__reel cassette__reel--right" :style="rightReelStyle">
                    <div class="cassette__reel-hub">
                      <div class="cassette__spoke" />
                      <div class="cassette__spoke" />
                      <div class="cassette__spoke" />
                    </div>
                    <div class="cassette__tape-wrap" :style="{ borderWidth: rightTapeWidth + 'px' }" />
                  </div>
                </div>
                <div class="cassette__screws">
                  <div class="cassette__screw" />
                  <div class="cassette__screw" />
                  <div class="cassette__screw" />
                  <div class="cassette__screw" />
                  <div class="cassette__screw" />
                </div>
              </div>
            </div>
          </div>

          <!-- Lyrics scroll -->
          <div class="lyrics-scroll-area" ref="lyricsContainer">
            <div class="lyrics-scroll" ref="lyricsScroll">
              <div class="lyrics-spacer" />
              <template v-if="parsedLyrics.length">
                <div
                  v-for="(line, idx) in parsedLyrics"
                  :key="idx"
                  :ref="el => setLineRef(el, idx)"
                  class="lyrics-line"
                  :class="{
                    'lyrics-line--active': idx === currentLineIndex,
                    'lyrics-line--near': Math.abs(idx - currentLineIndex) === 1,
                  }"
                  @click="seekToLine(line.time)"
                >
                  <div class="lyrics-line__text">{{ line.text }}</div>
                  <div v-if="line.tText" class="lyrics-line__trans">{{ line.tText }}</div>
                </div>
              </template>
              <div v-else class="lyrics-empty">
                <Music :size="48" />
                <span>暂无歌词</span>
              </div>
              <div class="lyrics-spacer" />
            </div>
          </div>

          <!-- Playback controls -->
          <div class="lyrics-playback">
            <div class="lyrics-progress">
              <span class="lyrics-progress__time">{{ formatTime(playerStore.currentTime) }}</span>
              <div class="lyrics-progress__bar" @click="seekByClick">
                <div class="lyrics-progress__fill" :style="{ width: playerStore.progress + '%' }" />
                <div class="lyrics-progress__dot" :style="{ left: playerStore.progress + '%' }" />
              </div>
              <span class="lyrics-progress__time">{{ formatTime(playerStore.duration) }}</span>
            </div>
            <div class="lyrics-controls">
              <button class="lc-btn lc-btn--side" :class="{ 'lc-btn--active': isFavorited }" @click="toggleFavorite">
                <Heart :size="24" :fill="isFavorited ? '#f59e0b' : 'none'" :color="isFavorited ? '#f59e0b' : 'currentColor'" />
              </button>
              <div class="lc-center">
                <button class="lc-btn" @click="playerStore.prev"><SkipBack :size="20" /></button>
                <button class="lc-btn lc-btn--play" @click="playerStore.toggle">
                  <component :is="playerStore.isPlaying ? Pause : Play" :size="28" />
                </button>
                <button class="lc-btn" @click="playerStore.next"><SkipForward :size="20" /></button>
              </div>
              <div class="lc-volume">
                <component :is="volumeIcon" :size="20" class="lc-volume__icon" />
                <input
                  v-model.number="volumeValue"
                  type="range"
                  min="0"
                  max="100"
                  class="lc-volume__slider"
                  @input="onVolumeInput"
                />
              </div>
            </div>
          </div>
        </div>
      </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick, onBeforeUnmount } from 'vue'
import { ChevronDown, SkipBack, SkipForward, Play, Pause, Music, Heart, Volume2, Volume1, VolumeX } from 'lucide-vue-next'
import { usePlayerStore } from '@/stores/player'
import { useFavoriteStore } from '@/stores/favorite'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'
import api from '@/api'

const props = defineProps({ visible: Boolean })
defineEmits(['close'])

const playerStore = usePlayerStore()
const favoriteStore = useFavoriteStore()
const authStore = useAuthStore()

const lyricsContainer = ref(null)
const lyricsScroll = ref(null)
const lineRefs = ref({})
const rawLyrics = ref('')
const rawTLyrics = ref('')
const currentLineIndex = ref(-1)

const volumeValue = ref(Math.round(playerStore.volume * 100))

watch(() => playerStore.volume, (v) => {
  volumeValue.value = Math.round(v * 100)
})

const volumeIcon = computed(() => {
  if (volumeValue.value === 0) return VolumeX
  if (volumeValue.value < 50) return Volume1
  return Volume2
})

function onVolumeInput() {
  const v = Math.max(0, Math.min(100, volumeValue.value))
  volumeValue.value = v
  playerStore.setVolume(v / 100)
}

const isFavorited = computed(() => {
  if (!playerStore.currentSong) return false
  return favoriteStore.isFavorited(playerStore.currentSong.id)
})

async function toggleFavorite() {
  if (!authStore.isLoggedIn) {
    ElMessage.warning('请先登录')
    return
  }
  const song = playerStore.currentSong
  if (!song) return
  try {
    if (isFavorited.value) {
      await favoriteStore.remove(song.id)
      ElMessage.success('已取消收藏')
    } else {
      await favoriteStore.add(song)
      ElMessage.success('已添加到我喜欢的音乐')
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.message || '操作失败')
  }
}

const songCover = computed(() =>
  playerStore.currentSong?.pic_url || playerStore.currentSong?.al?.picUrl || ''
)

const songName = computed(() =>
  playerStore.currentSong?.name || playerStore.currentSong?.song_name || ''
)
const songArtist = computed(() =>
  playerStore.currentSong?.artists || playerStore.currentSong?.artist_string || ''
)
const songAlbum = computed(() =>
  playerStore.currentSong?.album || playerStore.currentSong?.album_name || ''
)

const progressRatio = computed(() => {
  if (!playerStore.duration) return 0
  return playerStore.currentTime / playerStore.duration
})

const MAX_TAPE = 14
const MIN_TAPE = 3
const leftTapeWidth = computed(() => MAX_TAPE - (MAX_TAPE - MIN_TAPE) * progressRatio.value)
const rightTapeWidth = computed(() => MIN_TAPE + (MAX_TAPE - MIN_TAPE) * progressRatio.value)

const reelRotation = computed(() => playerStore.currentTime * 120)
const leftReelStyle = computed(() => ({
  '--reel-rotation': `${reelRotation.value}deg`
}))
const rightReelStyle = computed(() => ({
  '--reel-rotation': `${reelRotation.value}deg`
}))

function parseLRC(lrc) {
  if (!lrc) return []
  const lines = []
  const regex = /\[(\d{2}):(\d{2})(?:\.(\d{2,3}))?\](.*)/g
  let match
  while ((match = regex.exec(lrc)) !== null) {
    const min = parseInt(match[1])
    const sec = parseInt(match[2])
    const ms = match[3] ? parseInt(match[3].padEnd(3, '0')) : 0
    const time = min * 60 + sec + ms / 1000
    const text = match[4].trim()
    if (text) lines.push({ time, text })
  }
  return lines.sort((a, b) => a.time - b.time)
}

const parsedLyrics = computed(() => {
  const main = parseLRC(rawLyrics.value)
  const trans = parseLRC(rawTLyrics.value)
  if (!trans.length) return main

  const transMap = new Map()
  trans.forEach(t => {
    const key = Math.round(t.time * 10)
    transMap.set(key, t.text)
  })
  return main.map(line => ({
    ...line,
    tText: transMap.get(Math.round(line.time * 10)) || ''
  }))
})

function setLineRef(el, idx) {
  if (el) lineRefs.value[idx] = el
}

watch(() => playerStore.currentTime, (time) => {
  if (!parsedLyrics.value.length) return
  let idx = -1
  for (let i = parsedLyrics.value.length - 1; i >= 0; i--) {
    if (time >= parsedLyrics.value[i].time) {
      idx = i
      break
    }
  }
  if (idx !== currentLineIndex.value) {
    currentLineIndex.value = idx
    scrollToLine(idx)
  }
})

function scrollToLine(idx) {
  const el = lineRefs.value[idx]
  const container = lyricsContainer.value
  if (!el || !container) return
  nextTick(() => {
    const containerRect = container.getBoundingClientRect()
    const lineRect = el.getBoundingClientRect()
    const offset = lineRect.top - containerRect.top - containerRect.height / 2 + lineRect.height / 2
    container.scrollBy({ top: offset, behavior: 'smooth' })
  })
}

async function fetchLyrics(songId) {
  rawLyrics.value = ''
  rawTLyrics.value = ''
  currentLineIndex.value = -1
  if (!songId) return
  try {
    const res = await api.get('/song', { params: { id: songId, type: 'lyric' } })
    if (res.data.success && res.data.data) {
      rawLyrics.value = res.data.data?.lrc?.lyric || ''
      rawTLyrics.value = res.data.data?.tlyric?.lyric || ''
    }
  } catch { /* silently fail */ }
}

watch(() => playerStore.currentSong, (song) => {
  if (song) fetchLyrics(song.id)
}, { immediate: true })

watch(() => props.visible, (v) => {
  if (v) {
    nextTick(() => scrollToLine(currentLineIndex.value))
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = ''
  }
})

onBeforeUnmount(() => {
  document.body.style.overflow = ''
})

function seekToLine(time) {
  playerStore.seek(time)
}

function seekByClick(e) {
  const rect = e.currentTarget.getBoundingClientRect()
  const percent = (e.clientX - rect.left) / rect.width
  playerStore.seek(percent * playerStore.duration)
}

function formatTime(seconds) {
  if (!seconds || isNaN(seconds)) return '0:00'
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${mins}:${secs.toString().padStart(2, '0')}`
}
</script>

<style lang="scss" scoped>
.lyrics-page {
  position: fixed;
  inset: 0;
  z-index: 2000;
  background: var(--bg-deep);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.lyrics-page__inner {
  display: flex;
  flex-direction: column;
  height: 100%;
  max-width: 1200px;
  width: 100%;
  margin: 0 auto;
  padding: 0 32px;
}

/* ── Top bar ────────────────────── */
.lyrics-top {
  display: flex;
  align-items: center;
  padding: 20px 0 12px;
  gap: 12px;
  flex-shrink: 0;

  &__title {
    flex: 1;
    text-align: center;
    min-width: 0;
  }

  &__name {
    font-size: 16px;
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  &__artist {
    font-size: 13px;
    color: var(--text-muted);
    margin-top: 2px;
  }
}

.lyrics-close {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: none;
  background: var(--btn-bg);
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  flex-shrink: 0;

  &:hover {
    background: var(--btn-hover-bg);
    color: var(--text-primary);
  }
}

/* ── Mobile-only elements ─────── */
.lyrics-mobile-header,
.lyrics-mobile-cover {
  display: none !important;
}

/* ── Desktop song info (right-column header) ── */
.lyrics-song-info {
  grid-column: 2;
  grid-row: 1;
  padding-bottom: 28px;
  min-width: 0;

  &__name {
    font-size: 30px;
    font-weight: 700;
    color: var(--text-primary);
    line-height: 1.3;
    overflow: hidden;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    margin-bottom: 10px;
  }

  &__meta {
    display: flex;
    flex-wrap: wrap;
    gap: 6px 16px;
    align-items: center;
  }

  &__album,
  &__artist {
    font-size: 14px;
    color: var(--text-muted);
  }

  &__album::before {
    content: '专辑：';
    opacity: 0.6;
  }

  &__artist::before {
    content: '歌手：';
    opacity: 0.6;
  }
}

/* Hide top-bar title on all devices (mobile has cover+title below, desktop has right-column info) */
.lyrics-top__title,
.lyrics-top__spacer {
  display: none !important;
}

.lyrics-top {
  padding: 16px 0 8px;
}

/* ── Body layout (CSS Grid) ─────── */
.lyrics-body {
  display: grid;
  grid-template-columns: 44% 1fr;
  grid-template-rows: auto 1fr auto;
  gap: 0 64px;
  flex: 1;
  min-height: 0;
  padding-bottom: 36px;
}

.lyrics-visual {
  grid-column: 1;
  grid-row: 1 / 3; /* Only span rows 1 to 2, leaving row 3 for playback */
  display: flex;
  align-items: center;
  justify-content: center;
}

.lyrics-scroll-area {
  grid-column: 2;
  grid-row: 2;
  overflow-y: auto;
  min-height: 0;
  mask-image: linear-gradient(
    to bottom,
    transparent 0%,
    #000 10%,
    #000 90%,
    transparent 100%
  );
  -webkit-mask-image: linear-gradient(
    to bottom,
    transparent 0%,
    #000 10%,
    #000 90%,
    transparent 100%
  );
  scroll-behavior: smooth;

  &::-webkit-scrollbar {
    width: 0;
  }
}

.lyrics-playback {
  grid-column: 1 / -1; /* Span across both columns on desktop */
  grid-row: 3;
  display: flex;
  flex-direction: column;
  align-items: center; 
  gap: 16px;
  padding-top: 24px;
}

/* ── Cassette Tape ──────────────── */
.cassette__body {
  width: 420px;
  height: 262px;
  border-radius: 20px;
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 18px 26px;
}

[data-theme="night"] .cassette__body {
  background: linear-gradient(145deg, #12151c, #0c0f14);
  border: 1px solid rgba(94, 234, 212, 0.22);
  box-shadow:
    0 0 40px rgba(45, 212, 191, 0.08),
    inset 0 1px 0 rgba(255, 255, 255, 0.05);
}

[data-theme="day"] .cassette__body {
  background: linear-gradient(145deg, #fffefb, #f8fafc);
  border: 1px solid var(--card-border);
  box-shadow: var(--shadow-lg);
}

.cassette__label {
  text-align: center;
  padding: 6px 16px 10px;
  border-radius: 8px;
  width: 80%;
  margin-bottom: 8px;
}

[data-theme="night"] .cassette__label {
  background: linear-gradient(135deg, rgba(45, 212, 191, 0.1), rgba(129, 140, 248, 0.08));
  border: 1px solid rgba(94, 234, 212, 0.18);
}

[data-theme="day"] .cassette__label {
  background: var(--bg-elevated);
  border: 1px solid var(--border-color);
}

.cassette__label-text {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  letter-spacing: 0.03em;
}

.cassette__label-sub {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 1px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Reels window ───────────────── */
.cassette__window {
  width: 72%;
  height: 112px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 20px;
  position: relative;
}

[data-theme="night"] .cassette__window {
  background: rgba(0, 0, 0, 0.45);
  border: 1px solid rgba(94, 234, 212, 0.12);
}

[data-theme="day"] .cassette__window {
  background: var(--avatar-bg);
  border: 1px solid var(--border-color);
}

.cassette__reel {
  width: 76px;
  height: 76px;
  border-radius: 50%;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

[data-theme="night"] .cassette__reel {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(129, 140, 248, 0.22);
}

[data-theme="day"] .cassette__reel {
  background: #fff;
  border: 1px solid var(--border-color);
}

.cassette__reel-hub {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  position: absolute;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2;
}

.cassette--playing .cassette__reel-hub {
  animation: reel-spin 2s linear infinite;
}

[data-theme="night"] .cassette__reel-hub {
  background: #12151c;
  border: 2px solid rgba(94, 234, 212, 0.45);
}

[data-theme="day"] .cassette__reel-hub {
  background: #fff;
  border: 2px solid var(--accent);
}

.cassette__spoke {
  position: absolute;
  width: 2px;
  height: 22px;
  border-radius: 1px;
  top: 50%;
  left: 50%;
  transform-origin: center center;

  &:nth-child(1) { transform: translate(-50%, -50%) rotate(0deg); }
  &:nth-child(2) { transform: translate(-50%, -50%) rotate(60deg); }
  &:nth-child(3) { transform: translate(-50%, -50%) rotate(120deg); }
}

[data-theme="night"] .cassette__spoke {
  background: rgba(94, 234, 212, 0.45);
}

[data-theme="day"] .cassette__spoke {
  background: var(--text-muted);
}

.cassette__tape-wrap {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  border-style: solid;
  transition: border-width 0.3s linear;
  pointer-events: none;
}

[data-theme="night"] .cassette__tape-wrap {
  border-color: rgba(139, 92, 246, 0.35);
}

[data-theme="day"] .cassette__tape-wrap {
  border-color: rgba(26, 26, 46, 0.2);
}

/* ── Tape path line ─────────────── */
.cassette__tape-path {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  height: 100%;
  margin: 0 -4px;
  z-index: 1;
}

.cassette__tape-line {
  width: 100%;
  height: 2px;
}

[data-theme="night"] .cassette__tape-line {
  background: linear-gradient(90deg, rgba(45, 212, 191, 0.4), rgba(129, 140, 248, 0.4));
}

[data-theme="day"] .cassette__tape-line {
  background: var(--border-hover);
}

/* ── Screws ─────────────────────── */
.cassette__screws {
  display: flex;
  justify-content: space-between;
  width: 90%;
  margin-top: auto;
  padding-top: 8px;
}

.cassette__screw {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

[data-theme="night"] .cassette__screw {
  background: rgba(94, 234, 212, 0.15);
  border: 1px solid rgba(94, 234, 212, 0.28);
}

[data-theme="day"] .cassette__screw {
  background: var(--avatar-bg);
  border: 1px solid var(--border-color);
}

@keyframes reel-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* ── Controls ───────────────────── */
.lyrics-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  position: relative;
  min-height: 56px;
  width: 100%;
  max-width: 600px;
}

.lc-center {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 16px;
}

.lc-volume {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  z-index: 1;

  &__icon {
    flex-shrink: 0;
    color: var(--text-secondary);
  }

  &__slider {
    width: 80px;
    height: 6px;
    -webkit-appearance: none;
    appearance: none;
    background: var(--progress-bg);
    border-radius: 3px;
    outline: none;

    &::-webkit-slider-thumb {
      -webkit-appearance: none;
      appearance: none;
      width: 14px;
      height: 14px;
      border-radius: 50%;
      cursor: pointer;
      transition: transform 0.15s;
    }

    &::-moz-range-thumb {
      width: 14px;
      height: 14px;
      border-radius: 50%;
      border: none;
      cursor: pointer;
      transition: transform 0.15s;
    }
  }
}

[data-theme="night"] .lc-volume__slider {
  &::-webkit-slider-thumb {
    background: var(--accent);
    box-shadow: 0 0 6px rgba(var(--accent-rgb), 0.45);
  }
  &::-moz-range-thumb {
    background: var(--accent);
    box-shadow: 0 0 6px rgba(var(--accent-rgb), 0.45);
  }
}

[data-theme="day"] .lc-volume__slider {
  &::-webkit-slider-thumb {
    background: var(--accent);
    border: 2px solid var(--card-border);
  }
  &::-moz-range-thumb {
    background: var(--accent);
    border: 2px solid var(--card-border);
  }
}

.lc-btn {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: none;
  background: var(--btn-bg);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    background: var(--btn-hover-bg);
    color: var(--text-primary);
  }

  &:active {
    transform: scale(0.92);
  }

  &--side {
    background: transparent;
    box-shadow: none !important;
    border: none !important;
    z-index: 1;
    
    &:hover {
      background: var(--bg-hover);
    }
  }

  &--active {
    color: #f59e0b;
  }

  &--play {
    width: 56px;
    height: 56px;
    background: var(--accent-btn-bg);
    color: var(--accent-btn-text);
    border: var(--border-width) solid var(--btn-border);
    box-shadow: var(--btn-shadow);

    &:hover {
      opacity: 0.9;
    }
  }
}

[data-theme="day"] .lc-btn {
  border: 2px solid var(--border-color);
  box-shadow: var(--shadow-sm);

  &:hover {
    box-shadow: var(--shadow-hover);
    transform: translate(1px, 1px);
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

/* ── Progress ───────────────────── */
.lyrics-progress {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  max-width: 600px;

  &__time {
    font-size: 12px;
    color: var(--text-muted);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    min-width: 36px;
    text-align: center;
  }

  &__bar {
    flex: 1;
    height: 4px;
    border-radius: 2px;
    background: var(--progress-bg);
    cursor: pointer;
    position: relative;
    transition: height 0.15s;

    &:hover {
      height: 6px;
    }
  }

  &__fill {
    height: 100%;
    border-radius: 2px;
    background: var(--progress-fill);
    transition: width 0.1s linear;
  }

  &__dot {
    position: absolute;
    top: 50%;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    transform: translate(-50%, -50%);
    opacity: 0;
    transition: opacity 0.15s;
  }

  &__bar:hover &__dot {
    opacity: 1;
  }
}

[data-theme="night"] .lyrics-progress__dot {
  background: var(--accent);
  box-shadow: 0 0 8px rgba(var(--accent-rgb), 0.55);
}

[data-theme="day"] .lyrics-progress__dot {
  background: var(--accent);
  border: 2px solid var(--card-bg);
}

/* ── Lyrics lines ───────────────── */
.lyrics-scroll {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 0 16px;
}

.lyrics-spacer {
  height: 40vh;
  flex-shrink: 0;
}

.lyrics-line {
  padding: 10px 16px;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.35s ease;

  &__text {
    font-size: 20px;
    font-weight: 500;
    color: var(--text-faint);
    line-height: 1.5;
    transition: all 0.35s ease;
  }

  &__trans {
    font-size: 14px;
    color: var(--text-faint);
    margin-top: 4px;
    opacity: 0.6;
    transition: all 0.35s ease;
  }

  &:hover {
    background: var(--bg-elevated);
  }

  &--near .lyrics-line__text {
    color: var(--text-muted);
  }

  &--active {
    .lyrics-line__text {
      font-size: 24px;
      font-weight: 700;
      color: var(--text-primary);
    }

    .lyrics-line__trans {
      color: var(--text-secondary);
      opacity: 1;
    }
  }
}

[data-theme="night"] .lyrics-line--active .lyrics-line__text {
  text-shadow: 0 0 20px rgba(var(--accent-rgb), 0.35);
  background: linear-gradient(90deg, #5eead4, #a5b4fc, #f472b6);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  background-size: 200% 100%;
  animation: lyric-gradient 4s ease infinite;
}

@keyframes lyric-gradient {
  0%, 100% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
}

[data-theme="day"] .lyrics-line--active .lyrics-line__text {
  color: var(--accent);
  -webkit-text-fill-color: var(--accent);
}

.lyrics-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding-top: 20vh;
  color: var(--text-faint);
  font-size: 16px;
}

/* ── Slide transition ───────────── */
.lyrics-slide-enter-active,
.lyrics-slide-leave-active {
  transition: transform 0.45s cubic-bezier(0.4, 0, 0.2, 1),
              opacity 0.35s ease;
}

.lyrics-slide-enter-from {
  transform: translateY(100%);
  opacity: 0;
}

.lyrics-slide-leave-to {
  transform: translateY(100%);
  opacity: 0;
}

/* ── Responsive ─────────────────── */
@media (max-width: 860px) {
  .lyrics-top__title,
  .lyrics-top__spacer {
    display: none;
  }

  .lyrics-top {
    padding: 12px 0 8px;
  }

  .lyrics-body {
    grid-template-columns: 1fr;
    grid-template-rows: auto auto 1fr auto;
    gap: 0;
    padding-bottom: 0;
  }

  .lyrics-song-info {
    display: none;
  }

  .lyrics-mobile-cover {
    display: flex !important;
    justify-content: center;
    align-items: center;
    grid-column: 1;
    grid-row: 1;
    padding: 10px 0 20px;
    
    &__img {
      width: 260px;
      height: 260px;
      border-radius: 50%;
      object-fit: cover;
      border: 6px solid rgba(255, 255, 255, 0.1);
      box-shadow: 0 16px 32px rgba(0, 0, 0, 0.4), 0 4px 12px rgba(0, 0, 0, 0.3);
      animation: float-cover 6s ease-in-out infinite, spin-cover 20s linear infinite;
    }
  }

  .cassette--playing .lyrics-mobile-cover__img {
    animation-play-state: running;
  }

  .lyrics-mobile-cover__img:not(.cassette--playing *) {
    animation-play-state: paused;
  }

  @keyframes float-cover {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(-8px); }
  }

  @keyframes spin-cover {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .lyrics-mobile-header {
    display: flex !important;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    grid-column: 1;
    grid-row: 2;
    padding: 0 0 16px;
    text-align: center;
    gap: 4px;

    &__name {
      font-size: 18px;
      font-weight: 700;
      color: var(--text-primary);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      max-width: 100%;
    }

    &__artist {
      font-size: 13px;
      color: var(--text-muted);
    }
  }

  .lyrics-visual {
    display: none;
  }

  .lyrics-scroll-area {
    grid-column: 1;
    grid-row: 3;
    min-height: 0;
    mask-image: linear-gradient(
      to bottom,
      transparent 0%,
      #000 8%,
      #000 85%,
      transparent 100%
    );
    -webkit-mask-image: linear-gradient(
      to bottom,
      transparent 0%,
      #000 8%,
      #000 85%,
      transparent 100%
    );
  }

  .lyrics-playback {
    grid-column: 1;
    grid-row: 4;
    border-top: var(--border-width) solid var(--border-color);
    background: var(--bg-player);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    margin: 0 -32px;
    padding: 14px 32px;
    padding-bottom: calc(14px + env(safe-area-inset-bottom, 0px));
    gap: 12px;
  }

  .lyrics-progress {
    order: 1;
  }

  .lyrics-controls {
    order: 2;
  }

  .lyrics-line {
    text-align: center;
  }
}

@media (max-width: 640px) {
  .lyrics-page__inner {
    padding: 0 16px;
  }

  .lyrics-top {
    padding: 12px 0 8px;
  }

  .lyrics-mobile-header {
    padding: 6px 0 10px;

    &__name {
      font-size: 16px;
    }
  }

  .lyrics-playback {
    margin: 0 -16px;
    padding: 12px 16px;
    padding-bottom: calc(12px + env(safe-area-inset-bottom, 0px));
    gap: 10px;
  }

  .lyrics-line {
    padding: 8px 12px;

    &__text {
      font-size: 17px;
    }

    &--active .lyrics-line__text {
      font-size: 20px;
    }
  }

  .lyrics-controls {
    gap: 14px;
    min-height: 48px;
  }

  .lc-volume__slider {
    width: 64px;
  }

  .lc-btn {
    width: 38px;
    height: 38px;

    &--play {
      width: 48px;
      height: 48px;
    }
  }

  .lyrics-progress {
    max-width: 100%;
    padding: 0;
  }

  .lyrics-spacer {
    height: 15vh;
  }
}
</style>
