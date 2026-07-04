<template>
  <div class="dashboard">
    <h2 class="page-h2">总览</h2>

    <!-- 顶部统计卡片 -->
    <el-row :gutter="20" class="stat-row">
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <el-statistic title="媒资总数" :value="data?.total_media ?? 0">
            <template #prefix><el-icon class="stat-icon primary"><Film /></el-icon></template>
          </el-statistic>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <el-statistic title="存储空间" :value="formattedSize.value" :suffix="formattedSize.unit">
            <template #prefix><el-icon class="stat-icon success"><Coin /></el-icon></template>
          </el-statistic>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <el-statistic title="用户档案" :value="data?.total_profiles ?? 0">
            <template #prefix><el-icon class="stat-icon warning"><User /></el-icon></template>
          </el-statistic>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="never" class="stat-card">
          <el-statistic title="待刮削" :value="scrapePending">
            <template #prefix><el-icon class="stat-icon danger"><Warning /></el-icon></template>
          </el-statistic>
        </el-card>
      </el-col>
    </el-row>

    <!-- 类型分布 & 可用性状态 -->
    <el-row :gutter="20" class="content-row">
      <el-col :xs="24" :md="12">
        <el-card shadow="never" class="panel">
          <template #header>
            <div class="panel-header">
              <span>媒资类型分布</span>
              <el-tag size="small" type="info">实时</el-tag>
            </div>
          </template>
          <div class="bar-chart">
            <div
              v-for="item in data?.type_counts ?? []"
              :key="item.type"
              class="bar-row"
            >
              <span class="bar-label">{{ mediaTypeLabel(item.type) }}</span>
              <div class="bar-track">
                <div
                  class="bar-fill"
                  :style="{
                    width: barWidth(item.count, maxTypeCount) + '%',
                    background: typeColorMap[item.type] || 'var(--mh-primary)',
                  }"
                />
              </div>
              <span class="bar-value">{{ item.count }}</span>
            </div>
            <el-empty v-if="!data?.type_counts?.length" description="暂无数据" :image-size="60" />
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :md="12">
        <el-card shadow="never" class="panel">
          <template #header>
            <div class="panel-header">
              <span>刮削状态</span>
              <el-button text type="primary" @click="refresh">
                <el-icon><Refresh /></el-icon>刷新
              </el-button>
            </div>
          </template>
          <div class="status-grid">
            <div
              v-for="item in data?.scrape_counts ?? []"
              :key="item.status"
              class="status-card"
            >
              <el-tag :type="scrapeStatusColor(item.status)" size="small" effect="dark">
                {{ scrapeStatusLabel(item.status) }}
              </el-tag>
              <span class="status-count">{{ item.count }}</span>
            </div>
            <el-empty v-if="!data?.scrape_counts?.length" description="暂无数据" :image-size="60" />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 可用性状态 & 每日趋势 -->
    <el-row :gutter="20" class="content-row">
      <el-col :xs="24" :md="10">
        <el-card shadow="never" class="panel">
          <template #header>
            <div class="panel-header"><span>可用性状态</span></div>
          </template>
          <div class="status-grid">
            <div
              v-for="item in data?.avail_counts ?? []"
              :key="item.status"
              class="status-card"
            >
              <el-tag :type="availStatusColor(item.status)" size="small" effect="plain">
                {{ availStatusLabel(item.status) }}
              </el-tag>
              <span class="status-count">{{ item.count }}</span>
            </div>
            <el-empty v-if="!data?.avail_counts?.length" description="暂无数据" :image-size="60" />
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :md="14">
        <el-card shadow="never" class="panel">
          <template #header>
            <div class="panel-header">
              <span>每日新增趋势</span>
              <el-tag size="small" type="info">近 7 天</el-tag>
            </div>
          </template>
          <div class="trend-chart">
            <div class="trend-bars">
              <div
                v-for="item in data?.daily_trend ?? []"
                :key="item.date"
                class="trend-col"
              >
                <span class="trend-val">{{ item.count }}</span>
                <div
                  class="trend-bar"
                  :style="{ height: trendBarHeight(item.count) + '%' }"
                />
                <span class="trend-date">{{ formatDay(item.date) }}</span>
              </div>
            </div>
            <el-empty v-if="!data?.daily_trend?.length" description="暂无数据" :image-size="60" />
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { http } from '@/api/client'

interface CountItem {
  type: string
  count: number
}

interface StatusItem {
  status: string
  count: number
}

interface TrendItem {
  date: string
  count: number
}

interface DashboardData {
  total_media: number
  type_counts: CountItem[]
  scrape_counts: StatusItem[]
  avail_counts: StatusItem[]
  daily_trend: TrendItem[]
  total_size: number
  total_profiles: number
}

interface DashboardResponse {
  data: DashboardData
}

const data = ref<DashboardData | null>(null)
const loading = ref(false)

const typeColorMap: Record<string, string> = {
  movie: '#6B8EFF',
  tvshow: '#FFB88C',
  anime: '#5ECA9E',
  documentary: '#FF8B8B',
}

async function refresh() {
  loading.value = true
  try {
    const res = await http.get<DashboardResponse>('/api/v1/dashboard/stats')
    data.value = res.data
  } catch {
    // error handled by interceptor
  } finally {
    loading.value = false
  }
}

const scrapePending = computed(() => {
  const item = data.value?.scrape_counts?.find((s) => s.status === 'pending')
  return item?.count ?? 0
})

const maxTypeCount = computed(() => {
  const counts = data.value?.type_counts?.map((i) => i.count) ?? []
  return Math.max(...counts, 1)
})

