<template>
  <div class="home">
    <header class="topbar">
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
            <button class="btn btn-primary" @click="playItem(heroItem)">
              ▶ 播放
            </button>
            <button class="btn btn-secondary" @click="openDetail(heroItem)">
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

      <section v-for="row in rows" :key="row.id" class="row">
        <h2 class="row-title">
          {{ row.title }}
          <span v-if="row.subtitle" class="row-subtitle">{{ row.subtitle }}</span>
        </h2>
        <div class="row-cards" :class="`card-style-${row.card_style || 'poster'}`">
          <div
            v-for="item in row.items"
            :key="item.media_id"
            class="card"
            @click="openDetail(item)"
          >
            <div class="card-poster">
              <img v-if="item.poster_url" :src="item.poster_url" :alt="item.title" loading="lazy" />
              <span v-else class="poster-placeholder">{{ item.title.slice(0, 2) }}</span>
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
          onClick: () => window.open('/admin', '_blank'),
        }"
      />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { feedApi, recommendApi, historyApi, type Feed, type FeedItem, type FeedRow } from '@/api'
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

const heroItem = computed<FeedItem | null>(() => {
  for (const row of rows.value) {
    if (row.type === 'hero-banner' && row.items?.length > 0) {
      return row.items[0]
    }
  }
  for (const row of rows.value) {
    if (row.items?.length > 0) return row.items[0]
  }
  return null
})

const heroBg = computed(() => {
  const url = heroItem.value?.backdrop_url || heroItem.value?.poster_url
  return url ? { backgroundImage: `url(${url})` } : {}
})

async function loadFeed() {
  loading.value = true
  try {
    const data = await feedApi.get('web')
    rows.value = data.rows.filter((r) => r.type !== 'divider' && r.type !== 'text-banner')
  } catch (e: any) {
    console.error('拉取 Feed 失败', e)
    window.toast?.(`加载失败：${e?.message || '网络错误'}`, 'error', 4000)
  } finally {
    loading.value = false
  }
}

async function loadRecommendations() {
  try {
    const profileId = currentProfileId.value

    let continueItems: FeedItem[] = []
    if (profileId) {
      try {
        const cw = await historyApi.getContinueWatching(10)
        continueItems = cw.data.map((h: any) => ({
          media_id: h.media_id,
          title: h.media?.title || '',
          year: h.media?.year,
          poster_url: h.media?.poster_url,
          backdrop_url: h.media?.backdrop_url,
          rating: h.media?.rating || 0,
          type: h.media?.type || 'movie',
          progress: h.progress,
          overview: h.media?.overview,
          genres: h.media?.genres,
        }))
      } catch (e) {
        // ignore
      }
    }

    let hotItems: FeedItem[] = []
    try {
      const hot = await recommendApi.hot(15)
      hotItems = hot.map((m: any) => ({
        media_id: m.id,
        title: m.title,
        year: m.year,
        poster_url: m.poster_url,
        backdrop_url: m.backdrop_url,
        rating: m.rating,
        type: m.type,
        overview: m.overview,
        genres: m.genres,
      }))
    } catch (e) {
      // ignore
    }

    const prependRows: FeedRow[] = []

    if (continueItems.length > 0) {
      prependRows.push({
        id: 'continue-watching',
        type: 'shelf',
        title: '继续观看',
        subtitle: '从上次的地方继续',
        card_style: 'poster',
        items: continueItems,
      })
    }

    if (hotItems.length > 0) {
      prependRows.push({
        id: 'hot',
        type: 'shelf',
        title: '热门推荐',
        subtitle: '本周最火',
        card_style: 'poster',
        items: hotItems,
      })
    }

    rows.value = [...prependRows, ...rows.value]
  } catch (e) {
    console.error('推荐加载失败', e)
  }
}

function progressPct(item: FeedItem) {
  if (!item.progress || !item.duration) return 0
  return Math.min(100, (item.progress / item.duration) * 100)
}

function openDetail(item: FeedItem) {
  router.push(`/media/${item.media_id}`)
}

function playItem(item: FeedItem) {
  router.push(`/play/${item.media_id}`)
}

function onSearch() {
  if (!searchQuery.value.trim()) return
  router.push({ path: '/search', query: { q: searchQuery.value } })
}

