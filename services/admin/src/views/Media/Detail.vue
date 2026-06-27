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
            <img v-if="posterSrc" :src="posterSrc" :alt="media.title" />
            <span v-else>{{ media.title.slice(0, 2) }}</span>
          </div>
          <el-upload
            class="poster-upload"
            :show-file-list="false"
            accept="image/jpeg,image/png,image/webp"
            :disabled="posterUploading"
            :http-request="onPosterUpload"
          >
            <el-button size="small" :loading="posterUploading" style="width: 100%; margin-top: 12px">
              <el-icon><Upload /></el-icon>
              {{ posterUploading ? '上传中…' : '上传/替换海报' }}
            </el-button>
          </el-upload>
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
              <el-tag v-if="media.availability_status" :type="availTagType(media.availability_status)" size="large">
                {{ availLabel(media.availability_status) }}
              </el-tag>
              <el-tag v-for="r in contentRatings" :key="r.rating + r.country" size="large" type="warning">
                {{ r.rating }}
              </el-tag>
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
            <el-form-item label="存储路径">
              <el-input
                v-model="editForm.storage_path"
                placeholder="/media/movies/片名 (2010)/片名.mkv（容器内路径，勿填 /volume1/...）"
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="onSave" :loading="saving">保存</el-button>
              <el-button @click="editing = false">取消</el-button>
            </el-form-item>
          </el-form>
        </div>
      </div>

      <el-card v-if="showScrapeMatch" shadow="never" class="scrape-match-card">
        <template #header>
          <div class="match-header">
            <span>TMDB 候选匹配</span>
            <el-button size="small" :loading="candidatesLoading" @click="loadCandidates">
              刷新候选
            </el-button>
          </div>
        </template>
        <p class="match-hint">刮削未成功时，从下列候选中选择正确条目，无需手动输入标题。</p>
        <div v-loading="candidatesLoading" class="candidate-grid">
          <div
            v-for="c in scrapeCandidates"
            :key="`${c.type}-${c.tmdb_id}`"
            class="candidate-card"
          >
            <div class="candidate-poster">
              <img v-if="c.poster_url" :src="c.poster_url" :alt="c.title" />
              <span v-else>{{ c.title.slice(0, 2) }}</span>
            </div>
            <div class="candidate-body">
              <div class="candidate-title">
                {{ c.title }}
                <span v-if="c.year" class="candidate-year">({{ c.year }})</span>
              </div>
              <div class="candidate-meta">
                <el-tag size="small" type="info">{{ c.type === 'tv' ? '剧集' : '电影' }}</el-tag>
                <span v-if="c.runtime"> {{ c.runtime }} 分钟</span>
                <span v-if="c.rating"> · {{ c.rating.toFixed(1) }}</span>
              </div>
              <p v-if="c.overview" class="candidate-overview">{{ c.overview }}</p>
              <el-button
                type="primary"
                size="small"
                :loading="applyingKey === `${c.type}-${c.tmdb_id}`"
                @click="onApplyCandidate(c)"
              >
                确认匹配
              </el-button>
            </div>
          </div>
          <el-empty v-if="!candidatesLoading && scrapeCandidates.length === 0" description="暂无候选，可尝试刷新或修改文件夹名后重新刮削" />
        </div>
      </el-card>

      <el-card v-if="castCredits.length" shadow="never" class="credits-card">
        <template #header><span>演职员</span></template>
        <div class="credits-row">
          <div
            v-for="c in castCredits.slice(0, 20)"
            :key="c.id"
            class="credit-item"
            role="button"
            tabindex="0"
            @click="openCreditDetail(c)"
            @keyup.enter="openCreditDetail(c)"
          >
            <div class="credit-avatar">
              <img
                v-if="creditAvatar(c)"
                :src="creditAvatar(c)"
                :alt="c.person?.name"
                loading="lazy"
              />
              <span v-else>{{ c.person?.name?.slice(0, 1) || '?' }}</span>
            </div>
            <div class="credit-name">{{ c.person?.name }}</div>
            <div v-if="c.character_name" class="credit-role">饰 {{ c.character_name }}</div>
          </div>
        </div>
      </el-card>

      <el-dialog
        v-model="creditDialogVisible"
        :title="selectedCredit?.person?.name || '演职员'"
        width="520px"
        destroy-on-close
      >
        <div v-if="selectedCredit?.person" class="credit-detail">
          <div class="credit-detail-head">
            <div class="credit-detail-avatar">
              <img
                v-if="creditAvatar(selectedCredit)"
                :src="creditAvatar(selectedCredit)"
                :alt="selectedCredit.person.name"
              />
              <span v-else>{{ selectedCredit.person.name?.slice(0, 1) || '?' }}</span>
            </div>
            <div class="credit-detail-meta">
              <div v-if="selectedCredit.character_name" class="credit-detail-role">
                饰 {{ selectedCredit.character_name }}
              </div>
              <div v-if="selectedCredit.person.known_for_department" class="credit-detail-dept">
                {{ selectedCredit.person.known_for_department }}
              </div>
              <div v-if="personMeta(selectedCredit.person)" class="credit-detail-extra">
                {{ personMeta(selectedCredit.person) }}
              </div>
            </div>
          </div>
          <p v-if="selectedCredit.person.biography" class="credit-detail-bio">
            {{ selectedCredit.person.biography }}
          </p>
          <el-empty v-else description="暂无人物介绍" :image-size="64" />
        </div>
      </el-dialog>

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
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { mediaApi, type ScrapeCandidate } from '@/api/media'
import { catalogApi, type MediaCredit, type ContentRating } from '@/api/catalog'
import type { MediaDetail } from '@/api/types'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const rescanning = ref(false)
const saving = ref(false)
const editing = ref(false)
const media = ref<MediaDetail | null>(null)
const castCredits = ref<MediaCredit[]>([])
const contentRatings = ref<ContentRating[]>([])
const creditDialogVisible = ref(false)
const selectedCredit = ref<MediaCredit | null>(null)
const scrapeCandidates = ref<ScrapeCandidate[]>([])
const candidatesLoading = ref(false)
const applyingKey = ref('')
const posterUploading = ref(false)
const posterCacheBust = ref(0)

