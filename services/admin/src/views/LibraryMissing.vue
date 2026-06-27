<template>
  <div class="library-missing-page">
    <div class="page-header">
      <h2 class="page-h2">猜你喜欢 · 库外推荐</h2>
      <div class="header-actions">
        <el-button @click="load" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <el-alert
      type="info"
      show-icon
      :closable="false"
      class="tip-alert"
      title="入库向导"
      description="以下条目来自「猜你喜欢」中的 TMDB 推荐，尚未加入媒体库。可一键搜索资源并添加到 qBittorrent 下载队列。"
    />

    <el-table v-loading="loading" :data="items" stripe>
      <el-table-column label="封面" width="80">
        <template #default="{ row }">
          <el-image v-if="row.poster_url" :src="row.poster_url" fit="cover" class="poster-thumb" />
          <div v-else class="poster-placeholder">{{ row.title?.slice(0, 1) || '?' }}</div>
        </template>
      </el-table-column>
      <el-table-column label="标题" min-width="220">
        <template #default="{ row }">
          <div class="title-cell">
            <span>{{ row.title }}</span>
            <el-tag size="small" type="warning">未入库</el-tag>
          </div>
          <div class="sub-meta">
            {{ row.year || '—' }} · {{ typeLabel(row.media_type) }}
            <span v-if="row.rating"> · ⭐ {{ row.rating.toFixed(1) }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="TMDB" width="90" prop="tmdb_id" />
      <el-table-column label="简介" min-width="280" show-overflow-tooltip prop="overview" />
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="openSearch(row as LibraryMissingItem)">搜索入库</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="!loading && items.length === 0" description="暂无库外推荐，或 TMDB 未配置" />

    <el-dialog v-model="searchDialog" :title="`搜索资源：${activeItem?.title || ''}`" width="720">
      <div class="search-bar">
        <el-input v-model="searchQuery" placeholder="搜索关键词" @keyup.enter="runSearch" />
        <el-button type="primary" :loading="searching" @click="runSearch">搜索</el-button>
      </div>
      <el-alert
        v-if="searchMessage"
        :type="searchStatus === 'ok' ? 'info' : 'warning'"
        :title="searchMessage"
        show-icon
        :closable="false"
        class="search-alert"
      />
      <el-table v-loading="searching" :data="releases" max-height="420" empty-text="暂无结果">
        <el-table-column label="资源名" min-width="280" prop="title" show-overflow-tooltip />
        <el-table-column label="索引" prop="indexer" width="120" show-overflow-tooltip />
        <el-table-column label="大小" width="100">
          <template #default="{ row }">{{ formatSize(row.size) }}</template>
        </el-table-column>
        <el-table-column label="做种" width="70" prop="seeders" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button
              size="small"
              type="primary"
              :loading="downloadingHash === row.link"
              @click="downloadRelease(row as IndexerRelease)"
            >
              下载
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  libraryMissingApi,
  indexerApi,
  downloaderApi,
  type LibraryMissingItem,
  type IndexerRelease,
} from '@/api/client'

const loading = ref(false)
const items = ref<LibraryMissingItem[]>([])

const searchDialog = ref(false)
const activeItem = ref<LibraryMissingItem | null>(null)
const searchQuery = ref('')
const searching = ref(false)
const releases = ref<IndexerRelease[]>([])
const searchStatus = ref('')
const searchMessage = ref('')
const downloadingHash = ref('')

async function load() {
  loading.value = true
  try {
    const res = await libraryMissingApi.list({ limit: 50, discover_limit: 16 })
    items.value = res.data || []
  } catch (e: any) {
    ElMessage.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function openSearch(row: LibraryMissingItem) {
  activeItem.value = row
  searchQuery.value = [row.title, row.year].filter(Boolean).join(' ')
  searchDialog.value = true
  releases.value = []
  searchStatus.value = ''
  searchMessage.value = ''
  runSearch()
}

async function runSearch() {
  if (!searchQuery.value.trim()) {
    ElMessage.warning('请输入搜索词')
    return
  }
  searching.value = true
  try {
    const res = await indexerApi.search({
      q: searchQuery.value.trim(),
      type: activeItem.value?.media_type,
      limit: 25,
    })
    releases.value = res.data || []
    searchStatus.value = res.status
    searchMessage.value =
      res.status === 'unavailable'
        ? res.message || '索引器未配置'
        : res.status === 'error'
          ? res.message || '搜索失败'
          : `找到 ${releases.value.length} 条结果`
  } finally {
    searching.value = false
  }
}

async function downloadRelease(row: IndexerRelease) {
  if (!row.link) return
  downloadingHash.value = row.link
  try {
    await downloaderApi.add({ url: row.link, category: mapCategory(activeItem.value?.media_type) })
    ElMessage.success('已添加到下载队列，完成后将自动入库')
  } catch {
    ElMessage.error('添加下载失败')
  } finally {
    downloadingHash.value = ''
  }
}

function mapCategory(type?: string) {
  if (type === 'tvshow' || type === 'tv') return 'tvshow'
  if (type === 'anime') return 'anime'
  return 'movie'
}

function typeLabel(type?: string) {
  if (type === 'movie') return '电影'
  if (type === 'tvshow' || type === 'tv') return '剧集'
  if (type === 'anime') return '动画'
  return type || '—'
}

function formatSize(bytes: number) {
  if (!bytes) return '—'
  const gb = bytes / 1024 / 1024 / 1024
  if (gb >= 1) return `${gb.toFixed(1)} GB`
  return `${(bytes / 1024 / 1024).toFixed(0)} MB`
}

onMounted(load)
</script>

<style scoped lang="scss">
.library-missing-page {
  padding: 0 4px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.tip-alert {
  margin-bottom: 16px;
}

.poster-thumb {
  width: 48px;
  height: 72px;
  border-radius: 4px;
}

.poster-placeholder {
  width: 48px;
  height: 72px;
  border-radius: 4px;
  background: #334155;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  font-weight: 600;
}

.title-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sub-meta {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 4px;
}

.search-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.search-alert {
  margin-bottom: 12px;
}
</style>
