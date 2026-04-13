<template>
  <div class="setup-page">
    <div class="setup-card">
      <div class="setup-logo">
        <Headphones :size="48" class="logo-icon" />
        <h1>MeloVault</h1>
        <p class="subtitle">首次部署 — 创建超级管理员</p>
      </div>

      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
        class="setup-form"
        @submit.prevent="handleSubmit"
      >
        <el-form-item label="管理员用户名" prop="username">
          <el-input
            v-model="form.username"
            placeholder="设置管理员用户名"
            size="large"
            prefix-icon="User"
            :disabled="loading"
          />
        </el-form-item>

        <el-form-item label="邮箱" prop="email">
          <el-input
            v-model="form.email"
            placeholder="用于登录的邮箱地址"
            size="large"
            prefix-icon="Message"
            :disabled="loading"
          />
        </el-form-item>

        <el-form-item label="密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="至少 6 位密码"
            size="large"
            show-password
            :disabled="loading"
          />
        </el-form-item>

        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input
            v-model="form.confirmPassword"
            type="password"
            placeholder="再次输入密码"
            size="large"
            show-password
            :disabled="loading"
          />
        </el-form-item>

        <el-button
          size="large"
          :loading="loading"
          class="submit-btn"
          native-type="submit"
          @click="handleSubmit"
        >
          {{ loading ? '正在初始化...' : '创建超级管理员' }}
        </el-button>
      </el-form>

      <p class="tip">此页面仅在系统无任何账户时显示，初始化后将自动跳转到后台管理。</p>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Headphones } from 'lucide-vue-next'
import { useAdminStore } from '@/stores/admin'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const adminStore = useAdminStore()
const authStore = useAuthStore()

const formRef = ref(null)
const loading = ref(false)
const form = reactive({ username: '', email: '', password: '', confirmPassword: '' })

const validateConfirm = (rule, value, callback) => {
  if (value !== form.password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 50, message: '用户名长度须在2-50个字符之间', trigger: 'blur' }
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少6个字符', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    { validator: validateConfirm, trigger: 'blur' }
  ]
}

async function handleSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    const result = await adminStore.initSuperAdmin(form.username, form.email, form.password)
    if (result?.data?.tokens) {
      authStore.setTokens(result.data.tokens)
      authStore.user = result.data.user
    }
    ElMessage.success('超级管理员创建成功！正在跳转...')
    setTimeout(() => router.push('/admin'), 800)
  } catch (err) {
    const msg = err.response?.data?.message || '初始化失败，请重试'
    ElMessage.error(msg)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.setup-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-gradient);
  padding: 24px;
}

.setup-card {
  background: var(--card-bg);
  backdrop-filter: var(--card-backdrop);
  border: var(--border-width) solid var(--card-border);
  border-radius: var(--radius-lg);
  padding: 48px 40px;
  width: 100%;
  max-width: 440px;
  box-shadow: var(--shadow-lg);
}

.setup-logo {
  text-align: center;
  margin-bottom: 36px;
}

.logo-icon {
  color: var(--accent);
  margin-bottom: 12px;
}

.setup-logo h1 {
  font-size: 28px;
  font-weight: var(--title-weight);
  color: var(--text-primary);
  margin: 0 0 8px;
}

.subtitle {
  color: var(--text-muted);
  font-size: 14px;
  margin: 0;
}

.setup-form :deep(.el-form-item__label) {
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: var(--el-font-weight);
}

.setup-form :deep(.el-input__wrapper) {
  background: var(--bg-input);
  border: var(--border-width) solid var(--border-color);
  box-shadow: none !important;
  transition: border-color 0.3s;
}

.setup-form :deep(.el-input__wrapper:hover),
.setup-form :deep(.el-input__wrapper.is-focus) {
  border-color: var(--accent);
}

.setup-form :deep(.el-input__inner) {
  color: var(--text-primary);
}

.setup-form :deep(.el-input__inner::placeholder) {
  color: var(--text-faint);
}

.submit-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: var(--title-weight);
  background: var(--accent-btn-bg);
  border: var(--border-width) solid var(--btn-border);
  border-radius: var(--radius-sm);
  color: var(--accent-btn-text);
  box-shadow: var(--btn-shadow);
  margin-top: 8px;
  transition: all 0.2s;
}

.submit-btn:hover {
  opacity: 0.92;
  box-shadow: var(--btn-hover-shadow);
  transform: var(--btn-hover-transform);
}

.tip {
  text-align: center;
  color: var(--text-faint);
  font-size: 12px;
  margin-top: 24px;
  margin-bottom: 0;
  line-height: 1.6;
}
</style>
