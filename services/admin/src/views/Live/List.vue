<template>
  <div class="live-page">
    <!-- 顶部操作栏 -->
    <div class="live-header">
      <div class="header-left">
        <h2 class="page-title">直播管理</h2>
        <div class="live-stats">
          <span class="stat-item">
            <span class="stat-dot live"></span>
            {{ liveCount }} 直播中
          </span>
          <span class="stat-item">
            <span class="stat-dot"></span>
            {{ total }} 频道
          </span>
        </div>
      </div>
      <div class="header-actions">
        <el-button @click="openImportM3U" size="small">
          <el-icon><Upload /></el-icon>
          导入 M3U
        </el-button>
        <el-button @click="openCreate('iptv')" size="small">
          <el-icon><Link /></el-icon>
          添加 IPTV
        </el-button>
        <el-button type="primary" @click="openCreate('push')" size="small">
          <el-icon><Plus /></el-icon>
          创建推流
        </el-button>
      </div>
    </div>

    <!-- 电视墙主体 -->
    <div class="tv-wall">
      <!-- 频道列表 -->
      <aside class="channel-list">
        <!-- 搜索过滤 -->
        <div class="list-toolbar">
          <el-input
            v-model="filters.search"
            placeholder="搜索频道..."
            clearable
            size="small"
            @keyup.enter="loadRooms"
            @clear="loadRooms"
          />
          <el-select v-model="filters.group_title" placeholder="分组" clearable size="small" @change="loadRooms">
            <el-option v-for="g in groupOptions" :key="g.name" :label="`${g.name}`" :value="g.name" />
          </el-select>
        </div>

        <!-- 频道卡片列表 -->
        <div class="channels">
          <div
            v-for="room in rooms"
            :key="room.id"
            class="channel-card"
            :class="{ active: selectedRoom?.id === room.id, live: room.status === 'live' }"
            @click="selectRoom(room)"
          >
            <div class="channel-thumb">
              <img v-if="room.cover_url" :src="room.cover_url" :alt="room.title" />
              <span v-else class="thumb-placeholder">{{ room.title.slice(0, 2) }}</span>
              <div v-if="room.status === 'live'" class="live-badge">LIVE</div>
            </div>
            <div class="channel-info">
              <div class="channel-name">{{ room.title }}</div>
              <div class="channel-meta">
                <el-tag :type="room.room_type === 'iptv' ? 'success' : 'info'" size="small">
                  {{ room.room_type === 'iptv' ? 'IPTV' : '推流' }}
                </el-tag>
                <span v-if="room.group_title" class="group-name">{{ room.group_title }}</span>
              </div>
            </div>
          </div>

          <div v-if="rooms.length === 0 && !loading" class="empty-channels">
            <span>暂无频道</span>
          </div>
        </div>
      </aside>

      <!-- 预览区 -->
      <main class="preview-area">
        <template v-if="selectedRoom">
          <!-- 预览播放器区域 -->
          <div class="player-preview">
            <div class="player-frame">
              <div v-if="selectedRoom.status === 'live'" class="player-live">
                <div class="player-placeholder">
                  <div class="play-icon">▶</div>
                  <div class="play-hint">点击播放</div>
                </div>
              </div>
              <div v-else class="player-idle">
                <div class="idle-icon">📺</div>
                <div class="idle-text">{{ selectedRoom.status === 'ended' ? '直播已结束' : '等待开播' }}</div>
              </div>
            </div>

            <!-- 操作按钮 -->
            <div class="preview-actions">
              <el-button v-if="selectedRoom.status === 'live'" type="danger" size="small" @click="onStop(selectedRoom)">
                结束直播
              </el-button>
              <el-button size="small" @click="showStreamInfo(selectedRoom)">
                {{ selectedRoom.room_type === 'iptv' ? '源信息' : '推流信息' }}
              </el-button>
              <el-button type="danger" size="small" @click="onDelete(selectedRoom)">删除</el-button>
            </div>
          </div>

          <!-- 频道信息 -->
          <div class="preview-info">
            <h3 class="preview-title">{{ selectedRoom.title }}</h3>
            <div class="preview-tags">
              <el-tag :type="selectedRoom.room_type === 'iptv' ? 'success' : 'info'" size="small">
                {{ selectedRoom.room_type === 'iptv' ? 'IPTV 拉流' : '推流直播' }}
              </el-tag>
              <el-tag v-if="selectedRoom.group_title" size="small">{{ selectedRoom.group_title }}</el-tag>
              <el-tag :type="statusType(selectedRoom.status)" size="small">{{ statusLabel(selectedRoom.status) }}</el-tag>
            </div>
            <div v-if="selectedRoom.description" class="preview-desc">{{ selectedRoom.description }}</div>
            <div class="preview-time">
              <span v-if="selectedRoom.started_at">开始于 {{ formatTime(selectedRoom.started_at) }}</span>
              <span v-else>创建于 {{ formatTime(selectedRoom.created_at) }}</span>
            </div>
          </div>
        </template>

        <template v-else>
          <div class="preview-empty">
            <div class="empty-icon">📺</div>
            <div class="empty-text">选择频道查看预览</div>
          </div>
        </template>
      </main>
    </div>

    <!-- 创建对话框 -->
    <el-dialog v-model="createDialog" :title="createTitle" width="480">
      <el-form :model="createForm" label-position="top">
        <el-form-item label="标题" required>
          <el-input v-model="createForm.title" placeholder="例如：CCTV-1 / 周末电影之夜" maxlength="200" />
        </el-form-item>
        <el-form-item v-if="createForm.room_type === 'iptv'" label="IPTV 流地址 (m3u8)" required>
          <el-input v-model="createForm.source_url" placeholder="https://example.com/live/channel.m3u8" />
          <div class="field-hint">单路 HLS 流地址；若是 M3U 频道列表请使用「导入 M3U」</div>
        </el-form-item>
        <el-form-item label="简介">
          <el-input v-model="createForm.description" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
        <el-form-item label="封面 URL">
          <el-input v-model="createForm.cover_url" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialog = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="onCreate">创建</el-button>
      </template>
    </el-dialog>

    <!-- M3U 导入对话框 -->
    <el-dialog v-model="importDialog" title="导入 M3U 频道列表" width="520">
      <el-form label-position="top">
        <el-form-item label="M3U 列表地址">
          <el-input v-model="importForm.playlist_url" placeholder="https://live.zhi35.com/" />
          <div class="field-hint">支持聚合站首页或直接填写 M3U 地址</div>
        </el-form-item>
        <el-form-item label="或粘贴 M3U 内容">
          <el-input v-model="importForm.playlist_content" type="textarea" :rows="5" placeholder="#EXTM3U..." />
        </el-form-item>
        <el-form-item>
          <el-button :loading="previewing" @click="onPreviewM3U">解析预览</el-button>
        </el-form-item>
        <el-form-item v-if="preview" label="频道分组">
          <el-select v-model="importForm.groups" multiple collapse-tags placeholder="不选则导入全部分组" style="width: 100%">
            <el-option v-for="g in preview.groups" :key="g.name" :label="`${g.name} (${g.count})`" :value="g.name" />
          </el-select>
          <div class="field-hint">共 {{ preview.total }} 个频道</div>
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="importForm.auto_sync">导入后启用定时同步</el-checkbox>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="importDialog = false">取消</el-button>
        <el-button type="primary" :loading="importing" :disabled="!preview" @click="onImportM3U">开始导入</el-button>
      </template>
    </el-dialog>

    <!-- 推流/源信息对话框 -->
    <el-dialog v-model="streamDialog" :title="infoDialogTitle" width="480">
      <template v-if="selectedRoom">
        <div class="stream-info">
          <template v-if="selectedRoom.room_type === 'iptv'">
            <div class="info-row">
              <label>源地址</label>
              <div class="copy-row">
                <el-input :model-value="selectedRoom.source_url || ''" readonly size="small" />
                <el-button size="small" @click="copy(selectedRoom.source_url || '', '源地址')">复制</el-button>
              </div>
            </div>
            <div class="info-row">
              <label>播放地址</label>
              <div class="copy-row">
                <el-input :model-value="playUrl" readonly size="small" />
                <el-button size="small" @click="copy(playUrl, '播放地址')">复制</el-button>
              </div>
            </div>
          </template>
          <template v-else>
            <div class="info-row">
              <label>RTMP 服务器</label>
              <div class="copy-row">
                <el-input :model-value="rtmpServer" readonly size="small" />
                <el-button size="small" @click="copy(rtmpServer, '服务器')">复制</el-button>
              </div>
            </div>
            <div class="info-row">
              <label>Stream Key</label>
              <div class="copy-row">
                <el-input :model-value="selectedRoom.stream_key" readonly size="small" />
                <el-button size="small" @click="copy(selectedRoom.stream_key, 'Key')">复制</el-button>
              </div>
            </div>
          </template>
        </div>
        <div v-if="selectedRoom.room_type !== 'iptv'" class="obs-hint">
          <p><strong>OBS 推流设置</strong></p>
          <p>服务：自定义 | 服务器：<code>{{ rtmpServer }}</code> | 串流密钥：<code>{{ selectedRoom.stream_key }}</code></p>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Link, Upload } from '@element-plus/icons-vue'
