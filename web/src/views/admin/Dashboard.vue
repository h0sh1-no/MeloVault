<template>
  <div class="dashboard">
    <div v-if="loading" class="loading-state">
      <el-skeleton :rows="6" animated />
    </div>

    <template v-else>
      <!-- Stats cards -->
      <div class="stats-grid">
        <div class="stat-card" v-for="card in statCards" :key="card.label">
          <div class="stat-icon" :class="card.color">
            <component :is="card.icon" :size="22" />
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ card.value ?? '—' }}</div>
            <div class="stat-label">{{ card.label }}</div>
          </div>
        </div>
      </div>

      <!-- Trend chart -->
      <div class="section" v-if="trends.length">
        <div class="section-header">
          <h3 class="section-title">趋势概览</h3>
          <el-radio-group v-model="trendDays" size="small" @change="loadTrends">
            <el-radio-button :value="7">7天</el-radio-button>
            <el-radio-button :value="14">14天</el-radio-button>
            <el-radio-button :value="30">30天</el-radio-button>
          </el-radio-group>
        </div>
        <div class="chart-container">
          <v-chart :option="trendChartOption" autoresize style="height: 320px" />
        </div>
      </div>

      <!-- Bottom row -->
      <div class="bottom-grid">
        <!-- Recent activity -->
        <div class="section">
          <h3 class="section-title">最近活动</h3>
          <div class="activity-list" v-if="recentActivity.length">
            <div class="activity-item" v-for="item in recentActivity" :key="item.id">
              <div class="activity-dot" :class="actionColor(item.action)" />
              <div class="activity-content">
                <span class="activity-user">{{ item.username || '匿名' }}</span>
                <span class="activity-action">{{ actionLabel(item.action) }}</span>
                <span class="activity-ip" v-if="item.ip">{{ item.ip }}</span>
              </div>
              <div class="activity-time">{{ timeAgo(item.created_at) }}</div>
            </div>
          </div>
          <div class="empty-state" v-else>暂无活动记录</div>
        </div>

        <!-- Quick actions -->
        <div class="section">
          <h3 class="section-title">快捷操作</h3>
          <div class="quick-actions">
            <router-link to="/admin/users" class="action-card">
              <Users :size="24" />
              <span>管理用户</span>
            </router-link>
            <router-link to="/admin/analytics" class="action-card">
              <BarChart3 :size="24" />
              <span>用户分析</span>
            </router-link>
            <router-link to="/admin/activity" class="action-card">
              <Activity :size="24" />
              <span>活动日志</span>
            </router-link>
            <router-link to="/admin/downloads" class="action-card">
              <Download :size="24" />
              <span>查看下载</span>
            </router-link>
            <router-link to="/admin/netease" class="action-card">
              <Music2 :size="24" />
              <span>配置网易云</span>
            </router-link>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { Users, Download, Music2, UserPlus, Activity, Wifi, PlayCircle, BarChart3 } from 'lucide-vue-next'
import { ElMessage } from 'element-plus'
import { useAdminStore } from '@/stores/admin'
import { useThemeStore } from '@/stores/theme'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent])

const adminStore = useAdminStore()
const loading = ref(true)
const stats = ref(null)
const trends = ref([])
const recentActivity = ref([])
const trendDays = ref(7)

const statCards = computed(() => [
  { label: '总用户数', value: stats.value?.total_users, icon: Users, color: 'purple' },
  { label: '今日新增', value: stats.value?.today_new_users, icon: UserPlus, color: 'blue' },
  { label: '今日播放', value: stats.value?.today_plays, icon: PlayCircle, color: 'green' },
  { label: '今日下载', value: stats.value?.today_downloads, icon: Download, color: 'orange' },
  { label: '今日活跃', value: stats.value?.today_active_users, icon: Activity, color: 'cyan' },
  { label: '当前在线', value: stats.value?.online_users, icon: Wifi, color: 'emerald' },
  { label: '总收藏数', value: stats.value?.total_favorites, icon: Music2, color: 'pink' },
  { label: '总播放数', value: stats.value?.total_plays, icon: PlayCircle, color: 'amber' }
])

const themeStore = useThemeStore()

