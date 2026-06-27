<template>
  <div class="live-list-page">
    <header class="topbar mh-topbar">
      <button class="back-btn" @click="$router.push('/')">← 首页</button>
      <h1 class="page-title">直播间</h1>
    </header>

    <main class="content">
      <LoadingState v-if="loading" message="加载直播间…" />

      <template v-else>
        <div v-if="groups.length > 0 || pushCount > 0" class="filters">
          <div class="group-tabs">
            <button
              class="tab"
              :class="{ active: activeGroup === '' }"
              @click="activeGroup = ''"
            >
              全部 ({{ totalCount }})
            </button>
            <button
              v-if="pushCount > 0"
              class="tab"
              :class="{ active: activeGroup === '__push__' }"
              @click="activeGroup = '__push__'"
            >
              推流 ({{ pushCount }})
            </button>
            <button
              v-for="g in groups"
              :key="g.name"
              class="tab"
              :class="{ active: activeGroup === g.name }"
              @click="activeGroup = g.name"
            >
              {{ g.name }} ({{ g.count }})
            </button>
          </div>
          <input
            v-model="search"
            class="search-input"
            type="search"
            placeholder="搜索频道…"
          />
        </div>

        <div v-if="filteredRooms.length === 0" class="empty">
          <p>{{ search ? '没有匹配的频道' : '暂无直播间' }}</p>
          <p v-if="!search" class="hint">管理员可在 CMS 后台创建直播间或导入 M3U 列表</p>
        </div>

        <div v-else class="room-grid">
          <article
            v-for="room in filteredRooms"
            :key="room.id"
            class="room-card"
            @click="$router.push(`/live/${room.id}`)"
          >
            <div class="room-cover" :style="coverStyle(room)">
              <span v-if="room.status === 'live'" class="live-badge">LIVE</span>
              <span v-else-if="room.status === 'idle'" class="idle-badge">
                {{ room.room_type === 'iptv' ? 'IPTV' : '待开播' }}
              </span>
            </div>
            <div class="room-info">
              <h3>{{ room.title }}</h3>
              <p v-if="room.description && !room.playlist_url" class="desc">{{ room.description }}</p>
              <div class="meta">
                <span v-if="room.group_title" class="group">{{ room.group_title }}</span>
                <span v-if="room.status === 'live' && room.started_at">
                  开播于 {{ formatTime(room.started_at) }}
                </span>
              </div>
            </div>
          </article>
        </div>
      </template>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import LoadingState from '@/components/LoadingState.vue'
import { liveApi, type LiveRoom, type LiveGroupStat } from '@/api'

const loading = ref(true)
const rooms = ref<LiveRoom[]>([])
const groups = ref<LiveGroupStat[]>([])
const activeGroup = ref('')
const search = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

const pushCount = computed(() =>
  rooms.value.filter((r) => r.room_type !== 'iptv').length,
)

const totalCount = computed(() => rooms.value.length)

const filteredRooms = computed(() => {
  let list = rooms.value
  if (activeGroup.value === '__push__') {
    list = list.filter((r) => r.room_type !== 'iptv')
  } else if (activeGroup.value) {
    const g = activeGroup.value
    list = list.filter((r) => (r.group_title || '未分组') === g)
  }
  const q = search.value.trim().toLowerCase()
  if (q) {
    list = list.filter((r) => r.title.toLowerCase().includes(q))
  }
  return [...list].sort((a, b) => {
    const order = { live: 0, idle: 1, ended: 2 }
    const sd = (order[a.status] ?? 9) - (order[b.status] ?? 9)
    if (sd !== 0) return sd
    return a.title.localeCompare(b.title, 'zh-CN')
  })
})

function coverStyle(room: LiveRoom) {
  if (room.cover_url) {
    return { backgroundImage: `url(${room.cover_url})` }
  }
  return {}
}

function formatTime(iso: string) {
  return new Date(iso).toLocaleString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

async function load() {
  try {
    const [roomList, groupList] = await Promise.all([
      liveApi.list({ page_size: 500 }),
      liveApi.groups(),
    ])
    rooms.value = roomList
    groups.value = groupList.filter((g) => g.name !== '未分组' || g.count > 0)
  } catch (e) {
    console.error('加载直播间失败', e)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  load()
  pollTimer = setInterval(load, 30000)
})

onBeforeUnmount(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style lang="scss" scoped>
.live-list-page {
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

.back-btn {
  background: none;
  border: none;
  color: var(--mh-text-secondary);
  cursor: pointer;
  font-size: 14px;

  &:hover { color: var(--mh-text); }
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
}

.content {
  padding: 0 var(--mh-space-6) var(--mh-space-8);
  max-width: 1200px;
}

.filters {
  margin-bottom: var(--mh-space-5);
  display: flex;
  flex-direction: column;
  gap: var(--mh-space-3);
}

.group-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tab {
  background: var(--mh-surface);
  border: 1px solid transparent;
  color: var(--mh-text-secondary);
  padding: 6px 14px;
  border-radius: 20px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;

  &:hover {
    color: var(--mh-text);
    border-color: rgba(255,255,255,0.1);
  }

  &.active {
    background: var(--mh-primary, #6366f1);
    color: #fff;
  }
}

.search-input {
  max-width: 320px;
  padding: 8px 14px;
  border-radius: 8px;
  border: 1px solid rgba(255,255,255,0.1);
  background: var(--mh-surface);
  color: var(--mh-text);
  font-size: 14px;
  outline: none;

  &:focus {
    border-color: var(--mh-primary, #6366f1);
  }
}

.empty {
  text-align: center;
  padding: 80px 20px;
  color: var(--mh-text-secondary);

  .hint { font-size: 13px; margin-top: 8px; opacity: 0.7; }
}

.room-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--mh-space-5);
}

.room-card {
  background: var(--mh-surface);
  border-radius: var(--mh-radius);
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;

  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--mh-shadow-md);
  }
}

.room-cover {
  aspect-ratio: 16 / 9;
  background: linear-gradient(135deg, #1a1a2e, #16213e);
  background-size: cover;
  background-position: center;
  position: relative;
}

.live-badge {
  position: absolute;
  top: 12px;
  left: 12px;
  background: #e50914;
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 4px;
  letter-spacing: 0.05em;
  animation: pulse 1.5s infinite;
}

.idle-badge {
  position: absolute;
  top: 12px;
  left: 12px;
  background: rgba(255,255,255,0.15);
  color: #fff;
  font-size: 11px;
  padding: 3px 8px;
  border-radius: 4px;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.room-info {
  padding: var(--mh-space-4);

  h3 {
    margin: 0 0 6px;
    font-size: 16px;
    font-weight: 600;
  }

  .desc {
    margin: 0 0 8px;
    font-size: 13px;
    color: var(--mh-text-secondary);
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .meta {
    font-size: 12px;
    color: var(--mh-text-muted);
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .group {
    background: rgba(255,255,255,0.08);
    padding: 1px 6px;
    border-radius: 4px;
  }
}
</style>