const posterSrc = computed(() => {
  const url = media.value?.poster_url
  if (!url) return ''
  if (url.includes('?')) return url
  if (url.startsWith('/api/')) return `${url}${posterCacheBust.value ? `?t=${posterCacheBust.value}` : ''}`
  return url
})

const showScrapeMatch = computed(() => {
  const s = media.value?.scrape_status
  return s === 'failed' || s === 'pending'
})
const editForm = reactive({
  title: '',
  original_title: '',
  year: undefined as number | undefined,
  rating: 0,
  genresStr: '',
  overview: '',
  storage_path: '',
})

function creditAvatar(c: MediaCredit) {
  const p = c.person
  if (!p) return ''
  if (p.profile_url) return p.profile_url
  if (p.profile_path?.startsWith('http')) return p.profile_path
  if (p.profile_path) return `https://image.tmdb.org/t/p/w185${p.profile_path}`
  return ''
}

function openCreditDetail(c: MediaCredit) {
  selectedCredit.value = c
  creditDialogVisible.value = true
}

function personMeta(p: NonNullable<MediaCredit['person']>) {
  const parts: string[] = []
  if (p.place_of_birth) parts.push(p.place_of_birth)
  if (p.birthday) parts.push(p.birthday.slice(0, 10))
  return parts.join(' · ')
}

async function load() {
  const id = route.params.id as string
  loading.value = true
  try {
    const res = await mediaApi.get(id)
    media.value = res.data
    const [credits, ratings] = await Promise.all([
      catalogApi.credits(id, 'actor').catch(() => ({ data: [] as MediaCredit[] })),
      catalogApi.ratings(id).catch(() => ({ data: [] as ContentRating[] })),
    ])
    castCredits.value = credits.data || []
    contentRatings.value = ratings.data || []
    Object.assign(editForm, {
      title: res.data.title,
      original_title: res.data.original_title || '',
      year: res.data.year,
      rating: res.data.rating,
      genresStr: (res.data.genres || []).join(', '),
      overview: res.data.overview || '',
      storage_path: res.data.storage_path || '',
    })
    if (showScrapeMatch.value) {
      loadCandidates()
    } else {
      scrapeCandidates.value = []
    }
  } finally {
    loading.value = false
  }
}

async function onSave() {
  if (!editForm.title?.trim()) {
    ElMessage.warning('标题不能为空')
    return
  }
  if (!editForm.storage_path?.trim()) {
    ElMessage.warning('存储路径不能为空')
    return
  }
  saving.value = true
  try {
    const id = media.value!.id
    await mediaApi.update(id, {
      title: editForm.title.trim(),
      original_title: editForm.original_title,
      year: editForm.year ?? null,
      rating: editForm.rating,
      genres: editForm.genresStr.split(',').map((s) => s.trim()).filter(Boolean),
      overview: editForm.overview,
      storage_path: editForm.storage_path.trim(),
    } as any)
    ElMessage.success('保存成功，重新刮削不会覆盖已修改的标题')
    editing.value = false
    await load()
  } catch {
    // 错误由 axios 拦截器提示
  } finally {
    saving.value = false
  }
}

