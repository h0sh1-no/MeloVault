<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-header">
        <h1>MeloVault</h1>
        <p>登录您的账户</p>
      </div>

      <el-form v-if="showEmailForm" ref="formRef" :model="form" :rules="rules" @submit.prevent="handleLogin">
        <el-form-item prop="email">
          <el-input
            v-model="form.email"
            placeholder="邮箱"
            size="large"
            :prefix-icon="Message"
          />
        </el-form-item>
        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="密码"
            size="large"
            :prefix-icon="Lock"
            show-password
          />
        </el-form-item>
        <el-form-item v-if="hasLegalDocs" prop="agreedTerms">
          <div class="terms-agreement">
            <el-checkbox v-model="form.agreedTerms">
              我已阅读并同意
            </el-checkbox>
            <span v-if="termsDoc" class="terms-link" @click="showLegalDialog('terms')">《{{ termsDoc.title }}》</span>
            <span v-if="termsDoc && disclaimerDoc">和</span>
            <span v-if="disclaimerDoc" class="terms-link" @click="showLegalDialog('disclaimer')">《{{ disclaimerDoc.title }}》</span>
          </div>
        </el-form-item>
        <el-form-item>
          <el-button
            :loading="loading"
            class="submit-btn"
            native-type="submit"
          >
            登录
          </el-button>
        </el-form-item>
      </el-form>

      <template v-if="showEmailForm && showLinuxdoBtn">
        <el-divider>或</el-divider>
      </template>

      <template v-if="!showEmailForm && !showLinuxdoBtn && settingsLoaded">
        <div class="no-login-methods">
          <p>管理员已关闭所有登录方式</p>
        </div>
      </template>

      <template v-if="!showEmailForm && showLinuxdoBtn && hasLegalDocs">
        <el-form-item>
          <div class="terms-agreement">
            <el-checkbox v-model="form.agreedTerms">
              我已阅读并同意
            </el-checkbox>
            <span v-if="termsDoc" class="terms-link" @click="showLegalDialog('terms')">《{{ termsDoc.title }}》</span>
            <span v-if="termsDoc && disclaimerDoc">和</span>
            <span v-if="disclaimerDoc" class="terms-link" @click="showLegalDialog('disclaimer')">《{{ disclaimerDoc.title }}》</span>
          </div>
        </el-form-item>
      </template>

      <el-button
        v-if="showLinuxdoBtn"
        class="oauth-btn"
        size="large"
        @click="handleLinuxdoLogin"
      >
        <svg viewBox="0 0 24 24" width="20" height="20" style="margin-right: 8px;">
          <path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
        </svg>
        使用 Linuxdo 登录
      </el-button>

      <!-- Legal document dialog -->
      <el-dialog
        v-model="legalDialogVisible"
        :title="legalDialogTitle"
        width="600px"
        class="legal-dialog"
        destroy-on-close
      >
        <div class="legal-content" v-html="legalDialogContent"></div>
      </el-dialog>

      <div class="auth-footer" v-if="showRegisterLink">
        <span>还没有账户?</span>
        <router-link to="/register">立即注册</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Message, Lock } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useSiteSettingsStore } from '@/stores/siteSettings'
import api from '@/api'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const siteStore = useSiteSettingsStore()

const formRef = ref(null)
const loading = ref(false)
const settingsLoaded = ref(false)

const allowEmailLogin = computed(() => siteStore.features.allow_email_login)
const allowLinuxdoLogin = computed(() => siteStore.features.allow_linuxdo_login)
const linuxdoConfigured = computed(() => siteStore.features.linuxdo_configured !== false)
const allowRegister = computed(() => siteStore.features.allow_register)
const showEmailForm = computed(() => allowEmailLogin.value)
const showLinuxdoBtn = computed(() => allowLinuxdoLogin.value && linuxdoConfigured.value)
const showRegisterLink = computed(() => allowRegister.value)

const termsDoc = ref(null)
const disclaimerDoc = ref(null)
const legalDialogVisible = ref(false)
const legalDialogTitle = ref('')
const legalDialogContent = ref('')

const hasLegalDocs = computed(() => termsDoc.value || disclaimerDoc.value)

const form = reactive({
  email: '',
  password: '',
  agreedTerms: false
})

const validateAgreed = (_rule, value, callback) => {
  if (!value) {
    callback(new Error('请阅读并同意服务条款'))
  } else {
    callback()
  }
}