const chartColors = computed(() => themeStore.isDayMode ? {
  tooltipBg: '#ffffff', tooltipBorder: 'rgba(0,0,0,0.1)', tooltipText: '#333',
  legendText: 'rgba(0,0,0,0.5)', axisLine: 'rgba(0,0,0,0.1)',
  axisLabel: 'rgba(0,0,0,0.45)', splitLine: 'rgba(0,0,0,0.06)'
} : {
  tooltipBg: '#1e1e32', tooltipBorder: 'rgba(255,255,255,0.1)', tooltipText: '#e2e8f0',
  legendText: 'rgba(255,255,255,0.5)', axisLine: 'rgba(255,255,255,0.1)',
  axisLabel: 'rgba(255,255,255,0.45)', splitLine: 'rgba(255,255,255,0.05)'
})

const trendChartOption = computed(() => ({
  tooltip: { trigger: 'axis', backgroundColor: chartColors.value.tooltipBg, borderColor: chartColors.value.tooltipBorder, textStyle: { color: chartColors.value.tooltipText } },
  legend: { data: ['播放', '下载', '新用户', '登录'], textStyle: { color: chartColors.value.legendText }, bottom: 0 },
  grid: { top: 16, right: 16, bottom: 40, left: 48 },
  xAxis: {
    type: 'category',
    data: trends.value.map(t => t.date?.slice(5)),
    axisLine: { lineStyle: { color: chartColors.value.axisLine } },
    axisLabel: { color: chartColors.value.axisLabel }
  },
  yAxis: {
    type: 'value',
    splitLine: { lineStyle: { color: chartColors.value.splitLine } },
    axisLabel: { color: chartColors.value.axisLabel }
  },
  series: [
    { name: '播放', type: 'line', smooth: true, data: trends.value.map(t => t.play_count), itemStyle: { color: '#a78bfa' }, areaStyle: { color: 'rgba(167,139,250,0.1)' } },
    { name: '下载', type: 'line', smooth: true, data: trends.value.map(t => t.down_count), itemStyle: { color: '#fb923c' } },
    { name: '新用户', type: 'line', smooth: true, data: trends.value.map(t => t.user_count), itemStyle: { color: '#60a5fa' } },
    { name: '登录', type: 'line', smooth: true, data: trends.value.map(t => t.login_count), itemStyle: { color: '#34d399' } }
  ]
}))

function actionLabel(action) {
  const map = { login: '登录', play: '播放歌曲', download: '下载歌曲', search: '搜索', favorite: '收藏', browse: '浏览' }
  return map[action] || action
}

function actionColor(action) {
  const map = { login: 'blue', play: 'purple', download: 'orange', search: 'green', favorite: 'pink', browse: 'cyan' }
  return map[action] || 'gray'
}

function timeAgo(dateStr) {
  if (!dateStr) return ''
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins}分钟前`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}小时前`
  return `${Math.floor(hours / 24)}天前`
}

async function loadTrends() {
  try {
    trends.value = await adminStore.getTrends(trendDays.value) || []
  } catch {}
}

