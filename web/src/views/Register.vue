<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-header">
        <h1>MeloVault</h1>
        <p>{{ registerDisabled ? '注册已关闭' : '创建您的账户' }}</p>
      </div>

      <!-- Registration disabled message -->
      <div v-if="registerDisabled && settingsLoaded" class="register-disabled">
        <el-alert type="info" :closable="false" show-icon>
          {{ disabledReason }}
        </el-alert>
        <div class="auth-footer" style="margin-top: 20px;">
          <span>已有账户?</span>
          <router-link to="/login">立即登录</router-link>
        </div>
      </div>

      <!-- Normal registration form -->
      <template v-else-if="settingsLoaded">
        <el-form ref="formRef" :model="form" :rules="rules" @submit.prevent="handleRegister">
          <el-form-item prop="username">
            <el-input
              v-model="form.username"
              placeholder="用户名"
              size="large"
              :prefix-icon="User"
            />
          </el-form-item>
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
          <el-form-item prop="confirmPassword">
            <el-input
              v-model="form.confirmPassword"
              type="password"
              placeholder="确认密码"
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
              注册
            </el-button>
          </el-form-item>
        </el-form>

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

        <div class="auth-footer">
          <span>已有账户?</span>
          <router-link to="/login">立即登录</router-link>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { User, Message, Lock } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { useSiteSettingsStore } from '@/stores/siteSettings'
import api from '@/api'

const router = useRouter()
const authStore = useAuthStore()
const siteStore = useSiteSettingsStore()

const formRef = ref(null)
const loading = ref(false)
const settingsLoaded = ref(false)

const registerDisabled = computed(() =>
  !siteStore.features.allow_register || !siteStore.features.allow_email_register
)
const disabledReason = computed(() => {
  if (!siteStore.features.allow_register) return '管理员已关闭注册功能，暂不接受新用户注册'
  if (!siteStore.features.allow_email_register) return '管理员已关闭邮箱注册，请尝试其他方式'
  return ''
})

const termsDoc = ref(null)
const disclaimerDoc = ref(null)
const legalDialogVisible = ref(false)
const legalDialogTitle = ref('')
const legalDialogContent = ref('')

const hasLegalDocs = computed(() => termsDoc.value || disclaimerDoc.value)

const form = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  agreedTerms: false
})

const validateConfirmPassword = (_rule, value, callback) => {
  if (value !== form.password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const validateAgreed = (_rule, value, callback) => {
  if (!value) {
    callback(new Error('请阅读并同意服务条款'))
  } else {
    callback()
  }
}

const rules = computed(() => {
  const base = {
    username: [
      { required: true, message: '请输入用户名', trigger: 'blur' },
      { min: 2, max: 50, message: '用户名长度在2-50个字符', trigger: 'blur' }
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
      { required: true, message: '请确认密码', trigger: 'blur' },
      { validator: validateConfirmPassword, trigger: 'blur' }
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
})

async function handleRegister() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    const result = await authStore.register(form.username, form.email, form.password)
    if (result.success) {
      ElMessage.success('注册成功')
      router.push('/')
    } else {
      ElMessage.error(result.message || '注册失败')
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '注册失败')
  } finally {
    loading.value = false
  }
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

.register-disabled {
  text-align: center;

  :deep(.el-alert) {
    background: var(--bg-elevated);
    border: var(--border-width) solid var(--border-color);
  }
}

:deep(.el-checkbox__label) {
  color: var(--text-muted);
  font-size: 13px;
  padding-right: 0;
}
</style>
