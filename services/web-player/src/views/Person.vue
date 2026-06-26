<template>
  <div v-loading="loading" class="person-page">
    <header class="topbar">
      <button class="back-btn" @click="$router.back()">← 返回</button>
      <span class="breadcrumb">
        <span class="link" @click="$router.push('/')">首页</span>
        <template v-if="fromMediaTitle">
          <span class="sep">/</span>
          <span class="link" @click="goFromMedia">{{ fromMediaTitle }}</span>
        </template>
        <span class="sep">/</span>
        <span>{{ person?.name || '影人' }}</span>
      </span>
    </header>

    <main v-if="person" class="content">
      <section class="profile">
        <div class="avatar">
          <img v-if="avatarUrl" :src="avatarUrl" :alt="person.name" />
          <span v-else>{{ person.name?.slice(0, 1) || '?' }}</span>
        </div>
        <div class="profile-meta">
          <h1 class="name">{{ person.name }}</h1>
          <p v-if="roleLabel" class="role">{{ roleLabel }}</p>
          <p v-if="person.known_for_department" class="dept">{{ person.known_for_department }}</p>
          <p v-if="extraMeta" class="extra">{{ extraMeta }}</p>
          <p v-if="person.biography" class="bio">{{ person.biography }}</p>
          <p v-else-if="!loading" class="empty">暂无人物介绍</p>
        </div>
      </section>

      <section class="section">
        <h2 class="section-title">参演作品</h2>
        <div v-if="works.length" class="works-row">
          <button
            v-for="work in works"
            :key="work.id || `tmdb-${work.tmdb_id}`"
            type="button"
            class="work-card"
            @click="openWork(work)"
          >
            <div class="poster-card">
              <img v-if="work.poster_url" :src="work.poster_url" :alt="work.title" loading="lazy" />
              <span v-else>{{ work.title.slice(0, 2) }}</span>
              <div v-if="work.rating > 0" class="rating">⭐ {{ work.rating.toFixed(1) }}</div>
            </div>
            <div class="card-title">{{ work.title }}</div>
            <div class="card-meta">{{ work.year || '—' }} · {{ typeLabel(work.type) }}</div>
          </button>
        </div>
        <p v-else-if="!loading" class="empty">暂无参演作品</p>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { catalogApi, mediaApi, historyApi, type Person, type PersonWork } from '@/api'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const person = ref<Person | null>(null)
const works = ref<PersonWork[]>([])
const fromMediaTitle = ref('')

const fromMediaId = computed(() => (route.query.from as string) || '')
const fromTmdbType = computed(() => (route.query.from_tmdb_type as string) || '')
const fromTmdbId = computed(() => (route.query.from_tmdb_id as string) || '')
const roleName = computed(() => (route.query.role as string) || '')

const avatarUrl = computed(() => {
  const p = person.value
  if (!p) return ''
  if (p.profile_url) return p.profile_url
  if (p.profile_path) {
    if (p.profile_path.startsWith('http')) return p.profile_path
    return `https://image.tmdb.org/t/p/w300${p.profile_path}`
  }
  return ''
})

const roleLabel = computed(() => (roleName.value ? `饰 ${roleName.value}` : ''))

const extraMeta = computed(() => {
  const p = person.value
  if (!p) return ''
  const parts: string[] = []
  if (p.place_of_birth) parts.push(p.place_of_birth)
  if (p.birthday) parts.push(p.birthday.slice(0, 10))
  return parts.join(' · ')
})

async function load() {
  loading.value = true
  try {
    await historyApi.ensureProfileId()
    let personId: string
    if (route.name === 'person-tmdb') {
      const p = await catalogApi.personByTmdb(Number(route.params.tmdbId))
      person.value = p
      personId = p.id
    } else {
      personId = route.params.id as string
      person.value = await catalogApi.person(personId)
    }
    const list = await catalogApi.personWorks(personId, {
      excludeMediaId: fromMediaId.value || undefined,
    })
    works.value = list.filter((w) => {
      if (fromTmdbId.value && w.tmdb_id === Number(fromTmdbId.value)) return false
      return true
    })

    if (fromMediaId.value) {
      mediaApi.get(fromMediaId.value).then((m) => {
        fromMediaTitle.value = m.title
      }).catch(() => {
        fromMediaTitle.value = ''
      })
    } else if (fromTmdbType.value && fromTmdbId.value) {
      mediaApi.getTmdb(fromTmdbType.value, Number(fromTmdbId.value)).then((m) => {
        fromMediaTitle.value = m.title
      }).catch(() => {
        fromMediaTitle.value = ''
      })
    } else {
      fromMediaTitle.value = ''
    }
  } finally {
    loading.value = false
  }
}

