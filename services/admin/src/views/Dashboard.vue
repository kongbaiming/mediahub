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
              <el-button text type="primary" @click="$router.push('/scrape')">
                刮削中心
              </el-button>
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
        color: ['#6c63ff', '#ec4899', '#3ecfcf', '#f59e0b'],
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

.stat-row {
  margin-bottom: var(--mh-space-5);
}

.content-row {
  margin-bottom: var(--mh-space-5);
}

.chart-container {
  height: 280px;
}

.scrape-status {
  display: flex;
  flex-direction: column;
  gap: var(--mh-space-3);
}

.status-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--mh-space-2) var(--mh-space-3);
  background: var(--mh-admin-surface-muted);
  border-radius: var(--mh-radius-sm);
  border: 1px solid var(--mh-border-light);
  transition: background var(--mh-duration) var(--mh-ease);

  &:hover {
    background: var(--mh-admin-surface);
  }
}

.status-count {
  font-weight: 700;
  font-size: 16px;
  font-family: var(--mh-font-display);
  color: var(--mh-text);
}

.recent-card {
  cursor: pointer;
  transition: transform var(--mh-duration) var(--mh-ease);

  &:hover {
    transform: translateY(-3px);

    .poster-card {
      box-shadow: var(--mh-shadow-md);
    }
  }
}

.recent-title {
  margin-top: var(--mh-space-2);
  font-size: 13px;
  font-weight: 600;
  color: var(--mh-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.recent-meta {
  margin-top: var(--mh-space-1);
  font-size: 12px;
  color: var(--mh-text-secondary);
  display: flex;
  align-items: center;
  gap: var(--mh-space-2);
}
</style>
