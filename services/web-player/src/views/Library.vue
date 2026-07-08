<template>
  <div class="library-page mh-page">
    <AppTopbar
      variant="sub"
      :show-back="false"
      title="我的片库"
    >
      <template #start>
        <button type="button" class="mh-back-btn" @click="$router.push('/')">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18" aria-hidden="true">
            <path d="M15 18l-6-6 6-6" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          <span>首页</span>
        </button>
      </template>
    </AppTopbar>

    <main class="library-content mh-page-body mh-animate-in">
      <el-tabs v-model="activeTab" class="library-tabs" @tab-change="onTabChange">
        <el-tab-pane label="继续观看" name="continue">
          <LoadingState v-if="loading" message="加载中…" />
          <EmptyState
            v-else-if="!continueItems.length"
            icon="▶"
            title="暂无在看"
            description="去首页找一部喜欢的作品开始观看吧。"
            :action="{ label: '浏览首页', onClick: () => $router.push('/') }"
          />
          <div v-else class="mh-media-grid">
            <MediaPosterCard
              v-for="item in continueItems"
              :key="item.media_id"
              :title="item.title || '未知'"
              :poster-url="item.poster_url"
              :subtitle="String(item.year || '')"
              landscape
              :progress="item.progress"
              :duration="item.duration"
              @click="$router.push(`/play/${item.media_id}`)"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane label="想看" name="want">
          <LoadingState v-if="loading" message="加载中…" />
          <EmptyState
            v-else-if="!wantItems.length"
            icon="☆"
            title="暂无想看"
            description="在详情页点击「想看」，把感兴趣的作品收藏在这里。"
          />
          <div v-else class="mh-media-grid">
            <MediaPosterCard
              v-for="item in wantItems"
              :key="item.media_id || `tmdb-${item.tmdb_id}`"
              :title="item.title || '未知'"
              :poster-url="item.poster_url"
              :subtitle="String(item.year || '')"
              @click="openDetail(item)"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane label="收藏" name="favorites">
          <LoadingState v-if="loading" message="加载中…" />
          <EmptyState
            v-else-if="!favItems.length"
            icon="★"
            title="暂无收藏"
            description="在详情页点击「收藏」，快速找到最爱的作品。"
          />
          <div v-else class="mh-media-grid">
            <MediaPosterCard
              v-for="item in favItems"
              :key="item.media_id"
              :title="item.title || item.media_id"
              :poster-url="item.poster_url"
              :subtitle="String(item.year || '')"
              @click="$router.push(`/media/${item.media_id}`)"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { libraryApi, mediaApi, type LibraryItem } from '@/api'
import AppTopbar from '@/components/AppTopbar.vue'
import MediaPosterCard from '@/components/MediaPosterCard.vue'
import LoadingState from '@/components/LoadingState.vue'
import EmptyState from '@/components/EmptyState.vue'

const router = useRouter()
const activeTab = ref('continue')
const loading = ref(false)

const continueItems = ref<LibraryItem[]>([])
const wantItems = ref<LibraryItem[]>([])
const favItems = ref<{ media_id: string; title?: string; poster_url?: string; year?: number }[]>([])

async function onTabChange(tab: string | number) {
  loading.value = true
  try {
    switch (tab) {
      case 'continue':
        continueItems.value = await libraryApi.continueWatching(24)
        break
      case 'want':
        wantItems.value = await libraryApi.wantList()
        break
      case 'favorites': {
        const favIds = await libraryApi.favoritesList()
        favItems.value = await Promise.all(
          favIds.map(async (f) => {
            try {
              const detail = await mediaApi.get(f.media_id)
              return { media_id: f.media_id, title: detail.title, poster_url: detail.poster_url, year: detail.year }
            } catch {
              return { media_id: f.media_id }
            }
          }),
        )
        break
      }
    }
  } catch {
    window.toast?.('加载失败', 'error', 2500)
  } finally {
    loading.value = false
  }
}

function openDetail(item: LibraryItem) {
  if (item.media_id) {
    router.push(`/media/${item.media_id}`)
  } else if (item.tmdb_id && item.media_type) {
    router.push(`/media/tmdb/${item.media_type}/${item.tmdb_id}`)
  }
}

onMounted(() => onTabChange(activeTab.value))
</script>

<style lang="scss" scoped>
.library-content {
  padding-left: var(--mh-page-gutter);
  padding-right: var(--mh-page-gutter);
  padding-bottom: var(--mh-space-10);
}

.library-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: var(--mh-space-6);
  }

  :deep(.el-tabs__nav-wrap::after) {
    background: var(--mh-outline);
  }

  :deep(.el-tabs__item) {
    color: var(--mh-text-muted);
    font-size: 15px;
    font-weight: 500;
    padding: 0 var(--mh-space-5);
    height: 44px;

    &.is-active {
      color: var(--mh-text);
    }

    &:hover {
      color: var(--mh-text-secondary);
    }
  }

  :deep(.el-tabs__active-bar) {
    background: var(--mh-primary);
    height: 3px;
    border-radius: 2px;
  }
}
</style>
