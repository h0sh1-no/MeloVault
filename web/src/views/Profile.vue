<template>
  <div class="profile-page">
    <div class="profile-card">
      <div class="profile-header">
        <el-avatar :size="80" :src="authStore.user?.avatar">
          {{ authStore.user?.username?.charAt(0).toUpperCase() }}
        </el-avatar>
        <div class="profile-info">
          <h2>{{ authStore.user?.username }}</h2>
          <p>{{ authStore.user?.email }}</p>
          <p class="provider">
            <el-tag size="small" type="info">
              {{ authStore.user?.provider === 'linuxdo' ? 'Linuxdo' : '邮箱注册' }}
            </el-tag>
          </p>
        </div>
      </div>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="个人信息" name="profile">
          <el-form ref="profileFormRef" :model="profileData" :rules="profileRules" :label-width="isMobile ? undefined : '80px'" :label-position="isMobile ? 'top' : 'right'">
            <el-form-item label="用户名" prop="username">
              <el-input v-model="profileData.username" />
            </el-form-item>
            <el-form-item label="头像">
              <el-input v-model="profileData.avatar" placeholder="头像URL（可选）" />
            </el-form-item>
            <el-form-item>
              <el-button class="save-btn" :loading="profileLoading" @click="updateProfile">
                保存修改
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="音质设置" name="quality">
          <div class="settings-section" v-loading="settingsStore.loading">
            <div class="settings-group">
              <div class="settings-group-title">
                <Music :size="18" />
                <span>播放音质</span>
              </div>
              <p class="settings-group-desc">选择在线播放时使用的音质，更高音质需要更多带宽</p>
              <div class="quality-grid">
                <div
                  v-for="q in qualityOptions"
                  :key="q.value"
                  class="quality-card"
                  :class="{ 'quality-card--active': localStreamingQuality === q.value }"
                  @click="setStreamingQuality(q.value)"
                >
                  <div class="quality-card-name">{{ q.label }}</div>
                  <div class="quality-card-desc">{{ q.desc }}</div>
                  <div class="quality-card-check" v-if="localStreamingQuality === q.value">
                    <Check :size="14" />
                  </div>
                </div>
              </div>
            </div>

            <el-divider />

            <div class="settings-group">
              <div class="settings-group-title">
                <HardDriveDownload :size="18" />
                <span>下载音质</span>
              </div>
              <p class="settings-group-desc">选择下载时使用的音质，更高音质文件体积更大</p>
              <div class="quality-grid">
                <div
                  v-for="q in qualityOptions"
                  :key="q.value"
                  class="quality-card"
                  :class="{ 'quality-card--active': localDownloadQuality === q.value }"
                  @click="setDownloadQuality(q.value)"
                >
                  <div class="quality-card-name">{{ q.label }}</div>
                  <div class="quality-card-desc">{{ q.desc }}</div>
                  <div class="quality-card-check" v-if="localDownloadQuality === q.value">
                    <Check :size="14" />
                  </div>
                </div>
              </div>
            </div>

            <el-divider />

            <div class="settings-group">
              <div class="settings-group-title">
                <Volume2 :size="18" />
                <span>默认音量</span>
              </div>
              <p class="settings-group-desc">设置播放器的默认音量大小</p>
              <div class="volume-setting">
                <el-slider
                  v-model="localVolume"
                  :min="0"
                  :max="100"
                  :step="1"
                  :format-tooltip="(v) => v + '%'"
                  @change="saveVolume"
                />
                <span class="volume-label">{{ localVolume }}%</span>
              </div>
            </div>

            <el-divider />

            <div class="settings-group">
              <div class="settings-group-title">
                <Repeat :size="18" />
                <span>播放模式</span>
              </div>
              <p class="settings-group-desc">设置默认的循环播放模式</p>
              <el-radio-group v-model="localRepeatMode" @change="saveRepeatMode" class="repeat-radio-group">
                <el-radio-button value="none">不循环</el-radio-button>
                <el-radio-button value="one">单曲循环</el-radio-button>
                <el-radio-button value="all">列表循环</el-radio-button>
              </el-radio-group>
            </div>

            <div class="settings-saved-hint" v-if="savedHint">
              <Check :size="14" />
              <span>设置已自动保存到云端</span>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="修改密码" name="password" v-if="authStore.user?.provider === 'email'">
          <el-form ref="passwordForm" :model="passwordForm" :rules="passwordRules" :label-width="isMobile ? undefined : '100px'" :label-position="isMobile ? 'top' : 'right'">
            <el-form-item label="当前密码" prop="old_password">
              <el-input v-model="passwordForm.old_password" type="password" show-password />
            </el-form-item>
            <el-form-item label="新密码" prop="new_password">
              <el-input v-model="passwordForm.new_password" type="password" show-password />
            </el-form-item>
            <el-form-item label="确认密码" prop="confirm_password">
              <el-input v-model="passwordForm.confirm_password" type="password" show-password />
            </el-form-item>
            <el-form-item>
              <el-button class="save-btn" :loading="passwordLoading" @click="changePassword">
                修改密码
              </el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, watch, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Music, HardDriveDownload, Volume2, Repeat, Check } from 'lucide-vue-next'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { usePlayerStore } from '@/stores/player'

