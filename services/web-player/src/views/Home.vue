<template>
  <div class="home mh-page" :class="homeSchemaClass">
    <header class="topbar mh-topbar">
      <button type="button" class="logo" @click="$router.push('/')">
        <span class="logo-mark" aria-hidden="true">
          <svg viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z" /></svg>
        </span>
        <span class="logo-text">MediaHub</span>
      </button>

      <form class="search-box" @submit.prevent="onSearch">
        <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <circle cx="11" cy="11" r="7" />
          <path d="M20 20l-3-3" stroke-linecap="round" />
        </svg>
        <input
          v-model="searchQuery"
          type="search"
          class="mh-search-field"
          placeholder="搜索电影、剧集、影人…"
          autocomplete="off"
        />
      </form>

      <nav class="user-area" aria-label="主导航">
        <button type="button" class="mh-icon-btn" title="我的片库" @click="$router.push('/library')">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
            <path d="M4 19.5A2.5 2.5 0 016.5 17H20" stroke-linecap="round" />
            <path d="M6.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15A2.5 2.5 0 016.5 2z" />
          </svg>
        </button>
        <button type="button" class="mh-icon-btn" title="直播" @click="$router.push('/live')">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
            <path d="M4.5 8.5c4-3.5 11-3.5 15 0" stroke-linecap="round" />
            <path d="M7.5 11.5c2.5-2 7.5-2 10 0" stroke-linecap="round" />
            <circle cx="12" cy="16" r="2" fill="currentColor" stroke="none" />
          </svg>
        </button>
        <button
          type="button"
          class="mh-icon-btn profile-btn"
          :title="currentProfile?.name || '切换档案'"
          @click="showProfile = true"
        >
          {{ currentProfile?.name?.slice(0, 1) || '?' }}
        </button>
        <button type="button" class="mh-icon-btn" title="搜索页" @click="$router.push('/search')">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
            <circle cx="11" cy="11" r="7" />
            <path d="M20 20l-3-3" stroke-linecap="round" />
          </svg>
        </button>
      </nav>
    </header>

    <ProfileSwitcher v-model="showProfile" @switched="onProfileSwitched" />

    <main class="feed-main mh-animate-in">
      <section v-if="loading && rows.length === 0" class="feed-skeleton">
        <h2 class="feed-skeleton__title">正在加载推荐…</h2>
        <div class="feed-skeleton__cards">
          <SkeletonCard v-for="n in 6" :key="`skeleton-${n}`" />
        </div>
      </section>

      <FeedRowRenderer
        v-for="row in visibleRows"
        :key="row.id"
        :row="row"
        :immersive="isImmersiveLayout"
        @open="openDetail"
        @play="playItem"
      />

      <EmptyState
        v-if="!loading && visibleRows.length === 0"
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
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { feedApi, type FeedItem, type FeedRow } from '@/api'
import FeedRowRenderer from '@/components/FeedRowRenderer.vue'
import ProfileSwitcher from '@/views/ProfileSwitcher.vue'
import SkeletonCard from '@/components/SkeletonCard.vue'
import EmptyState from '@/components/EmptyState.vue'
import { syncProfiles, getActiveProfileId, type LocalProfile } from '@/utils/profiles'

const router = useRouter()
const loading = ref(false)
const rows = ref<FeedRow[]>([])
const feedGlobal = ref<Record<string, unknown>>({})
const searchQuery = ref('')
const showProfile = ref(false)
const currentProfileId = ref('')
const feedVersion = ref(0)
let feedPollTimer: ReturnType<typeof setInterval> | null = null

const profiles = ref<LocalProfile[]>([])
const currentProfile = computed(() =>
  profiles.value.find((p) => p.id === currentProfileId.value),
)

const isImmersiveLayout = computed(
  () => feedGlobal.value?.layout_schema === 'immersive',
)

const homeSchemaClass = computed(() => {
  const schema = feedGlobal.value?.layout_schema
  return schema && typeof schema === 'string' ? `home--${schema}` : ''
})

