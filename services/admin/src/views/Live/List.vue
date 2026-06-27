<template>
  <div class="live-page">
    <div class="page-header">
      <h2 class="page-h2">直播管理</h2>
      <div class="header-actions">
        <el-button @click="openImportM3U">
          <el-icon><Upload /></el-icon>
          导入 M3U
        </el-button>
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
        <p><strong>M3U 列表：</strong>支持导入 GitHub、live.zhi35.com 等托管的 M3U 频道列表（首页地址会自动解析）。</p>
      </template>
    </el-alert>

    <el-card v-if="playlists.length > 0" shadow="never" class="m3u-sync-panel">
      <template #header>
        <span>M3U 自动同步</span>
      </template>
      <el-table :data="playlists" size="small" stripe>
        <el-table-column label="列表来源" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <el-link :href="row.url" target="_blank" type="primary">{{ shortUrl(row.url) }}</el-link>
          </template>
        </el-table-column>
        <el-table-column label="频道数" width="80" prop="count" />
        <el-table-column label="自动同步" width="100">
          <template #default="{ row }">
            <el-switch
              :model-value="row.sync_enabled"
              @change="(v: boolean) => saveSyncConfig(row, { sync_enabled: v })"
            />
          </template>
        </el-table-column>
        <el-table-column label="同步频率" width="150">
          <template #default="{ row }">
            <el-select
              :model-value="row.interval_minutes"
              size="small"
              :disabled="!row.sync_enabled"
              @change="(v: number) => saveSyncConfig(row, { interval_minutes: v })"
            >
              <el-option v-for="o in syncIntervalOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="上次同步" min-width="180">
          <template #default="{ row }">
            <span v-if="row.last_sync_at">{{ formatTime(row.last_sync_at) }}</span>
            <span v-else class="text-muted">尚未同步</span>
            <el-tag
              v-if="row.last_sync_status"
              size="small"
              :type="row.last_sync_status === 'ok' ? 'success' : 'danger'"
              class="sync-status-tag"
            >
              {{ row.last_sync_status === 'ok' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button
              link
              type="primary"
              size="small"
              :loading="syncingUrl === row.url"
              @click="onSyncM3U(row.url)"
            >
              立即同步
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <div class="filter-bar">
      <el-input
        v-model="filters.search"
        placeholder="搜索标题"
        clearable
        style="width: 200px"
        @keyup.enter="loadRooms"
        @clear="loadRooms"
      />
      <el-select v-model="filters.room_type" placeholder="类型" clearable style="width: 110px" @change="loadRooms">
        <el-option label="推流" value="push" />
        <el-option label="IPTV" value="iptv" />
      </el-select>
      <el-select v-model="filters.group_title" placeholder="分组" clearable style="width: 140px" @change="loadRooms">
        <el-option v-for="g in groupOptions" :key="g.name" :label="`${g.name} (${g.count})`" :value="g.name" />
      </el-select>
      <el-select v-model="filters.status" placeholder="状态" clearable style="width: 110px" @change="loadRooms">
        <el-option label="直播中" value="live" />
        <el-option label="待开播" value="idle" />
        <el-option label="已结束" value="ended" />
      </el-select>
      <el-button @click="loadRooms">查询</el-button>
      <el-button
        v-if="selectedIds.length > 0"
        type="danger"
        :loading="batchDeleting"
        @click="onBatchDelete"
      >
        批量删除 ({{ selectedIds.length }})
      </el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="rooms"
      stripe
      @selection-change="onSelectionChange"
    >
      <el-table-column type="selection" width="48" />
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
      <el-table-column label="分组" width="120" show-overflow-tooltip>
        <template #default="{ row }">
          {{ row.group_title || '-' }}
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

    <div v-if="total > pageSize" class="pagination">
      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="loadRooms"
      />
    </div>

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
          <div class="field-hint">单路 HLS 流地址；若是 M3U 频道列表请使用「导入 M3U」</div>
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

    <!-- M3U 导入对话框 -->
    <el-dialog v-model="importDialog" title="导入 M3U 频道列表" width="560">
      <el-form label-position="top">
        <el-form-item label="M3U 列表地址">
          <el-input
            v-model="importForm.playlist_url"
            placeholder="https://live.zhi35.com/ 或 https://example.com/playlist.m3u8"
          />
          <div class="field-hint">
            支持聚合站首页（如 live.zhi35.com 会自动解析为 /iptv.m3u），也可填写 URL 或直接粘贴下方内容
          </div>
        </el-form-item>
        <el-form-item label="或粘贴 M3U 内容">
          <el-input
            v-model="importForm.playlist_content"
            type="textarea"
            :rows="6"
            placeholder="#EXTM3U&#10;#EXTINF:-1,频道名&#10;http://..."
          />
          <div class="field-hint">NAS 无法访问 GitHub 时，可在电脑浏览器打开链接复制内容粘贴到这里</div>
        </el-form-item>
        <el-form-item>
          <el-button :loading="previewing" @click="onPreviewM3U">解析预览</el-button>
        </el-form-item>
        <el-form-item v-if="preview" label="频道分组">
          <el-select
            v-model="importForm.groups"
            multiple
            collapse-tags
            collapse-tags-tooltip
            placeholder="不选则导入全部分组"
            style="width: 100%"
          >
            <el-option
              v-for="g in preview.groups"
              :key="g.name"
              :label="`${g.name} (${g.count})`"
              :value="g.name"
            />
          </el-select>
          <div v-if="preview" class="field-hint">共 {{ preview.total }} 个频道</div>
        </el-form-item>
        <el-form-item>
          <el-checkbox v-model="importForm.replace">
            替换同源频道（删除此前从该 M3U 地址导入的频道后重新导入）
          </el-checkbox>
        </el-form-item>
        <el-form-item label="自动同步">
          <el-checkbox v-model="importForm.auto_sync">导入后启用定时同步</el-checkbox>
        </el-form-item>
        <el-form-item v-if="importForm.auto_sync" label="同步频率">
          <el-select v-model="importForm.auto_sync_interval_minutes" style="width: 100%">
            <el-option v-for="o in syncIntervalOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="importDialog = false">取消</el-button>
        <el-button type="primary" :loading="importing" :disabled="!preview" @click="onImportM3U">
          开始导入
        </el-button>
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

.m3u-sync-panel {
  margin-bottom: var(--mh-space-4);
  border-radius: var(--mh-radius-sm);
}

.sync-status-tag {
  margin-left: 8px;
}

.text-muted {
  color: var(--mh-text-muted, #888);
}

.filter-bar {
  display: flex;
  flex-wrap: wrap;
  gap: var(--mh-space-2);
  margin-bottom: var(--mh-space-4);
}

.pagination {
  margin-top: var(--mh-space-4);
  display: flex;
  justify-content: flex-end;
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
