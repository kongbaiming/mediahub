<template>
  <div v-loading="loading" class="detail-page">
    <header class="topbar">
      <button class="back-btn" @click="$router.back()">← 返回</button>
      <span class="breadcrumb">
        <span @click="$router.push('/')" class="link">首页</span>
        <span class="sep">/</span>
        <span>{{ media?.title }}</span>
      </span>
      <div class="actions" v-if="!isExternal">
        <button class="want-btn" :class="{ active: wantListed }" @click="toggleWant">
          {{ wantListed ? '✓ 想看' : '+ 想看' }}
        </button>
        <button class="fav-btn" :class="{ active: favorited }" @click="toggleFavorite">
          {{ favorited ? '★ 已收藏' : '☆ 收藏' }}
        </button>
      </div>
    </header>

    <div v-if="media" class="hero" :style="heroBg">
      <div class="hero-overlay"></div>
      <div class="hero-content">
        <div class="meta-top">
          <span v-if="isExternal" class="external-badge">未入库</span>
          <span class="type-badge">{{ typeLabel(media.type) }}</span>
          <span v-if="media.year">{{ media.year }}</span>
          <span v-if="contentRating" class="rating-badge">{{ contentRating }}</span>
          <span v-if="media.rating">⭐ {{ media.rating.toFixed(1) }}</span>
          <span v-if="media.runtime">{{ media.runtime }} 分钟</span>
        </div>
        <h1 class="title">
          {{ media.title }}
        </h1>
        <div v-if="media.original_title" class="original-title">{{ media.original_title }}</div>

        <div v-if="media.genres?.length" class="genres">
          <span v-for="g in media.genres" :key="g" class="genre-tag">{{ g }}</span>
        </div>

        <p v-if="media.overview" class="overview">{{ media.overview }}</p>

        <div v-if="!isExternal" class="actions-row">
          <button
            v-if="!isSeries"
            class="btn mh-btn mh-btn--primary"
            @click="$router.push(`/play/${media.id}`)"
          >
            ▶ 播放
          </button>
          <button
            v-else-if="firstEpisodeId"
            class="btn mh-btn mh-btn--primary"
            @click="$router.push(`/play/${media.id}?episode_id=${firstEpisodeId}`)"
          >
            ▶ 播放第 1 集
          </button>
        </div>
      </div>
    </div>

    <section v-if="media" class="info-section">
      <div v-if="castCredits.length" class="section">
        <h2 class="section-title">演职员</h2>
        <div class="credits-row">
          <button
            v-for="c in castCredits.slice(0, 16)"
            :key="c.id"
            type="button"
            class="credit-card"
            @click="openPerson(c)"
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
          </button>
        </div>
      </div>

      <div v-if="trailers.length" class="section">
        <h2 class="section-title">预告 / 花絮</h2>
        <div class="extras-row">
          <a
            v-for="ex in trailers"
            :key="ex.id"
            class="extra-card"
            :href="extraUrl(ex)"
            target="_blank"
            rel="noopener noreferrer"
          >
            <span class="extra-icon">▶</span>
            <span class="extra-title">{{ ex.title || extraTypeLabel(ex.extra_type) }}</span>
          </a>
        </div>
      </div>

      <div v-if="!isExternal && (media.video_codec || media.audio_codec || media.resolution || media.has_subtitle != null)" class="section">
        <h2 class="section-title">详细信息</h2>
        <div class="info-grid">
          <div v-if="media.video_codec" class="info-item">
            <span class="info-label">视频编码</span>
            <span class="info-value">{{ media.video_codec }}</span>
          </div>
          <div v-if="media.audio_codec" class="info-item">
            <span class="info-label">音频编码</span>
            <span class="info-value">{{ media.audio_codec }}</span>
          </div>
          <div v-if="media.resolution" class="info-item">
            <span class="info-label">分辨率</span>
            <span class="info-value">{{ media.resolution }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">字幕</span>
            <span class="info-value">{{ media.has_subtitle ? '含字幕' : '无字幕' }}</span>
          </div>
        </div>
      </div>

      <!-- 剧集选集 -->
      <div v-if="!isExternal && isSeries && seasonsWithEpisodes.length" class="section">
        <h2 class="section-title">选集</h2>
        <div v-for="season in seasonsWithEpisodes" :key="season.id" class="season-block">
          <h3 class="season-title">
            {{ season.title || `第 ${season.season_number} 季` }}
          </h3>
          <div class="episode-list">
            <button
              v-for="ep in season.episodes"
              :key="ep.id"
              class="episode-btn"
              @click="$router.push(`/play/${media.id}?episode_id=${ep.id}`)"
            >
              <span class="ep-num">第 {{ ep.episode_number }} 集</span>
              <span class="ep-title">{{ ep.title || `第 ${ep.episode_number} 集` }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- 相似推荐 -->
      <div v-if="!isExternal && similar.length > 0" class="section">
        <h2 class="section-title">相似推荐</h2>
        <div class="similar-row">
          <button
            v-for="m in similar"
            :key="m.id"
            type="button"
            class="similar-card"
            @click="openSimilar(m.id)"
          >
            <div class="poster-card">
              <img v-if="m.poster_url" :src="m.poster_url" :alt="m.title" loading="lazy" />
              <span v-else>{{ m.title.slice(0, 2) }}</span>
              <div v-if="m.rating > 0" class="rating">⭐ {{ m.rating.toFixed(1) }}</div>
            </div>
            <div class="card-title">{{ m.title }}</div>
            <div class="card-meta">{{ m.year }} · {{ typeLabel(m.type) }}</div>
          </button>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  mediaApi,
  catalogApi,
  libraryApi,
  recommendApi,
  historyApi,
  type MediaDetail,
  type MediaSummary,
  type EpisodeDetail,
  type SeasonDetail,
  type MediaCredit,
  type MediaExtra,
} from '@/api'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const media = ref<MediaDetail | null>(null)
const favorited = ref(false)
const wantListed = ref(false)
const similar = ref<MediaSummary[]>([])
const castCredits = ref<MediaCredit[]>([])
const contentRating = ref('')
const trailers = ref<MediaExtra[]>([])

const isTmdbDetail = computed(() => route.name === 'tmdb-detail')
const isExternal = computed(() => isTmdbDetail.value || !!media.value?.external)

const isSeries = computed(() => media.value?.type === 'tvshow' || media.value?.type === 'anime')

const seasonsWithEpisodes = computed((): Array<SeasonDetail & { episodes: EpisodeDetail[] }> => {
  if (!media.value?.seasons) return []
  return media.value.seasons
    .map((s) => ({
      ...s,
      episodes: (s.episodes || [])
        .filter((ep) => ep.file_path)
        .sort((a, b) => a.episode_number - b.episode_number),
    }))
    .filter((s) => s.episodes.length > 0)
    .sort((a, b) => a.season_number - b.season_number)
})

const episodes = computed((): EpisodeDetail[] => {
  const out: EpisodeDetail[] = []
  for (const s of seasonsWithEpisodes.value) {
    out.push(...s.episodes)
  }
  return out
})

const firstEpisodeId = computed(() => episodes.value[0]?.id)

const heroBg = computed(() => {
  const url = media.value?.backdrop_url || media.value?.poster_url
  return url ? { backgroundImage: `url(${url})` } : {}
})

async function loadLocal(id: string) {
  const data = await mediaApi.get(id)
  media.value = data

  const [simRes, credits, ratings, extras, wants, favs] = await Promise.all([
    recommendApi.similar(id, 12).catch(() => [] as MediaSummary[]),
    catalogApi.credits(id, 'actor').catch(() => [] as MediaCredit[]),
    catalogApi.ratings(id).catch(() => []),
    catalogApi.extras(id, 'trailer').catch(() => [] as MediaExtra[]),
    libraryApi.wantList().catch(() => []),
    libraryApi.favoritesList().catch(() => []),
  ])

  similar.value = simRes.filter((m) => m.id !== id)
  castCredits.value = credits
  contentRating.value = ratings[0]?.rating || ''
  trailers.value = extras
  wantListed.value = wants.some((w) => w.media_id === id)
  favorited.value = favs.some((f) => f.media_id === id)
}

async function loadTmdb(type: string, tmdbId: number) {
  const data = await mediaApi.getTmdb(type, tmdbId)
  if (data.local_media_id) {
    await router.replace(`/media/${data.local_media_id}`)
    return
  }
  media.value = {
    id: '',
    title: data.title,
    original_title: data.original_title,
    year: data.year,
    type: data.type,
    rating: data.rating,
    poster_url: data.poster_url,
    backdrop_url: data.backdrop_url,
    overview: data.overview,
    runtime: data.runtime,
    genres: data.genres || [],
    external: true,
    tmdb_id: data.tmdb_id,
  }
  castCredits.value = data.credits || []
  similar.value = []
  contentRating.value = ''
  trailers.value = []
  wantListed.value = false
  favorited.value = false
}

async function load() {
  loading.value = true
  try {
    await historyApi.ensureProfileId()
    if (isTmdbDetail.value) {
      await loadTmdb(route.params.type as string, Number(route.params.tmdbId))
    } else {
      await loadLocal(route.params.id as string)
    }
  } finally {
    loading.value = false
  }
}

async function toggleWant() {
  if (!media.value) return
  try {
    if (wantListed.value) {
      await libraryApi.removeWant(media.value.id)
      wantListed.value = false
      window.toast?.('已从想看移除', 'info', 2000)
    } else {
      await libraryApi.addWant(media.value.id)
      wantListed.value = true
      window.toast?.('已加入想看', 'success', 2000)
    }
  } catch {
    // ignore
  }
}

async function toggleFavorite() {
  if (!media.value) return
  try {
    const { added } = await libraryApi.toggleFavorite(media.value.id)
    favorited.value = added
    window.toast?.(added ? '已加入收藏' : '已取消收藏', 'success', 2000)
  } catch {
    // ignore
  }
}

function creditAvatar(c: MediaCredit) {
  const p = c.person
  if (!p) return ''
  if (p.profile_url) return p.profile_url
  if (p.profile_path) return profileImage(p.profile_path)
  return ''
}

function openPerson(c: MediaCredit) {
  const personId = c.person?.id
  const tmdbPersonId = c.person?.tmdb_person_id
  const query: Record<string, string | undefined> = {
    role: c.character_name || undefined,
  }
  if (!isExternal.value && media.value?.id) {
    query.from = media.value.id
  } else if (isTmdbDetail.value && media.value?.tmdb_id) {
    query.from_tmdb_type = route.params.type as string
    query.from_tmdb_id = String(media.value.tmdb_id)
  }
  if (personId) {
    router.push({ path: `/person/${personId}`, query })
    return
  }
  if (tmdbPersonId) {
    router.push({ path: `/person/tmdb/${tmdbPersonId}`, query })
  }
}

function openSimilar(mediaId: string) {
  if (mediaId === route.params.id) return
  router.push(`/media/${mediaId}`)
}

function profileImage(path: string) {
  if (path.startsWith('http')) return path
  return `https://image.tmdb.org/t/p/w185${path}`
}

function extraUrl(ex: MediaExtra) {
  if (ex.external_url) return ex.external_url
  return '#'
}

function extraTypeLabel(t: string) {
  return ({ trailer: '预告片', teaser: '先导', clip: '片段' } as Record<string, string>)[t] || t
}

function typeLabel(t: string) {
  return ({ movie: '电影', tvshow: '剧集', anime: '动画', documentary: '纪录片' } as any)[t] || t
}

watch(
  () => [route.name, route.params.id, route.params.type, route.params.tmdbId],
  (cur, prev) => {
    if (!cur[0]) return
    if (prev && cur.every((v, i) => v === prev[i])) return
    window.scrollTo({ top: 0, behavior: 'instant' })
    load()
  },
)

onMounted(load)
</script>

<style lang="scss" scoped>
.detail-page {
  min-height: 100vh;
  background: var(--mh-bg, #0a0a12);
  color: var(--mh-text, #f0f0f5);
}

.topbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 60px;
  background: rgba(15, 23, 42, 0.95);
  backdrop-filter: blur(20px);
  z-index: 100;
  display: flex;
  align-items: center;
  padding: 0 40px;
  gap: 16px;
}

.back-btn {
  background: rgba(255, 255, 255, 0.1);
  border: none;
  color: #fff;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;

  &:hover { background: rgba(255, 255, 255, 0.2); }
}

.breadcrumb {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #94a3b8;

  .link {
    cursor: pointer;
    color: #cbd5e1;

    &:hover { color: #fff; }
  }

  .sep {
    color: #475569;
  }
}

.actions {
  display: flex;
  gap: 8px;
}

.fav-btn {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid var(--mh-outline, rgba(255, 255, 255, 0.08));
  color: var(--mh-text, #fff);
  padding: 8px 16px;
  border-radius: 10px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: background var(--mh-duration, 200ms) ease;

  &.active {
    background: rgba(251, 191, 36, 0.15);
    color: var(--mh-warning, #fbbf24);
    border-color: rgba(251, 191, 36, 0.35);
  }

  &:hover { background: rgba(255, 255, 255, 0.1); }
}

.want-btn {
  background: rgba(108, 99, 255, 0.12);
  border: 1px solid rgba(108, 99, 255, 0.25);
  color: #c4bfff;
  padding: 8px 16px;
  border-radius: 10px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;

  &.active {
    background: var(--mh-primary-muted, rgba(108, 99, 255, 0.2));
    color: #fff;
    border-color: var(--mh-primary, #6c63ff);
  }

  &:hover { background: rgba(108, 99, 255, 0.2); }
}

.rating-badge {
  background: rgba(255, 255, 255, 0.12);
  border: 1px solid rgba(255, 255, 255, 0.2);
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.04em;
}

.credits-row {
  display: flex;
  gap: var(--mh-space-4, 16px);
  overflow-x: auto;
  padding-bottom: var(--mh-space-2, 8px);
}

.credit-card {
  flex-shrink: 0;
  width: 96px;
  text-align: center;
  border: none;
  background: transparent;
  padding: 0;
  cursor: pointer;
  color: inherit;
  font: inherit;

  &:hover .credit-avatar {
    border-color: var(--mh-primary, #6c63ff);
  }
}

.credit-avatar {
  width: 72px;
  height: 72px;
  margin: 0 auto var(--mh-space-2, 8px);
  border-radius: var(--mh-radius-full, 9999px);
  overflow: hidden;
  background: var(--mh-surface-variant, #1e1e2e);
  border: 1px solid var(--mh-outline, rgba(255, 255, 255, 0.08));
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: 600;
  color: var(--mh-text-muted, #6b6b80);

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.credit-name {
  font-size: 12px;
  font-weight: 600;
  color: var(--mh-text, #f0f0f5);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.credit-role {
  font-size: 11px;
  color: var(--mh-text-muted, #6b6b80);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.extras-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--mh-space-3, 12px);
}

.extra-card {
  display: inline-flex;
  align-items: center;
  gap: var(--mh-space-2, 8px);
  padding: var(--mh-space-3, 12px) var(--mh-space-4, 16px);
  background: rgba(108, 99, 255, 0.12);
  border: 1px solid rgba(108, 99, 255, 0.25);
  border-radius: var(--mh-radius-md, 12px);
  color: var(--mh-text, #f0f0f5);
  text-decoration: none;
  transition: background var(--mh-duration, 200ms) ease;

  &:hover {
    background: rgba(108, 99, 255, 0.2);
  }
}

.extra-icon {
  color: var(--mh-primary, #6c63ff);
  font-size: 12px;
}

.extra-title {
  font-size: 14px;
  font-weight: 500;
}

.hero {
  position: relative;
  height: 70vh;
  min-height: 480px;
  background-size: cover;
  background-position: center;
  display: flex;
  align-items: flex-end;
  padding: 0 80px 60px;
  margin-top: 60px;
}

.hero-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    180deg,
    rgba(15, 23, 42, 0.3) 0%,
    rgba(15, 23, 42, 0.4) 40%,
    rgba(15, 23, 42, 0.9) 80%,
    rgba(15, 23, 42, 1) 100%
  );
}

.hero-content {
  position: relative;
  z-index: 1;
  max-width: 800px;
  color: #fff;
}

.meta-top {
  display: flex;
  gap: 12px;
  font-size: 13px;
  color: #cbd5e1;
  margin-bottom: 12px;
  align-items: center;
}

.type-badge {
  background: rgba(99, 102, 241, 0.8);
  color: #fff;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 12px;
}

.external-badge {
  background: rgba(251, 191, 36, 0.2);
  border: 1px solid rgba(251, 191, 36, 0.45);
  color: #fcd34d;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.title {
  font-size: 48px;
  font-weight: 800;
  margin: 0 0 8px;
  text-shadow: 0 2px 12px rgba(0, 0, 0, 0.6);
}

.original-title {
  font-size: 16px;
  color: #94a3b8;
  margin-bottom: 16px;
}

.genres {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.genre-tag {
  background: rgba(255, 255, 255, 0.1);
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  color: #cbd5e1;
}

.overview {
  font-size: 15px;
  line-height: 1.6;
  color: #cbd5e1;
  margin-bottom: 24px;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.actions-row {
  display: flex;
  gap: 12px;
}

.btn {
  height: 44px;
  padding: 0 24px;
  border-radius: 6px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  border: none;

  &-primary {
    background: #fff;
    color: #0f172a;
    &:hover { background: #e2e8f0; }
  }
  &-secondary {
    background: rgba(255, 255, 255, 0.2);
    color: #fff;
    backdrop-filter: blur(10px);
    &:hover { background: rgba(255, 255, 255, 0.3); }
  }
}

.info-section {
  padding: 60px 80px 80px;
  max-width: 1400px;
  margin: 0 auto;
}

.section {
  margin-bottom: 48px;
}

.section-title {
  font-size: 22px;
  font-weight: 600;
  margin: 0 0 20px;
  color: #fff;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
  padding: 20px;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.info-label {
  display: block;
  font-size: 12px;
  color: #94a3b8;
  margin-bottom: 4px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.info-value {
  font-size: 14px;
  color: #e2e8f0;
  font-weight: 500;
}

.similar-row {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 16px;
}

.similar-card {
  border: none;
  background: transparent;
  padding: 0;
  text-align: left;
  color: inherit;
  font: inherit;
  cursor: pointer;
  transition: transform 0.2s;

  &:hover {
    transform: translateY(-4px);
  }
}

.similar-card .poster-card {
  aspect-ratio: 2/3;
  background: linear-gradient(135deg, #1e293b, #0f172a);
  border-radius: 8px;
  overflow: hidden;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.3);
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 8px;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.similar-card .rating {
  position: absolute;
  top: 6px;
  left: 6px;
  background: rgba(0, 0, 0, 0.7);
  color: #fbbf24;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.card-title {
  font-size: 13px;
  font-weight: 500;
  color: #e2e8f0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-meta {
  font-size: 11px;
  color: #94a3b8;
}

.season-block {
  margin-bottom: 24px;
}

.season-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--mh-text-secondary, #cbd5e1);
  margin: 0 0 12px;
}

.episode-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.episode-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  text-align: left;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  color: #e2e8f0;
  cursor: pointer;

  &:hover {
    background: rgba(99, 102, 241, 0.15);
    border-color: rgba(99, 102, 241, 0.35);
  }
}

.ep-num {
  flex-shrink: 0;
  font-size: 13px;
  color: #94a3b8;
  min-width: 72px;
}

.ep-title {
  font-size: 14px;
}
</style>
