<template>
  <div class="dashboard">
    <h2 class="page-h2">总览</h2>

    <el-row :gutter="20" class="stat-row">
      <el-col :xs="24" :sm="12" :md="6">
        <div class="stat-card stat-primary">
          <div class="stat-label">媒资总数</div>
          <div class="stat-value">{{ stats?.total || 0 }}</div>
          <el-icon class="stat-bg-icon"><Film /></el-icon>
        </div>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <div class="stat-card stat-success">
          <div class="stat-label">电影</div>
          <div class="stat-value">{{ stats?.by_type?.movie || 0 }}</div>
          <el-icon class="stat-bg-icon"><VideoCamera /></el-icon>
        </div>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <div class="stat-card stat-warning">
          <div class="stat-label">剧集</div>
          <div class="stat-value">{{ stats?.by_type?.tvshow || 0 }}</div>
          <el-icon class="stat-bg-icon"><Monitor /></el-icon>
        </div>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <div class="stat-card stat-danger">
          <div class="stat-label">未刮削</div>
          <div class="stat-value">{{ stats?.by_scrape?.pending || 0 }}</div>
          <el-icon class="stat-bg-icon"><Warning /></el-icon>
        </div>
      </el-col>
    </el-row>

    <el-row :gutter="20" class="content-row">
      <el-col :xs="24" :md="14">
        <el-card shadow="never" class="panel">
          <template #header>
            <div class="panel-header">
              <span>媒资类型分布</span>
              <el-tag size="small">实时</el-tag>
            </div>
          </template>
          <div ref="typeChartRef" class="chart-container"></div>
        </el-card>
      </el-col>

      <el-col :xs="24" :md="10">
        <el-card shadow="never" class="panel">
          <template #header>
            <div class="panel-header">
              <span>刮削状态</span>
              <el-button text type="primary" @click="refresh">刷新</el-button>
            </div>
          </template>
          <div class="scrape-status">
            <div
              v-for="(count, status) in stats?.by_scrape || {}"
              :key="status"
              class="status-item"
            >
              <div class="status-info">
                <el-tag :type="scrapeStatusColor(status)" size="small">
                  {{ scrapeStatusLabel(status) }}
                </el-tag>
              </div>
              <div class="status-count">{{ count }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="panel">
      <template #header>
        <div class="panel-header">
          <span>最近添加</span>
          <el-button text type="primary" @click="$router.push('/media')">
            查看全部
            <el-icon><ArrowRight /></el-icon>
          </el-button>
        </div>
      </template>
      <el-row :gutter="16">
        <el-col v-for="m in recent" :key="m.id" :xs="12" :sm="8" :md="4">
          <div class="recent-card" @click="$router.push(`/media/${m.id}`)">
            <div class="poster-card">
              <img v-if="m.poster_url" :src="m.poster_url" :alt="m.title" />
              <span v-else>{{ m.title.slice(0, 2) }}</span>
            </div>
            <div class="recent-title">{{ m.title }}</div>
            <div class="recent-meta">
              <span v-if="m.year">{{ m.year }}</span>
              <el-rate
                v-if="m.rating"
                :model-value="m.rating / 2"
                disabled
                size="small"
                show-score
                :show-text="false"
              />
            </div>
          </div>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import * as echarts from 'echarts'
import { mediaApi, type MediaListParams } from '@/api/media'
import type { Stats, MediaSummary } from '@/api/types'

const stats = ref<Stats | null>(null)
const recent = ref<MediaSummary[]>([])
const typeChartRef = ref<HTMLElement>()

async function refresh() {
  try {
    const [s, r] = await Promise.all([
      mediaApi.stats(),
      mediaApi.list({ page: 1, page_size: 8, sort: 'created_at', order: 'desc' }),
    ])
    stats.value = s.data
    recent.value = r.items
    renderChart()
  } catch (e) {
    // toast handled by interceptor
  }
}

function renderChart() {
  if (!typeChartRef.value || !stats.value) return
  const chart = echarts.init(typeChartRef.value)
  const data = Object.entries(stats.value.by_type || {}).map(([k, v]) => ({
    name: mediaTypeLabel(k),
    value: v,
  }))
  chart.setOption({
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [
      {
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: true,
        itemStyle: { borderRadius: 8, borderColor: '#fff', borderWidth: 2 },
        label: { show: true, formatter: '{b}\n{c}' },
        data,
        color: ['#6366f1', '#ec4899', '#14b8a6', '#f59e0b'],
      },
    ],
  })
}

function scrapeStatusLabel(s: string) {
  return ({ pending: '待刮削', scraping: '刮削中', done: '已完成', failed: '失败' } as any)[s] || s
}

function scrapeStatusColor(s: string): 'success' | 'warning' | 'danger' | 'info' {
  return ({ done: 'success', scraping: 'warning', failed: 'danger' } as any)[s] || 'info'
}

function mediaTypeLabel(s: string) {
  return ({ movie: '电影', tvshow: '剧集', anime: '动画', documentary: '纪录片' } as any)[s] || s
}

onMounted(() => {
  refresh()
  window.addEventListener('resize', () => {
    if (typeChartRef.value) {
      echarts.getInstanceByDom(typeChartRef.value)?.resize()
    }
  })
})
</script>

<style lang="scss" scoped>
.dashboard {
  max-width: 1400px;
}

.page-h2 {
  margin: 0 0 20px;
  font-size: 22px;
  font-weight: 600;
  color: #1e293b;
}

.stat-row {
  margin-bottom: 20px;
}

.stat-card {
  position: relative;
  padding: 20px;
  border-radius: 12px;
  color: #fff;
  overflow: hidden;
  margin-bottom: 16px;

  &.stat-primary { background: linear-gradient(135deg, #6366f1, #4f46e5); }
  &.stat-success { background: linear-gradient(135deg, #10b981, #047857); }
  &.stat-warning { background: linear-gradient(135deg, #f59e0b, #d97706); }
  &.stat-danger  { background: linear-gradient(135deg, #ef4444, #b91c1c); }
}

.stat-label {
  font-size: 13px;
  opacity: 0.85;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  margin-top: 8px;
}

.stat-bg-icon {
  position: absolute;
  right: -10px;
  bottom: -10px;
  font-size: 100px;
  opacity: 0.18;
}

.content-row {
  margin-bottom: 20px;
}

.panel {
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  margin-bottom: 16px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 500;
}

.chart-container {
  height: 280px;
}

.scrape-status {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.status-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #f8fafc;
  border-radius: 6px;
}

.status-count {
  font-weight: 600;
  font-size: 16px;
  color: #1e293b;
}

.recent-card {
  cursor: pointer;
  transition: transform 0.15s;

  &:hover {
    transform: translateY(-2px);
  }
}

.recent-title {
  margin-top: 8px;
  font-size: 13px;
  font-weight: 500;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.recent-meta {
  margin-top: 4px;
  font-size: 12px;
  color: #64748b;
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
