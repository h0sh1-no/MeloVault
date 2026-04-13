import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'

const DEFAULT_TITLE = 'MeloVault'

function updateDocumentTitle(song) {
  if (song) {
    const name = song.name || song.song_name || ''
    const artists = song.artists || song.artist_string || ''
    document.title = artists ? `${name} - ${artists} | ${DEFAULT_TITLE}` : `${name} | ${DEFAULT_TITLE}`
  } else {
    document.title = DEFAULT_TITLE
  }
}

export const usePlayerStore = defineStore('player', () => {
  const currentSong = ref(null)
  const playlist = ref([])
  const currentIndex = ref(-1)
  const isPlaying = ref(false)
  const currentTime = ref(0)
  const duration = ref(0)
  const volume = ref(parseFloat(localStorage.getItem('volume') || '0.8'))
  const repeatMode = ref('none') // none, one, all
  const audioElement = ref(null)
  const streamingQuality = ref(localStorage.getItem('streamingQuality') || 'jymaster')
  const downloadQuality = ref(localStorage.getItem('downloadQuality') || 'jymaster')

  // Update document title when song changes
  watch(currentSong, (song) => {
    updateDocumentTitle(song)
  }, { immediate: true })

  const progress = computed(() => {
    if (duration.value === 0) return 0
    return (currentTime.value / duration.value) * 100
  })

  function setAudioElement(el) {
    audioElement.value = el
    if (el) {
      el.volume = volume.value
    }
  }

  function play(song, songList = null) {
    if (songList) {
      playlist.value = songList
      currentIndex.value = songList.findIndex(s => s.id === song.id)
    } else if (!playlist.value.find(s => s.id === song.id)) {
      playlist.value = [song]
      currentIndex.value = 0
    } else {
      currentIndex.value = playlist.value.findIndex(s => s.id === song.id)
    }
    currentSong.value = song
    isPlaying.value = true
  }

  function pause() {
    isPlaying.value = false
    if (audioElement.value) {
      audioElement.value.pause()
    }
  }

  function resume() {
    if (currentSong.value && audioElement.value) {
      isPlaying.value = true
      audioElement.value.play()
    }
  }

  function toggle() {
    if (isPlaying.value) {
      pause()
    } else {
      resume()
    }
  }

  function next() {
    if (playlist.value.length === 0) return
    if (currentIndex.value < playlist.value.length - 1) {
      currentIndex.value++
    } else if (repeatMode.value === 'all') {
      currentIndex.value = 0
    } else {
      return
    }
    currentSong.value = playlist.value[currentIndex.value]
    isPlaying.value = true
  }

  function prev() {
    if (playlist.value.length === 0) return
    if (currentIndex.value > 0) {
      currentIndex.value--
    } else if (repeatMode.value === 'all') {
      currentIndex.value = playlist.value.length - 1
    } else {
      return
    }
    currentSong.value = playlist.value[currentIndex.value]
    isPlaying.value = true
  }

  function seek(time) {
    if (audioElement.value) {
      audioElement.value.currentTime = time
      currentTime.value = time
    }
  }

  function setVolume(v) {
    volume.value = v
    localStorage.setItem('volume', v.toString())
    if (audioElement.value) {
      audioElement.value.volume = v
    }
  }

  function toggleRepeat() {
    const modes = ['none', 'one', 'all']
    const idx = modes.indexOf(repeatMode.value)
    repeatMode.value = modes[(idx + 1) % modes.length]
  }

  function setStreamingQuality(q) {
    streamingQuality.value = q
    localStorage.setItem('streamingQuality', q)
  }

  function setDownloadQuality(q) {
    downloadQuality.value = q
    localStorage.setItem('downloadQuality', q)
  }

  function clearPlaylist() {
    playlist.value = []
    currentIndex.value = -1
    currentSong.value = null
    isPlaying.value = false
    document.title = DEFAULT_TITLE
  }

  function updateTime(time) {
    currentTime.value = time
  }

  function updateDuration(d) {
    duration.value = d
  }

  function onEnded() {
    if (repeatMode.value === 'one') {
      if (audioElement.value) {
        audioElement.value.currentTime = 0
        audioElement.value.play()
      }
    } else {
      next()
    }
  }

  return {
    currentSong,
    playlist,
    currentIndex,
    isPlaying,
    currentTime,
    duration,
    volume,
    repeatMode,
    streamingQuality,
    downloadQuality,
    progress,
    setAudioElement,
    play,
    pause,
    resume,
    toggle,
    next,
    prev,
    seek,
    setVolume,
    toggleRepeat,
    setStreamingQuality,
    setDownloadQuality,
    clearPlaylist,
    updateTime,
    updateDuration,
    onEnded
  }
})