onMounted(async () => {
  try {
    const [overview, trendsData, activityData] = await Promise.all([
      adminStore.getAnalyticsOverview(),
      adminStore.getTrends(trendDays.value),
      adminStore.getActivityLogs(1, 10)
    ])
    stats.value = overview
    trends.value = trendsData || []
    recentActivity.value = activityData?.list || []
  } catch {
    ElMessage.error('获取统计数据失败')
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.dashboard { max-width: 1200px; }
.loading-state { padding: 20px 0; }

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 14px;
  margin-bottom: 28px;
}

.stat-card {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 14px;
  padding: 18px;
  display: flex;
  align-items: center;
  gap: 14px;
  transition: border-color 0.2s;
}
.stat-card:hover { border-color: rgba(255, 255, 255, 0.15); }

.stat-icon {
  width: 44px; height: 44px; border-radius: 11px;
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
}
.stat-icon.purple  { background: rgba(124, 58, 237, 0.2); color: #a78bfa; }
.stat-icon.blue    { background: rgba(59, 130, 246, 0.2); color: #60a5fa; }
.stat-icon.green   { background: rgba(16, 185, 129, 0.2); color: #34d399; }
.stat-icon.orange  { background: rgba(249, 115, 22, 0.2); color: #fb923c; }
.stat-icon.cyan    { background: rgba(6, 182, 212, 0.2); color: #22d3ee; }
.stat-icon.emerald { background: rgba(16, 185, 129, 0.2); color: #6ee7b7; }
.stat-icon.pink    { background: rgba(236, 72, 153, 0.2); color: #f472b6; }
.stat-icon.amber   { background: rgba(245, 158, 11, 0.2); color: #fbbf24; }

.stat-value { font-size: 26px; font-weight: 700; color: #fff; line-height: 1.1; }
.stat-label { font-size: 12px; color: rgba(255, 255, 255, 0.45); margin-top: 3px; }

.section { margin-bottom: 28px; }
.section-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.section-title {
  font-size: 12px; font-weight: 600; color: rgba(255, 255, 255, 0.6);
  margin: 0 0 16px; text-transform: uppercase; letter-spacing: 0.05em;
}
.section-header .section-title { margin-bottom: 0; }

.chart-container {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 14px;
  padding: 16px;
}

.bottom-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.activity-list { display: flex; flex-direction: column; gap: 2px; }
.activity-item {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 12px; border-radius: 8px;
  transition: background 0.15s;
}
.activity-item:hover { background: rgba(255, 255, 255, 0.03); }

.activity-dot {
  width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0;
}
.activity-dot.blue   { background: #60a5fa; }
.activity-dot.purple { background: #a78bfa; }
.activity-dot.orange { background: #fb923c; }
.activity-dot.green  { background: #34d399; }
.activity-dot.pink   { background: #f472b6; }
.activity-dot.cyan   { background: #22d3ee; }
.activity-dot.gray   { background: rgba(255,255,255,0.3); }

.activity-content { flex: 1; font-size: 13px; display: flex; gap: 6px; align-items: center; flex-wrap: wrap; }
.activity-user { color: rgba(255, 255, 255, 0.85); font-weight: 500; }
.activity-action { color: rgba(255, 255, 255, 0.5); }
.activity-ip { color: rgba(255, 255, 255, 0.3); font-size: 12px; font-family: monospace; }
.activity-time { font-size: 12px; color: rgba(255, 255, 255, 0.3); white-space: nowrap; }

.empty-state {
  text-align: center; padding: 32px; color: rgba(255, 255, 255, 0.3); font-size: 14px;
}

.quick-actions { display: flex; gap: 10px; flex-wrap: wrap; }
.action-card {
  display: flex; align-items: center; gap: 8px;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.07);
  border-radius: 10px;
  color: rgba(255, 255, 255, 0.7);
  text-decoration: none; font-size: 13px; font-weight: 500;
  transition: all 0.2s;
}
.action-card:hover {
  border-color: #7c3aed; color: #a78bfa;
  background: rgba(124, 58, 237, 0.1);
}

@media (max-width: 768px) {
  .bottom-grid { grid-template-columns: 1fr; }
  .stats-grid { grid-template-columns: repeat(2, 1fr); }
}

/* Day theme overrides */
[data-theme="day"] .stat-card {
  background: var(--card-bg); border-color: var(--card-border); box-shadow: var(--shadow-sm);
}
[data-theme="day"] .stat-card:hover { border-color: var(--border-hover); }

[data-theme="day"] .stat-value { color: var(--text-primary); }
[data-theme="day"] .stat-label { color: var(--text-muted); }

[data-theme="day"] .section-title { color: var(--text-muted); }

[data-theme="day"] .chart-container {
  background: var(--card-bg); border-color: var(--card-border); box-shadow: var(--shadow-sm);
}

[data-theme="day"] .activity-item:hover { background: rgba(0,0,0,0.03); }
[data-theme="day"] .activity-dot.gray { background: rgba(0,0,0,0.2); }

[data-theme="day"] .activity-user { color: var(--text-primary); }
[data-theme="day"] .activity-action { color: var(--text-muted); }
[data-theme="day"] .activity-ip { color: var(--text-faint); }
[data-theme="day"] .activity-time { color: var(--text-faint); }

[data-theme="day"] .empty-state { color: var(--text-faint); }

[data-theme="day"] .action-card {
  background: var(--card-bg); border-color: var(--card-border);
  color: var(--text-secondary); box-shadow: var(--shadow-sm);
}
[data-theme="day"] .action-card:hover {
  border-color: var(--accent); color: var(--accent);
  background: rgba(230,57,70,0.05);
}
</style>
