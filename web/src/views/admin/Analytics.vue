<template>
  <div class="analytics-page">
    <div v-if="loading" class="loading-state">
      <el-skeleton :rows="8" animated />
    </div>

    <template v-else>
      <!-- Time range selector -->
      <div class="toolbar">
        <el-radio-group v-model="days" size="small" @change="loadData">
          <el-radio-button :value="7">近7天</el-radio-button>
          <el-radio-button :value="14">近14天</el-radio-button>
          <el-radio-button :value="30">近30天</el-radio-button>
          <el-radio-button :value="90">近90天</el-radio-button>
        </el-radio-group>
      </div>

      <div class="charts-grid">
        <!-- Province distribution -->
        <div class="chart-card full-width">
          <h3 class="chart-title">省份分布</h3>
          <div class="province-layout">
            <div class="chart-wrap">
              <v-chart :option="provinceBarOption" autoresize style="height: 400px" />
            </div>
            <div class="province-table-wrap">
              <el-table :data="provinceStats" class="province-table" max-height="400" size="small" stripe>
                <el-table-column type="index" label="#" width="40" />
                <el-table-column prop="province" label="省份" min-width="80" />
                <el-table-column prop="count" label="请求数" width="90" align="right" sortable />
                <el-table-column label="占比" width="90" align="right">
                  <template #default="{ row }">{{ ((row.count / totalRequests) * 100).toFixed(1) }}%</template>
                </el-table-column>
              </el-table>
            </div>
          </div>
        </div>

        <!-- Action distribution pie -->
        <div class="chart-card">
          <h3 class="chart-title">活动类型分布</h3>
          <v-chart :option="actionPieOption" autoresize style="height: 340px" />
        </div>

        <!-- Trends chart -->
        <div class="chart-card">
          <h3 class="chart-title">活动趋势</h3>
          <v-chart :option="trendOption" autoresize style="height: 340px" />
        </div>
      </div>

      <!-- Online users -->
      <div class="section">
        <h3 class="chart-title">在线用户 (15分钟内活跃)</h3>
        <el-table :data="onlineUsers" class="online-table" stripe size="small" v-if="onlineUsers.length">
          <el-table-column prop="username" label="用户" min-width="120">
            <template #default="{ row }">
              <div class="user-cell">
                <el-avatar :size="28" class="tiny-avatar">{{ row.username?.[0]?.toUpperCase() }}</el-avatar>
                <span>{{ row.username }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="last_ip" label="IP" width="140">
            <template #default="{ row }"><code class="ip-code">{{ row.last_ip }}</code></template>
          </el-table-column>
          <el-table-column prop="province" label="省份" width="100" />
          <el-table-column prop="action" label="最后操作" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="actionTagType(row.action)">{{ actionLabel(row.action) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="最后活跃" width="140">
            <template #default="{ row }">{{ timeAgo(row.last_seen) }}</template>
          </el-table-column>
        </el-table>
        <div class="empty-state" v-else>当前无活跃用户</div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, PieChart, LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { ElMessage } from 'element-plus'
import { useAdminStore } from '@/stores/admin'
import { useThemeStore } from '@/stores/theme'

use([CanvasRenderer, BarChart, PieChart, LineChart, GridComponent, TooltipComponent, LegendComponent])

const adminStore = useAdminStore()
const themeStore = useThemeStore()
const loading = ref(true)
const days = ref(30)
const provinceStats = ref([])
const onlineUsers = ref([])
const trends = ref([])
const activityStats = ref([])

const totalRequests = computed(() => provinceStats.value.reduce((s, p) => s + p.count, 0) || 1)

const chartColors = computed(() => themeStore.isDayMode ? {
  tooltipBg: '#ffffff', tooltipBorder: 'rgba(0,0,0,0.1)', tooltipText: '#333',
  legendText: 'rgba(0,0,0,0.5)', axisLine: 'rgba(0,0,0,0.1)',
  axisLabel: 'rgba(0,0,0,0.45)', splitLine: 'rgba(0,0,0,0.06)',
  pieLabel: 'rgba(0,0,0,0.65)'
} : {
  tooltipBg: '#1e1e32', tooltipBorder: 'rgba(255,255,255,0.1)', tooltipText: '#e2e8f0',
  legendText: 'rgba(255,255,255,0.5)', axisLine: 'rgba(255,255,255,0.1)',
  axisLabel: 'rgba(255,255,255,0.45)', splitLine: 'rgba(255,255,255,0.05)',
  pieLabel: 'rgba(255,255,255,0.65)'
})

const provinceBarOption = computed(() => {
  const top = provinceStats.value.slice(0, 15)
  return {
    tooltip: { trigger: 'axis', backgroundColor: chartColors.value.tooltipBg, borderColor: chartColors.value.tooltipBorder, textStyle: { color: chartColors.value.tooltipText } },
    grid: { top: 8, right: 16, bottom: 24, left: 80 },
    xAxis: {
      type: 'value',
      splitLine: { lineStyle: { color: chartColors.value.splitLine } },
      axisLabel: { color: chartColors.value.axisLabel }
    },
    yAxis: {
      type: 'category',
      data: top.map(p => p.province).reverse(),
      axisLine: { lineStyle: { color: chartColors.value.axisLine } },
      axisLabel: { color: chartColors.value.axisLabel }
    },
    series: [{
      type: 'bar',
      data: top.map(p => p.count).reverse(),
      itemStyle: {
        borderRadius: [0, 6, 6, 0],
        color: { type: 'linear', x: 0, y: 0, x2: 1, y2: 0, colorStops: [{ offset: 0, color: '#7c3aed' }, { offset: 1, color: '#a78bfa' }] }
      },
      barWidth: 18
    }]
  }
})

const actionPieOption = computed(() => {
  const colors = { play: '#a78bfa', download: '#fb923c', login: '#60a5fa', search: '#34d399', favorite: '#f472b6', browse: '#22d3ee' }
  const labels = { play: '播放', download: '下载', login: '登录', search: '搜索', favorite: '收藏', browse: '浏览' }
  return {
    tooltip: { trigger: 'item', backgroundColor: chartColors.value.tooltipBg, borderColor: chartColors.value.tooltipBorder, textStyle: { color: chartColors.value.tooltipText } },
    series: [{
      type: 'pie',
      radius: ['42%', '70%'],
      center: ['50%', '50%'],
      data: activityStats.value.map(a => ({
        name: labels[a.action] || a.action,
        value: a.count,
        itemStyle: { color: colors[a.action] || '#666' }
      })),
      label: { color: chartColors.value.pieLabel, fontSize: 12 },
      emphasis: { itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0,0,0,0.3)' } }
    }]
  }
})

const trendOption = computed(() => ({
  tooltip: { trigger: 'axis', backgroundColor: chartColors.value.tooltipBg, borderColor: chartColors.value.tooltipBorder, textStyle: { color: chartColors.value.tooltipText } },
  legend: { data: ['播放', '下载', '登录'], textStyle: { color: chartColors.value.legendText }, bottom: 0 },
  grid: { top: 12, right: 12, bottom: 36, left: 40 },
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
    { name: '播放', type: 'line', smooth: true, data: trends.value.map(t => t.play_count), itemStyle: { color: '#a78bfa' } },
    { name: '下载', type: 'line', smooth: true, data: trends.value.map(t => t.down_count), itemStyle: { color: '#fb923c' } },
    { name: '登录', type: 'line', smooth: true, data: trends.value.map(t => t.login_count), itemStyle: { color: '#60a5fa' } }
  ]
}))

function actionLabel(a) {
  return { login: '登录', play: '播放', download: '下载', search: '搜索', favorite: '收藏', browse: '浏览' }[a] || a
}

function actionTagType(a) {
  return { login: 'primary', play: '', download: 'warning', search: 'success', favorite: 'danger' }[a] || 'info'
}

function timeAgo(d) {
  if (!d) return ''
  const mins = Math.floor((Date.now() - new Date(d).getTime()) / 60000)
  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins}分钟前`
  return `${Math.floor(mins / 60)}小时前`
}

async function loadData() {
  try {
    const [prov, online, trend, actLogs] = await Promise.all([
      adminStore.getProvinceStats(days.value),
      adminStore.getOnlineUsers(),
      adminStore.getTrends(days.value),
      adminStore.getActivityLogs(1, 1000, {})
    ])
    provinceStats.value = prov || []
    onlineUsers.value = online || []
    trends.value = trend || []

    // Aggregate action stats from activity logs
    const actionMap = {}
    for (const log of (actLogs?.list || [])) {
      actionMap[log.action] = (actionMap[log.action] || 0) + 1
    }
    activityStats.value = Object.entries(actionMap).map(([action, count]) => ({ action, count }))
      .sort((a, b) => b.count - a.count)
  } catch {
    ElMessage.error('加载分析数据失败')
  }
}

onMounted(async () => {
  await loadData()
  loading.value = false
})
</script>

<style scoped>
.analytics-page {
  max-width: 1200px;
  --tbl-bg: transparent;
  --tbl-text: rgba(255,255,255,0.85);
  --tbl-header-bg: rgba(255,255,255,0.04);
  --tbl-header-text: rgba(255,255,255,0.5);
  --tbl-stripe: rgba(255,255,255,0.02);
}

[data-theme="day"] .analytics-page {
  --tbl-bg: #ffffff;
  --tbl-text: rgba(0,0,0,0.85);
  --tbl-header-bg: #fafafa;
  --tbl-header-text: rgba(0,0,0,0.5);
  --tbl-stripe: rgba(0,0,0,0.02);
}

.loading-state { padding: 20px 0; }
.toolbar { margin-bottom: 20px; }

.charts-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 28px;
}

.chart-card {
  background: var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: 14px;
  padding: 20px;
}
.chart-card.full-width { grid-column: 1 / -1; }

.chart-title {
  font-size: 14px; font-weight: 600; color: var(--text-secondary);
  margin: 0 0 16px;
}

.province-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.chart-wrap, .province-table-wrap { min-width: 0; }

:deep(.province-table .el-table) { background: var(--tbl-bg) !important; color: var(--tbl-text); }
:deep(.province-table .el-table th) { background: var(--tbl-header-bg) !important; color: var(--tbl-header-text); font-size: 12px; }
:deep(.province-table .el-table tr) { background: transparent !important; }
:deep(.province-table .el-table--striped .el-table__body tr.el-table__row--striped td) { background: var(--tbl-stripe) !important; }

.section { margin-bottom: 28px; }

:deep(.online-table .el-table) { background: var(--tbl-bg) !important; color: var(--tbl-text); }
:deep(.online-table .el-table th) { background: var(--tbl-header-bg) !important; color: var(--tbl-header-text); font-size: 12px; }
:deep(.online-table .el-table tr) { background: transparent !important; }
:deep(.online-table .el-table--striped .el-table__body tr.el-table__row--striped td) { background: var(--tbl-stripe) !important; }

.user-cell { display: flex; align-items: center; gap: 8px; }
.tiny-avatar { flex-shrink: 0; }
.ip-code { font-size: 12px; color: var(--text-muted); background: var(--bg-elevated); padding: 2px 6px; border-radius: 4px; }

.empty-state { text-align: center; padding: 32px; color: var(--text-faint); font-size: 14px; }

@media (max-width: 768px) {
  .charts-grid { grid-template-columns: 1fr; }
  .province-layout { grid-template-columns: 1fr; }
}
</style>
