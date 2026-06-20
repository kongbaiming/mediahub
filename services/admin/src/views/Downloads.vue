<template>
  <div class="downloads-page">
    <div class="page-header">
      <h2 class="page-h2">下载管理</h2>
      <div class="header-actions">
        <el-button @click="checkCompleted" :loading="checking">
          <el-icon><Refresh /></el-icon>
          检查已完成（自动入库）
        </el-button>
        <el-button @click="addDialog = true" type="primary">
          <el-icon><Plus /></el-icon>
          添加下载
        </el-button>
      </div>
    </div>

    <el-card shadow="never" class="stats-card">
      <div class="stats-row">
        <div class="stat">
          <div class="stat-label">活跃下载</div>
          <div class="stat-value">{{ activeCount }}</div>
        </div>
        <div class="stat">
          <div class="stat-label">已完成</div>
          <div class="stat-value">{{ completedCount }}</div>
        </div>
        <div class="stat">
          <div class="stat-label">下载速度</div>
          <div class="stat-value">{{ formatSpeed(totalDownSpeed) }}</div>
        </div>
        <div class="stat">
          <div class="stat-label">上传速度</div>
          <div class="stat-value">{{ formatSpeed(totalUpSpeed) }}</div>
        </div>
      </div>
    </el-card>

    <el-table v-loading="loading" :data="items" stripe>
      <el-table-column label="名称" min-width="280">
        <template #default="{ row }">
          <div class="name-cell">
            <el-icon class="file-icon"><VideoCamera /></el-icon>
            <span>{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="分类" prop="category" width="100">
        <template #default="{ row }">
          <el-tag size="small">{{ row.category || '-' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="进度" width="200">
        <template #default="{ row }">
          <el-progress
            :percentage="Math.round(row.progress * 100)"
            :status="row.state === 'downloading' ? '' : (row.progress >= 1 ? 'success' : '')"
          />
          <div class="progress-meta">
            {{ formatSize(row.size * row.progress) }} / {{ formatSize(row.size) }}
          </div>
        </template>
      </el-table-column>
      <el-table-column label="下载速度" width="100">
        <template #default="{ row }">
          {{ row.dlspeed > 0 ? formatSpeed(row.dlspeed) : '-' }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="stateType(row.state)" size="small">{{ stateLabel(row.state) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button v-if="row.state === 'downloading'" size="small" @click.stop="pause(row.hash)">
            暂停
          </el-button>
          <el-button v-if="row.state === 'paused'" size="small" type="success" @click.stop="resume(row.hash)">
            继续
          </el-button>
          <el-button size="small" type="danger" @click.stop="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="addDialog" title="添加下载任务" width="500">
      <el-form :model="addForm" label-position="top">
        <el-form-item label="种子链接 / Magnet">
          <el-input
            v-model="addForm.url"
            type="textarea"
            :rows="3"
            placeholder="magnet:?xt=urn:btih:... 或 https://..."
          />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="addForm.category" placeholder="选择分类">
            <el-option label="电影" value="movie" />
            <el-option label="剧集" value="tvshow" />
            <el-option label="动画" value="anime" />
            <el-option label="纪录片" value="documentary" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addDialog = false">取消</el-button>
        <el-button type="primary" @click="onAdd" :loading="adding">添加</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { downloaderApi, type Download } from '@/api/client'

const loading = ref(false)
const checking = ref(false)
const adding = ref(false)
const items = ref<Download[]>([])
const addDialog = ref(false)
const addForm = ref({ url: '', category: 'movie' })

let refreshTimer: number | undefined

const activeCount = computed(() =>
  items.value.filter((i) => i.state === 'downloading' || i.state === 'stalled').length
)
const completedCount = computed(() =>
  items.value.filter((i) => i.progress >= 1 || i.state === 'completed').length
)
const totalDownSpeed = computed(() =>
  items.value.reduce((sum, i) => sum + (i.dlspeed || 0), 0)
)
const totalUpSpeed = computed(() =>
  items.value.reduce((sum, i) => sum + (i.upspeed || 0), 0)
)

async function load() {
  loading.value = true
  try {
    const res = await downloaderApi.list()
    items.value = res.data
  } finally {
    loading.value = false
  }
}

async function onAdd() {
  if (!addForm.value.url.trim()) {
    ElMessage.warning('请输入下载链接')
    return
  }
  adding.value = true
  try {
    await downloaderApi.add({
      url: addForm.value.url,
      category: addForm.value.category,
    })
    ElMessage.success('已添加到 qBittorrent')
    addDialog.value = false
    addForm.value.url = ''
    await load()
  } finally {
    adding.value = false
  }
}

async function pause(hash: string) {
  try {
    await downloaderApi.pause(hash)
    ElMessage.success('已暂停')
    load()
  } catch {}
}

async function resume(hash: string) {
  try {
    await downloaderApi.resume(hash)
    ElMessage.success('已继续')
    load()
  } catch {}
}

async function remove(row: Download) {
  try {
    await ElMessageBox.confirm(
      `确认删除「${row.name}」？${row.progress < 1 ? '\n\n⚠️ 文件未下载完，建议保留文件' : ''}`,
      '删除确认',
      { type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await downloaderApi.remove(row.hash, row.progress >= 1)
    ElMessage.success('已删除')
    load()
  } catch {}
}

async function checkCompleted() {
  checking.value = true
  try {
    const res = await downloaderApi.checkCompleted() as { status?: string; imported?: number; message?: string }
    if (res.status === 'unavailable') {
      ElMessage.warning(res.message || '下载器不可用，请检查 qBittorrent 连接')
      return
    }
    ElMessage.success(`扫描完成，自动入库 ${res.imported ?? 0} 部`)
    load()
  } finally {
    checking.value = false
  }
}

function formatSize(bytes: number) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = bytes
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(2)} ${units[i]}`
}

function formatSpeed(bytesPerSec: number) {
  if (!bytesPerSec) return '0 B/s'
  return formatSize(bytesPerSec) + '/s'
}

function stateType(state: string): any {
  return ({
    downloading: 'primary',
    completed: 'success',
    paused: 'warning',
    error: 'danger',
    stalled: 'warning',
  } as any)[state] || 'info'
}

function stateLabel(state: string) {
  return ({
    downloading: '下载中',
    completed: '已完成',
    paused: '已暂停',
    error: '出错',
    stalled: '停滞',
    queued: '排队',
    uploading: '做种',
  } as any)[state] || state
}

onMounted(() => {
  load()
  // 每 5 秒刷新
  refreshTimer = window.setInterval(load, 5000)
})

onBeforeUnmount(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<style lang="scss" scoped>
.downloads-page {
  max-width: 1400px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  color: #1e293b;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.stats-card {
  margin-bottom: 20px;
  border-radius: 12px;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 24px;
}

.stat-label {
  font-size: 13px;
  color: #64748b;
  margin-bottom: 4px;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: #1e293b;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-icon {
  color: #6366f1;
  font-size: 18px;
}

.progress-meta {
  font-size: 11px;
  color: #94a3b8;
  margin-top: 2px;
}
</style>
