<template>
  <div class="person-page">
    <header class="person-topbar mh-topbar mh-sub-topbar">
      <button class="back-btn" @click="$router.back()">← 返回</button>
    </header>

    <main class="person-content" v-if="person">
      <div class="person-hero">
        <div class="person-avatar-large">
          <img v-if="person.profile_url" :src="person.profile_url" :alt="person.name" />
          <span v-else class="avatar-placeholder">{{ person.name.slice(0, 1) }}</span>
        </div>
        <div class="person-info">
          <h1 class="person-name">{{ person.name }}</h1>
          <div v-if="person.known_for_department" class="person-dept">
            {{ deptLabel(person.known_for_department) }}
          </div>
          <div v-if="person.birthday" class="person-meta">
            {{ person.birthday }}
            <span v-if="person.place_of_birth"> · {{ person.place_of_birth }}</span>
          </div>
          <p v-if="person.biography" class="person-bio">{{ person.biography }}</p>
        </div>
      </div>

      <section v-if="works.length" class="works-section">
        <h2 class="section-title">参演作品</h2>
        <div class="works-grid">
          <div
            v-for="w in works"
            :key="w.id"
            class="card"
            @click="$router.push(`/media/${w.id}`)"
          >
            <div class="poster-card">
              <img v-if="w.poster_url" :src="w.poster_url" :alt="w.title" loading="lazy" />
              <span v-else class="poster-placeholder">{{ (w.title || '').slice(0, 2) }}</span>
              <div v-if="w.rating > 0" class="rating">⭐ {{ w.rating.toFixed(1) }}</div>
            </div>
            <div class="card-title">{{ w.title }}</div>
            <div class="card-meta">{{ w.year }}</div>
          </div>
        </div>
      </section>
    </main>

    <div v-else-if="loading" class="loading">加载中...</div>
    <div v-else class="loading">影人不存在</div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { catalogApi, type PersonBrief, type MediaSummary } from '@/api'

const route = useRoute()
const loading = ref(true)
const person = ref<PersonBrief | null>(null)
const works = ref<MediaSummary[]>([])

function deptLabel(d: string) {
  return ({ Acting: '演员', Directing: '导演', Writing: '编剧', Production: '制片' } as Record<string, string>)[d] || d
}

onMounted(async () => {
  const id = route.params.id as string
  try {
    person.value = await catalogApi.person(id)
    works.value = await catalogApi.personWorks(id)
  } catch {
    // not found
  } finally {
    loading.value = false
  }
})
</script>

<style lang="scss" scoped>
.person-page {
  min-height: 100vh;
  background: var(--mh-bg);
  color: var(--mh-text);
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

.person-content {
  padding: calc(var(--mh-topbar-height) + var(--mh-space-6)) var(--mh-page-gutter) var(--mh-space-10);
}

.person-hero {
  display: flex;
  gap: var(--mh-space-8);
  margin-bottom: var(--mh-space-10);
}

.person-avatar-large {
  width: 180px;
  height: 180px;
  flex-shrink: 0;
  border-radius: 50%;
  overflow: hidden;
  background: linear-gradient(145deg, var(--mh-surface-variant), var(--mh-bg));
  border: 3px solid var(--mh-outline);
  display: flex;
  align-items: center;
  justify-content: center;

  img { width: 100%; height: 100%; object-fit: cover; }
}

.avatar-placeholder {
  font-size: 64px;
  font-weight: 700;
  color: var(--mh-text-muted);
}

.person-info { flex: 1; }

.person-name {
  font-size: 28px;
  font-weight: 700;
  margin-bottom: var(--mh-space-2);
}

.person-dept {
  font-size: 14px;
  color: var(--mh-primary);
  margin-bottom: var(--mh-space-2);
}

.person-meta {
  font-size: 14px;
  color: var(--mh-text-muted);
  margin-bottom: var(--mh-space-4);
}

.person-bio {
  font-size: 14px;
  line-height: 1.7;
  color: var(--mh-text-secondary);
}

.section-title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: var(--mh-space-4);
}

.works-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: var(--mh-space-4);
}

.card {
  cursor: pointer;
  transition: transform var(--mh-duration) var(--mh-ease-spring);
  &:hover { transform: translateY(-4px); }
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
  img { width: 100%; height: 100%; object-fit: cover; }
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
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-meta {
  font-size: 12px;
  color: var(--mh-text-muted);
}

.loading {
  text-align: center;
  padding: 80px 0;
  font-size: 16px;
  color: var(--mh-text-muted);
}
</style>