function isPlayableWork(work: PersonWork) {
  return !work.external && !!work.id && work.id !== '00000000-0000-0000-0000-000000000000'
}

function openWork(work: PersonWork) {
  if (isPlayableWork(work)) {
    router.push(`/media/${work.id}`)
    return
  }
  if (work.tmdb_id && work.type) {
    router.push(`/media/tmdb/${work.type}/${work.tmdb_id}`)
  }
}

function goFromMedia() {
  if (fromMediaId.value) {
    router.push(`/media/${fromMediaId.value}`)
  } else if (fromTmdbType.value && fromTmdbId.value) {
    router.push(`/media/tmdb/${fromTmdbType.value}/${fromTmdbId.value}`)
  }
}

function typeLabel(t: string) {
  return ({ movie: '电影', tvshow: '剧集', anime: '动画', documentary: '纪录片' } as any)[t] || t
}

watch(
  () => [route.name, route.params.id, route.params.tmdbId, route.query.from, route.query.role],
  (cur, prev) => {
    if (!cur[0]) return
    if (prev && cur.every((v, i) => v === prev[i])) return
    window.scrollTo({ top: 0, behavior: 'instant' })
    load()
  },
)

onMounted(load)
</script>

<style lang="scss" scoped>
.person-page {
  min-height: 100vh;
  background: var(--mh-bg, #0a0a12);
  color: var(--mh-text, #f0f0f5);
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
  gap: 16px;
}

.back-btn {
  background: rgba(255, 255, 255, 0.1);
  border: none;
  color: #fff;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;

  &:hover { background: rgba(255, 255, 255, 0.2); }
}

.breadcrumb {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #94a3b8;
  min-width: 0;

  .link {
    cursor: pointer;
    color: #cbd5e1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 240px;

    &:hover { color: #fff; }
  }

  .sep { color: #475569; flex-shrink: 0; }

  > span:last-child {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

.content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 84px 40px 80px;
}

.profile {
  display: flex;
  gap: 32px;
  margin-bottom: 48px;
  align-items: flex-start;
}

.avatar {
  flex-shrink: 0;
  width: 180px;
  height: 180px;
  border-radius: 9999px;
  overflow: hidden;
  background: var(--mh-surface-variant, #1e1e2e);
  border: 2px solid rgba(255, 255, 255, 0.08);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 56px;
  font-weight: 700;
  color: var(--mh-text-muted, #6b6b80);

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.profile-meta {
  flex: 1;
  min-width: 0;
}

.name {
  margin: 0 0 8px;
  font-size: 36px;
  font-weight: 800;
}

.role,
.dept,
.extra {
  margin: 0 0 6px;
  font-size: 14px;
  color: var(--mh-text-muted, #94a3b8);
}

.role {
  color: #c4bfff;
}

.bio {
  margin: 20px 0 0;
  font-size: 15px;
  line-height: 1.75;
  color: #d8d8e8;
  white-space: pre-wrap;
}

.empty {
  margin: 0;
  font-size: 14px;
  color: var(--mh-text-muted, #6b6b80);
}

.section-title {
  font-size: 22px;
  font-weight: 600;
  margin: 0 0 20px;
}

.works-row {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 16px;
}

.work-card {
  border: none;
  background: transparent;
  padding: 0;
  text-align: left;
  color: inherit;
  font: inherit;
  cursor: pointer;
  transition: transform 0.2s;

  &:hover {
    transform: translateY(-4px);
  }
}

.poster-card {
  aspect-ratio: 2/3;
  background: linear-gradient(135deg, #1e293b, #0f172a);
  border-radius: 8px;
  overflow: hidden;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.3);
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 8px;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.rating {
  position: absolute;
  top: 6px;
  left: 6px;
  background: rgba(0, 0, 0, 0.7);
  color: #fbbf24;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.card-title {
  font-size: 13px;
  font-weight: 500;
  color: #e2e8f0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-meta {
  font-size: 11px;
  color: #94a3b8;
}

@media (max-width: 768px) {
  .content {
    padding: 76px 20px 48px;
  }

  .profile {
    flex-direction: column;
    align-items: center;
    text-align: center;
  }

  .avatar {
    width: 140px;
    height: 140px;
    font-size: 44px;
  }

  .name {
    font-size: 28px;
  }
}
</style>
