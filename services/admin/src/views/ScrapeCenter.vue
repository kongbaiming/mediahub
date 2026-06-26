<template>
  <div class="scrape-center-page">
    <div class="page-header">
      <h2 class="page-h2">刮削中心</h2>
      <div class="header-actions">
        <el-button @click="refresh" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button
          v-if="activeTab !== 'done'"
          type="primary"
          :loading="batchLoading"
          @click="retryByStatus"
        >
          重试当前 Tab 全部
        </el-button>
        <el-button
          type="warning"
          :disabled="selectedIds.length === 0"
          :loading="batchLoading"
          @click="retrySelected"
        >
          重试选中 ({{ selectedIds.length }})
        </el-button>
      </div>
    </div>

    <el-row :gutter="16" class="stat-row">
      <el-col :xs="12" :sm="6">
        <div class="stat-card stat-warning" @click="switchTab('pending')">
          <div class="stat-label">待刮削</div>
          <div class="stat-value">{{ stats?.by_scrape?.pending || 0 }}</div>
        </div>
      </el-col>
      <el-col :xs="12" :sm="6">
        <div class="stat-card stat-primary" @click="switchTab('scraping')">
          <div class="stat-label">刮削中</div>
          <div class="stat-value">{{ stats?.by_scrape?.scraping || 0 }}</div>
        </div>
      </el-col>
      <el-col :xs="12" :sm="6">
        <div class="stat-card stat-danger" @click="switchTab('failed')">
          <div class="stat-label">失败</div>
          <div class="stat-value">{{ stats?.by_scrape?.failed || 0 }}</div>
        </div>
      </el-col>
      <el-col :xs="12" :sm="6">
        <div class="stat-card stat-success" @click="switchTab('done')">
          <div class="stat-label">已完成</div>
          <div class="stat-value">{{ stats?.by_scrape?.done || 0 }}</div>
        </div>
      </el-col>
    </el-row>

    <el-card shadow="never" class="panel">
      <el-tabs v-model="activeTab" @tab-change="onTabChange">
        <el-tab-pane label="待刮削" name="pending" />
        <el-tab-pane label="刮削中" name="scraping" />
        <el-tab-pane label="失败" name="failed" />
        <el-tab-pane label="已完成" name="done" />
      </el-tabs>

      <el-table
        v-loading="loading"
        :data="items"
        stripe
        @selection-change="onSelectionChange"
      >
        <el-table-column v-if="activeTab !== 'done'" type="selection" width="48" />
        <el-table-column label="媒资" min-width="280">
          <template #default="{ row }">
            <div class="media-cell" @click="$router.push(`/media/${row.id}`)">
              <img v-if="row.poster_url" :src="row.poster_url" class="poster" alt="" />
              <div v-else class="poster placeholder">{{ row.title.slice(0, 2) }}</div>
              <div class="media-info">
                <div class="title">{{ row.title }}</div>
                <div class="meta">
                  <el-tag size="small" type="info">{{ mediaTypeLabel(row.type) }}</el-tag>
                  <span v-if="row.year">{{ row.year }}</span>
                </div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="scrapeStatusColor(row.scrape_status)" size="small">
              {{ scrapeStatusLabel(row.scrape_status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="失败原因" min-width="240">
          <template #default="{ row }">
            <span v-if="row.scrape_error" class="error-text">{{ row.scrape_error }}</span>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.scrape_status !== 'done'"
              size="small"
              type="primary"
              link
              :loading="retryingId === row.id"
              @click="retryOne(row.id)"
            >
              重试
            </el-button>
            <el-button size="small" link @click="$router.push(`/media/${row.id}`)">
              详情
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination" v-if="total > pageSize">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next, total"
          background
          @current-change="loadItems"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage } from 'element-plus'
import { mediaApi } from '@/api/media'
import type { MediaSummary, Stats } from '@/api/types'

const loading = ref(false)
const batchLoading = ref(false)
const retryingId = ref('')
const items = ref<MediaSummary[]>([])
const stats = ref<Stats | null>(null)
const selectedIds = ref<string[]>([])
const activeTab = ref<'pending' | 'scraping' | 'failed' | 'done'>('pending')
const page = ref(1)
const pageSize = 20
const total = ref(0)
let pollTimer: ReturnType<typeof setInterval> | null = null

function mediaTypeLabel(t: string) {
  return ({ movie: '电影', tvshow: '剧集', anime: '动画', documentary: '纪录片' } as Record<string, string>)[t] || t
}

function scrapeStatusLabel(s?: string) {
  return ({ pending: '待刮削', scraping: '刮削中', done: '已完成', failed: '失败' } as Record<string, string>)[s || ''] || s
}

function scrapeStatusColor(s?: string): 'success' | 'warning' | 'danger' | 'info' {
  return ({ done: 'success', scraping: 'warning', failed: 'danger' } as Record<string, 'success' | 'warning' | 'danger' | 'info'>)[s || ''] || 'info'
}

function onSelectionChange(rows: MediaSummary[]) {
  selectedIds.value = rows.map((r) => r.id)
}

function switchTab(tab: typeof activeTab.value) {
  activeTab.value = tab
  page.value = 1
  loadItems()
}

function onTabChange() {
  page.value = 1
  selectedIds.value = []
  loadItems()
}

async function loadStats() {
  try {
    const res = await mediaApi.stats()
    stats.value = res.data
  } catch {
    /* ignore */
  }
}

async function loadItems() {
  loading.value = true
  try {
    const res = await mediaApi.list({
      scrape_status: activeTab.value,
      page: page.value,
      page_size: pageSize,
      sort: 'created_at',
      order: 'desc',
    })
    items.value = res.items
    total.value = res.total
  } catch (e: any) {
    ElMessage.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function refresh() {
  await Promise.all([loadStats(), loadItems()])
}

async function retryOne(id: string) {
  retryingId.value = id
  try {
    await mediaApi.rescan(id)
    ElMessage.success('已加入刮削队列')
    await refresh()
  } catch (e: any) {
    ElMessage.error(e?.message || '重试失败')
  } finally {
    retryingId.value = ''
  }
}

async function retrySelected() {
  if (selectedIds.value.length === 0) return
  batchLoading.value = true
  try {
    const res = await mediaApi.batchRescan({ ids: selectedIds.value })
    ElMessage.success(`已加入 ${res.queued} 条刮削任务`)
    selectedIds.value = []
    await refresh()
  } catch (e: any) {
    ElMessage.error(e?.message || '批量重试失败')
  } finally {
    batchLoading.value = false
  }
}

async function retryByStatus() {
  batchLoading.value = true
  try {
    const res = await mediaApi.batchRescan({ scrape_status: activeTab.value })
    ElMessage.success(`已加入 ${res.queued} 条刮削任务`)
    await refresh()
  } catch (e: any) {
    ElMessage.error(e?.message || '批量重试失败')
  } finally {
    batchLoading.value = false
  }
}

onMounted(async () => {
  await refresh()
  pollTimer = setInterval(() => {
    if (activeTab.value === 'scraping' || activeTab.value === 'pending') {
      refresh()
    }
  }, 15000)
})

onBeforeUnmount(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped lang="scss">
.scrape-center-page {
  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 20px;
    flex-wrap: wrap;
    gap: 12px;
  }

  .header-actions {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .stat-row {
    margin-bottom: 20px;
  }

  .stat-card {
    padding: 16px 20px;
    border-radius: var(--mh-radius-lg);
    cursor: pointer;
    transition: transform 0.15s ease, box-shadow 0.15s ease;

    &:hover {
      transform: translateY(-2px);
      box-shadow: var(--mh-shadow-md);
    }

    .stat-label {
      font-size: 13px;
      color: var(--mh-text-secondary);
      margin-bottom: 6px;
    }

    .stat-value {
      font-size: 28px;
      font-weight: 700;
      font-family: var(--mh-font-display);
    }

    &.stat-warning { background: linear-gradient(135deg, #fef3c7, #fde68a); color: #92400e; }
    &.stat-primary { background: linear-gradient(135deg, #e0e7ff, #c7d2fe); color: #3730a3; }
    &.stat-danger { background: linear-gradient(135deg, #fee2e2, #fecaca); color: #991b1b; }
    &.stat-success { background: linear-gradient(135deg, #d1fae5, #a7f3d0); color: #065f46; }
  }

  .media-cell {
    display: flex;
    align-items: center;
    gap: 12px;
    cursor: pointer;

    .poster {
      width: 40px;
      height: 60px;
      border-radius: 4px;
      object-fit: cover;
      flex-shrink: 0;

      &.placeholder {
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--mh-bg-muted);
        font-size: 12px;
        color: var(--mh-text-muted);
      }
    }

    .title {
      font-weight: 600;
      color: var(--mh-text-primary);
    }

    .meta {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-top: 4px;
      font-size: 12px;
      color: var(--mh-text-secondary);
    }
  }

  .error-text {
    color: var(--mh-danger);
    font-size: 13px;
    word-break: break-word;
  }

  .muted {
    color: var(--mh-text-muted);
  }

  .pagination {
    margin-top: 16px;
    display: flex;
    justify-content: flex-end;
  }
}
</style>
