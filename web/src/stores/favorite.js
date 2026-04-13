import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/api'

export const useFavoriteStore = defineStore('favorite', () => {
  const favorites = ref(new Map())
  const favoriteList = ref([])
  const total = ref(0)

  async function fetchFavorites(page = 1, pageSize = 20) {
    try {
      const res = await api.get('/api/favorites', { params: { page, page_size: pageSize } })
      if (res.data.success) {
        favoriteList.value = res.data.data.list
        total.value = res.data.data.total
        // Update map
        res.data.data.list.forEach(item => {
          favorites.value.set(item.song_id, item)
        })
      }
      return res.data
    } catch (e) {
      throw e
    }
  }

  async function checkFavorites(songIds) {
    if (!songIds || songIds.length === 0) return
    try {
      const res = await api.post('/api/favorites/batch-check', { song_ids: songIds })
      if (res.data.success) {
        const result = res.data.data.favorites
        for (const [id, isFav] of Object.entries(result)) {
          if (isFav) {
            favorites.value.set(parseInt(id), true)
          } else {
            favorites.value.delete(parseInt(id))
          }
        }
      }
    } catch (e) {
      console.error('check favorites error:', e)
    }
  }

  async function add(song) {
    try {
      const songId = song.song_id || song.id
      const payload = { ...song, song_id: songId }
      const res = await api.post('/api/favorites', payload)
      if (res.data.success) {
        favorites.value.set(songId, res.data.data)
      }
      return res.data
    } catch (e) {
      throw e
    }
  }

  async function remove(songId) {
    try {
      const res = await api.delete(`/api/favorites/${songId}`)
      if (res.data.success) {
        favorites.value.delete(songId)
      }
      return res.data
    } catch (e) {
      throw e
    }
  }

  function isFavorited(songId) {
    return favorites.value.has(songId)
  }

  return {
    favorites,
    favoriteList,
    total,
    fetchFavorites,
    checkFavorites,
    add,
    remove,
    isFavorited
  }
})