const maxTrendCount = computed(() => {
  const counts = data.value?.daily_trend?.map((i) => i.count) ?? []
  return Math.max(...counts, 1)
})

const formattedSize = computed(() => {
  const bytes = data.value?.total_size ?? 0
  if (bytes === 0) return { value: 0, unit: 'B' }
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const val = parseFloat((bytes / Math.pow(1024, i)).toFixed(1))
  return { value: val, unit: units[i] }
})

function barWidth(count: number, max: number): number {
  return max > 0 ? Math.round((count / max) * 100) : 0
}

function trendBarHeight(count: number): number {
  return maxTrendCount.value > 0 ? Math.round((count / maxTrendCount.value) * 100) : 0
}

function formatDay(dateStr: string): string {
  const d = new Date(dateStr)
  return `${d.getMonth() + 1}/${d.getDate()}`
}

function mediaTypeLabel(t: string): string {
  const map: Record<string, string> = {
    movie: '电影',
    tvshow: '剧集',
    anime: '动画',
    documentary: '纪录片',
  }
  return map[t] || t
}

function scrapeStatusLabel(s: string): string {
  const map: Record<string, string> = {
    pending: '待刮削',
    scraping: '刮削中',
    done: '已完成',
    failed: '失败',
  }
  return map[s] || s
}

function scrapeStatusColor(s: string): '' | 'success' | 'warning' | 'danger' | 'info' {
  const map: Record<string, '' | 'success' | 'warning' | 'danger' | 'info'> = {
    done: 'success',
    scraping: 'warning',
    failed: 'danger',
    pending: 'info',
  }
  return map[s] ?? ''
}

function availStatusLabel(s: string): string {
  const map: Record<string, string> = {
    available: '可用',
    processing: '处理中',
    missing: '缺失',
    unreleased: '未上映',
  }
  return map[s] || s
}

function availStatusColor(s: string): '' | 'success' | 'warning' | 'danger' | 'info' {
  const map: Record<string, '' | 'success' | 'warning' | 'danger' | 'info'> = {
    available: 'success',
    processing: 'warning',
    missing: 'danger',
    unreleased: 'info',
  }
  return map[s] ?? ''
}

onMounted(() => {
  refresh()
})
</script>

<style lang="scss" scoped>
.stat-row {
  margin-bottom: var(--mh-space-5, 20px);
}

.content-row {
  margin-bottom: var(--mh-space-5, 20px);
}

.stat-card {
  :deep(.el-card__body) {
    padding: 20px;
  }

  :deep(.el-statistic__head) {
    font-size: 13px;
    color: rgba(255, 255, 255, 0.7);
  }

  :deep(.el-statistic__content) {
    font-size: 28px;
    font-weight: 700;
    font-family: var(--mh-font-display, inherit);
    color: #fff;
  }
}

.stat-icon {
  font-size: 20px;
  margin-right: 4px;

  &.primary { color: rgba(255, 255, 255, 0.9); }
  &.success { color: rgba(255, 255, 255, 0.9); }
  &.warning { color: rgba(255, 255, 255, 0.9); }
  &.danger  { color: rgba(255, 255, 255, 0.9); }
}

.panel {
  height: 100%;
  background: var(--mh-surface) !important;
  border: 1px solid var(--mh-border) !important;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: var(--mh-text);
}

/* ---------- CSS 横向柱状图（类型分布） ---------- */
.bar-chart {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 4px 0;
}

.bar-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.bar-label {
  width: 56px;
  flex-shrink: 0;
  text-align: right;
  font-size: 13px;
  color: var(--mh-text-secondary);
}

.bar-track {
  flex: 1;
  height: 22px;
  background: var(--mh-surface-secondary);
  border-radius: 11px;
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  border-radius: 11px;
  transition: width 0.6s cubic-bezier(0.22, 1, 0.36, 1);
  min-width: 4px;
}

.bar-value {
  width: 36px;
  flex-shrink: 0;
  font-size: 14px;
  font-weight: 700;
  color: var(--mh-text);
  text-align: right;
}

/* ---------- 状态网格 ---------- */
.status-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}

.status-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px 12px;
  background: var(--mh-surface-secondary);
  border-radius: var(--mh-radius-md);
  border: 1px solid var(--mh-border);
  transition: all var(--mh-duration-fast) var(--mh-ease);

  &:hover {
    background: var(--mh-surface-tertiary);
    transform: translateY(-1px);
  }
}

.status-count {
  font-weight: 700;
  font-size: 22px;
  font-family: var(--mh-font-display, inherit);
  color: var(--mh-text);
}

/* ---------- CSS 纵向柱状图（每日趋势） ---------- */
.trend-chart {
  padding: 8px 0;
}

.trend-bars {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  height: 200px;
  padding-bottom: 28px;
  position: relative;
}

.trend-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
  position: relative;
}

.trend-val {
  font-size: 12px;
  font-weight: 600;
  color: var(--mh-text);
  margin-bottom: 4px;
}

.trend-bar {
  width: 100%;
  max-width: 48px;
  border-radius: 6px 6px 0 0;
  background: linear-gradient(180deg, var(--mh-primary) 0%, rgba(107, 142, 255, 0.5) 100%);
  transition: height 0.6s cubic-bezier(0.22, 1, 0.36, 1);
  min-height: 4px;
  margin-top: auto;
}

.trend-date {
  position: absolute;
  bottom: 0;
  font-size: 11px;
  color: var(--mh-text-tertiary);
  white-space: nowrap;
}
</style>
