<template>
  <div class="home">
    <header class="topbar mh-topbar">
      <div class="logo">
        <span class="logo-icon">▶</span>
        <span class="logo-text">MediaHub</span>
      </div>
      <div class="search-box">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="搜索电影 / 剧集..."
          @keyup.enter="onSearch"
        />
      </div>
      <div class="user-area">
        <button class="icon-btn" @click="showProfile = true" :title="currentProfile?.name">
          {{ currentProfile?.name?.slice(0, 1) || '?' }}
        </button>
        <button class="icon-btn" @click="$router.push('/search')">🔍</button>
      </div>
    </header>

    <ProfileSwitcher v-model="showProfile" @switched="onProfileSwitched" />

    <Transition name="hero-fade">
      <div v-if="heroItem && !loading" class="hero" :style="heroBg" :key="heroItem.media_id">
        <div class="hero-overlay"></div>
        <div class="hero-content">
          <h1 class="hero-title">{{ heroItem.title }}</h1>
          <div class="hero-meta">
            <span v-if="heroItem.year">{{ heroItem.year }}</span>
            <span v-if="heroItem.rating">⭐ {{ heroItem.rating.toFixed(1) }}</span>
            <span v-for="g in heroItem.genres?.slice(0, 3)" :key="g">{{ g }}</span>
          </div>
          <p v-if="heroItem.overview" class="hero-overview">{{ heroItem.overview }}</p>
          <div class="hero-actions">
            <button class="btn mh-btn mh-btn--primary" @click="playItem(heroItem)">
              ▶ 播放
            </button>
            <button class="btn mh-btn mh-btn--secondary" @click="openDetail(heroItem)">
              ℹ 详情
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <main class="rows">
      <!-- Skeleton: 首次加载 -->
      <section v-if="loading && rows.length === 0" class="row">
        <h2 class="row-title">热门推荐</h2>
        <div class="row-cards">
          <SkeletonCard v-for="n in 6" :key="`skeleton-${n}`" />
        </div>
      </section>

      <section v-for="row in displayRows" :key="row.id" class="row">
        <h2 class="row-title">
          {{ row.title }}
          <span v-if="row.subtitle" class="row-subtitle">{{ row.subtitle }}</span>
        </h2>
        <div class="row-cards" :class="`card-style-${row.card_style || 'poster'}`">
          <div
            v-for="item in row.items"
            :key="item.external ? `tmdb-${item.tmdb_id}` : item.media_id"
            class="card"
            :class="{ 'card-external': item.external }"
            @click="openDetail(item)"
          >
            <div class="card-poster">
              <img v-if="item.poster_url" :src="item.poster_url" :alt="item.title" loading="lazy" />
              <span v-else class="poster-placeholder">{{ item.title.slice(0, 2) }}</span>
              <div v-if="item.external" class="external-badge">TMDB</div>
              <div v-if="item.progress && item.progress > 0" class="progress-bar">
                <div class="progress-fill" :style="{ width: progressPct(item) + '%' }"></div>
              </div>
              <div v-if="item.rating > 0" class="rating">⭐ {{ item.rating.toFixed(1) }}</div>
              <div v-if="item.progress && item.progress > 0" class="resume-badge">继续</div>
            </div>
            <div class="card-title">{{ item.title }}</div>
          </div>
        </div>
      </section>

      <!-- Empty -->
      <EmptyState
        v-if="!loading && rows.length === 0"
        icon="🎬"
        title="暂无内容"
        description="CMS 里还没有发布布局或媒资。打开后台添加内容吧。"
        :action="{
          label: '前往 CMS Admin',
          onClick: openCmsAdmin,
        }"
      />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { feedApi, historyApi, type FeedItem, type FeedRow } from '@/api'
import ProfileSwitcher from '@/views/ProfileSwitcher.vue'
import SkeletonCard from '@/components/SkeletonCard.vue'
import EmptyState from '@/components/EmptyState.vue'

const router = useRouter()
const loading = ref(false)
const rows = ref<FeedRow[]>([])
const searchQuery = ref('')
const showProfile = ref(false)
const currentProfileId = ref('')

const profiles = ref<Array<{id: string; name: string; is_kid: boolean}>>([])
const currentProfile = computed(() =>
  profiles.value.find((p) => p.id === currentProfileId.value)
)

/** 列表区：仅用 CMS Feed，排除 hero 行与空行 */
const displayRows = computed(() =>
  rows.value.filter(
    (r) =>
      r.type !== 'hero-banner' &&
      r.type !== 'divider' &&
      r.type !== 'text-banner' &&
      (r.items?.length ?? 0) > 0,
  ),
)

