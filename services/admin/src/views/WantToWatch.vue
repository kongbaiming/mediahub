<template>
  <div class="want-page">
    <div class="page-header">
      <h2 class="page-h2">播放端想看</h2>
      <div class="header-actions">
        <el-button @click="load" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <el-alert
      v-if="indexerStatus === 'unavailable'"
      type="warning"
      show-icon
      :closable="false"
      class="tip-alert"
      title="索引器未配置"
      description="请在 API 环境变量中设置 INDEXER_URL 与 INDEXER_API_KEY（Prowlarr），才能在线搜索资源。"
    />

    <el-table v-loading="loading" :data="items" stripe>
      <el-table-column label="封面" width="80">
        <template #default="{ row }">
          <el-image
            v-if="row.poster_url"
            :src="row.poster_url"
            fit="cover"
            class="poster-thumb"
          />
          <div v-else class="poster-placeholder">{{ row.title?.slice(0, 1) || '?' }}</div>
        </template>
      </el-table-column>
      <el-table-column label="标题" min-width="200">
        <template #default="{ row }">
          <div class="title-cell">
            <span>{{ row.title }}</span>
            <el-tag v-if="row.external" size="small" type="warning">未入库</el-tag>
            <el-tag v-else-if="row.in_library" size="small" type="success">已入库</el-tag>
          </div>
          <div class="sub-meta">{{ row.year || '—' }} · {{ typeLabel(row.media_type) }}</div>
        </template>
      </el-table-column>
      <el-table-column label="成员" prop="profile_name" width="120" />
      <el-table-column label="TMDB" width="100">
        <template #default="{ row }">
          {{ row.tmdb_id || '—' }}
        </template>
      </el-table-column>
      <el-table-column label="标记时间" width="170">
        <template #default="{ row }">
          {{ formatTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button
            v-if="row.local_media_id"
            size="small"
            @click="openMedia(row.local_media_id)"
          >
            查看媒资
          </el-button>
          <el-button
            v-if="!row.in_library"
            size="small"
            type="primary"
            @click="openSearch(row)"
          >
            搜索入库
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-empty
      v-if="!loading && items.length === 0"
      description="暂无播放端标记的想看"
    >
      <template #default>
        <p class="empty-hint">在 Web 播放端打开库外影片详情，点击「加入想看」后，会出现在此列表。</p>
      </template>
    </el-empty>

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
            <el-button size="small" type="primary" :loading="downloadingHash === row.link" @click="downloadRelease(row)">
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
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { adminWantApi, indexerApi, downloaderApi, type AdminWantItem, type IndexerRelease } from '@/api/client'

const router = useRouter()
const loading = ref(false)
const items = ref<AdminWantItem[]>([])
const indexerStatus = ref('')

const searchDialog = ref(false)
const activeItem = ref<AdminWantItem | null>(null)
const searchQuery = ref('')
const searching = ref(false)
const releases = ref<IndexerRelease[]>([])
const searchStatus = ref('')
const searchMessage = ref('')
const downloadingHash = ref('')

async function load() {
  loading.value = true
  try {
    const res = await adminWantApi.list()
    items.value = res.data || []
  } finally {
    loading.value = false
  }
}

function openMedia(id: string) {
  router.push(`/media/${id}`)
}

function openSearch(row: AdminWantItem) {
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
    indexerStatus.value = res.status
  } finally {
    searching.value = false
  }
}

async function downloadRelease(row: IndexerRelease) {
  if (!row.link) return
  downloadingHash.value = row.link
  try {
    const category = mapCategory(activeItem.value?.media_type)
    await downloaderApi.add({ url: row.link, category })
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
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let n = bytes
  let i = 0
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024
    i++
  }
  return `${n.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

function formatTime(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString('zh-CN')
}

onMounted(load)
</script>

<style lang="scss" scoped>
.want-page {
  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--mh-space-5);
  }

  .page-h2 {
    margin: 0;
    font-size: 22px;
    font-weight: 600;
  }
}

.tip-alert {
  margin-bottom: var(--mh-space-4);
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
  background: var(--mh-admin-surface-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
}

.title-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sub-meta {
  font-size: 12px;
  color: var(--mh-text-muted);
  margin-top: 4px;
}

.search-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}

.search-alert {
  margin-bottom: 12px;
}

.empty-hint {
  margin: 0;
  font-size: 13px;
  color: var(--mh-text-muted);
  max-width: 420px;
  line-height: 1.5;
}
</style>