const rules = computed(() => {
  const base = {
    email: [
      { required: true, message: '请输入邮箱', trigger: 'blur' },
      { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' }
    ],
    password: [
      { required: true, message: '请输入密码', trigger: 'blur' },
      { min: 6, message: '密码至少6个字符', trigger: 'blur' }
    ]
  }
  if (hasLegalDocs.value) {
    base.agreedTerms = [{ validator: validateAgreed, trigger: 'change' }]
  }
  return base
})

async function fetchLegalDocs() {
  try {
    const [termsRes, disclaimerRes] = await Promise.all([
      api.get('/api/legal/terms'),
      api.get('/api/legal/disclaimer')
    ])
    termsDoc.value = termsRes.data?.data || null
    disclaimerDoc.value = disclaimerRes.data?.data || null
  } catch {
    // Legal docs not configured, skip
  }
}

function showLegalDialog(type) {
  const doc = type === 'terms' ? termsDoc.value : disclaimerDoc.value
  if (!doc) return
  legalDialogTitle.value = doc.title
  legalDialogContent.value = doc.content
  legalDialogVisible.value = true
}

onMounted(async () => {
  if (!siteStore.loaded) {
    try { await siteStore.fetch() } catch { /* proceed with defaults */ }
  }
  settingsLoaded.value = true
  await fetchLegalDocs()

  if (route.query.error === 'register_disabled') {
    ElMessage.warning('管理员已关闭注册，LinuxDO 新用户无法登录')
  }
})

async function handleLogin() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    const result = await authStore.login(form.email, form.password)
    if (result.success) {
      ElMessage.success('登录成功')
      const redirect = route.query.redirect || '/'
      router.push(redirect)
    } else {
      ElMessage.error(result.message || '登录失败')
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '登录失败')
  } finally {
    loading.value = false
  }
}

function handleLinuxdoLogin() {
  if (hasLegalDocs.value && !form.agreedTerms) {
    ElMessage.warning('请先阅读并同意服务条款')
    return
  }
  window.location.href = '/api/auth/linuxdo'
}
</script>

<style lang="scss" scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-gradient);
  padding: 24px;
}

.auth-card {
  width: 100%;
  max-width: 400px;
  background: var(--card-bg);
  backdrop-filter: var(--card-backdrop);
  border: var(--border-width) solid var(--card-border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-lg);
  padding: 40px;
  transition: all 0.4s;

  @media (max-width: 640px) {
    padding: 28px 20px;
    border-radius: var(--radius-lg);
  }
}

.auth-header {
  text-align: center;
  margin-bottom: 32px;

  h1 {
    font-size: 28px;
    font-weight: var(--title-weight);
    text-transform: var(--title-transform);
    letter-spacing: var(--title-letter-spacing);
    margin: 0 0 8px;
    color: var(--text-primary);
  }

  p {
    color: var(--text-muted);
    margin: 0;
  }
}

[data-theme="night"] .auth-header h1 {
  background: var(--accent-gradient);
  background-size: 200% 200%;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

[data-theme="day"] .auth-header h1 {
  color: var(--text-primary);
}

:deep(.el-input__wrapper) {
  background: var(--bg-input);
  border: var(--border-width) solid var(--border-color);
  border-radius: var(--radius-sm);
  transition: all 0.3s;

  &:hover, &.is-focus {
    border-color: var(--accent);
  }

  .el-input__inner {
    color: var(--text-primary);

    &::placeholder {
      color: var(--text-faint);
    }
  }
}

[data-theme="day"] :deep(.el-input__wrapper) {
  box-shadow: var(--shadow-sm);

  &:hover, &.is-focus {
    box-shadow: var(--shadow-hover);
  }
}

:deep(.el-form-item__error) {
  color: #f56c6c;
}

.submit-btn {
  width: 100%;
  background: var(--accent-btn-bg);
  border: var(--border-width) solid var(--btn-border);
  border-radius: var(--radius-sm);
  color: var(--accent-btn-text);
  font-weight: var(--title-weight);
  font-size: 16px;
  height: 42px;
  box-shadow: var(--btn-shadow);
  transition: all 0.2s;

  &:hover {
    opacity: 0.92;
    box-shadow: var(--btn-hover-shadow);
    transform: var(--btn-hover-transform);
  }
}

:deep(.el-divider__text) {
  background: var(--card-bg);
  color: var(--text-faint);
}

:deep(.el-divider) {
  border-color: var(--divider-color);
}

.oauth-btn {
  width: 100%;
  background: var(--btn-bg);
  border: var(--border-width) solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-weight: var(--el-font-weight);
  box-shadow: var(--btn-shadow);
  transition: all 0.2s;

  &:hover {
    background: var(--btn-hover-bg);
    box-shadow: var(--btn-hover-shadow);
    transform: var(--btn-hover-transform);
  }
}

.auth-footer {
  text-align: center;
  margin-top: 24px;
  color: var(--text-muted);
  font-size: 14px;

  a {
    color: var(--accent);
    text-decoration: none;
    margin-left: 4px;
    font-weight: var(--el-font-weight);

    &:hover {
      text-decoration: underline;
    }
  }
}

.terms-agreement {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  font-size: 13px;
  color: var(--text-muted);
  line-height: 1.6;
}

.terms-link {
  color: var(--accent);
  cursor: pointer;
  font-weight: 500;

  &:hover {
    text-decoration: underline;
  }
}

.no-login-methods {
  text-align: center;
  padding: 24px 0;
  color: var(--text-muted);
  font-size: 14px;
}

:deep(.el-checkbox__label) {
  color: var(--text-muted);
  font-size: 13px;
  padding-right: 0;
}
</style>
