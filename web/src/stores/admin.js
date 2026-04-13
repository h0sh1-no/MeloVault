import { ref } from 'vue'
import { defineStore } from 'pinia'
import api from '@/api'

export const useAdminStore = defineStore('admin', () => {
  // null = not yet checked, false = not initialized, true = initialized
  const setupInitialized = ref(null)

  async function checkSetupStatus() {
    // Let errors propagate so the router guard can decide what to do
    const res = await api.get('/api/setup/status')
    const val = res.data?.data?.initialized
    // Only accept explicit boolean; unknown format keeps state as null
    setupInitialized.value = typeof val === 'boolean' ? val : null
    return setupInitialized.value
  }

  async function initSuperAdmin(username, email, password) {
    const res = await api.post('/api/setup/init', { username, email, password })
    setupInitialized.value = true
    return res.data
  }

  async function getStats() {
    const res = await api.get('/api/admin/stats')
    return res.data?.data
  }

  async function listUsers(page = 1, pageSize = 20, search = '') {
    const res = await api.get('/api/admin/users', {
      params: { page, page_size: pageSize, search: search || undefined }
    })
    return res.data?.data
  }

  async function getUser(id) {
    const res = await api.get(`/api/admin/users/${id}`)
    return res.data?.data
  }

  async function updateUser(id, username, role) {
    const res = await api.put(`/api/admin/users/${id}`, { username, role })
    return res.data
  }

  async function deleteUser(id) {
    const res = await api.delete(`/api/admin/users/${id}`)
    return res.data
  }

  async function listDownloads(page = 1, pageSize = 20, search = '') {
    const res = await api.get('/api/admin/downloads', {
      params: { page, page_size: pageSize, search: search || undefined }
    })
    return res.data?.data
  }

  async function getNeteaseQRKey() {
    const res = await api.get('/api/admin/netease/qr/key')
    return res.data?.data
  }

  async function checkNeteaseQRStatus(key) {
    const res = await api.get('/api/admin/netease/qr/check', { params: { key } })
    return res.data?.data
  }

  async function setNeteaseCookie(cookie) {
    const res = await api.post('/api/admin/netease/cookie', { cookie })
    return res.data
  }

  async function listNeteaseAccounts() {
    const res = await api.get('/api/admin/netease/accounts')
    return res.data?.data
  }

  async function addNeteaseAccount(nickname, cookie) {
    const res = await api.post('/api/admin/netease/accounts', { nickname, cookie })
    return res.data
  }

  async function updateNeteaseAccount(id, data) {
    const res = await api.put(`/api/admin/netease/accounts/${id}`, data)
    return res.data
  }

  async function deleteNeteaseAccount(id) {
    const res = await api.delete(`/api/admin/netease/accounts/${id}`)
    return res.data
  }

  async function getAnalyticsOverview() {
    const res = await api.get('/api/admin/analytics/overview')
    return res.data?.data
  }

  async function getActivityLogs(page = 1, pageSize = 20, filters = {}) {
    const res = await api.get('/api/admin/analytics/activity', {
      params: {
        page,
        page_size: pageSize,
        action: filters.action || undefined,
        user_id: filters.userId || undefined,
        ip: filters.ip || undefined,
        search: filters.search || undefined
      }
    })
    return res.data?.data
  }

  async function getOnlineUsers(minutes = 15) {
    const res = await api.get('/api/admin/analytics/online', { params: { minutes } })
    return res.data?.data
  }

  async function getProvinceStats(days = 30) {
    const res = await api.get('/api/admin/analytics/provinces', { params: { days } })
    return res.data?.data
  }

  async function getTrends(days = 7) {
    const res = await api.get('/api/admin/analytics/trends', { params: { days } })
    return res.data?.data
  }

  async function getUserActivity(userId, page = 1, pageSize = 20, action = '') {
    const res = await api.get(`/api/admin/users/${userId}/activity`, {
      params: { page, page_size: pageSize, action: action || undefined }
    })
    return res.data?.data
  }

  async function createUser(username, email, password, role) {
    const res = await api.post('/api/admin/users', { username, email, password, role })
    return res.data
  }

  async function resetPassword(id, password) {
    const res = await api.put(`/api/admin/users/${id}/password`, { password })
    return res.data
  }

  async function getUserDownloads(userId, page = 1, pageSize = 20) {
    const res = await api.get(`/api/admin/users/${userId}/downloads`, {
      params: { page, page_size: pageSize }
    })
    return res.data?.data
  }

  async function getLegalDocuments(type = 'terms') {
    const res = await api.get('/api/admin/legal', { params: { type } })
    return res.data?.data
  }

  async function saveLegalDocument(type, title, content) {
    const res = await api.post('/api/admin/legal', { type, title, content })
    return res.data
  }

  return {
    setupInitialized,
    checkSetupStatus,
    initSuperAdmin,
    getStats,
    listUsers,
    getUser,
    updateUser,
    deleteUser,
    listDownloads,
    getNeteaseQRKey,
    checkNeteaseQRStatus,
    setNeteaseCookie,
    listNeteaseAccounts,
    addNeteaseAccount,
    updateNeteaseAccount,
    deleteNeteaseAccount,
    getAnalyticsOverview,
    getActivityLogs,
    getOnlineUsers,
    getProvinceStats,
    getTrends,
    getUserActivity,
    createUser,
    resetPassword,
    getUserDownloads,
    getLegalDocuments,
    saveLegalDocument
  }
})
