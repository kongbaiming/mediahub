<template>
  <div class="home" :class="homeSchemaClass">
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
        <button class="icon-btn" @click="$router.push('/library')" title="我的片库">📚</button>
        <button class="icon-btn" @click="$router.push('/live')" title="直播间">📡</button>
        <button class="icon-btn" @click="showProfile = true" :title="currentProfile?.name">
          {{ currentProfile?.name?.slice(0, 1) || '?' }}
        </button>
        <button class="icon-btn" @click="$router.push('/search')">🔍</button>
      </div>
    </header>

    <ProfileSwitcher v-model="showProfile" @switched="onProfileSwitched" />

    <main class="feed-main">
      <section v-if="loading && rows.length === 0" class="feed-skeleton">
        <h2 class="feed-skeleton__title">加载中...</h2>
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

/** 按 CMS 编排顺序展示全部可见行（含 Hero、公告、分隔线等） */
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
.home {
  min-height: 100vh;
  background:
    radial-gradient(ellipse 80% 50% at 50% -20%, rgba(124, 92, 255, 0.12), transparent),
    radial-gradient(ellipse 60% 40% at 100% 50%, rgba(74, 215, 255, 0.06), transparent),
    var(--mh-bg);
}

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
  }
}
</style>
