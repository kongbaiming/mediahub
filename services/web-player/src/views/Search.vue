<template>
  <div class="search-page">
    <header class="topbar">
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
  background: #0f172a;
  color: #e2e8f0;
}

.topbar {
  position: sticky;
  top: 0;
  background: rgba(15, 23, 42, 0.95);
  backdrop-filter: blur(20px);
  z-index: 10;
  display: flex;
  align-items: center;
  padding: 16px 40px;
  gap: 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.back-btn {
  background: rgba(255, 255, 255, 0.1);
  border: none;
  color: #fff;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;

  &:hover { background: rgba(255, 255, 255, 0.2); }
}

.search-box {
  flex: 1;
  max-width: 600px;

  input {
    width: 100%;
    height: 40px;
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    color: #fff;
    padding: 0 16px;
    font-size: 15px;
    outline: none;

    &::placeholder { color: #64748b; }

    &:focus {
      background: rgba(255, 255, 255, 0.12);
      border-color: #6366f1;
    }
  }
}

.results {
  padding: 32px 40px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 20px;
}

.card {
  cursor: pointer;
  transition: transform 0.2s;

  &:hover { transform: translateY(-4px); }
}

.poster-card {
  position: relative;
  aspect-ratio: 2/3;
  background: linear-gradient(135deg, #1e293b, #0f172a);
  border-radius: 8px;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.3);
  font-size: 28px;
  font-weight: 700;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
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

.card-title {
  margin-top: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #e2e8f0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-meta {
  font-size: 12px;
  color: #94a3b8;
}

.loading, .empty {
  text-align: center;
  padding: 80px 0;
  font-size: 16px;
  color: #94a3b8;
}
</style>
