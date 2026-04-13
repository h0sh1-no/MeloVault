<template>
  <div class="site-settings-page">
    <!-- Site URL card -->
    <div class="settings-card">
      <h3 class="card-title">站点域名配置</h3>
      <p class="card-desc">配置站点实际访问地址，用于 OAuth 回调跳转和 CORS 跨域策略，防止域名不匹配导致登录失效</p>

      <div class="oauth-form">
        <div class="form-group">
          <label class="form-label">站点 URL</label>
          <el-input
            v-model="siteUrlForm.site_url"
            placeholder="例如 https://music.example.com"
            clearable
          />
          <div class="form-hint">
            填入前端实际访问的完整域名（含协议），留空则使用环境变量 FRONTEND_URL
          </div>
        </div>
        <div class="form-actions">
          <el-button
            :loading="savingSiteUrl"
            class="save-oauth-btn"
            @click="saveSiteUrl"
          >
            保存域名
          </el-button>
          <el-tag
            v-if="siteUrlForm.site_url"
            type="success"
            size="small"
            effect="plain"
          >已配置</el-tag>
          <el-tag
            v-else
            type="info"
            size="small"
            effect="plain"
          >使用环境变量</el-tag>
        </div>
      </div>
    </div>

    <!-- Auth config card -->
    <div class="settings-card">
      <h3 class="card-title">登录注册配置</h3>
      <p class="card-desc">控制用户注册和登录方式的可用性</p>

      <div class="feature-list">
        <!-- Master registration switch -->
        <div class="feature-item">
          <div class="feature-header">
            <div class="feature-icon"><UserPlus :size="20" /></div>
            <div class="feature-info">
              <div class="feature-name">开放注册</div>
              <div class="feature-desc">关闭后将禁止所有新用户注册</div>
            </div>
            <el-switch
              v-model="form.allow_register"
              @change="save"
              :loading="saving"
            />
          </div>
        </div>

        <!-- Email registration -->
        <div class="feature-item" :class="{ 'is-disabled': !form.allow_register }">
          <div class="feature-header">
            <div class="feature-icon"><Message :size="20" /></div>
            <div class="feature-info">
              <div class="feature-name">邮箱注册</div>
              <div class="feature-desc">允许用户通过邮箱 + 密码的方式注册账号</div>
            </div>
            <el-switch
              v-model="form.allow_email_register"
              @change="save"
              :loading="saving"
              :disabled="!form.allow_register"
            />
          </div>
        </div>

        <!-- LinuxDO registration -->
        <div class="feature-item" :class="{ 'is-disabled': !form.allow_register }">
          <div class="feature-header">
            <div class="feature-icon"><Globe :size="20" /></div>
            <div class="feature-info">
              <div class="feature-name">LinuxDO 注册</div>
              <div class="feature-desc">允许通过 LinuxDO OAuth 自动创建新账号</div>
            </div>
            <el-switch
              v-model="form.allow_linuxdo_register"
              @change="save"
              :loading="saving"
              :disabled="!form.allow_register"
            />
          </div>
        </div>

        <el-divider />

        <!-- Email login -->
        <div class="feature-item">
          <div class="feature-header">
            <div class="feature-icon"><Lock :size="20" /></div>
            <div class="feature-info">
              <div class="feature-name">邮箱登录</div>
              <div class="feature-desc">允许已注册用户使用邮箱 + 密码登录</div>
            </div>
            <el-switch
              v-model="form.allow_email_login"
              @change="handleLoginToggle('email', $event)"
              :loading="saving"
            />
          </div>
        </div>

        <!-- LinuxDO login -->
        <div class="feature-item">
          <div class="feature-header">
            <div class="feature-icon"><Globe :size="20" /></div>
            <div class="feature-info">
              <div class="feature-name">LinuxDO 登录</div>
              <div class="feature-desc">允许用户通过 LinuxDO OAuth 登录</div>
            </div>
            <el-switch
              v-model="form.allow_linuxdo_login"
              @change="handleLoginToggle('linuxdo', $event)"
              :loading="saving"
            />
          </div>
        </div>
      </div>

      <div class="settings-tip" v-if="!form.allow_email_login && !form.allow_linuxdo_login">
        <el-alert type="warning" :closable="false" show-icon>
          所有登录方式均已关闭，普通用户将无法登录。管理员仍可通过已有会话访问。
        </el-alert>
      </div>
    </div>

    <!-- LinuxDO OAuth card -->
    <div class="settings-card">
      <h3 class="card-title">LinuxDO 登录配置</h3>
      <p class="card-desc">配置 LinuxDO OAuth 凭据以启用 LinuxDO 登录功能，无需修改环境变量</p>

      <div class="oauth-form">
        <div class="form-group">
          <label class="form-label">Client ID</label>
          <el-input
            v-model="oauthForm.linuxdo_client_id"
            placeholder="填入 LinuxDO OAuth Client ID"
            clearable
          />
        </div>
        <div class="form-group">
          <label class="form-label">Client Secret</label>
          <el-input
            v-model="oauthForm.linuxdo_client_secret"
            :type="showSecret ? 'text' : 'password'"
            placeholder="填入 LinuxDO OAuth Client Secret"
            clearable
          >
            <template #suffix>
              <el-icon class="secret-toggle" @click="showSecret = !showSecret">
                <View v-if="!showSecret" />
                <Hide v-else />
              </el-icon>
            </template>
          </el-input>
          <div class="form-hint" v-if="hasExistingSecret && !oauthForm.linuxdo_client_secret">
            已配置密钥，留空则保持不变
          </div>
        </div>
        <div class="form-group">
          <label class="form-label">Redirect URI</label>
          <el-input
            v-model="oauthForm.linuxdo_redirect_uri"
            placeholder="例如 https://yourdomain.com/api/auth/linuxdo/callback"
            clearable
          />
        </div>
        <div class="form-actions">
          <el-button
            :loading="savingOAuth"
            class="save-oauth-btn"
            @click="saveOAuth"
          >
            保存配置
          </el-button>
          <el-tag
            v-if="linuxdoStatus !== null"
            :type="linuxdoStatus ? 'success' : 'info'"
            size="small"
            effect="plain"
          >
            {{ linuxdoStatus ? '已配置' : '未配置' }}
          </el-tag>
        </div>
      </div>
    </div>

    <!-- SMTP Email config card -->
    <div class="settings-card">
      <h3 class="card-title">邮箱服务配置</h3>
      <p class="card-desc">配置 SMTP 邮件服务器，用于发送验证码邮件。无需修改环境变量，保存后即时生效</p>

      <div class="oauth-form">
        <div class="form-group">
          <label class="form-label">SMTP 服务器</label>
          <el-input
            v-model="smtpForm.smtp_host"
            placeholder="例如 smtp.gmail.com"
            clearable
          />
        </div>
        <div class="form-group">
          <label class="form-label">端口</label>
          <el-input-number
            v-model="smtpForm.smtp_port"
            :min="1"
            :max="65535"
            controls-position="right"
            style="width: 100%"
          />
        </div>
        <div class="form-group">
          <label class="form-label">账号</label>
          <el-input
            v-model="smtpForm.smtp_user"
            placeholder="SMTP 登录用户名（通常为邮箱地址）"
            clearable
          />
        </div>
        <div class="form-group">
          <label class="form-label">密码</label>
          <el-input
            v-model="smtpForm.smtp_password"
            :type="showSmtpPassword ? 'text' : 'password'"
            placeholder="SMTP 登录密码或应用专用密码"
            clearable
          >
            <template #suffix>
              <el-icon class="secret-toggle" @click="showSmtpPassword = !showSmtpPassword">
                <View v-if="!showSmtpPassword" />
                <Hide v-else />
              </el-icon>
            </template>
          </el-input>
          <div class="form-hint" v-if="hasExistingSmtpPassword && !smtpForm.smtp_password">
            已配置密码，留空则保持不变
          </div>
        </div>
        <div class="form-group">
          <label class="form-label">发件人名称</label>
          <el-input
            v-model="smtpForm.smtp_from"
            placeholder="例如 MeloVault（可选，留空则使用账号地址）"
            clearable
          />
        </div>
        <div class="form-actions">
          <el-button
            :loading="savingSmtp"
            class="save-oauth-btn"
            @click="saveSmtp"
          >
            保存配置
          </el-button>
          <el-button
            :loading="sendingTestEmail"
            @click="sendTestEmail"
          >
            发送测试邮件
          </el-button>
          <el-tag
            v-if="smtpStatus !== null"
            :type="smtpStatus ? 'success' : 'info'"
            size="small"
            effect="plain"
          >
            {{ smtpStatus ? '已配置' : '未配置' }}
          </el-tag>
        </div>
      </div>
    </div>

    <!-- Feature flags card -->
    <div class="settings-card">
      <h3 class="card-title">功能开关</h3>
      <p class="card-desc">控制前台用户可使用的解析功能</p>

      <div class="feature-list">
        <div class="feature-item">
          <div class="feature-header">
            <div class="feature-icon"><List :size="20" /></div>
            <div class="feature-info">
              <div class="feature-name">歌单解析</div>
              <div class="feature-desc">允许通过网易云歌单 ID 或链接解析歌单内容</div>
            </div>
            <el-switch
              v-model="form.playlist_parse_enabled"
              @change="save"
              :loading="saving"
            />
          </div>
          <div class="feature-sub" v-if="form.playlist_parse_enabled">
            <el-checkbox
              v-model="form.playlist_parse_admin_only"
              @change="save"
              :disabled="saving"
            >仅管理员可用</el-checkbox>
          </div>
        </div>

        <div class="feature-item">
          <div class="feature-header">
            <div class="feature-icon"><DiscIcon :size="20" /></div>
            <div class="feature-info">
              <div class="feature-name">专辑解析</div>
              <div class="feature-desc">允许通过网易云专辑 ID 或链接解析专辑内容</div>
            </div>
            <el-switch
              v-model="form.album_parse_enabled"
              @change="save"
              :loading="saving"
            />
          </div>
          <div class="feature-sub" v-if="form.album_parse_enabled">
            <el-checkbox
              v-model="form.album_parse_admin_only"
              @change="save"
              :disabled="saving"
            >仅管理员可用</el-checkbox>
          </div>
        </div>
      </div>
    </div>

    <!-- Netease Real IP card -->
    <div class="settings-card">
      <h3 class="card-title">网易云 API 区域配置</h3>
      <p class="card-desc">服务器部署在海外时，通过伪造 X-Real-IP 头绕过网易云区域版权限制。填入一个中国大陆 IP 地址即可</p>

      <div class="oauth-form">
        <div class="form-group">
          <label class="form-label">Real IP 地址</label>
          <el-input
            v-model="neteaseIpForm.netease_real_ip"
            placeholder="例如 116.25.146.177"
            clearable
          />
          <div class="form-hint">
            填入中国大陆 IP 以解锁版权受限歌曲，留空则使用环境变量 NETEASE_REAL_IP
          </div>
        </div>
        <div class="form-actions">
          <el-button
            :loading="savingNeteaseIp"
            class="save-oauth-btn"
            @click="saveNeteaseIp"
          >
            保存配置
          </el-button>
          <el-tag
            v-if="neteaseIpForm.netease_real_ip"
            type="success"
            size="small"
            effect="plain"
          >已配置</el-tag>
          <el-tag
            v-else
            type="info"
            size="small"
            effect="plain"
          >使用环境变量</el-tag>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, computed, onMounted } from 'vue'