const heroItem = computed<FeedItem | null>(() => {
  const playable = (items: FeedItem[]) =>
    items.find((i) => !i.external && i.media_id)

  for (const row of rows.value) {
    if (row.type === 'hero-banner' && row.items?.length > 0) {
      const item = playable(row.items)
      if (item) return item
    }
  }
  for (const row of rows.value) {
    const item = playable(row.items || [])
    if (item) return item
  }
  return null
})

const heroBg = computed(() => {
  const url = heroItem.value?.backdrop_url || heroItem.value?.poster_url
  return url ? { backgroundImage: `url(${url})` } : {}
})

function openCmsAdmin() {
  window.open('/admin', '_blank')
}

async function loadFeed() {
  loading.value = true
  try {
    const data = await feedApi.get('web')
    rows.value = data.rows
  } catch (e: any) {
    console.error('拉取 Feed 失败', e)
    window.toast?.(`加载失败：${e?.message || '网络错误'}`, 'error', 4000)
  } finally {
    loading.value = false
  }
}

function progressPct(item: FeedItem) {
  if (!item.progress || !item.duration) return 0
  return Math.min(100, (item.progress / item.duration) * 100)
}

function openDetail(item: FeedItem) {
  if (item.external || !item.media_id) {
    window.toast?.('该内容尚未加入媒体库，可在 CMS 中搜索入库', 'info', 3500)
    return
  }
  router.push(`/media/${item.media_id}`)
}

function playItem(item: FeedItem) {
  if (item.external || !item.media_id) {
    window.toast?.('该内容尚未加入媒体库', 'info', 3000)
    return
  }
  router.push(`/play/${item.media_id}`)
}

function onSearch() {
  if (!searchQuery.value.trim()) return
  router.push({ path: '/search', query: { q: searchQuery.value } })
}

async function onProfileSwitched(p: any) {
  currentProfileId.value = p.id
  await loadFeed()
}

function loadProfiles() {
  const stored = localStorage.getItem('mediahub_profiles')
  if (stored) {
    try {
      profiles.value = JSON.parse(stored)
    } catch {
      profiles.value = []
    }
  }
  if (profiles.value.length === 0) {
    profiles.value = [{ id: '00000000-0000-0000-0000-000000000001', name: '我', is_kid: false }]
    localStorage.setItem('mediahub_profiles', JSON.stringify(profiles.value))
  }
  currentProfileId.value = localStorage.getItem('mediahub_profile_id') || profiles.value[0].id
}

onMounted(async () => {
  await historyApi.ensureProfileId()
  loadProfiles()

  currentProfileId.value = localStorage.getItem('mediahub_profile_id') || profiles.value[0]?.id || ''

  await loadFeed()
})
</script>

<style lang="scss" scoped>
.home {
  min-height: 100vh;
  background: var(--mh-bg);
}

.logo {
  display: flex;
  align-items: center;
  gap: var(--mh-space-2);
  font-size: 18px;
  font-weight: 700;
  font-family: var(--mh-font-display);
  color: var(--mh-text);
}

.logo-icon {
  color: var(--mh-primary);
}

.logo-text {
  background: linear-gradient(135deg, var(--mh-primary), var(--mh-accent));
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.search-box {
  flex: 1;
  max-width: min(400px, 40vw);

  input {
    width: 100%;
    height: 40px;
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid var(--mh-outline);
    border-radius: 10px;
    color: var(--mh-text);
    padding: 0 var(--mh-space-4);
    outline: none;
    transition: background var(--mh-duration) var(--mh-ease),
                border-color var(--mh-duration) var(--mh-ease),
                box-shadow var(--mh-duration) var(--mh-ease);

    &::placeholder {
      color: var(--mh-text-muted);
    }

    &:focus {
      background: rgba(255, 255, 255, 0.08);
      border-color: var(--mh-primary);
      box-shadow: 0 0 0 3px var(--mh-primary-muted);
    }
  }
}

.user-area {
  display: flex;
  align-items: center;
  gap: var(--mh-space-2);
}

.profile-switcher, .icon-btn {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid var(--mh-outline);
  width: 40px;
  height: 40px;
  border-radius: var(--mh-radius-full);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
  color: var(--mh-text);
  font-weight: 600;
  transition: background var(--mh-duration) var(--mh-ease),
              transform var(--mh-duration) var(--mh-ease),
              border-color var(--mh-duration) var(--mh-ease);

  &:hover {
    background: rgba(255, 255, 255, 0.1);
    border-color: var(--mh-outline-strong);
    transform: translateY(-1px);
  }

  &:active {
    transform: scale(0.97);
  }
}

.hero {
  position: relative;
  height: min(75vh, 820px);
  min-height: 480px;
  background-size: cover;
  background-position: center top;
  display: flex;
  align-items: center;
  padding: 0 clamp(var(--mh-space-6), 6vw, 80px);
  margin-top: var(--mh-topbar-height);
}

.hero-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    90deg,
    rgba(10, 10, 18, 0.96) 0%,
    rgba(10, 10, 18, 0.55) 45%,
    rgba(10, 10, 18, 0.15) 72%,
    rgba(10, 10, 18, 0.5) 100%
  );
}

