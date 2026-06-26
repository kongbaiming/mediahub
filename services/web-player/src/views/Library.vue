<template>
  <div class="library-page">
    <header class="topbar mh-topbar">
      <button class="back-btn" @click="$router.push('/')">← 首页</button>
      <h1 class="page-title">我的片库</h1>
      <button class="icon-btn" @click="loadAll" title="刷新">↻</button>
    </header>

    <div class="tabs">
      <button
        v-for="t in tabs"
        :key="t.key"
        class="tab-btn"
        :class="{ active: activeTab === t.key }"
        @click="activeTab = t.key"
      >
        {{ t.label }}
        <span v-if="t.count != null" class="tab-count">{{ t.count }}</span>
      </button>
    </div>

    <div v-loading="loading" class="content">
      <div v-if="activeTab === 'continue'" class="grid">
        <div
          v-for="item in continueItems"
          :key="item.media_id"
          class="card"
          @click="playContinue(item)"
        >
          <div class="poster">
            <img
              v-if="item.media?.poster_url"
              :src="item.media.poster_url"
              :alt="item.media?.title"
              loading="lazy"
            />
            <span v-else class="placeholder">{{ item.media?.title?.slice(0, 2) || '?' }}</span>
            <div v-if="item.progress && item.duration" class="progress-bar">
              <div
                class="progress-fill"
                :style="{ width: Math.min(100, (item.progress / item.duration) * 100) + '%' }"
              />
            </div>
          </div>
          <div class="title">{{ item.media?.title || '未知' }}</div>
          <div class="meta">继续观看 · {{ formatProgress(item) }}</div>
        </div>
        <EmptyState
          v-if="!loading && continueItems.length === 0"
          icon="▶"
          title="暂无续播"
          description="开始观看后，进度会出现在这里"
        />
      </div>

      <div v-else-if="activeTab === 'favorites'" class="grid">
        <div
          v-for="item in favoriteItems"
          :key="item.media_id"
          class="card"
          @click="openMedia(item.media_id)"
        >
          <div class="poster">
            <img
              v-if="item.media?.poster_url"
              :src="item.media.poster_url"
              :alt="item.media?.title"
              loading="lazy"
            />
            <span v-else class="placeholder">{{ item.media?.title?.slice(0, 2) || '?' }}</span>
          </div>
          <div class="title">{{ item.media?.title || '未知' }}</div>
        </div>
        <EmptyState
          v-if="!loading && favoriteItems.length === 0"
          icon="★"
          title="暂无收藏"
          description="在详情页点击收藏即可添加"
        />
      </div>

      <div v-else class="grid">
        <div
          v-for="item in wantItems"
          :key="item.media_id"
          class="card"
          @click="openMedia(item.media_id)"
        >
          <div class="poster">
            <img
              v-if="item.media?.poster_url"
              :src="item.media.poster_url"
              :alt="item.media?.title"
              loading="lazy"
            />
            <span v-else class="placeholder">{{ item.media?.title?.slice(0, 2) || '?' }}</span>
          </div>
          <div class="title">{{ item.media?.title || '未知' }}</div>
        </div>
        <EmptyState
          v-if="!loading && wantItems.length === 0"
          icon="+"
          title="暂无想看"
          description="在详情页标记「想看」即可添加"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { libraryApi, type LibraryItem } from '@/api'
import { syncProfiles } from '@/utils/profiles'
import EmptyState from '@/components/EmptyState.vue'

type TabKey = 'continue' | 'favorites' | 'want'

const router = useRouter()
const loading = ref(false)
const activeTab = ref<TabKey>('continue')
const continueItems = ref<LibraryItem[]>([])
const favoriteItems = ref<LibraryItem[]>([])
const wantItems = ref<LibraryItem[]>([])

const tabs = computed(() => [
  { key: 'continue' as TabKey, label: '继续观看', count: continueItems.value.length || undefined },
  { key: 'favorites' as TabKey, label: '收藏', count: favoriteItems.value.length || undefined },
  { key: 'want' as TabKey, label: '想看', count: wantItems.value.length || undefined },
])