import { List, Message, Lock, View, Hide } from '@element-plus/icons-vue'
import { Disc3 as DiscIcon, UserPlus, Globe, Mail } from 'lucide-vue-next'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useSiteSettingsStore } from '@/stores/siteSettings'
import api from '@/api'

const siteStore = useSiteSettingsStore()
const saving = ref(false)
const savingOAuth = ref(false)
const savingSiteUrl = ref(false)
const savingNeteaseIp = ref(false)
const savingSmtp = ref(false)
const sendingTestEmail = ref(false)
const showSecret = ref(false)
const showSmtpPassword = ref(false)
const linuxdoStatus = ref(null)
const smtpStatus = ref(null)
const hasExistingSecret = ref(false)
const hasExistingSmtpPassword = ref(false)

const form = reactive({
  playlist_parse_enabled: true,
  playlist_parse_admin_only: false,
  album_parse_enabled: true,
  album_parse_admin_only: false,
  allow_register: true,
  allow_email_register: true,
  allow_linuxdo_register: true,
  allow_email_login: true,
  allow_linuxdo_login: true,
})

const oauthForm = reactive({
  linuxdo_client_id: '',
  linuxdo_client_secret: '',
  linuxdo_redirect_uri: '',
})

const siteUrlForm = reactive({
  site_url: '',
})

