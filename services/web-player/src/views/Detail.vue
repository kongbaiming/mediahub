<template>
  <div v-loading="loading" class="detail-page">
    <header class="topbar">
      <button class="back-btn" @click="$router.back()">← 返回</button>
      <span class="breadcrumb">
        <span @click="$router.push('/')" class="link">首页</span>
        <span class="sep">/</span>
        <span>{{ media?.title }}</span>
      </span>
      <div class="actions">
        <button class="fav-btn" :class="{ active: favorited }" @click="toggleFavorite">
          {{ favorited ? '★ 已收藏' : '☆ 收藏' }}
        </button>
      </div>
    </header>

    <div v-if="media" class="hero" :style="heroBg">
      <div class="hero-overlay"></div>
      <div class="hero-content">
        <div class="meta-top">
          <span class="type-badge">{{ typeLabel(media.type) }}</span>
          <span v-if="media.year">{{ media.year }}</span>
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

        <div class="actions-row">
          <button class="btn btn-primary" @click="$router.push(`/play/${media.id}`)">▶ 播放</button>
          <button class="btn btn-secondary" @click="$router.push(`/play/${media.id}`)">
            ℹ 更多
          </button>
        </div>
      </div>
    </div>

    <section v-if="media" class="info-section">
      <div class="section">
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

      <!-- 相似推荐 -->
      <div v-if="similar.length > 0" class="section">
        <h2 class="section-title">相似推荐</h2>
        <div class="similar-row">
          <div
            v-for="m in similar"
            :key="m.media_id"
            class="similar-card"
            @click="$router.push(`/media/${m.media_id}`)"
          >
            <div class="poster-card">
              <img v-if="m.poster_url" :src="m.poster_url" :alt="m.title" loading="lazy" />
              <span v-else>{{ m.title.slice(0, 2) }}</span>
              <div v-if="m.rating > 0" class="rating">⭐ {{ m.rating.toFixed(1) }}</div>
            </div>
            <div class="card-title">{{ m.title }}</div>
            <div class="card-meta">{{ m.year }} · {{ typeLabel(m.type) }}</div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { mediaApi, historyApi, recommendApi, type MediaDetail, type MediaSummary } from '@/api'

const route = useRoute()
const loading = ref(false)
const media = ref<MediaDetail | null>(null)
const favorited = ref(false)
const similar = ref<MediaSummary[]>([])

const heroBg = computed(() => {
  const url = media.value?.backdrop_url || media.value?.poster_url
  return url ? { backgroundImage: `url(${url})` } : {}
})

async function load() {
  const id = route.params.id as string
  loading.value = true
  try {
    const data = await mediaApi.get(id)
    media.value = data

    // 加载相似推荐
    try {
      const simRes = await recommendApi.similar(id, 12)
      similar.value = simRes.filter((m) => m.id !== id)
    } catch (e) {
      similar.value = []
    }
  } finally {
    loading.value = false
  }
}

async function toggleFavorite() {
  if (!media.value) return
  try {
    await historyApi.toggleFavorite({
      media_id: media.value.id,
      type: 'want',
    })
    favorited.value = !favorited.value
    window.toast?.(favorited.value ? '已加入收藏' : '已取消收藏', 'success', 2000)
  } catch {
    // ignore
  }
}

function typeLabel(t: string) {
  return ({ movie: '电影', tvshow: '剧集', anime: '动画', documentary: '纪录片' } as any)[t] || t
}

onMounted(load)
</script>

<style lang="scss" scoped>
.detail-page {
  min-height: 100vh;
  background: #0f172a;
  color: #e2e8f0;
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
  background: rgba(255, 255, 255, 0.1);
  border: none;
  color: #fff;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;

  &.active {
    background: rgba(251, 191, 36, 0.2);
    color: #fbbf24;
    border: 1px solid rgba(251, 191, 36, 0.4);
  }

  &:hover { background: rgba(255, 255, 255, 0.2); }
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
</style>
