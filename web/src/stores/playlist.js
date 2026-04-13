import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api'

export const usePlaylistStore = defineStore('playlist', () => {
  const myPlaylists = ref([])
  const total = ref(0)
  const loading = ref(false)

  async function fetchMyPlaylists(page = 1, pageSize = 50) {
    loading.value = true
    try {
      const res = await api.get('/api/playlists', { params: { page, page_size: pageSize } })
      if (res.data.success) {
        myPlaylists.value = res.data.data.list || []
        total.value = res.data.data.total
      }
      return res.data
    } finally {
      loading.value = false
    }
  }

  async function createPlaylist(name, description = '', coverURL = '') {
    const res = await api.post('/api/playlists', { name, description, cover_url: coverURL })
    if (res.data.success) {
      myPlaylists.value.unshift(res.data.data)
      total.value++
    }
    return res.data
  }

  async function updatePlaylist(id, data) {
    const res = await api.put(`/api/playlists/${id}`, data)
    if (res.data.success) {
      const idx = myPlaylists.value.findIndex(p => p.id === id)
      if (idx !== -1) {
        myPlaylists.value[idx] = res.data.data
      }
    }
    return res.data
  }

  async function deletePlaylist(id) {
    const res = await api.delete(`/api/playlists/${id}`)
    if (res.data.success) {
      myPlaylists.value = myPlaylists.value.filter(p => p.id !== id)
      total.value--
    }
    return res.data
  }

  async function getPlaylistDetail(id) {
    const res = await api.get(`/api/playlists/${id}`)
    return res.data
  }

  async function addSongToPlaylist(playlistId, song) {
    const res = await api.post(`/api/playlists/${playlistId}/songs`, song)
    return res.data
  }

  async function removeSongFromPlaylist(playlistId, songId) {
    const res = await api.delete(`/api/playlists/${playlistId}/songs/${songId}`)
    return res.data
  }

  async function getSharedPlaylist(id, sharerId) {
    const params = {}
    if (sharerId) params.sharer = sharerId
    const res = await api.get(`/api/shared/playlist/${id}`, { params })
    return res.data
  }

  return {
    myPlaylists,
    total,
    loading,
    fetchMyPlaylists,
    createPlaylist,
    updatePlaylist,
    deletePlaylist,
    getPlaylistDetail,
    addSongToPlaylist,
    removeSongFromPlaylist,
    getSharedPlaylist
  }
})