const neteaseIpForm = reactive({
  netease_real_ip: '',
})

const smtpForm = reactive({
  smtp_host: '',
  smtp_port: 587,
  smtp_user: '',
  smtp_password: '',
  smtp_from: '',
})

async function fetchAdminSettings() {
  try {
    const res = await api.get('/api/admin/site-settings')
    if (res.data.success && res.data.data) {
      const d = res.data.data
      Object.assign(form, {
        playlist_parse_enabled: d.playlist_parse_enabled,
        playlist_parse_admin_only: d.playlist_parse_admin_only,
        album_parse_enabled: d.album_parse_enabled,
        album_parse_admin_only: d.album_parse_admin_only,
        allow_register: d.allow_register,
        allow_email_register: d.allow_email_register,
        allow_linuxdo_register: d.allow_linuxdo_register,
        allow_email_login: d.allow_email_login,
        allow_linuxdo_login: d.allow_linuxdo_login,
      })
      oauthForm.linuxdo_client_id = d.linuxdo_client_id || ''
      oauthForm.linuxdo_redirect_uri = d.linuxdo_redirect_uri || ''
      siteUrlForm.site_url = d.site_url || ''
      neteaseIpForm.netease_real_ip = d.netease_real_ip || ''
      smtpForm.smtp_host = d.smtp_host || ''
      smtpForm.smtp_port = d.smtp_port || 587
      smtpForm.smtp_user = d.smtp_user || ''
      smtpForm.smtp_from = d.smtp_from || ''
      hasExistingSmtpPassword.value = !!d.smtp_password
      smtpStatus.value = !!(d.smtp_host && d.smtp_user && d.smtp_password)
      hasExistingSecret.value = !!d.linuxdo_client_secret
      linuxdoStatus.value = !!(d.linuxdo_client_id && d.linuxdo_client_secret && d.linuxdo_redirect_uri)
    }
  } catch {
    await siteStore.fetch()
    Object.assign(form, siteStore.features)
  }
}

