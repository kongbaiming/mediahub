<template>
  <div class="media-list-page">
    <div class="page-header">
      <h2 class="page-h2">媒资库</h2>
      <el-button type="primary" @click="$router.push('/media-create')">
        <el-icon><Plus /></el-icon>
        新建媒资
      </el-button>
    </div>

    <el-card v-if="scanConfig" shadow="never" class="scan-config-panel">
      <template #header>
        <div class="scan-header">
          <span>自动入库扫描</span>
          <el-button type="primary" size="small" :loading="scanning" @click="onScanNow">
            立即扫描
          </el-button>
        </div>
      </template>
      <div class="scan-config-row">
        <div class="scan-field">
          <span class="label">自动扫描</span>
          <el-switch
            :model-value="scanConfig.enabled"
            @change="(v: boolean) => saveScanConfig({ enabled: v })"
          />
        </div>
        <div class="scan-field">
          <span class="label">扫描频率</span>
          <el-select
            :model-value="scanConfig.interval_minutes"
            size="small"
            style="width: 140px"
            :disabled="!scanConfig.enabled"
            @change="(v: number) => saveScanConfig({ interval_minutes: v })"
          >
            <el-option v-for="o in scanIntervalOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </div>
        <div class="scan-field scan-roots">
          <span class="label">扫描目录</span>
          <span class="roots">{{ scanConfig.roots?.join('、') || '-' }}</span>
        </div>
        <div class="scan-field scan-last">
          <span class="label">上次扫描</span>
          <span v-if="scanConfig.last_scan_at">
            {{ formatTime(scanConfig.last_scan_at) }}
            <el-tag
              size="small"
              :type="scanConfig.last_scan_status === 'ok' ? 'success' : 'danger'"
              class="status-tag"
            >
              {{ scanConfig.last_scan_status === 'ok' ? '成功' : '失败' }}
            </el-tag>
            <span v-if="scanConfig.last_scan_message" class="scan-msg">{{ scanConfig.last_scan_message }}</span>
          </span>
          <span v-else class="text-muted">尚未扫描</span>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="filter-card">
      <el-form inline :model="filter" @submit.prevent="reload">
        <el-form-item label="搜索">
          <el-input
            v-model="filter.q"
            placeholder="标题 / 原始标题"
            clearable
            :prefix-icon="Search"
            style="width: 220px"
            @keyup.enter="reload"
          />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="filter.type" placeholder="全部" clearable style="width: 130px">
            <el-option label="电影" value="movie" />
            <el-option label="剧集" value="tvshow" />
            <el-option label="动画" value="anime" />
            <el-option label="纪录片" value="documentary" />
          </el-select>
        </el-form-item>
        <el-form-item label="年份">
          <el-input-number v-model="filter.year" :min="1900" :max="2100" controls-position="right" style="width: 130px" />
        </el-form-item>
        <el-form-item label="最低评分">
          <el-input-number v-model="filter.min_rating" :min="0" :max="10" :step="0.5" :precision="1" controls-position="right" style="width: 130px" />
        </el-form-item>
        <el-form-item label="排序">
          <el-select v-model="filter.sort" style="width: 130px">
            <el-option label="添加时间" value="created_at" />
            <el-option label="评分" value="rating" />
            <el-option label="年份" value="year" />
            <el-option label="标题" value="title" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-radio-group v-model="filter.order">
            <el-radio-button label="desc">降序</el-radio-button>
            <el-radio-button label="asc">升序</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="reload">搜索</el-button>
          <el-button @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <div v-loading="loading" class="media-grid">
      <!-- Skeleton: 首次加载 -->
      <template v-if="loading && items.length === 0">
        <div v-for="n in 8" :key="`skel-${n}`" class="media-card skeleton">
          <div class="poster-card skeleton-poster"></div>
          <div class="skeleton-line" style="width: 80%"></div>
          <div class="skeleton-line" style="width: 50%"></div>
        </div>
      </template>

      <div v-if="!loading && items.length === 0" class="empty">
        <el-empty description="暂无媒资">
          <el-button type="primary" @click="$router.push('/media-create')">手动添加</el-button>
        </el-empty>
      </div>

      <div
        v-for="m in items"
        :key="m.id"
        class="media-card"
        @click="$router.push(`/media/${m.id}`)"
      >
        <div class="poster-card">
          <img v-if="m.poster_url" :src="m.poster_url" :alt="m.title" />
          <span v-else class="poster-placeholder">{{ m.title.slice(0, 2) }}</span>
          <div class="rating-badge" v-if="m.rating > 0">
            <el-icon><StarFilled /></el-icon>
            {{ m.rating.toFixed(1) }}
          </div>
          <div class="type-badge">{{ mediaTypeLabel(m.type) }}</div>
        </div>
        <div class="media-info">
          <div class="media-title" :title="m.title">{{ m.title }}</div>
          <div class="media-meta">
            <span v-if="m.year">{{ m.year }}</span>
            <span v-if="m.genres?.length" class="genre">{{ m.genres.slice(0, 2).join(' · ') }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="pagination" v-if="total > pageSize">
      <el-pagination
        v-model:current-page="filter.page"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next, jumper, total"
        background
        @current-change="reload"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Search, Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { mediaApi, type MediaListParams } from '@/api/media'
import { scannerApi, SCAN_INTERVAL_OPTIONS, type MediaScanConfig } from '@/api/client'
import type { MediaSummary } from '@/api/types'

const loading = ref(false)
const scanning = ref(false)
const scanConfig = ref<MediaScanConfig | null>(null)
const scanIntervalOptions = SCAN_INTERVAL_OPTIONS
const items = ref<MediaSummary[]>([])
const total = ref(0)
const pageSize = 24

const filter = reactive<MediaListParams & { page: number }>({
  page: 1,
  page_size: pageSize,
  type: '',
  q: '',
  sort: 'created_at',
  order: 'desc',
  year: undefined,
  min_rating: undefined,
})

async function reload() {
  loading.value = true
  try {
    const res = await mediaApi.list({ ...filter, page_size: pageSize })
    items.value = res.items
    total.value = res.total
    ElMessage.success(`加载了 ${res.items.length} 条媒资`)
  } catch (e: any) {
    ElMessage.error(`加载失败：${e?.response?.data?.message || e?.message || '未知错误'}`)
  } finally {
    loading.value = false
  }
}

function resetFilter() {
  Object.assign(filter, {
    page: 1,
    page_size: pageSize,
    type: '',
    q: '',
    sort: 'created_at',
    order: 'desc',
    year: undefined,
    min_rating: undefined,
  })
  reload()
}

function mediaTypeLabel(s: string) {
  return ({ movie: '电影', tvshow: '剧集', anime: '动画', documentary: '纪录片' } as any)[s] || s
}

function formatTime(iso: string) {
  return new Date(iso).toLocaleString('zh-CN')
}

async function loadScanConfig() {
  try {
    const resp = await scannerApi.getConfig()
    scanConfig.value = resp.data
  } catch {
    // ignore
  }
}

async function saveScanConfig(patch: Partial<{ enabled: boolean; interval_minutes: number }>) {
  if (!scanConfig.value) return
  const enabled = patch.enabled ?? scanConfig.value.enabled
  const interval_minutes = patch.interval_minutes ?? scanConfig.value.interval_minutes
  try {
    const resp = await scannerApi.updateConfig({ enabled, interval_minutes })
    scanConfig.value = resp.data
    ElMessage.success('扫描配置已保存')
  } catch (e: any) {
    ElMessage.error(e?.message || '保存失败')
    await loadScanConfig()
  }
}

async function onScanNow() {
  scanning.value = true
  try {
    const resp = await scannerApi.scan()
    const r = resp.data
    ElMessage.success(`扫描完成：新增 ${r.added}，共 ${r.total} 个文件`)
    await Promise.all([loadScanConfig(), reload()])
  } catch (e: any) {
    ElMessage.error(e?.message || '扫描失败')
    await loadScanConfig()
  } finally {
    scanning.value = false
  }
}

onMounted(() => {
  loadScanConfig()
  reload()
})
</script>

<style lang="scss" scoped>
.media-list-page {
  max-width: 1600px;
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

.scan-config-panel {
  margin-bottom: 20px;
  border-radius: 12px;
}

.scan-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.scan-config-row {
  display: flex;
  flex-wrap: wrap;
  gap: 20px 32px;
  align-items: flex-start;
}

.scan-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;

  .label {
    color: #64748b;
    font-size: 12px;
  }
}

.scan-roots .roots {
  color: #334155;
  font-family: var(--mh-font-mono, monospace);
  font-size: 12px;
}

.scan-last {
  flex: 1;
  min-width: 240px;
}

.status-tag {
  margin-left: 8px;
}

.scan-msg {
  display: block;
  margin-top: 4px;
  color: #64748b;
  font-size: 12px;
}

.text-muted {
  color: #94a3b8;
}

.filter-card {
  margin-bottom: 20px;
  border-radius: 12px;
}

.media-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 20px;
  min-height: 300px;
}

