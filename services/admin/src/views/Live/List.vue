<template>
  <div class="live-page">
    <div class="page-header">
      <h2 class="page-h2">直播管理</h2>
      <div class="header-actions">
        <el-button @click="openCreate('iptv')">
          <el-icon><Link /></el-icon>
          添加 IPTV
        </el-button>
        <el-button type="primary" @click="openCreate('push')">
          <el-icon><Plus /></el-icon>
          创建推流间
        </el-button>
      </div>
    </div>

    <el-alert
      type="info"
      :closable="false"
      show-icon
      class="tip-alert"
      title="直播类型说明"
    >
      <template #default>
        <p><strong>推流直播：</strong>创建后用 OBS 推流，Stream Key 填入串流密钥。</p>
        <p><strong>IPTV 拉流：</strong>填写外部 m3u8 地址，无需推流，创建后可直接播放。</p>
      </template>
    </el-alert>

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
      <el-table-column label="类型" width="100">
        <template #default="{ row }">
          <el-tag :type="row.room_type === 'iptv' ? 'success' : ''" size="small">
            {{ row.room_type === 'iptv' ? 'IPTV' : '推流' }}
          </el-tag>
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
      <el-table-column label="操作" width="300" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="showStreamInfo(row)">
            {{ row.room_type === 'iptv' ? '源信息' : '推流信息' }}
          </el-button>
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
    <el-dialog v-model="createDialog" :title="createTitle" width="520">
      <el-form :model="createForm" label-position="top">
        <el-form-item label="标题" required>
          <el-input v-model="createForm.title" placeholder="例如：CCTV-1 / 周末电影之夜" maxlength="200" />
        </el-form-item>
        <el-form-item v-if="createForm.room_type === 'iptv'" label="IPTV 流地址 (m3u8)" required>
          <el-input
            v-model="createForm.source_url"
            placeholder="https://example.com/live/channel.m3u8"
          />
          <div class="field-hint">支持 http/https 的 HLS (m3u8) 直播源</div>
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

    <!-- 推流/源信息对话框 -->
    <el-dialog v-model="streamDialog" :title="infoDialogTitle" width="560">
      <template v-if="selectedRoom">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="直播间 ID">{{ selectedRoom.id }}</el-descriptions-item>
          <el-descriptions-item label="类型">
            {{ selectedRoom.room_type === 'iptv' ? 'IPTV 拉流' : '推流直播' }}
          </el-descriptions-item>

          <template v-if="selectedRoom.room_type === 'iptv'">
            <el-descriptions-item label="IPTV 源地址">
              <div class="copy-field">
                <el-input :model-value="selectedRoom.source_url || ''" readonly />
                <el-button @click="copy(selectedRoom.source_url || '', 'IPTV 源地址')">复制</el-button>
              </div>
            </el-descriptions-item>
            <el-descriptions-item label="播放地址">
              <div class="copy-field">
                <el-input :model-value="playUrl" readonly />
                <el-button @click="copy(playUrl, '播放地址')">复制</el-button>
              </div>
            </el-descriptions-item>
          </template>

          <template v-else>
            <el-descriptions-item label="RTMP 服务器">
              <div class="copy-field">
                <el-input :model-value="rtmpServer" readonly />
                <el-button @click="copy(rtmpServer, 'RTMP 服务器')">复制</el-button>
              </div>
            </el-descriptions-item>
            <el-descriptions-item label="Stream Key">
              <div class="copy-field">
                <el-input :model-value="selectedRoom.stream_key" readonly />
                <el-button @click="copy(selectedRoom.stream_key, 'Stream Key')">复制</el-button>
              </div>
            </el-descriptions-item>
            <el-descriptions-item label="完整 RTMP 地址">
              <div class="copy-field">
                <el-input :model-value="selectedRoom.rtmp_url || ''" readonly />
                <el-button @click="copy(selectedRoom.rtmp_url || '', 'RTMP 地址')">复制</el-button>
              </div>
            </el-descriptions-item>
          </template>
        </el-descriptions>

        <div v-if="selectedRoom.room_type !== 'iptv'" class="obs-tip">
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
import { Plus, Link } from '@element-plus/icons-vue'
import { liveApi, type LiveRoom, type LiveRoomType } from '@/api/live'
import { copyToClipboard } from '@/utils/clipboard'

const loading = ref(false)
const creating = ref(false)
const rooms = ref<LiveRoom[]>([])
const createDialog = ref(false)
const streamDialog = ref(false)
const selectedRoom = ref<LiveRoom | null>(null)
const createForm = ref({
  title: '',
  description: '',
  cover_url: '',
  room_type: 'push' as LiveRoomType,
  source_url: '',
})

let pollTimer: ReturnType<typeof setInterval> | null = null

const createTitle = computed(() =>
  createForm.value.room_type === 'iptv' ? '添加 IPTV 频道' : '创建推流直播间',
)

const infoDialogTitle = computed(() =>
  selectedRoom.value?.room_type === 'iptv' ? 'IPTV 源信息' : '推流信息',
)

const playUrl = computed(() => {
  if (!selectedRoom.value) return ''
  const path = selectedRoom.value.play_url || `/api/v1/live/rooms/${selectedRoom.value.id}/playlist.m3u8`
  if (path.startsWith('http')) return path
  return `${window.location.origin}${path}`
})

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

function openCreate(type: LiveRoomType) {
  createForm.value = {
    title: '',
    description: '',
    cover_url: '',
    room_type: type,
    source_url: '',
  }
  createDialog.value = true
}

async function onCreate() {
  if (!createForm.value.title.trim()) {
    ElMessage.warning('请填写标题')
    return
  }
  if (createForm.value.room_type === 'iptv' && !createForm.value.source_url.trim()) {
    ElMessage.warning('请填写 IPTV 流地址')
    return
  }
  creating.value = true
  try {
    const resp = await liveApi.create(createForm.value)
    createDialog.value = false
    ElMessage.success(createForm.value.room_type === 'iptv' ? 'IPTV 频道已添加' : '直播间已创建')
    await loadRooms()
    showStreamInfo(resp.data)
  } catch (e: any) {
    ElMessage.error(e?.message || '创建失败')
  } finally {
    creating.value = false
  }
}

async function showStreamInfo(room: LiveRoom) {
  streamDialog.value = true
  selectedRoom.value = room
  try {
    const resp = await liveApi.get(room.id)
    selectedRoom.value = resp.data
  } catch {
    // 列表数据兜底
  }
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

async function copy(text: string, label = '内容') {
  if (!text?.trim()) {
    ElMessage.warning(`${label} 为空`)
    return
  }
  const ok = await copyToClipboard(text)
  if (ok) {
    ElMessage.success(`${label} 已复制`)
  } else {
    ElMessage.warning(`${label} 自动复制失败，请手动选中输入框内容复制`)
  }
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

.header-actions {
  display: flex;
  gap: var(--mh-space-2);
}

.tip-alert {
  margin-bottom: var(--mh-space-4);

  p {
    margin: 0 0 4px;
    font-size: 13px;
  }
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

.field-hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--mh-text-muted, #888);
}

.copy-field {
  display: flex;
  gap: var(--mh-space-2);
  width: 100%;

  .el-input {
    flex: 1;
  }
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
