<template>
  <div class="live-page">
    <div class="page-header">
      <h2 class="page-h2">直播管理</h2>
      <el-button type="primary" @click="openCreate">
        <el-icon><Plus /></el-icon>
        创建直播间
      </el-button>
    </div>

    <el-alert
      type="info"
      :closable="false"
      show-icon
      class="tip-alert"
      title="推流说明"
      description="创建直播间后，使用 OBS 等推流软件，服务器填写 RTMP 地址，串流密钥填写 Stream Key。推流成功后状态自动变为「直播中」。"
    />

    <el-table v-loading="loading" :data="rooms" stripe>
      <el-table-column label="标题" min-width="200">
        <template #default="{ row }">
          <div class="title-cell">
            <el-tag v-if="row.status === 'live'" type="danger" size="small" effect="dark" class="live-dot">
              LIVE
            </el-tag>
            <span>{{ row.title }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="开始时间" width="170">
        <template #default="{ row }">
          {{ row.started_at ? formatTime(row.started_at) : '-' }}
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="170">
        <template #default="{ row }">
          {{ formatTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="280" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="showStreamInfo(row)">推流信息</el-button>
          <el-button
            v-if="row.status === 'live'"
            size="small"
            type="warning"
            @click="onStop(row)"
          >
            结束
          </el-button>
          <el-button size="small" type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 创建对话框 -->
    <el-dialog v-model="createDialog" title="创建直播间" width="480">
      <el-form :model="createForm" label-position="top">
        <el-form-item label="标题" required>
          <el-input v-model="createForm.title" placeholder="例如：周末电影之夜" maxlength="200" />
        </el-form-item>
        <el-form-item label="简介">
          <el-input v-model="createForm.description" type="textarea" :rows="3" placeholder="可选" />
        </el-form-item>
        <el-form-item label="封面 URL">
          <el-input v-model="createForm.cover_url" placeholder="可选，http(s)://..." />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialog = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="onCreate">创建</el-button>
      </template>
    </el-dialog>

    <!-- 推流信息对话框 -->
    <el-dialog v-model="streamDialog" title="推流信息" width="560">
      <template v-if="selectedRoom">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="直播间 ID">{{ selectedRoom.id }}</el-descriptions-item>
          <el-descriptions-item label="RTMP 服务器">
            <code>{{ rtmpServer }}</code>
            <el-button link type="primary" @click="copy(rtmpServer)">复制</el-button>
          </el-descriptions-item>
          <el-descriptions-item label="Stream Key">
            <code>{{ selectedRoom.stream_key }}</code>
            <el-button link type="primary" @click="copy(selectedRoom.stream_key)">复制</el-button>
          </el-descriptions-item>
          <el-descriptions-item label="完整 RTMP 地址">
            <code class="break-all">{{ selectedRoom.rtmp_url }}</code>
            <el-button link type="primary" @click="copy(selectedRoom.rtmp_url || '')">复制</el-button>
          </el-descriptions-item>
        </el-descriptions>
        <div class="obs-tip">
          <p><strong>OBS 设置：</strong></p>
          <ul>
            <li>设置 → 推流 → 服务选「自定义」</li>
            <li>服务器：<code>{{ rtmpServer }}</code></li>
            <li>串流密钥：<code>{{ selectedRoom.stream_key }}</code></li>
          </ul>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { liveApi, type LiveRoom } from '@/api/live'

const loading = ref(false)
const creating = ref(false)
const rooms = ref<LiveRoom[]>([])
const createDialog = ref(false)
const streamDialog = ref(false)
const selectedRoom = ref<LiveRoom | null>(null)
const createForm = ref({ title: '', description: '', cover_url: '' })

let pollTimer: ReturnType<typeof setInterval> | null = null

const rtmpServer = computed(() => {
  if (!selectedRoom.value?.rtmp_url) return ''
  const url = selectedRoom.value.rtmp_url
  const idx = url.lastIndexOf('/')
  return idx > 0 ? url.slice(0, idx) : url
})

function statusLabel(s: string) {
  return { idle: '待开播', live: '直播中', ended: '已结束' }[s] || s
}

function statusType(s: string) {
  return { idle: 'info', live: 'danger', ended: '' }[s] || 'info'
}

function formatTime(iso: string) {
  return new Date(iso).toLocaleString('zh-CN')
}

async function loadRooms() {
  loading.value = true
  try {
    const resp = await liveApi.list({ page_size: 50 })
    rooms.value = resp.data
  } catch (e: any) {
    ElMessage.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  createForm.value = { title: '', description: '', cover_url: '' }
  createDialog.value = true
}

async function onCreate() {
  if (!createForm.value.title.trim()) {
    ElMessage.warning('请填写标题')
    return
  }
  creating.value = true
  try {
    const resp = await liveApi.create(createForm.value)
    createDialog.value = false
    ElMessage.success('直播间已创建')
    await loadRooms()
    showStreamInfo(resp.data)
  } catch (e: any) {
    ElMessage.error(e?.message || '创建失败')
  } finally {
    creating.value = false
  }
}

function showStreamInfo(room: LiveRoom) {
  selectedRoom.value = room
  streamDialog.value = true
}

async function onStop(room: LiveRoom) {
  await ElMessageBox.confirm(`确定结束「${room.title}」的直播？`, '结束直播')
  try {
    await liveApi.stop(room.id)
    ElMessage.success('已结束')
    await loadRooms()
  } catch (e: any) {
    ElMessage.error(e?.message || '操作失败')
  }
}

async function onDelete(room: LiveRoom) {
  await ElMessageBox.confirm(`确定删除「${room.title}」？`, '删除直播间', { type: 'warning' })
  try {
    await liveApi.delete(room.id)
    ElMessage.success('已删除')
    await loadRooms()
  } catch (e: any) {
    ElMessage.error(e?.message || '删除失败')
  }
}

function copy(text: string) {
  navigator.clipboard.writeText(text).then(
    () => ElMessage.success('已复制'),
    () => ElMessage.error('复制失败'),
  )
}

onMounted(() => {
  loadRooms()
  pollTimer = setInterval(loadRooms, 10000)
})

onBeforeUnmount(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style lang="scss" scoped>
.live-page {
  max-width: 1200px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--mh-space-4);
}

.tip-alert {
  margin-bottom: var(--mh-space-4);
}

.title-cell {
  display: flex;
  align-items: center;
  gap: var(--mh-space-2);
}

.live-dot {
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

code {
  font-family: var(--mh-font-mono, monospace);
  font-size: 13px;
  background: var(--mh-admin-surface-muted);
  padding: 2px 6px;
  border-radius: 4px;
}

.break-all {
  word-break: break-all;
}

.obs-tip {
  margin-top: var(--mh-space-4);
  padding: var(--mh-space-3);
  background: var(--mh-admin-surface-muted);
  border-radius: var(--mh-radius-sm);
  font-size: 13px;

  ul {
    margin: var(--mh-space-2) 0 0;
    padding-left: 1.2em;
  }
}
</style>