function formatProgress(item: LibraryItem) {
  if (!item.duration) return ''
  const pct = Math.round(((item.progress || 0) / item.duration) * 100)
  return `${pct}%`
}

function openMedia(id: string) {
  router.push(`/media/${id}`)
}

function playContinue(item: LibraryItem) {
  const q = item.episode_id ? `?episode_id=${item.episode_id}` : ''
  router.push(`/play/${item.media_id}${q}`)
}

async function loadAll() {
  loading.value = true
  try {
    await syncProfiles()
    const [cw, fav, want] = await Promise.all([
      libraryApi.continueWatching(24),
      libraryApi.favoritesList(),
      libraryApi.wantList(),
    ])
    continueItems.value = dedupeByMedia(cw.map(normalizeHistory))
    favoriteItems.value = dedupeByMedia(fav.map(normalizeFavorite))
    wantItems.value = dedupeByMedia(want.map(normalizeFavorite))
  } catch {
    window.toast?.('加载片库失败', 'error', 2500)
  } finally {
    loading.value = false
  }
}

function normalizeHistory(h: any): LibraryItem {
  return {
    media_id: h.media_id || h.MediaID || h.media?.id,
    media: h.media,
    progress: h.progress,
    duration: h.duration,
    updated_at: h.updated_at,
    episode_id: h.episode_id,
  }
}

function normalizeFavorite(f: any): LibraryItem {
  return {
    media_id: f.media_id || f.MediaID || f.media?.id,
    media: f.media,
  }
}

function dedupeByMedia(items: LibraryItem[]): LibraryItem[] {
  const seen = new Set<string>()
  const out: LibraryItem[] = []
  for (const item of items) {
    const id = item.media_id || item.media?.id
    if (!id || seen.has(id)) continue
    seen.add(id)
    out.push(item)
  }
  return out
}

onMounted(loadAll)
</script>

<style lang="scss" scoped>
.library-page {
  min-height: 100vh;
  background: var(--mh-bg);
  color: var(--mh-text);
}

.topbar {
  display: flex;
  align-items: center;
  gap: var(--mh-space-4);
  padding: var(--mh-space-4) var(--mh-space-6);
}

.back-btn,
.icon-btn {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid var(--mh-outline);
  color: var(--mh-text-secondary);
  border-radius: var(--mh-radius-sm);
  padding: 8px 12px;
  cursor: pointer;

  &:hover {
    background: rgba(255, 255, 255, 0.1);
  }
}

.page-title {
  flex: 1;
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}

.tabs {
  display: flex;
  gap: var(--mh-space-2);
  padding: 0 var(--mh-space-6) var(--mh-space-4);
  border-bottom: 1px solid var(--mh-outline);
}

.tab-btn {
  background: transparent;
  border: none;
  color: var(--mh-text-muted);
  padding: 10px 16px;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  font-size: 15px;

  &.active {
    color: var(--mh-text);
    border-bottom-color: var(--mh-primary);
  }
}

.tab-count {
  margin-left: 6px;
  font-size: 12px;
  opacity: 0.7;
}

.content {
  padding: var(--mh-space-6);
  min-height: 400px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: var(--mh-space-4);
}

.card {
  cursor: pointer;
  transition: transform var(--mh-duration) var(--mh-ease);

  &:hover {
    transform: translateY(-3px);
  }
}

.poster {
  aspect-ratio: 2/3;
  border-radius: var(--mh-radius-md);
  overflow: hidden;
  background: var(--mh-surface);
  position: relative;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 24px;
  color: var(--mh-text-muted);
}

.progress-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 4px;
  background: rgba(0, 0, 0, 0.5);
}

.progress-fill {
  height: 100%;
  background: var(--mh-primary);
}

.title {
  margin-top: var(--mh-space-2);
  font-size: 14px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta {
  font-size: 12px;
  color: var(--mh-text-muted);
  margin-top: 2px;
}
</style>