async function onPosterUpload(options: { file: File }) {
  if (!media.value) return
  const file = options.file
  if (!file.type.startsWith('image/')) {
    ElMessage.warning('请选择图片文件')
    return
  }
  if (file.size > 10 * 1024 * 1024) {
    ElMessage.warning('海报不能超过 10MB')
    return
  }
  posterUploading.value = true
  try {
    const res = await mediaApi.uploadPoster(media.value.id, file)
    posterCacheBust.value = Date.now()
    if (media.value && res.data?.poster_url) {
      media.value.poster_url = res.data.poster_url
    }
    ElMessage.success('海报已更新')
    await load()
  } catch {
    // 错误由 axios 拦截器提示
  } finally {
    posterUploading.value = false
  }
}

async function onRescan() {
  rescanning.value = true
  try {
    await mediaApi.rescan(media.value!.id)
    ElMessage.success('已加入刮削队列')
    await load()
  } finally {
    rescanning.value = false
  }
}

async function loadCandidates() {
  if (!media.value) return
  candidatesLoading.value = true
  try {
    const res = await mediaApi.scrapeCandidates(media.value.id)
    scrapeCandidates.value = res.data || []
  } catch {
    scrapeCandidates.value = []
  } finally {
    candidatesLoading.value = false
  }
}

async function onApplyCandidate(c: ScrapeCandidate) {
  if (!media.value) return
  const key = `${c.type}-${c.tmdb_id}`
  applyingKey.value = key
  try {
    await mediaApi.applyScrapeMatch(media.value.id, {
      tmdb_id: c.tmdb_id,
      type: c.type,
    })
    ElMessage.success('匹配成功')
    await load()
  } catch {
    // 错误由 axios 拦截器提示
  } finally {
    applyingKey.value = ''
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

function availLabel(s: string) {
  return ({
    available: '可播放',
    processing: '处理中',
    missing: '缺文件',
    unreleased: '未发布',
  } as Record<string, string>)[s] || s
}

function availTagType(s: string): 'success' | 'warning' | 'danger' | 'info' {
  return ({
    available: 'success',
    processing: 'warning',
    missing: 'danger',
    unreleased: 'info',
  } as Record<string, string>)[s] as any || 'info'
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

.scrape-match-card {
  margin-top: 16px;
  border-radius: 12px;
}

.match-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.match-hint {
  margin: 0 0 16px;
  font-size: 13px;
  color: #64748b;
}

.candidate-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.candidate-card {
  display: flex;
  gap: 12px;
  padding: 12px;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  background: #fff;
}

.candidate-poster {
  width: 72px;
  flex-shrink: 0;
  aspect-ratio: 2/3;
  border-radius: 6px;
  overflow: hidden;
  background: #1e293b;
  color: rgba(255, 255, 255, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.candidate-title {
  font-weight: 600;
  color: #1e293b;
  line-height: 1.3;
}

.candidate-year {
  color: #94a3b8;
  font-weight: 400;
}

.candidate-meta {
  margin: 6px 0;
  font-size: 12px;
  color: #64748b;
}

.candidate-overview {
  margin: 0 0 10px;
  font-size: 12px;
  color: #64748b;
  line-height: 1.5;
}

.seasons-card {
  margin-top: 16px;
  border-radius: 12px;
}

.credits-card {
  margin-top: 16px;
  border-radius: 12px;
}

.credits-row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

.credit-item {
  width: 88px;
  text-align: center;
  cursor: pointer;
  border-radius: 10px;
  padding: 4px;
  transition: background 0.15s;

  &:hover {
    background: rgba(108, 99, 255, 0.08);
  }
}

.credit-avatar {
  width: 64px;
  height: 64px;
  margin: 0 auto 6px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--mh-primary, #6c63ff), #ec4899);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 20px;
  overflow: hidden;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.credit-name {
  font-size: 12px;
  font-weight: 600;
  color: var(--mh-text, #1a1a28);
}

.credit-role {
  font-size: 11px;
  color: var(--mh-text-secondary, #64748b);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.credit-detail-head {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.credit-detail-avatar {
  flex-shrink: 0;
  width: 96px;
  height: 96px;
  border-radius: 12px;
  overflow: hidden;
  background: linear-gradient(135deg, var(--mh-primary, #6c63ff), #ec4899);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  font-weight: 600;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.credit-detail-meta {
  flex: 1;
  min-width: 0;
  padding-top: 4px;
}

.credit-detail-role {
  font-size: 15px;
  font-weight: 600;
  color: var(--mh-text, #1a1a28);
  margin-bottom: 6px;
}

.credit-detail-dept {
  font-size: 13px;
  color: var(--mh-text-secondary, #64748b);
  margin-bottom: 4px;
}

.credit-detail-extra {
  font-size: 12px;
  color: var(--mh-text-secondary, #64748b);
}

.credit-detail-bio {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: var(--mh-text, #334155);
  white-space: pre-wrap;
  max-height: 360px;
  overflow-y: auto;
}
</style>