onMounted(fetchAdminSettings)

function handleLoginToggle(type, val) {
  const otherKey = type === 'email' ? 'allow_linuxdo_login' : 'allow_email_login'
  if (!val && !form[otherKey]) {
    ElMessage.warning('至少需要保留一种登录方式')
    if (type === 'email') form.allow_email_login = true
    else form.allow_linuxdo_login = true
    return
  }
  save()
}

async function save() {
  saving.value = true
  try {
    await siteStore.update({ ...form })
    ElMessage.success('已保存')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function saveOAuth() {
  savingOAuth.value = true
  try {
    const payload = {
      linuxdo_client_id: oauthForm.linuxdo_client_id,
      linuxdo_redirect_uri: oauthForm.linuxdo_redirect_uri,
    }
    if (oauthForm.linuxdo_client_secret) {
      payload.linuxdo_client_secret = oauthForm.linuxdo_client_secret
    }
    const res = await siteStore.update(payload)
    hasExistingSecret.value = !!(oauthForm.linuxdo_client_secret || hasExistingSecret.value)
    oauthForm.linuxdo_client_secret = ''
    linuxdoStatus.value = !!(oauthForm.linuxdo_client_id && hasExistingSecret.value && oauthForm.linuxdo_redirect_uri)
    ElMessage.success('LinuxDO 配置已保存')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    savingOAuth.value = false
  }
}

async function saveSiteUrl() {
  savingSiteUrl.value = true
  try {
    await siteStore.update({ site_url: siteUrlForm.site_url.replace(/\/+$/, '') })
    ElMessage.success('站点域名已保存')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    savingSiteUrl.value = false
  }
}

async function saveNeteaseIp() {
  savingNeteaseIp.value = true
  try {
    await siteStore.update({ netease_real_ip: neteaseIpForm.netease_real_ip.trim() })
    ElMessage.success('网易云 Real IP 已保存，即时生效')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    savingNeteaseIp.value = false
  }
}

async function saveSmtp() {
  savingSmtp.value = true
  try {
    const payload = {
      smtp_host: smtpForm.smtp_host,
      smtp_port: smtpForm.smtp_port,
      smtp_user: smtpForm.smtp_user,
      smtp_from: smtpForm.smtp_from,
    }
    if (smtpForm.smtp_password) {
      payload.smtp_password = smtpForm.smtp_password
    }
    await siteStore.update(payload)
    hasExistingSmtpPassword.value = !!(smtpForm.smtp_password || hasExistingSmtpPassword.value)
    smtpForm.smtp_password = ''
    smtpStatus.value = !!(smtpForm.smtp_host && smtpForm.smtp_user && hasExistingSmtpPassword.value)
    ElMessage.success('SMTP 配置已保存，即时生效')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    savingSmtp.value = false
  }
}

async function sendTestEmail() {
  try {
    const { value: email } = await ElMessageBox.prompt('请输入接收测试邮件的邮箱地址', '发送测试邮件', {
      confirmButtonText: '发送',
      cancelButtonText: '取消',
      inputPlaceholder: 'test@example.com',
      inputPattern: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
      inputErrorMessage: '请输入有效的邮箱地址',
    })
    sendingTestEmail.value = true
    const res = await api.post('/api/admin/site-settings/test-email', { email })
    if (res.data.success) {
      ElMessage.success('测试邮件已发送至 ' + email)
    } else {
      ElMessage.error(res.data.message || '发送失败')
    }
  } catch (e) {
    if (e !== 'cancel' && e?.toString() !== 'cancel') {
      ElMessage.error(e?.response?.data?.message || '发送测试邮件失败')
    }
  } finally {
    sendingTestEmail.value = false
  }
}
</script>

<style lang="scss" scoped>
.site-settings-page {
  max-width: 640px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.settings-card {
  background: var(--card-bg);
  border: var(--border-width) solid var(--card-border);
  border-radius: var(--radius);
  padding: 28px;
  box-shadow: var(--shadow-sm);
}

.card-title {
  font-size: 18px;
  font-weight: var(--title-weight);
  color: var(--text-primary);
  margin: 0 0 4px;
}

.card-desc {
  font-size: 13px;
  color: var(--text-muted);
  margin: 0 0 24px;
}

.feature-list {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.feature-item {
  padding: 16px;
  background: var(--bg-elevated);
  border: var(--border-width) solid var(--border-color);
  border-radius: var(--radius-sm);
  transition: all 0.2s;
}

.feature-header {
  display: flex;
  align-items: center;
  gap: 14px;
}

.feature-icon {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-sm);
  background: var(--tag-bg);
  color: var(--accent);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.feature-info {
  flex: 1;
  min-width: 0;
}

.feature-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.feature-desc {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 2px;
}

.feature-sub {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-color);

  :deep(.el-checkbox__label) {
    color: var(--text-secondary);
    font-size: 13px;
  }

  :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
    background-color: var(--accent);
    border-color: var(--accent);
  }
}

.oauth-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
}

.form-hint {
  font-size: 12px;
  color: var(--text-muted);
}

.form-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 4px;
}