import { liveApi, type LiveRoom, type LiveRoomType, type M3UPreviewResult, type LivePlaylistStat, SYNC_INTERVAL_OPTIONS } from '@/api/live'
import { copyToClipboard } from '@/utils/clipboard'

const loading = ref(false)
const creating = ref(false)
const batchDeleting = ref(false)
const selectedIds = ref<string[]>([])
const syncingUrl = ref('')
const rooms = ref<LiveRoom[]>([])
const playlists = ref<LivePlaylistStat[]>([])
const syncIntervalOptions = SYNC_INTERVAL_OPTIONS
const groupOptions = ref<M3UPreviewResult['groups']>([])
const total = ref(0)
const page = ref(1)
const pageSize = 50
const filters = ref({
  search: '',
  room_type: '',
  group_title: '',
  status: '',
})
const createDialog = ref(false)
const importDialog = ref(false)
const streamDialog = ref(false)
const previewing = ref(false)
const importing = ref(false)
const preview = ref<M3UPreviewResult | null>(null)
const selectedRoom = ref<LiveRoom | null>(null)
const createForm = ref({
  title: '',
  description: '',
  cover_url: '',
  room_type: 'push' as LiveRoomType,
  source_url: '',
})
const importForm = ref({
  playlist_url: 'https://raw.githubusercontent.com/hujingguang/ChinaIPTV/main/cnTV_AutoUpdate.m3u8',
  playlist_content: '',
  groups: [] as string[],
  replace: false,
  auto_sync: true,
  auto_sync_interval_minutes: 1440,
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

const liveCount = computed(() => rooms.value.filter(r => r.status === 'live').length)

function selectRoom(room: LiveRoom) {
  selectedRoom.value = room
}

function statusLabel(s: string) {
  return { idle: '待开播', live: '直播中', ended: '已结束' }[s] || s
}

function statusType(s: string): 'primary' | 'success' | 'warning' | 'info' | 'danger' {
  return ({ idle: 'info', live: 'danger', ended: 'info' } as const)[s as 'idle' | 'live' | 'ended'] || 'info'
}

function formatTime(iso: string) {
  return new Date(iso).toLocaleString('zh-CN')
}

async function loadRooms() {
  loading.value = true
  try {
    const params: Record<string, string | number> = {
      page: page.value,
      page_size: pageSize,
    }
    if (filters.value.search.trim()) params.search = filters.value.search.trim()
    if (filters.value.room_type) params.room_type = filters.value.room_type
    if (filters.value.group_title) params.group_title = filters.value.group_title
    if (filters.value.status) params.status = filters.value.status
    const resp = await liveApi.list(params)
    rooms.value = resp.data
    total.value = resp.total
  } catch (e: any) {
    ElMessage.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadMeta() {
  try {
    const [gResp, pResp] = await Promise.all([liveApi.groups(), liveApi.playlists()])
    groupOptions.value = gResp.data
    playlists.value = pResp.data
  } catch {
    // ignore
  }
}

function shortUrl(url: string) {
  try {
    const u = new URL(url)
    const parts = u.pathname.split('/')
    return parts[parts.length - 1] || u.hostname
  } catch {
    return url.length > 40 ? url.slice(0, 40) + '…' : url
  }
}

async function onSyncM3U(url: string) {
  await ElMessageBox.confirm(
    '将删除该 M3U 来源下的旧频道并重新导入最新列表，是否继续？',
    '同步 M3U',
    { type: 'warning' },
  )
  syncingUrl.value = url
  try {
    const resp = await liveApi.syncM3U(url)
    const r = resp.data
    ElMessage.success(`同步完成：新增 ${r.created}，跳过 ${r.skipped}，失败 ${r.failed}`)
    await Promise.all([loadRooms(), loadMeta()])
  } catch (e: any) {
    ElMessage.error(e?.message || '同步失败')
  } finally {
    syncingUrl.value = ''
  }
}

async function saveSyncConfig(
  row: LivePlaylistStat,
  patch: Partial<{ sync_enabled: boolean; interval_minutes: number }>,
) {
  const enabled = patch.sync_enabled ?? row.sync_enabled
  const interval = patch.interval_minutes ?? row.interval_minutes
  try {
    await liveApi.updateSyncConfig({
      playlist_url: row.url,
      enabled,
      interval_minutes: interval,
    })
    const item = playlists.value.find((p) => p.url === row.url)
    if (item) {
      item.sync_enabled = enabled
      item.interval_minutes = interval
    }
    ElMessage.success('同步配置已保存')
  } catch (e: any) {
    ElMessage.error(e?.message || '保存失败')
    await loadMeta()
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

function openImportM3U() {
  preview.value = null
  importForm.value.playlist_content = ''
  importForm.value.groups = []
  importForm.value.replace = false
  importForm.value.auto_sync = true
  importForm.value.auto_sync_interval_minutes = 1440
  importDialog.value = true
}

async function onPreviewM3U() {
  if (!importForm.value.playlist_url.trim() && !importForm.value.playlist_content.trim()) {
    ElMessage.warning('请填写 M3U 地址或粘贴内容')
    return
  }
  previewing.value = true
  try {
    const resp = await liveApi.previewM3U({
      playlist_url: importForm.value.playlist_url.trim() || undefined,
      playlist_content: importForm.value.playlist_content.trim() || undefined,
    })
    preview.value = resp.data
    importForm.value.playlist_url = resp.data.playlist_url
    ElMessage.success(`解析成功，共 ${resp.data.total} 个频道`)
  } catch (e: any) {
    preview.value = null
    ElMessage.error(e?.message || '解析失败')
  } finally {
    previewing.value = false
  }
}

async function onImportM3U() {
  if (!preview.value) {
    ElMessage.warning('请先解析预览 M3U 列表')
    return
  }
  importing.value = true
  try {
    const resp = await liveApi.importM3U({
      playlist_url: importForm.value.playlist_url.trim() || undefined,
      playlist_content: importForm.value.playlist_content.trim() || undefined,
      groups: importForm.value.groups.length ? importForm.value.groups : undefined,
      replace: importForm.value.replace,
      auto_sync: importForm.value.auto_sync,
      auto_sync_interval_minutes: importForm.value.auto_sync_interval_minutes,
    })
    const r = resp.data
    importDialog.value = false
    ElMessage.success(`导入完成：新增 ${r.created}，跳过 ${r.skipped}，失败 ${r.failed}`)
    await Promise.all([loadRooms(), loadMeta()])
  } catch (e: any) {
    ElMessage.error(e?.message || '导入失败')
  } finally {
    importing.value = false
  }
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

function onSelectionChange(rows: LiveRoom[]) {
  selectedIds.value = rows.map((r) => r.id)
}

async function onBatchDelete() {
  const n = selectedIds.value.length
  if (n === 0) return
  await ElMessageBox.confirm(
    `确定删除选中的 ${n} 个直播源？此操作不可恢复。`,
    '批量删除',
    { type: 'warning' },
  )
  batchDeleting.value = true
  try {
    const resp = await liveApi.batchDelete(selectedIds.value)
    selectedIds.value = []
    ElMessage.success(`已删除 ${resp.deleted} 个直播源`)
    await Promise.all([loadRooms(), loadMeta()])
  } catch (e: any) {
    ElMessage.error(e?.message || '批量删除失败')
  } finally {
    batchDeleting.value = false
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
  loadMeta()
  pollTimer = setInterval(loadRooms, 15000)
})

onBeforeUnmount(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style lang="scss" scoped>
.live-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: var(--mh-space-4);
}

// 顶部栏
.live-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--mh-space-6);
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--mh-text);
  margin: 0;
}

.live-stats {
  display: flex;
  gap: var(--mh-space-4);
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--mh-text-secondary);
}

.stat-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--mh-text-tertiary);

  &.live {
    background: var(--mh-danger);
    animation: blink 1.5s infinite;
  }
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.header-actions {
  display: flex;
  gap: var(--mh-space-2);
}

// 电视墙布局
.tv-wall {
  flex: 1;
  display: grid;
  grid-template-columns: 320px 1fr;
  gap: var(--mh-space-4);
  min-height: 0;
}

// 频道列表
.channel-list {
  display: flex;
  flex-direction: column;
  background: var(--mh-surface);
  border: 1px solid var(--mh-border);
  border-radius: var(--mh-radius-lg);
  overflow: hidden;
}

.list-toolbar {
  display: flex;
  gap: var(--mh-space-2);
  padding: var(--mh-space-3);
  border-bottom: 1px solid var(--mh-border);
  flex-shrink: 0;

  .el-input {
    flex: 1;
  }
}

.channels {
  flex: 1;
  overflow-y: auto;
  padding: var(--mh-space-2);
}

.channel-card {
  display: flex;
  gap: var(--mh-space-3);
  padding: var(--mh-space-3);
  border-radius: var(--mh-radius-md);
  cursor: pointer;
  transition: background var(--mh-duration-fast) var(--mh-ease);
  margin-bottom: var(--mh-space-1);

  &:hover {
    background: var(--mh-surface-secondary);
  }

  &.active {
    background: var(--mh-primary-muted);
  }

  &.live .channel-thumb {
    border-color: var(--mh-danger);
  }
}

.channel-thumb {
  width: 64px;
  height: 48px;
  border-radius: var(--mh-radius-sm);
  overflow: hidden;
  flex-shrink: 0;
  background: var(--mh-surface-secondary);
  border: 2px solid transparent;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.thumb-placeholder {
  font-size: 14px;
  font-weight: 600;
  color: var(--mh-text-tertiary);
}

.live-badge {
  position: absolute;
  top: 2px;
  left: 2px;
  background: var(--mh-danger);
  color: #fff;
  font-size: 8px;
  font-weight: 700;
  padding: 1px 4px;
  border-radius: 3px;
}

.channel-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
}

.channel-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--mh-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.channel-meta {
  display: flex;
  align-items: center;
  gap: 6px;
}

.group-name {
  font-size: 11px;
  color: var(--mh-text-tertiary);
}

.empty-channels {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--mh-space-8);
  color: var(--mh-text-tertiary);
  font-size: 13px;
}

// 预览区
.preview-area {
  display: flex;
  flex-direction: column;
  background: var(--mh-surface);
  border: 1px solid var(--mh-border);
  border-radius: var(--mh-radius-lg);
  overflow: hidden;
}

.player-preview {
  background: #000;
  aspect-ratio: 16/9;
  display: flex;
  flex-direction: column;
}

.player-frame {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.player-live {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.player-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--mh-space-2);
}

.play-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: #fff;
}