const route = useRoute()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const playerStore = usePlayerStore()

const activeTab = ref(route.query.tab === 'quality' ? 'quality' : 'profile')

const isMobile = ref(window.innerWidth <= 640)
function onResize() { isMobile.value = window.innerWidth <= 640 }
onMounted(() => window.addEventListener('resize', onResize))
onUnmounted(() => window.removeEventListener('resize', onResize))

watch(() => route.query.tab, (tab) => {
  activeTab.value = tab === 'quality' ? 'quality' : 'profile'
})

const profileFormRef = ref(null)
const profileData = reactive({
  username: '',
  avatar: ''
})
const profileRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 50, message: '用户名长度在2-50个字符', trigger: 'blur' }
  ]
}
const profileLoading = ref(false)
const savedHint = ref(false)
let savedTimer = null

const qualityOptions = [
  { value: 'jymaster', label: '超清母带', desc: '最高音质，约 24bit/192kHz' },
  { value: 'jyeffect', label: '高清环绕声', desc: '空间音频，沉浸体验' },
  { value: 'sky', label: '沉浸环绕声', desc: '杜比全景声效果' },
  { value: 'hires', label: 'Hi-Res', desc: '高解析度，约 24bit/96kHz' },
  { value: 'lossless', label: '无损', desc: 'FLAC 无损，约 16bit/44.1kHz' },
  { value: 'exhigh', label: '极高', desc: '320kbps MP3' },
  { value: 'standard', label: '标准', desc: '128kbps，节省流量' },
]

const localStreamingQuality = ref(playerStore.streamingQuality)
const localDownloadQuality = ref(playerStore.downloadQuality)
const localVolume = ref(Math.round(playerStore.volume * 100))
const localRepeatMode = ref(playerStore.repeatMode)

function showSavedHint() {
  savedHint.value = true
  if (savedTimer) clearTimeout(savedTimer)
  savedTimer = setTimeout(() => { savedHint.value = false }, 2000)
}

async function setStreamingQuality(q) {
  localStreamingQuality.value = q
  await settingsStore.update({ streaming_quality: q })
  showSavedHint()
}

async function setDownloadQuality(q) {
  localDownloadQuality.value = q
  await settingsStore.update({ download_quality: q })
  showSavedHint()
}

async function saveVolume(val) {
  await settingsStore.update({ volume: val / 100 })
  showSavedHint()
}

async function saveRepeatMode(mode) {
  await settingsStore.update({ repeat_mode: mode })
  showSavedHint()
}

const passwordForm = reactive({
  old_password: '',
  new_password: '',
  confirm_password: ''
})
const validateConfirmPassword = (rule, value, callback) => {
  if (value !== passwordForm.new_password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}
const passwordRules = {
  old_password: [
    { required: true, message: '请输入当前密码', trigger: 'blur' }
  ],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码至少6个字符', trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}
const passwordLoading = ref(false)
const passwordFormRef = ref(null)

onMounted(async () => {
  if (authStore.user) {
    profileData.username = authStore.user.username || ''
    profileData.avatar = authStore.user.avatar || ''
  }
  if (!settingsStore.loaded) {
    await settingsStore.fetch()
  }
  localStreamingQuality.value = playerStore.streamingQuality
  localDownloadQuality.value = playerStore.downloadQuality
  localVolume.value = Math.round(playerStore.volume * 100)
  localRepeatMode.value = playerStore.repeatMode
})

async function updateProfile() {
  const valid = await profileFormRef.value?.validate().catch(() => false)
  if (!valid) return

  profileLoading.value = true
  try {
    const res = await api.put('/api/user/profile', {
      username: profileData.username,
      avatar: profileData.avatar || null
    })
    if (res.data.success) {
      authStore.user = res.data.data
      ElMessage.success('修改成功')
    } else {
      ElMessage.error(res.data.message || '修改失败')
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '修改失败')
  } finally {
    profileLoading.value = false
  }
}

async function changePassword() {
  const valid = await passwordFormRef.value?.validate().catch(() => false)
  if (!valid) return

  passwordLoading.value = true
  try {
    const res = await api.put('/api/user/password', {
      old_password: passwordForm.old_password,
      new_password: passwordForm.new_password
    })
    if (res.data.success) {
      ElMessage.success('密码修改成功')
      passwordForm.old_password = ''
      passwordForm.new_password = ''
      passwordForm.confirm_password = ''
    } else {
      ElMessage.error(res.data.message || '修改失败')
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '修改失败')
  } finally {
    passwordLoading.value = false
  }
}
</script>

<style lang="scss" scoped>
.profile-page {
  max-width: 640px;
  margin: 0 auto;
  padding: 24px;

  @media (max-width: 640px) {
    padding: 16px 12px;
  }
}

.profile-card {
  background: var(--card-bg);
  backdrop-filter: var(--card-backdrop);
  border: var(--border-width) solid var(--card-border);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
  padding: 32px;
  transition: all 0.4s;

  @media (max-width: 640px) {
    padding: 20px 16px;
  }
}

.profile-header {
  display: flex;
  align-items: center;
  gap: 24px;
  margin-bottom: 32px;

  @media (max-width: 640px) {
    gap: 16px;
    margin-bottom: 24px;

    :deep(.el-avatar) {
      --el-avatar-size: 60px !important;
    }
  }

  :deep(.el-avatar) {
    border: var(--border-width) solid var(--card-border);
    box-shadow: var(--shadow-sm);
  }

  .profile-info {
    h2 {
      color: var(--text-primary);
      font-size: 24px;
      font-weight: var(--title-weight);
      margin: 0 0 8px;
    }

    p {
      color: var(--text-muted);
      font-size: 14px;
      margin: 0 0 8px;
    }

    .provider {
      margin-bottom: 0;
    }
  }
}

.save-btn {
  background: var(--accent-btn-bg);
  border: var(--border-width) solid var(--btn-border);
  color: var(--accent-btn-text);
  font-weight: var(--title-weight);
  box-shadow: var(--btn-shadow);
  transition: all 0.2s;

  &:hover {
    opacity: 0.9;
    box-shadow: var(--btn-hover-shadow);
    transform: var(--btn-hover-transform);
  }
}

.settings-section {
  padding: 8px 0;
}

.settings-group {
  .settings-group-title {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--text-primary);
    font-size: 16px;
    font-weight: var(--title-weight);
    margin-bottom: 6px;
  }

  .settings-group-desc {
    color: var(--text-faint);
    font-size: 13px;
    margin: 0 0 14px;
  }
}

.quality-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
  gap: 10px;

  @media (max-width: 640px) {
    grid-template-columns: 1fr;
    gap: 8px;
  }
}

