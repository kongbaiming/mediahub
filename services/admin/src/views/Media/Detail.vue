<template>
  <div v-loading="loading" class="media-detail">
    <div class="detail-header">
      <el-button text @click="$router.back()">
        <el-icon><ArrowLeft /></el-icon>
        返回
      </el-button>
      <div class="header-actions">
        <el-button @click="onRescan" :loading="rescanning">
          <el-icon><Refresh /></el-icon>
          重新刮削
        </el-button>
        <el-button @click="editing = !editing">
          <el-icon><Edit /></el-icon>
          {{ editing ? '取消编辑' : '编辑' }}
        </el-button>
        <el-button type="danger" @click="onDelete">
          <el-icon><Delete /></el-icon>
          删除
        </el-button>
      </div>
    </div>

    <div v-if="media" class="detail-content">
      <div class="detail-top">
        <div class="poster-section">
          <div class="poster-card poster-large">
            <img v-if="media.poster_url" :src="media.poster_url" :alt="media.title" />
            <span v-else>{{ media.title.slice(0, 2) }}</span>
          </div>
        </div>

        <div class="info-section">
          <div v-if="!editing">
            <h1 class="title">
              {{ media.title }}
              <span v-if="media.year" class="year">({{ media.year }})</span>
            </h1>
            <div v-if="media.original_title" class="original-title">{{ media.original_title }}</div>

            <div class="meta-row">
              <el-rate
                v-if="media.rating"
                :model-value="media.rating / 2"
                disabled
                show-score
                :show-text="false"
                size="large"
              />
              <el-tag size="large" type="primary">{{ mediaTypeLabel(media.type) }}</el-tag>
              <el-tag v-for="g in media.genres" :key="g" size="large">{{ g }}</el-tag>
              <el-tag v-if="media.has_subtitle" size="large" type="success">含字幕</el-tag>
            </div>

            <div class="info-grid">
              <div v-if="media.runtime" class="info-item">
                <span class="info-label">时长</span>
                <span class="info-value">{{ media.runtime }} 分钟</span>
              </div>
              <div v-if="media.resolution" class="info-item">
                <span class="info-label">分辨率</span>
                <span class="info-value">{{ media.resolution }}</span>
              </div>
              <div v-if="media.video_codec" class="info-item">
                <span class="info-label">视频编码</span>
                <span class="info-value">{{ media.video_codec }}</span>
              </div>
              <div v-if="media.audio_codec" class="info-item">
                <span class="info-label">音频编码</span>
                <span class="info-value">{{ media.audio_codec }}</span>
              </div>
              <div class="info-item">
                <span class="info-label">文件大小</span>
                <span class="info-value">{{ formatSize(media.file_size) }}</span>
              </div>
              <div v-if="media.tmdb_id" class="info-item">
                <span class="info-label">TMDB ID</span>
                <span class="info-value">{{ media.tmdb_id }}</span>
              </div>
            </div>

            <div v-if="media.overview" class="overview">
              <h3>简介</h3>
              <p>{{ media.overview }}</p>
            </div>

            <div class="storage-info">
              <el-text size="small" type="info">
                <el-icon><Folder /></el-icon>
                {{ media.storage_path }}
              </el-text>
            </div>

            <div class="scrape-status">
              <el-tag :type="scrapeStatusColor(media.scrape_status)">
                {{ scrapeStatusLabel(media.scrape_status) }}
              </el-tag>
              <span v-if="media.scrape_error" class="scrape-error">{{ media.scrape_error }}</span>
            </div>
          </div>

          <el-form v-else label-position="top" class="edit-form">
            <el-form-item label="标题">
              <el-input v-model="editForm.title" />
            </el-form-item>
            <el-form-item label="原始标题">
              <el-input v-model="editForm.original_title" />
            </el-form-item>
            <el-form-item label="年份">
              <el-input-number v-model="editForm.year" :min="1900" :max="2100" controls-position="right" />
            </el-form-item>
            <el-form-item label="评分">
              <el-input-number v-model="editForm.rating" :min="0" :max="10" :step="0.1" :precision="1" controls-position="right" />
            </el-form-item>
            <el-form-item label="类型（标签，逗号分隔）">
              <el-input v-model="editForm.genresStr" />
            </el-form-item>
            <el-form-item label="简介">
              <el-input v-model="editForm.overview" type="textarea" :rows="5" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="onSave" :loading="saving">保存</el-button>
              <el-button @click="editing = false">取消</el-button>
            </el-form-item>
          </el-form>
        </div>
      </div>

      <el-card v-if="media.seasons?.length" shadow="never" class="seasons-card">
        <template #header><span>季 / 集</span></template>
        <el-collapse>
          <el-collapse-item
            v-for="s in media.seasons"
            :key="s.id"
            :title="`第 ${s.season_number} 季${s.title ? '：' + s.title : ''} (${s.episodes?.length || 0} 集)`"
          >
            <el-table :data="s.episodes || []" size="small">
              <el-table-column label="集" prop="episode_number" width="80" />
              <el-table-column label="标题" prop="title" />
              <el-table-column label="时长" width="120">
                <template #default="{ row }">
                  {{ row.duration ? `${row.duration}s` : '-' }}
                </template>
              </el-table-column>
              <el-table-column label="文件" width="80">
                <template #default="{ row }">
                  <el-icon v-if="row.file_path" color="green"><CircleCheck /></el-icon>
                </template>
              </el-table-column>
            </el-table>
          </el-collapse-item>
        </el-collapse>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { mediaApi } from '@/api/media'
