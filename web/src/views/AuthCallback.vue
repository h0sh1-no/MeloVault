<template>
  <div class="auth-callback">
    <div class="loading">
      <el-icon class="is-loading" :size="48"><Loading /></el-icon>
      <p>正在登录...</p>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Loading } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

onMounted(async () => {
  const accessToken = route.query.access_token
  const refreshToken = route.query.refresh_token

  if (accessToken && refreshToken) {
    authStore.setTokens({ access_token: accessToken, refresh_token: refreshToken })
    await authStore.fetchUser()
    ElMessage.success('登录成功')
    router.push('/')
  } else {
    ElMessage.error('登录失败')
    router.push('/login')
  }
})
</script>

<style lang="scss" scoped>
.auth-callback {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-gradient);

  .loading {
    text-align: center;

    .el-icon {
      color: var(--accent);
    }

    p {
      color: var(--text-secondary);
      margin-top: 16px;
    }
  }
}
</style>