.hero-content {
  position: relative;
  z-index: 1;
  max-width: 640px;
  color: var(--mh-text);
}

.hero-title {
  font-size: clamp(36px, 5vw, 56px);
  font-weight: 800;
  font-family: var(--mh-font-display);
  margin: 0 0 var(--mh-space-4);
  text-shadow: 0 4px 24px rgba(0, 0, 0, 0.5);
  line-height: 1.08;
  letter-spacing: -0.03em;
}

.hero-meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--mh-space-4);
  font-size: 14px;
  color: var(--mh-text-secondary);
  margin-bottom: var(--mh-space-4);
}

.hero-overview {
  font-size: 15px;
  line-height: 1.65;
  color: var(--mh-text-secondary);
  margin-bottom: var(--mh-space-6);
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--mh-space-3);
}

.rows {
  padding: 0 clamp(var(--mh-space-4), 4vw, var(--mh-space-10)) 80px;
  margin-top: -80px;
  position: relative;
  z-index: 2;
}

.row {
  margin-bottom: var(--mh-space-12);
}

.row-title {
  font-size: clamp(18px, 2vw, 22px);
  font-weight: 600;
  font-family: var(--mh-font-display);
  margin: 0 0 var(--mh-space-4);
  color: var(--mh-text);
  letter-spacing: -0.02em;
}

.row-subtitle {
  font-size: 13px;
  font-weight: 400;
  color: var(--mh-text-muted);
  margin-left: var(--mh-space-3);
}

.row-cards {
  display: flex;
  gap: var(--mh-space-4);
  overflow-x: auto;
  padding-bottom: var(--mh-space-4);
  scroll-behavior: smooth;
  scrollbar-width: thin;

  &::-webkit-scrollbar {
    height: 6px;
  }

  &::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.15);
    border-radius: 3px;
  }
}

.card {
  flex-shrink: 0;
  cursor: pointer;
  width: 200px;
  transition: transform var(--mh-duration) var(--mh-ease-spring);

  &:hover {
    transform: scale(1.04);
    z-index: 10;

    .card-poster {
      box-shadow: var(--mh-shadow-lg), var(--mh-shadow-glow);
    }
  }

  &:active {
    transform: scale(0.98);
  }
}

.card-style-landscape .card {
  width: 280px;
}

.card-style-landscape .card-poster {
  aspect-ratio: 16/9;
}

.card-poster {
  position: relative;
  aspect-ratio: 2/3;
  background: linear-gradient(145deg, var(--mh-surface-variant), var(--mh-bg));
  border-radius: var(--mh-radius-md);
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.25);
  font-size: 32px;
  font-weight: 700;
  transition: box-shadow var(--mh-duration) var(--mh-ease);
  border: 1px solid var(--mh-outline);

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.external-badge {
  position: absolute;
  top: var(--mh-space-2);
  right: var(--mh-space-2);
  padding: 2px 8px;
  border-radius: 6px;
  background: rgba(108, 99, 255, 0.92);
  color: #fff;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.04em;
  backdrop-filter: blur(8px);
}

.card-external .card-poster {
  outline: 1px dashed rgba(108, 99, 255, 0.45);
}

.progress-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: rgba(0, 0, 0, 0.5);
}

.progress-fill {
  height: 100%;
  background: var(--mh-primary);
}

.rating {
  position: absolute;
  top: var(--mh-space-2);
  left: var(--mh-space-2);
  background: rgba(0, 0, 0, 0.72);
  backdrop-filter: blur(8px);
  color: var(--mh-warning);
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.resume-badge {
  position: absolute;
  bottom: var(--mh-space-2);
  left: var(--mh-space-2);
  background: rgba(108, 99, 255, 0.92);
  color: #fff;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 500;
}

.card-title {
  margin-top: var(--mh-space-2);
  font-size: 13px;
  font-weight: 500;
  color: var(--mh-text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.hero-fade-enter-active, .hero-fade-leave-active {
  transition: opacity 0.6s var(--mh-ease);
}

.hero-fade-enter-from, .hero-fade-leave-to {
  opacity: 0;
}
</style>