import type { MediaDetail } from '@/api/types'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const rescanning = ref(false)
const saving = ref(false)
const editing = ref(false)
const media = ref<MediaDetail | null>(null)
const editForm = reactive({
  title: '',
  original_title: '',
  year: undefined as number | undefined,
  rating: 0,
  genresStr: '',
  overview: '',
})

async function load() {
  const id = route.params.id as string
  loading.value = true
  try {
    const res = await mediaApi.get(id)
    media.value = res.data
    Object.assign(editForm, {
      title: res.data.title,
      original_title: res.data.original_title || '',
      year: res.data.year,
      rating: res.data.rating,
      genresStr: (res.data.genres || []).join(', '),
      overview: res.data.overview || '',
    })
  } finally {
    loading.value = false
  }
}

async function onSave() {
  saving.value = true
  try {
    const id = media.value!.id
    await mediaApi.update(id, {
      title: editForm.title,
      original_title: editForm.original_title,
      year: editForm.year,
      rating: editForm.rating,
      genres: editForm.genresStr.split(',').map((s) => s.trim()).filter(Boolean),
      overview: editForm.overview,
    } as any)
    ElMessage.success('保存成功')
    editing.value = false
    await load()
  } finally {
    saving.value = false
  }
}

async function onRescan() {
  rescanning.value = true
  try {
    await mediaApi.rescan(media.value!.id)
    ElMessage.success('已加入刮削队列')
  } finally {
    rescanning.value = false
  }
}

async function onDelete() {
  try {
    await ElMessageBox.confirm(`确认删除「${media.value!.title}」？此操作不可恢复`, '删除确认', {
      type: 'warning',
    })
    await mediaApi.delete(media.value!.id)
    ElMessage.success('已删除')
    router.back()
  } catch {
    // user cancelled
  }
}

function mediaTypeLabel(s: string) {
  return ({ movie: '电影', tvshow: '剧集', anime: '动画', documentary: '纪录片' } as any)[s] || s
}
function scrapeStatusLabel(s: string) {
  return ({ pending: '待刮削', scraping: '刮削中', done: '已完成', failed: '失败' } as any)[s] || s
}
function scrapeStatusColor(s: string): any {
  return ({ done: 'success', scraping: 'warning', failed: 'danger' } as any)[s] || 'info'
}
function formatSize(bytes: number) {
  if (!bytes) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = bytes
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(2)} ${units[i]}`
}

onMounted(load)
</script>

<style lang="scss" scoped>
.media-detail {
  max-width: 1400px;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.detail-top {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 32px;
  margin-bottom: 24px;
}

.poster-large {
  width: 100%;
  aspect-ratio: 2/3;
  background: linear-gradient(135deg, #1e293b, #0f172a);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.3);
  font-size: 48px;
  font-weight: 600;
  overflow: hidden;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.title {
  margin: 0;
  font-size: 28px;
  font-weight: 700;
  color: #1e293b;
}

.year {
  color: #94a3b8;
  font-weight: 400;
  font-size: 20px;
  margin-left: 4px;
}

.original-title {
  margin-top: 4px;
  font-size: 15px;
  color: #64748b;
}

.meta-row {
  margin-top: 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.info-grid {
  margin-top: 24px;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 12px;
  padding: 16px;
  background: #f8fafc;
  border-radius: 8px;
}

.info-label {
  display: block;
  font-size: 12px;
  color: #64748b;
  margin-bottom: 4px;
}

.info-value {
  font-size: 14px;
  color: #1e293b;
  font-weight: 500;
}

.overview {
  margin-top: 24px;

  h3 {
    font-size: 15px;
    margin: 0 0 8px;
    color: #1e293b;
  }
  p {
    margin: 0;
    line-height: 1.7;
    color: #475569;
    font-size: 14px;
  }
}

.storage-info {
  margin-top: 16px;
}

.scrape-status {
  margin-top: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.scrape-error {
  font-size: 12px;
  color: #ef4444;
}

.seasons-card {
  margin-top: 16px;
  border-radius: 12px;
}

.edit-form {
  max-width: 600px;
}
</style>