.media-card {
  cursor: pointer;
  transition: transform 0.2s;

  &:hover {
    transform: translateY(-4px);

    .poster-card {
      box-shadow: 0 8px 20px rgba(0, 0, 0, 0.15);
    }
  }
}

.poster-card {
  position: relative;
  aspect-ratio: 2/3;
  border-radius: 8px;
  overflow: hidden;
  background: linear-gradient(135deg, #1e293b, #0f172a);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  font-size: 24px;
  font-weight: 600;
  transition: box-shadow 0.2s;
}

.poster-placeholder {
  color: rgba(255, 255, 255, 0.3);
}

.poster-card img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.rating-badge {
  position: absolute;
  top: 8px;
  left: 8px;
  background: rgba(0, 0, 0, 0.65);
  color: #fbbf24;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 2px;
  font-weight: 600;
}

.type-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  background: rgba(99, 102, 241, 0.9);
  color: #fff;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
}

.media-info {
  margin-top: 8px;
}

.media-title {
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.media-meta {
  margin-top: 4px;
  font-size: 12px;
  color: #64748b;
  display: flex;
  gap: 8px;
}

.genre {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.pagination {
  margin-top: 32px;
  display: flex;
  justify-content: center;
}

.empty {
  grid-column: 1 / -1;
}

/* Skeleton 状态 */
.media-card.skeleton {
  cursor: default;

  &:hover {
    transform: none;
  }
}

.skeleton-poster {
  background: linear-gradient(90deg, #f1f5f9 0%, #e2e8f0 50%, #f1f5f9 100%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

.skeleton-line {
  height: 12px;
  margin-top: 8px;
  background: linear-gradient(90deg, #f1f5f9 0%, #e2e8f0 50%, #f1f5f9 100%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 4px;
}

@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
</style>