function onProfileSwitched(p: any) {
  currentProfileId.value = p.id
  loadFeed()
  loadRecommendations()
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
  loadProfiles()

  let profileId = localStorage.getItem('mediahub_profile_id')
  if (!profileId) {
    profileId = profiles.value[0]?.id || '00000000-0000-0000-0000-000000000001'
    localStorage.setItem('mediahub_profile_id', profileId)
    currentProfileId.value = profileId
  }

  await loadFeed()
  await loadRecommendations()
})
</script>

<style lang="scss" scoped>
.home {
  min-height: 100vh;
  background: #0f172a;
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
  gap: 32px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.logo {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 18px;
  font-weight: 700;
  color: #fff;
}

.logo-icon {
  color: #6366f1;
}

.search-box {
  flex: 1;
  max-width: 400px;

  input {
    width: 100%;
    height: 36px;
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    color: #fff;
    padding: 0 16px;
    outline: none;
    transition: all 0.2s;

    &::placeholder {
      color: #64748b;
    }

    &:focus {
      background: rgba(255, 255, 255, 0.12);
      border-color: #6366f1;
    }
  }
}

.user-area {
  display: flex;
  align-items: center;
  gap: 8px;
}

.profile-switcher, .icon-btn {
  background: rgba(255, 255, 255, 0.08);
  border: none;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  color: #fff;
  font-weight: 600;
  transition: all 0.15s;

  &:hover { background: rgba(255, 255, 255, 0.15); }
}

.hero {
  position: relative;
  height: 75vh;
  min-height: 480px;
  background-size: cover;
  background-position: center;
  display: flex;
  align-items: center;
  padding: 0 80px;
  margin-top: 60px;
}

.hero-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    90deg,
    rgba(15, 23, 42, 0.95) 0%,
    rgba(15, 23, 42, 0.6) 40%,
    rgba(15, 23, 42, 0.2) 70%,
    rgba(15, 23, 42, 0.4) 100%
  );
}

.hero-content {
  position: relative;
  z-index: 1;
  max-width: 600px;
  color: #fff;
}

.hero-title {
  font-size: 56px;
  font-weight: 800;
  margin: 0 0 16px;
  text-shadow: 0 2px 12px rgba(0, 0, 0, 0.6);
  line-height: 1.1;
}

.hero-meta {
  display: flex;
  gap: 16px;
  font-size: 14px;
  color: #cbd5e1;
  margin-bottom: 16px;
}

.hero-overview {
  font-size: 15px;
  line-height: 1.6;
  color: #cbd5e1;
  margin-bottom: 24px;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.hero-actions {
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
  transition: all 0.15s;

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

.rows {
  padding: 0 40px 80px;
  margin-top: -80px;
  position: relative;
  z-index: 2;
}

.row {
  margin-bottom: 48px;
}

.row-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0 0 16px;
  color: #fff;
}

.row-subtitle {
  font-size: 13px;
  font-weight: 400;
  color: #94a3b8;
  margin-left: 12px;
}

.row-cards {
  display: flex;
  gap: 16px;
  overflow-x: auto;
  padding-bottom: 16px;
  scroll-behavior: smooth;
  scrollbar-width: thin;

  &::-webkit-scrollbar {
    height: 6px;
  }

  &::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.2);
    border-radius: 3px;
  }
}

.card {
  flex-shrink: 0;
  cursor: pointer;
  transition: transform 0.2s;
  width: 200px;

  &:hover {
    transform: scale(1.05);
    z-index: 10;

    .card-poster {
      box-shadow: 0 12px 24px rgba(0, 0, 0, 0.5);
    }
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
  background: linear-gradient(135deg, #1e293b, #0f172a);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.3);
  font-size: 32px;
  font-weight: 700;
  transition: box-shadow 0.2s;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.progress-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 4px;
  background: rgba(0, 0, 0, 0.6);
}

.progress-fill {
  height: 100%;
  background: #6366f1;
}

.rating {
  position: absolute;
  top: 8px;
  left: 8px;
  background: rgba(0, 0, 0, 0.7);
  color: #fbbf24;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
}

.resume-badge {
  position: absolute;
  bottom: 8px;
  left: 8px;
  background: rgba(99, 102, 241, 0.9);
  color: #fff;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
}

.card-title {
  margin-top: 8px;
  font-size: 13px;
  color: #e2e8f0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Hero 切换动画 */
.hero-fade-enter-active, .hero-fade-leave-active {
  transition: opacity 0.6s cubic-bezier(0.16, 1, 0.3, 1);
}
.hero-fade-enter-from, .hero-fade-leave-to {
  opacity: 0;
}

/* 卡片 hover 缩放 */
.card {
  transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}
</style>