.quality-card {
  position: relative;
  padding: 14px 16px;
  border-radius: var(--radius-sm);
  border: var(--border-width) solid var(--border-color);
  background: var(--bg-elevated);
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    border-color: var(--accent);
    background: var(--bg-elevated-hover);
  }

  &--active {
    border-color: var(--accent);
    box-shadow: var(--shadow-sm);

    .quality-card-name {
      color: var(--accent);
    }
  }

  .quality-card-name {
    color: var(--text-primary);
    font-size: 14px;
    font-weight: 500;
    margin-bottom: 4px;
  }

  .quality-card-desc {
    color: var(--text-faint);
    font-size: 12px;
    line-height: 1.4;
  }

  .quality-card-check {
    position: absolute;
    top: 10px;
    right: 10px;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: var(--accent);
    display: flex;
    align-items: center;
    justify-content: center;
    color: #fff;
  }
}

[data-theme="day"] .quality-card {
  box-shadow: var(--shadow-sm);

  &:hover {
    box-shadow: var(--shadow);
  }

  &--active {
    box-shadow: var(--shadow);
  }
}

[data-theme="night"] .quality-card--active {
  background: rgba(var(--accent-rgb), 0.08);
}

.volume-setting {
  display: flex;
  align-items: center;
  gap: 16px;

  .el-slider {
    flex: 1;
    max-width: 300px;

    @media (max-width: 640px) {
      max-width: none;
    }
  }

  .volume-label {
    color: var(--text-secondary);
    font-size: 14px;
    font-variant-numeric: tabular-nums;
    min-width: 40px;
    text-align: right;
  }
}

.repeat-radio-group {
  :deep(.el-radio-button__inner) {
    background: var(--bg-elevated);
    border-color: var(--border-color);
    color: var(--text-secondary);

    &:hover {
      color: var(--text-primary);
    }
  }

  :deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
    background: var(--accent);
    border-color: var(--accent);
    color: #fff;
    box-shadow: -1px 0 0 0 var(--accent);
  }
}

.settings-saved-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin-top: 20px;
  padding: 10px;
  border-radius: var(--radius-sm);
  background: var(--success-bg);
  color: var(--success-color);
  border: var(--border-width) solid transparent;
  font-size: 13px;
  animation: fadeIn 0.3s ease;
}

[data-theme="day"] .settings-saved-hint {
  border-color: var(--success-color);
  box-shadow: 2px 2px 0 var(--success-color);
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(4px); }
  to { opacity: 1; transform: translateY(0); }
}

:deep(.el-divider) {
  border-color: var(--divider-color);
  margin: 20px 0;
}

:deep(.el-tabs__item) {
  color: var(--text-muted);
  font-weight: var(--el-font-weight);

  &.is-active {
    color: var(--accent);
  }
}

:deep(.el-tabs__nav-wrap::after) {
  background-color: var(--divider-color);
}

:deep(.el-tabs__active-bar) {
  background-color: var(--accent);
}

:deep(.el-form-item__label) {
  color: var(--text-secondary);
  font-weight: var(--el-font-weight);
}

:deep(.el-input__wrapper) {
  background: var(--bg-input);
  border: var(--border-width) solid var(--border-color);
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

:deep(.el-slider__runway) {
  background: var(--progress-bg);
}

:deep(.el-slider__bar) {
  background: var(--accent);
}

:deep(.el-slider__button) {
  border-color: var(--accent);
}
</style>
