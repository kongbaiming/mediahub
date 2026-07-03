<template>
  <div class="library-page">
    <header class="library-topbar mh-topbar">
      <h1 class="page-title">我的片库</h1>
      <button class="back-btn" @click="$router.push('/')">← 首页</button>
    </header>

    <main class="library-content">
      <el-tabs v-model="activeTab" class="library-tabs" @tab-change="onTabChange">
        <el-tab-pane label="继续观看" name="continue">
          <div v-if="loading" class="loading">加载中...</div>
          <div v-else-if="!continueItems.length" class="empty">暂无在看，去首页找一部开始吧</div>
          <div v-else class="media-grid">
            <div
              v-for="item in continueItems"
              :key="item.media_id"
              class="card"
              @click="$router.push(`/play/${item.media_id}`)"
            >
              <div class="poster-card landscape">
                <img v-if="item.poster_url" :src="item.poster_url" :alt="item.title" loading="lazy" />
                <span v-else class="poster-placeholder">{{ (item.title || '').slice(0, 2) }}</span>
                <div v-if="item.progress && item.duration" class="progress-bar">
                  <div class="progress-fill" :style="{ width: Math.min(100, (item.progress / item.duration) * 100) + '%' }" />
                </div>
              </div>
              <div class="card-title">{{ item.title }}</div>
              <div class="card-meta">{{ item.year }}</div>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="想看" name="want">
          <div v-if="loading" class="loading">加载中...</div>
          <div v-else-if="!wantItems.length" class="empty">暂无想看，点击详情页「想看」按钮添加</div>
          <div v-else class="media-grid">
            <div
              v-for="item in wantItems"
              :key="item.media_id"
              class="card"
              @click="openDetail(item)"
            >
              <div class="poster-card">
                <img v-if="item.poster_url" :src="item.poster_url" :alt="item.title" loading="lazy" />
                <span v-else class="poster-placeholder">{{ (item.title || '').slice(0, 2) }}</span>
              </div>
              <div class="card-title">{{ item.title }}</div>
              <div class="card-meta">{{ item.year }}</div>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="收藏" name="favorites">
          <div v-if="loading" class="loading">加载中...</div>
          <div v-else-if="!favItems.length" class="empty">暂无收藏</div>
          <div v-else class="media-grid">
            <div
              v-for="item in favItems"
              :key="item.media_id"
              class="card"
              @click="$router.push(`/media/${item.media_id}`)"
            >
              <div class="poster-card">
                <img v-if="item.poster_url" :src="item.poster_url" :alt="item.title" loading="lazy" />
                <span v-else class="poster-placeholder">{{ (item.title || '').slice(0, 2) }}</span>
              </div>
              <div class="card-title">{{ item.title || item.media_id }}</div>
              <div class="card-meta">{{ item.year }}</div>
            </div>
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
      case 'favorites':
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
  } catch {
    // silently fail
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
.library-page {
  min-height: 100vh;
  background: var(--mh-bg);
  color: var(--mh-text);
}

.library-topbar {
  justify-content: space-between;
}

.page-title {
  font-size: 22px;
  font-weight: 700;
}

.back-btn {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid var(--mh-outline);
  color: var(--mh-text);
  padding: var(--mh-space-2) var(--mh-space-4);
  border-radius: 10px;
  cursor: pointer;
  font-weight: 500;

  &:hover { background: rgba(255, 255, 255, 0.1); }
}

.library-content {
  padding: calc(var(--mh-topbar-height) + var(--mh-space-4)) var(--mh-page-gutter) var(--mh-space-10);
}

.library-tabs {
  :deep(.el-tabs__header) { margin-bottom: var(--mh-space-6); }
  :deep(.el-tabs__item) {
    color: var(--mh-text-muted);
    &.is-active { color: var(--mh-primary); }
  }
  :deep(.el-tabs__active-bar) { background-color: var(--mh-primary); }
}

.media-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: var(--mh-space-5);
}

.card {
  cursor: pointer;
  transition: transform var(--mh-duration) var(--mh-ease-spring);
  &:hover {
    transform: translateY(-4px);
    .poster-card { box-shadow: var(--mh-shadow-lg); }
  }
}

.poster-card {
  position: relative;
  aspect-ratio: 2/3;
  background: linear-gradient(145deg, var(--mh-surface-variant), var(--mh-bg));
  border-radius: var(--mh-radius-md);
  border: 1px solid var(--mh-outline);
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.25);
  font-size: 28px;
  font-weight: 700;
  transition: box-shadow var(--mh-duration) var(--mh-ease);

  &.landscape { aspect-ratio: 16/9; }

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
  height: 3px;
  background: rgba(255, 255, 255, 0.2);
}

.progress-fill {
  height: 100%;
  background: var(--mh-primary);
  transition: width 0.3s;
}

.card-title {
  margin-top: var(--mh-space-2);
  font-size: 14px;
  font-weight: 500;
  color: var(--mh-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-meta {
  font-size: 12px;
  color: var(--mh-text-muted);
}

.loading, .empty {
  text-align: center;
  padding: 80px 0;
  font-size: 16px;
  color: var(--mh-text-muted);
}
</style>