const visibleRows = computed(() =>
  rows.value.filter((r) => {
    if (r.type === 'hero-banner') {
      return (r.items?.length ?? 0) > 0
    }
    if (r.type === 'divider' || r.type === 'text-banner') {
      return !!(r.title || r.subtitle)
    }
    if (r.type === 'ranking') {
      return (r.items?.length ?? 0) > 0
    }
    return (r.items?.length ?? 0) > 0
  }),
)

function openCmsAdmin() {
  window.open('/admin', '_blank')
}

async function loadFeed() {
  loading.value = true
  try {
    const data = await feedApi.get('web')
    rows.value = data.rows
    feedGlobal.value = data.global || {}
  } catch (e: any) {
    console.error('拉取 Feed 失败', e)
    window.toast?.(`加载失败：${e?.message || '网络错误'}`, 'error', 4000)
  } finally {
    loading.value = false
  }
}

function openDetail(item: FeedItem) {
  if (item.media_id && !item.external) {
    router.push(`/media/${item.media_id}`)
    return
  }
  if (item.external && item.tmdb_id && item.type) {
    router.push(`/media/tmdb/${item.type}/${item.tmdb_id}`)
    return
  }
  window.toast?.('无法打开该内容', 'info', 2000)
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

async function onProfileSwitched(p: LocalProfile) {
  currentProfileId.value = p.id
  await loadFeed()
}

async function pollFeedVersion() {
  try {
    const v = await feedApi.version()
    if (feedVersion.value > 0 && v !== feedVersion.value) {
      await loadFeed()
    }
    feedVersion.value = v
  } catch {
    /* ignore */
  }
}

async function loadProfiles() {
  profiles.value = await syncProfiles()
  currentProfileId.value = getActiveProfileId() || profiles.value[0]?.id || ''
}

onMounted(async () => {
  await loadProfiles()
  await loadFeed()
  await pollFeedVersion()
  feedPollTimer = setInterval(pollFeedVersion, 5000)
})

onBeforeUnmount(() => {
  if (feedPollTimer) clearInterval(feedPollTimer)
})
</script>

<style lang="scss" scoped>
.home--immersive {
  :deep(.feed-hero) {
    margin-top: 0;
    padding-top: calc(var(--mh-topbar-height) + var(--mh-space-6));
  }
}

.logo {
  display: flex;
  align-items: center;
  gap: var(--mh-space-2);
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  color: var(--mh-text);
  font-size: 18px;
  font-weight: 700;
  font-family: var(--mh-font-display);
}

.logo-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--mh-radius-sm);
  background: var(--mh-primary-muted);
  color: var(--mh-primary);

  svg {
    width: 16px;
    height: 16px;
    margin-left: 2px;
  }
}

.logo-text {
  background: linear-gradient(135deg, var(--mh-primary), var(--mh-secondary));
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.search-box {
  flex: 1;
  max-width: min(480px, 42vw);
  position: relative;

  .mh-search-field {
    padding-left: 40px;
    height: 40px;
  }
}

.search-icon {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  width: 18px;
  height: 18px;
  color: var(--mh-text-muted);
  pointer-events: none;
}

.user-area {
  display: flex;
  align-items: center;
  gap: var(--mh-space-2);
}

.profile-btn {
  font-size: 15px;
  font-weight: 700;
  color: var(--mh-text);
}

.feed-main {
  padding-bottom: 80px;
  position: relative;
  z-index: 2;
}

.feed-skeleton {
  margin: calc(var(--mh-topbar-height) + var(--mh-space-6)) 0 var(--mh-space-12);
  padding: 0 var(--mh-page-gutter);

  &__title {
    font-size: clamp(18px, 2vw, 22px);
    font-weight: 600;
    margin: 0 0 var(--mh-space-4);
    color: var(--mh-text-muted);
  }

  &__cards {
    display: flex;
    gap: var(--mh-space-4);
    overflow-x: auto;
    padding-bottom: var(--mh-space-4);
    scroll-snap-type: x mandatory;

    :deep(.skeleton-card) {
      scroll-snap-align: start;
      flex: 0 0 auto;
    }
  }
}

@media (max-width: 720px) {
  .search-box {
    max-width: none;
  }

  .logo-text {
    display: none;
  }
}
</style>
