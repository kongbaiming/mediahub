<template>
  <div class="search-page">
    <header class="topbar mh-topbar">
      <button class="back-btn" @click="$router.back()">← 返回</button>
      <div class="search-box">
        <input
          v-model="query"
          type="text"
          placeholder="搜索电影 / 剧集..."
          autofocus
          @keyup.enter="search"
        />
      </div>
      <el-select v-model="filterType" placeholder="类型" size="small" style="width: 100px" @change="search">
        <el-option label="全部" value="" />
        <el-option label="电影" value="movie" />
        <el-option label="剧集" value="tvshow" />
        <el-option label="动画" value="anime" />
      </el-select>
    </header>

    <main class="results">
      <div v-if="loading" class="loading">搜索中...</div>
      <div v-else-if="!results.length && query" class="empty">
        没有找到「{{ query }}」相关结果
      </div>
      <div v-else-if="!query" class="empty">
        输入关键字开始搜索
      </div>
      <div v-else class="grid">
        <div
          v-for="m in results"
          :key="m.id"
          class="card"
          @click="$router.push(`/media/${m.id}`)"
        >
          <div class="poster-card">
            <img v-if="m.poster_url" :src="m.poster_url" :alt="m.title" loading="lazy" />
            <span v-else class="poster-placeholder">{{ m.title.slice(0, 2) }}</span>
            <div v-if="m.rating > 0" class="rating">⭐ {{ m.rating.toFixed(1) }}</div>
          </div>
          <div class="card-title">{{ m.title }}</div>
          <div class="card-meta">{{ m.year }} · {{ typeLabel(m.type) }}</div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { mediaApi, type MediaSummary } from '@/api'

const route = useRoute()
const query = ref('')
const filterType = ref('')
const loading = ref(false)
const results = ref<MediaSummary[]>([])

async function search() {
  if (!query.value.trim()) {
    results.value = []
    return
  }
  loading.value = true
  try {
    const res = await mediaApi.list({
      q: query.value,
      type: filterType.value,
      page_size: 30,
    })
    results.value = res.items
  } finally {
    loading.value = false
  }
}

function typeLabel(t: string) {
  return ({ movie: '电影', tvshow: '剧集', anime: '动画', documentary: '纪录片' } as any)[t] || t
}

onMounted(() => {
  if (route.query.q) {
    query.value = route.query.q as string
    search()
  }
})
</script>

<style lang="scss" scoped>
.search-page {
  min-height: 100vh;
  background: var(--mh-bg);
  color: var(--mh-text);
}

.topbar {
  position: sticky;
  top: 0;
  padding: var(--mh-space-4) clamp(var(--mh-space-4), 4vw, var(--mh-space-10));
}

.back-btn {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid var(--mh-outline);
  color: var(--mh-text);
  padding: var(--mh-space-2) var(--mh-space-4);
  border-radius: 10px;
  cursor: pointer;
  font-weight: 500;
  transition: background var(--mh-duration) var(--mh-ease);

  &:hover {
    background: rgba(255, 255, 255, 0.1);
  }
}

.search-box {
  flex: 1;
  max-width: 600px;

  input {
    width: 100%;
    height: 40px;
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid var(--mh-outline);
    border-radius: 10px;
    color: var(--mh-text);
    padding: 0 var(--mh-space-4);
    font-size: 15px;
    outline: none;
    transition: border-color var(--mh-duration) var(--mh-ease),
                box-shadow var(--mh-duration) var(--mh-ease);

    &::placeholder {
      color: var(--mh-text-muted);
    }

    &:focus {
      border-color: var(--mh-primary);
      box-shadow: 0 0 0 3px var(--mh-primary-muted);
    }
  }
}

.results {
  padding: var(--mh-space-8) clamp(var(--mh-space-4), 4vw, var(--mh-space-10));
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: var(--mh-space-5);
}

.card {
  cursor: pointer;
  transition: transform var(--mh-duration) var(--mh-ease-spring);

  &:hover {
    transform: translateY(-4px);

    .poster-card {
      box-shadow: var(--mh-shadow-lg);
    }
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

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
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
