<template>
  <div class="search-page mh-page">
    <AppTopbar variant="sub">
      <template #center>
        <form class="search-form" @submit.prevent="search">
          <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <circle cx="11" cy="11" r="7" />
            <path d="M20 20l-3-3" stroke-linecap="round" />
          </svg>
          <input
            v-model="query"
            type="search"
            class="mh-search-field"
            placeholder="搜索电影、剧集、影人…"
            autofocus
            @input="onQueryInput"
          />
        </form>
      </template>
      <template #end>
        <el-select v-model="filterType" placeholder="类型" size="small" class="type-filter" @change="search">
          <el-option label="全部" value="" />
          <el-option label="电影" value="movie" />
          <el-option label="剧集" value="tvshow" />
          <el-option label="动画" value="anime" />
        </el-select>
      </template>
    </AppTopbar>

    <main class="results mh-page-body mh-animate-in">
      <LoadingState v-if="loading" message="搜索中…" />

      <EmptyState
        v-else-if="!hasResults && query.trim()"
        icon="🔍"
        title="没有找到结果"
        :description="`未找到与「${query}」相关的内容，试试换个关键词或缩短搜索词。`"
      />

      <EmptyState
        v-else-if="!query.trim()"
        icon="✨"
        title="开始搜索"
        description="输入片名、演员或关键词，我们会同时搜索作品与影人。"
      />

      <template v-else>
        <section v-if="personResults.length" class="section">
          <h2 class="mh-section-title">影人</h2>
          <div class="person-grid">
            <button
              v-for="p in personResults"
              :key="p.person_id"
              type="button"
              class="person-card"
              @click="$router.push(`/person/${p.person_id}`)"
            >
              <div class="person-avatar">
                <img v-if="p.profile_url" :src="p.profile_url" :alt="p.name" loading="lazy" />
                <span v-else class="avatar-placeholder">{{ p.name.slice(0, 1) }}</span>
              </div>
              <div class="person-name">{{ p.name }}</div>
              <div v-if="p.known_for" class="person-dept">{{ deptLabel(p.known_for) }}</div>
            </button>
          </div>
        </section>

        <section v-if="mediaResults.length" class="section">
          <h2 class="mh-section-title">作品</h2>
          <div class="mh-media-grid">
            <MediaPosterCard
              v-for="m in mediaResults"
              :key="m.id"
              :title="m.title"
              :poster-url="m.poster_url"
              :rating="m.rating"
              :subtitle="`${m.year || '—'} · ${typeLabel(m.type)}`"
              @click="$router.push(`/media/${m.id}`)"
            />
          </div>
        </section>
      </template>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { mediaApi, type MediaSummary, type PersonSearchResult } from '@/api'
import AppTopbar from '@/components/AppTopbar.vue'
import MediaPosterCard from '@/components/MediaPosterCard.vue'
import LoadingState from '@/components/LoadingState.vue'
import EmptyState from '@/components/EmptyState.vue'

const route = useRoute()
const query = ref('')
const filterType = ref('')
const loading = ref(false)
const mediaResults = ref<MediaSummary[]>([])
const personResults = ref<PersonSearchResult[]>([])
let debounceTimer: ReturnType<typeof setTimeout> | null = null

const hasResults = computed(() => mediaResults.value.length > 0 || personResults.value.length > 0)

async function search() {
  if (!query.value.trim()) {
    mediaResults.value = []
    personResults.value = []
    return
  }
  loading.value = true
  try {
    if (filterType.value) {
      const res = await mediaApi.list({
        q: query.value,
        type: filterType.value,
        page_size: 30,
      })
      mediaResults.value = res.items
      personResults.value = []
    } else {
      const res = await mediaApi.searchAll(query.value, 30)
      mediaResults.value = res.media || []
      personResults.value = res.persons || []
    }
  } finally {
    loading.value = false
  }
}

function onQueryInput() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    search()
  }, 350)
}

function typeLabel(t: string) {
  return ({ movie: '电影', tvshow: '剧集', anime: '动画', documentary: '纪录片' } as Record<string, string>)[t] || t
}

function deptLabel(d: string) {
  return ({ Acting: '演员', Directing: '导演', Writing: '编剧', Production: '制片' } as Record<string, string>)[d] || d
}

onMounted(() => {
  if (route.query.q) {
    query.value = route.query.q as string
    search()
  }
})
</script>

<style lang="scss" scoped>
.search-form {
  position: relative;
  width: 100%;
  max-width: 560px;
  margin: 0 auto;

  .mh-search-field {
    padding-left: 40px;
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

.type-filter {
  width: 108px;
}

.results {
  padding-left: var(--mh-page-gutter);
  padding-right: var(--mh-page-gutter);
  padding-bottom: var(--mh-space-10);
}

.section {
  margin-bottom: var(--mh-space-10);
}

.person-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: var(--mh-space-5);
}

.person-card {
  cursor: pointer;
  text-align: center;
  background: none;
  border: none;
  padding: var(--mh-space-2);
  color: inherit;
  border-radius: var(--mh-radius-md);
  transition: transform var(--mh-duration) var(--mh-ease-spring);

  &:hover {
    transform: translateY(-4px);
  }

  &:focus-visible {
    outline: 2px solid var(--mh-primary);
    outline-offset: 2px;
  }
}

.person-avatar {
  width: 96px;
  height: 96px;
  margin: 0 auto var(--mh-space-2);
  border-radius: var(--mh-radius-full);
  overflow: hidden;
  background: linear-gradient(145deg, var(--mh-surface-variant), var(--mh-bg-elevated));
  border: 2px solid var(--mh-outline);
  display: flex;
  align-items: center;
  justify-content: center;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.avatar-placeholder {
  font-size: 28px;
  font-weight: 700;
  color: var(--mh-text-muted);
}

.person-name {
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.person-dept {
  font-size: 12px;
  color: var(--mh-text-muted);
  margin-top: 2px;
}
</style>