.save-oauth-btn {
  background: var(--accent-btn-bg);
  border: var(--border-width) solid var(--btn-border);
  border-radius: var(--radius-sm);
  color: var(--accent-btn-text);
  font-weight: 600;
  box-shadow: var(--btn-shadow);
  transition: all 0.2s;

  &:hover {
    opacity: 0.92;
    box-shadow: var(--btn-hover-shadow);
  }
}

.secret-toggle {
  cursor: pointer;
  color: var(--text-muted);
  transition: color 0.2s;

  &:hover {
    color: var(--text-primary);
  }
}

:deep(.oauth-form .el-input__wrapper) {
  background: var(--bg-input, var(--bg-elevated));
  border: var(--border-width) solid var(--border-color);
  border-radius: var(--radius-sm);

  &:hover, &.is-focus {
    border-color: var(--accent);
  }

  .el-input__inner {
    color: var(--text-primary);
  }
}

.feature-item.is-disabled {
  opacity: 0.5;
  pointer-events: none;
}

.settings-tip {
  margin-top: 16px;
}

:deep(.el-divider) {
  margin: 8px 0;
  border-color: var(--border-color);
}

:deep(.el-switch.is-checked .el-switch__core) {
  background-color: var(--accent);
  border-color: var(--accent);
}

:deep(.el-alert) {
  background: var(--bg-elevated);
  border: var(--border-width) solid var(--border-color);
}
</style>