.play-hint {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.6);
}

.player-idle {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--mh-space-2);
}

.idle-icon {
  font-size: 48px;
  opacity: 0.3;
}

.idle-text {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.4);
}

.preview-actions {
  display: flex;
  gap: var(--mh-space-2);
  padding: var(--mh-space-3);
  background: var(--mh-bg-elevated);
  border-top: 1px solid var(--mh-border);
}

.preview-info {
  padding: var(--mh-space-4);
  border-top: 1px solid var(--mh-border);
}

.preview-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--mh-text);
  margin: 0 0 var(--mh-space-2);
}

.preview-tags {
  display: flex;
  gap: var(--mh-space-2);
  flex-wrap: wrap;
  margin-bottom: var(--mh-space-3);
}

.preview-desc {
  font-size: 13px;
  color: var(--mh-text-secondary);
  margin-bottom: var(--mh-space-2);
}

.preview-time {
  font-size: 12px;
  color: var(--mh-text-tertiary);
}

.preview-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--mh-space-3);
}

.empty-icon {
  font-size: 64px;
  opacity: 0.2;
}

.empty-text {
  font-size: 14px;
  color: var(--mh-text-tertiary);
}

// 信息弹窗
.stream-info {
  display: flex;
  flex-direction: column;
  gap: var(--mh-space-3);
}

.info-row {
  label {
    display: block;
    font-size: 12px;
    color: var(--mh-text-secondary);
    margin-bottom: 4px;
  }
}

.copy-row {
  display: flex;
  gap: var(--mh-space-2);

  .el-input {
    flex: 1;
  }
}

.obs-hint {
  margin-top: var(--mh-space-4);
  padding: var(--mh-space-3);
  background: var(--mh-surface-secondary);
  border-radius: var(--mh-radius-md);
  font-size: 13px;

  p {
    margin: 0 0 var(--mh-space-1);
    color: var(--mh-text-secondary);
  }

  strong {
    color: var(--mh-text);
  }

  code {
    font-family: monospace;
    font-size: 12px;
    background: var(--mh-surface);
    padding: 1px 4px;
    border-radius: 3px;
  }
}

.field-hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--mh-text-tertiary);
}
</style>
